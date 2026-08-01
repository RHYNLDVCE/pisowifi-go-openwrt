package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/network"
	"pisowifi/internal/services"
	"pisowifi/internal/state"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// RegisterPortalRoutes mounts all captive-portal and user-facing routes.
func RegisterPortalRoutes(app *fiber.App) {
	// --- Portal UI ---
	app.Get("/", portalHome)
	app.Get("/status", portalStatus)
	app.Get("/rewards", rewardsPage)

	// --- Session actions ---
	app.Post("/connect", connectUser)
	app.Post("/pause", pauseUser)
	app.Post("/enable_slot", enableSlot)
	app.Post("/cancel_slot", cancelSlot)
	app.Post("/claim_free_time", claimFreeTime)
	app.Post("/redeem_points", redeemPoints)
	app.Post("/generate_voucher", generateVoucher)
	app.Post("/redeem_voucher", redeemVoucher)

	// --- WebSocket ---
	app.Get("/ws/:mac", websocket.New(portalWS))

	// --- Captive portal detection endpoints ---
	// All of these return a redirect to the portal homepage.
	// The portal identifies the user from c.IP() — no params needed.
	portalRedirect := func(c *fiber.Ctx) error {
		return c.Redirect("http://10.0.0.1/", fiber.StatusFound)
	}
	app.Get("/generate_204", portalRedirect)
	app.Get("/gen_204", portalRedirect)
	app.Get("/ncsi.txt", portalRedirect)
	app.Get("/connecttest.txt", portalRedirect)
	app.Get("/hotspot-detect.html", portalRedirect)
	app.Get("/library/test/success.html", portalRedirect)
	app.Get("/redirect", portalRedirect)

	// Catch-all — must be last
	app.Get("/*", portalRedirect)
}

// ---------------------------------------------------------------------------
// Home
// ---------------------------------------------------------------------------

