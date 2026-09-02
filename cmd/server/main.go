package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"pisowifi/internal/api"
	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/hardware"
	"pisowifi/internal/logger"
	"pisowifi/internal/mqtt"
	"pisowifi/internal/network"
	"pisowifi/internal/services"
	"pisowifi/internal/state"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/django/v3"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load .env (must happen before config reads env vars)
	if err := godotenv.Load(); err != nil {
		fmt.Println("[Main] .env not found — using system environment variables.")
	}

	// Re-initialize config package vars that depend on env (they were set at
	// package init time before .env was loaded, so we reload them here)
	config.ReloadSecrets()

	// 2. Init logger
	logger.Init()
	logger.SystemLog("Initializing PisoWifi System...")

	// 3. Load app config from config.json
	config.Load()

	// 4. Ensure required directories exist
	os.MkdirAll("static/banners/set", 0755)
	os.MkdirAll("static/sounds", 0755)

	// 5. Init database
	db.InitDB()

	// 6. Load users into in-memory state
	records := db.LoadUsers()
	for mac, rec := range records {
		// Reset any "connected" users to "paused" on startup
		status := rec.Status
		if status == "connected" {
			status = "paused"
			db.SyncUser(db.UserRecord{
				MAC: mac, IP: rec.IP, Time: rec.Time, Status: "paused",
				Balance: rec.Balance, FreeClaimed: rec.FreeClaimed, Points: rec.Points,
			})
		}
		state.Users.Set(mac, &state.UserRecord{
			IP:          rec.IP,
			Time:        rec.Time,
			Status:      status,
			Balance:     rec.Balance,
			FreeClaimed: rec.FreeClaimed,
			Points:      rec.Points,
			Hostname:    rec.Hostname,
		})
	}

	// 6b. Setup policy routing — forces port-80 HTTP replies from the Go server
	// to route through the OpenWrt router (10.0.0.1) even for same-subnet clients.
	// This is the counterpart to removing the hairpin masquerade rule in nftables:
	// the router's conntrack sees the return packet and applies reverse DNAT so
	// the browser sees the reply as coming from 10.0.0.1 (as expected).
	// Result: c.IP() in Fiber returns the real client IP, not 10.0.0.1.
	setupPolicyRouting()

	// 6c. Init MQTT client — connects to the OpenWrt router broker.
	// We pass network.InitFirewall so that the nftables rules are sent automatically
	// on startup, AND whenever the router reboots/reconnects.
	mqtt.Init(
		config.MQTTBroker,
		config.MQTTClientID,
		config.MQTTUsername,
		config.MQTTPassword,
		network.InitFirewall,
	)

	// 8. (conntrack flush is now handled by the router on init via MQTT)


	// 9. Init GPIO hardware
	hardware.Setup()

	// 10. Build Fiber app with HTML template engine
	// Go's html/template uses {{ .Variable }} — maps with string keys work naturally.
	engine := django.New("./templates", ".html")
	engine.Delims("{{", "}}")  // default, same as Jinja2

	app := fiber.New(fiber.Config{
		Views:                 engine,
		DisableStartupMessage: false,
		// Graceful shutdown is handled manually below
		StrictRouting: false,
		ServerHeader:  "PisoWifi",
	})

	app.Use(fiberrecover.New())
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Static files with browser caching
	app.Static("/static", "./static", fiber.Static{
		MaxAge:       86400,
		CacheControl: true,
	})
	app.Static("/admin/assets", "./admin-ui/dist/assets", fiber.Static{
		MaxAge:       3600 * 24 * 30, // 30 days for content-hashed assets
		CacheControl: true,
	})

	// Register all routes
	api.RegisterAdminRoutes(app)
	api.RegisterPortalRoutes(app) // portal catch-all must be last

	// 11. Start background goroutines
	services.StartBackgroundTasks()

	// 12. Graceful shutdown handler
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.SystemLog("System Shutting Down...")
		state.IsShuttingDown.Store(true)

		// Close all portal WebSocket connections
		state.Manager.CloseAll()

		// Turn off coin slot relay
		func() {
			defer func() { recover() }()
			hardware.TurnSlotOff()
		}()

		// Disconnect MQTT cleanly
		mqtt.Disconnect()

		// Wait for background tasks with a 3 second timeout
		waitCh := make(chan struct{})
		go func() {
			services.Wg.Wait()
			close(waitCh)
		}()
		select {
		case <-waitCh:
			logger.SystemLog("Background services exited cleanly.")
		case <-time.After(3 * time.Second):
			logger.SystemLog("Timeout waiting for background services. Forcing shutdown.")
		}

		// Run fail_safe.sh
		failSafePath := "fail_safe.sh"
		if _, err := os.Stat(failSafePath); err == nil {
			exec.Command(failSafePath).Run()
		}

		// Graceful HTTP shutdown
		app.Shutdown()

		// Safely close DB (triggers WAL checkpoint)
		db.CloseDB()
	}()

	// 13. Start HTTP server (blocks until shutdown)
	logger.SystemLog("PisoWifi System Started Successfully!")
	if err := app.Listen(":80"); err != nil {
		logger.SystemLog(fmt.Sprintf("Server error: %v", err))
	}
}

// setupPolicyRouting configures Linux policy routing on the Orange Pi so that
// HTTP replies from the portal (port 80) are always routed through the OpenWrt
// router, even when the client is on the same subnet as the Orange Pi.
//
// Why this is needed:
//   When a user requests http://10.0.0.1/, the router DNATs the packet to
//   10.0.0.2:80 WITHOUT masquerading the source IP (so c.IP() returns the real
//   client IP). On the return path, the OrangePi's kernel would normally try to
//   reach the client directly (same-subnet ARP), bypassing the router. The router
//   would never see the return packet and conntrack can't apply reverse DNAT,
//   so the client receives a reply from 10.0.0.2 instead of 10.0.0.1 and
//   the connection breaks. Policy routing fixes this by forcing all port-80
//   replies through the router, which applies reverse DNAT transparently.
func setupPolicyRouting() {
	gw := config.RouterIP

	cmds := [][]string{
		// Create routing table 80: all traffic in this table goes via the router
		{"ip", "route", "replace", "table", "80", "default", "via", gw},
		// Rule: packets marked 0x50 (80) use table 80
		{"ip", "rule", "add", "fwmark", "80", "table", "80", "priority", "100"},
	}
	for _, cmd := range cmds {
		exec.Command(cmd[0], cmd[1:]...).Run()
	}

	// Mark outgoing TCP packets from port 80 (portal replies) so they use table 80.
	// Check first to avoid duplicate rules on restart.
	checkCmd := exec.Command("iptables", "-t", "mangle", "-C", "OUTPUT",
		"-p", "tcp", "--sport", "80", "-j", "MARK", "--set-mark", "80")
	if checkCmd.Run() != nil {
		exec.Command("iptables", "-t", "mangle", "-A", "OUTPUT",
			"-p", "tcp", "--sport", "80", "-j", "MARK", "--set-mark", "80").Run()
	}

	logger.SystemLog(fmt.Sprintf("[ROUTING] Policy routing active: port-80 replies via %s", gw))
}