func portalHome(c *fiber.Ctx) error {
	// Force the URL in the browser to be 10.0.0.1
	host := c.Hostname()
	if host != "10.0.0.1" && host != "10.0.0.2" {
		return c.Redirect("http://10.0.0.1/", fiber.StatusFound)
	}

	// Identify client from source IP — same method as the old system.
	// c.IP() returns the real client IP because the router no longer masquerades
	// LAN→OrangePi traffic (hairpin masquerade was removed from nftables).
	clientIP := c.IP()
	clientMAC := network.GetMACByIP(clientIP)

	if clientMAC == "" {
		clientMAC = c.Query("mac")
	}
	clientMAC = strings.ToLower(strings.TrimSpace(clientMAC))

	if clientMAC == "" {
		// Ask the router to dump its ARP table, just in case we missed it
		network.RequestARPSync()

		// ARP cache miss - serve a loading page that refreshes to allow time for ARP sync
		c.Set("Refresh", "2; url=http://10.0.0.1/")
		return c.Type("html").SendString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="refresh" content="2;url=http://10.0.0.1/">
    <title>Loading Portal</title>
    <style>
        body { margin: 0; padding: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100vh; background-color: #0f172a; color: #f8fafc; font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; }
        .spinner { width: 44px; height: 44px; border: 4px solid rgba(255, 255, 255, 0.1); border-left-color: #3b82f6; border-radius: 50%; animation: spin 1s linear infinite; margin-bottom: 24px; }
        h2 { margin: 0 0 8px 0; font-size: 22px; font-weight: 600; letter-spacing: 0.5px; }
        p { margin: 0; font-size: 14px; color: #94a3b8; }
        @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
    </style>
</head>
<body>
    <div class="spinner"></div>
    <h2>Loading Portal...</h2>
    <p>Preparing your connection</p>
    <script>setTimeout(function(){window.location.reload();}, 2000);</script>
</body>
</html>`)
	}

	if _, ok := state.Users.Get(clientMAC); !ok {
		state.Users.Set(clientMAC, &state.UserRecord{
			Status: "new", Time: 0, Balance: 0, FreeClaimed: 0, Points: 0,
		})
	}
	state.Users.UpdateField(clientMAC, func(u *state.UserRecord) {
		u.IP = clientIP
		u.LastActive = float64(time.Now().UnixNano()) / 1e9
	})

	// Banners
	cfg := config.Get()
	banners := buildBannerList(cfg.BannerOrder)

	user := &state.UserRecord{}
	if u, ok := state.Users.Get(clientMAC); ok {
		user = u
	}

	s_insert := cfg.SoundInsert
	if s_insert == "" {
		s_insert = "insert_coin_sound.mp3"
	}
	s_coin := cfg.SoundCoin
	if s_coin == "" {
		s_coin = "coin-recieved.mp3"
	}

	coinPointMapJSON, _ := json.Marshal(cfg.CoinPointMap)
	pointsEnabledJSON := "false"
	if cfg.PointsEnabled {
		pointsEnabledJSON = "true"
	}
	fHours := cfg.FreeTimeDuration / 60
	fMins := cfg.FreeTimeDuration % 60
	freeTimeFormatted := fmt.Sprintf("%d:%02d", fHours, fMins)

	return c.Render("index", fiber.Map{
		"mac":                   clientMAC,
		"ip":                    clientIP,
		"time":                  user.Time,
		"status":                user.Status,
		"balance":               user.Balance,
		"points":                user.Points,
		"banners":               banners,
		"banner_text":           cfg.BannerText,
		"banner_link":           cfg.BannerLink,
		"portal_title":          cfg.PortalTitle,
		"portal_title_color":    cfg.PortalTitleColor,
		"portal_title_size":     cfg.PortalTitleSize,
		"portal_subtitle":       cfg.PortalSubtitle,
		"portal_subtitle_size":  cfg.PortalSubtitleSize,
		"coin_rates":            cfg.CoinRates,
		"points_enabled":        cfg.PointsEnabled,
		"voucher_enabled":       cfg.VoucherEnabled,
		"points_enabled_json":   pointsEnabledJSON,
		"coin_point_map":        cfg.CoinPointMap,
		"coin_point_map_json":   string(coinPointMapJSON),
		"free_time_enabled":     cfg.FreeTimeEnabled,
		"free_claimed":          user.FreeClaimed == 1,
		"free_duration":         cfg.FreeTimeDuration,
		"free_time_formatted":   freeTimeFormatted,
		"sound_insert_url":      fmt.Sprintf("/static/sounds/%s", s_insert),
		"sound_coin_url":        fmt.Sprintf("/static/sounds/%s", s_coin),
	})
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

func portalStatus(c *fiber.Ctx) error {
	// Derive MAC from real client IP — same as portalHome.
	clientIP := c.IP()
	clientMAC := network.GetMACByIP(clientIP)
	if clientMAC == "" {
		// Fallback: accept ?mac= from JS for users who loaded the page before
		// the ARP cache was populated (rare race on first connect).
		clientMAC = c.Query("mac")
	}
	clientMAC = strings.ToLower(strings.TrimSpace(clientMAC))
	cfg := config.Get()

	user := &state.UserRecord{Status: "new"}
	if u, ok := state.Users.Get(clientMAC); ok {
		user = u
	}
	if clientIP != "" && clientMAC != "" {
		state.Users.UpdateField(clientMAC, func(u *state.UserRecord) {
			u.IP = clientIP
		})
	}

	slotUser := state.GetSlotUser()
	isBusy := slotUser != "" && slotUser != clientMAC

	slotSecsLeft := 0
	if slotUser == clientMAC {
		left := cfg.SlotExpiryTimestamp - float64(time.Now().Unix())
		if left < 0 {
			left = 0
		}
		slotSecsLeft = int(left)
	}

	return c.JSON(fiber.Map{
		"time_remaining":   user.Time,
		"status":           user.Status,
		"balance":          user.Balance,
		"is_busy":          isBusy,
		"slot_seconds":     slotSecsLeft,
		"slot_max_seconds": cfg.SlotTimeout,
		"coin_rates":       cfg.CoinRates,
		"banner_text":      cfg.BannerText,
		"banner_link":      cfg.BannerLink,
		"points":           user.Points,
		"points_enabled":   cfg.PointsEnabled,
		"coin_point_map":   cfg.CoinPointMap,
	})
}

// ---------------------------------------------------------------------------
// WebSocket
// ---------------------------------------------------------------------------

func portalWS(c *websocket.Conn) {
	// The MAC in the URL (/ws/:mac) was rendered server-side by the template
	// (derived from the ARP cache), so it is already authoritative.
	// We additionally validate it against the current source IP as a sanity check.
	mac := strings.ToLower(strings.TrimSpace(c.Params("mac")))
	state.Manager.Connect(mac, c)
	defer state.Manager.Disconnect(mac)

	// Update stored IP from the WS connection's real source IP.
	var clientIP string
	if c.RemoteAddr() != nil {
		clientIP = strings.Split(c.RemoteAddr().String(), ":")[0]
	}
	if clientIP == "" || clientIP == "10.0.0.1" {
		// Fallback: derive from ARP cache if IP is empty or router's IP
		clientIP = network.GetIPByMAC(mac)
	}
	if clientIP != "" {
		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			u.IP = clientIP
			u.LastActive = float64(time.Now().UnixNano()) / 1e9
		})
	}

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			u.LastActive = float64(time.Now().UnixNano()) / 1e9
		})
	}
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

func connectUser(c *fiber.Ctx) error {
	mac := resolveMAC(c)
	if mac == "" {
		return c.JSON(fiber.Map{"result": "fail"})
	}
	result := services.ConnectUser(mac)
	return c.JSON(fiber.Map{"result": result})
}

func pauseUser(c *fiber.Ctx) error {
	mac := resolveMAC(c)
	if mac == "" {
		return c.JSON(fiber.Map{"result": "fail"})
	}
	result := services.PauseUser(mac)
	return c.JSON(fiber.Map{"result": result})
}

// ---------------------------------------------------------------------------
// Slot management
// ---------------------------------------------------------------------------

func enableSlot(c *fiber.Ctx) error {
	mac := resolveMAC(c)
	if mac == "" {
		return c.JSON(fiber.Map{"result": "fail"})
	}
	result, _, _, _, _ := services.EnableSlot(mac)
	return c.JSON(fiber.Map{"result": result})
}

func cancelSlot(c *fiber.Ctx) error {
	mac := resolveMAC(c)
	if mac == "" {
		return c.JSON(fiber.Map{"result": "fail"})
	}
	if services.CancelSlot(mac) {
		return c.JSON(fiber.Map{"result": "success"})
	}
	return c.JSON(fiber.Map{"result": "fail"})
}

// ---------------------------------------------------------------------------
// Free time
// ---------------------------------------------------------------------------

func claimFreeTime(c *fiber.Ctx) error {
	mac := resolveMAC(c)
	if mac == "" {
		return c.JSON(fiber.Map{"result": "fail"})
	}
	result := services.ClaimFreeTime(mac)
	return c.JSON(fiber.Map{"result": result})
}

// ---------------------------------------------------------------------------
// Rewards / Points
// ---------------------------------------------------------------------------

func rewardsPage(c *fiber.Ctx) error {
	mac := network.GetMACByIP(c.IP())
	if mac == "" {
		// Fallback: accept ?mac= in case of ARP cache miss
		mac = c.Query("mac")
	}
	mac = strings.ToLower(strings.TrimSpace(mac))
	if mac == "" {
		network.RequestARPSync()
		c.Set("Refresh", "2; url=/rewards")
		return c.Type("html").SendString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="refresh" content="2;url=/rewards">
    <title>Loading Rewards</title>
    <style>
        body { margin: 0; padding: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100vh; background-color: #0f172a; color: #f8fafc; font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; }
        .spinner { width: 44px; height: 44px; border: 4px solid rgba(255, 255, 255, 0.1); border-left-color: #10b981; border-radius: 50%; animation: spin 1s linear infinite; margin-bottom: 24px; }
        h2 { margin: 0 0 8px 0; font-size: 22px; font-weight: 600; letter-spacing: 0.5px; }
        p { margin: 0; font-size: 14px; color: #94a3b8; }
        @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
    </style>
</head>
<body>
    <div class="spinner"></div>
    <h2>Loading Rewards...</h2>
    <p>Preparing your account</p>
    <script>setTimeout(function(){window.location.reload();}, 2000);</script>
</body>
</html>`)
	}
	cfg := config.Get()

	if _, ok := state.Users.Get(mac); !ok {
		state.Users.Set(mac, &state.UserRecord{Status: "new"})
	}

	user := &state.UserRecord{}
	if u, ok := state.Users.Get(mac); ok {
		user = u
	}

	activeVouchers := db.GetActiveVouchersByUser(mac)

	return c.Render("rewards", fiber.Map{
		"mac":             mac,
		"points":          user.Points,
		"promos":          cfg.PointPromos,
		"enabled":         cfg.PointsEnabled,
		"voucher_enabled": cfg.VoucherEnabled,
		"banner_text":     cfg.BannerText,
		"banner_link":     cfg.BannerLink,
		"active_vouchers": activeVouchers,
		"voucher_point_promos": cfg.VoucherPointPromos,
	})
}

func redeemPoints(c *fiber.Ctx) error {
	var body struct {
		PromoID int64 `json:"promo_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": "Invalid body"})
	}

	mac := resolveMAC(c)
	if mac == "" {
		return c.JSON(fiber.Map{"status": "error", "message": "MAC address missing"})
	}
	// Pass the verified IP from the ARP cache (not the URL param) to services
	clientIP := network.GetIPByMAC(mac)

	status, msg := services.RedeemPoints(mac, clientIP, body.PromoID)
	return c.JSON(fiber.Map{"status": status, "message": msg})
}

// ---------------------------------------------------------------------------
// Vouchers
// ---------------------------------------------------------------------------

func generateVoucher(c *fiber.Ctx) error {
	var body struct {
		Type  string  `json:"type"`
		Value float64 `json:"value"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": "Invalid body"})
	}

	mac := resolveMAC(c)
	if mac == "" {
		return c.JSON(fiber.Map{"status": "error", "message": "MAC address missing"})
	}

	code, err := services.GenerateVoucher(mac, body.Type, body.Value)
	if err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "success", "code": code})
}

func redeemVoucher(c *fiber.Ctx) error {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": "Invalid body"})
	}

	mac := resolveMAC(c)
	if mac == "" {
		return c.JSON(fiber.Map{"status": "error", "message": "MAC address missing"})
	}

	err := services.RedeemVoucher(mac, body.Code)
	if err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "success", "message": "Voucher redeemed successfully!"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resolveMAC derives the authoritative MAC for the requesting client.
//
// Security model (identical to old on-router PisoWifi):
//   - Primary: c.IP() → router ARP cache (via MQTT-backed arpcache).
//     c.IP() returns the REAL client IP because the hairpin masquerade was
//     removed from nftables. The ARP cache is router-controlled and not
//     user-forgeable.
//   - Fallback: ?mac= param, only accepted when the ARP cache has no entry
//     yet (rare first-connect race before DHCP hook fires).
func resolveMAC(c *fiber.Ctx) string {
	// Primary: IP → ARP cache (authoritative)
	if mac := network.GetMACByIP(c.IP()); mac != "" {
		return strings.ToLower(strings.TrimSpace(mac))
	}
	// Fallback: accept ?mac= only for brand-new devices not yet in ARP cache
	return strings.ToLower(strings.TrimSpace(c.Query("mac")))
}

func buildBannerList(order []string) []string {
	bannerDir := "static/banners/set"
	entries, err := os.ReadDir(bannerDir)
	if err != nil {
		return []string{"/static/banners/default/banner_default.jpg"}
	}
	allFiles := map[string]bool{}
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
			allFiles[e.Name()] = true
		}
	}
	var banners []string
	for _, f := range order {
		if allFiles[f] {
			banners = append(banners, fmt.Sprintf("/static/banners/set/%s", f))
		}
	}
	for f := range allFiles {
		inList := false
		for _, b := range banners {
			if strings.HasSuffix(b, "/"+f) {
				inList = true
				break
			}
		}
		if !inList {
			banners = append(banners, fmt.Sprintf("/static/banners/set/%s", f))
		}
	}
	if len(banners) == 0 {
		banners = []string{"/static/banners/default/banner_default.jpg"}
	}
	return banners
}
