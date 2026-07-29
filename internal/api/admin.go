package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/infrastructure"
	"pisowifi/internal/logger"
	"pisowifi/internal/network"
	"pisowifi/internal/security"
	"pisowifi/internal/services"
	"pisowifi/internal/state"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// RegisterAdminRoutes mounts all admin/auth routes behind the security middleware.
func RegisterAdminRoutes(app *fiber.App) {
	// Public auth routes
	app.Get("/login", loginPage)
	app.Post("/auth", loginAction)
	app.Get("/logout", logoutAction)

	// Protected admin group
	admin := app.Group("/admin", security.AdminMiddleware)
	admin.Get("/api/dashboard_data", getDashboardData)
	admin.Get("/system_stats", getSystemStats)
	admin.Get("/get_infrastructure_devices", getInfrastructureDevices)
	admin.Get("/api/logs", getLogsJSON)
	admin.Get("/get_restart_schedule", getRestartSchedule)
	admin.Post("/set_restart_schedule", setRestartSchedule)
	admin.Get("/get_points_config", getPointsConfig)
	admin.Post("/save_points_config", savePointsConfig)
	admin.Post("/update_coin_settings", updateCoinSettings)
	admin.Post("/update_network_settings", updateNetworkSettings)
	admin.Post("/update_session_settings", updateSessionSettings)
	admin.Post("/update_portal_settings", updatePortalSettings)
	admin.Post("/reboot", rebootDevice)
	admin.Post("/clear_banners", clearBanners)
	admin.Post("/upload_banners", uploadBanners)
	admin.Post("/save_banner_order", saveBannerOrder)
	admin.Post("/delete_banner", deleteBanner)
	admin.Post("/upload_sound", uploadSound)
	admin.Post("/delete_sound", deleteSound)
	admin.Get("/backup", downloadBackup)
	admin.Post("/restore", restoreBackup)
	admin.Post("/reset_settings", resetSettings)
	admin.Get("/api/user/:mac", getSingleUser)
	admin.Post("/manage_time", adminManageTime)
	admin.Post("/manage_points", adminManagePoints)
	admin.Post("/block", adminBlock)
	admin.Post("/unblock", adminUnblock)
	admin.Post("/delete_user", adminDeleteUser)
	admin.Post("/rename_device", renameDevice)
	admin.Post("/add_device", addDevice)
	admin.Post("/remove_device", removeDevice)
	admin.Get("/api/vouchers", getVouchers)
	admin.Post("/update_voucher_settings", updateVoucherSettings)

	// WebSockets (not under the group middleware — Fiber WS upgrade is separate)
	app.Get("/admin/ws/system_stats", websocket.New(wsSystemStats))
	app.Get("/admin/ws/logs", websocket.New(wsLogs))

	// Catch-all to serve the React SPA index.html for any remaining /admin sub-route
	admin.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./admin-ui/dist/index.html")
	})
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

func loginPage(c *fiber.Ctx) error {
	return c.Render("login", fiber.Map{
		"Error": c.Query("error"),
	})
}

func loginAction(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	clientIP := c.IP()
	clientMAC := infrastructure.GetMACFromIP(clientIP)

	if Limiter.IsBlocked(clientIP) {
		logger.AuditLog("SECURITY_ALERT", clientIP, clientMAC, fmt.Sprintf("Rate limit exceeded for account '%s'", username))
		return c.Redirect("/login?error=Too many attempts. Try again in 5 minutes.", fiber.StatusSeeOther)
	}

	if db.VerifyAdmin(username, password) {
		logger.AuditLog("LOGIN_SUCCESS", clientIP, clientMAC, fmt.Sprintf("Account '%s' authenticated.", username))
		token, err := security.CreateToken(username)
		if err != nil {
			return c.Status(500).SendString("Token error")
		}
		Limiter.Clear(clientIP)
		c.Cookie(&fiber.Cookie{
			Name:     "admin_token",
			Value:    token,
			HTTPOnly: true,
			SameSite: "Lax",
		})
		return c.Redirect("/admin", fiber.StatusSeeOther)
	}

	Limiter.RecordAttempt(clientIP)
	logger.AuditLog("LOGIN_FAILED", clientIP, clientMAC, fmt.Sprintf("Failed attempt for account '%s'.", username))
	return c.Redirect("/login?error=Invalid Credentials", fiber.StatusSeeOther)
}

func logoutAction(c *fiber.Ctx) error {
	clientIP := c.IP()
	clientMAC := infrastructure.GetMACFromIP(clientIP)
	logger.AuditLog("LOGOUT", clientIP, clientMAC, "Administrator logged out.")
	c.ClearCookie("admin_token")
	return c.Redirect("/login", fiber.StatusFound)
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

const itemsPerPage = 10

func getDashboardData(c *fiber.Ctx) error {
	search := c.Query("search", "")
	page := c.QueryInt("page", 1)
	sortBy := c.Query("sort", "status")

	cfg := config.Get()
	bannerFiles := infrastructure.GetBanners(cfg.BannerOrder)
	soundFiles := infrastructure.GetSounds()
	logResult := infrastructure.GetSystemLogs(200, 0, "", "")

	data := services.GetPaginatedUsers(search, page, sortBy, itemsPerPage)

	// Merge settings into the data map
	data["slot_timeout"] = cfg.SlotTimeout
	data["sqm_enabled"] = cfg.SQMEnabled
	data["sqm_upload_mbps"] = cfg.SQMUploadMbps
	data["sqm_download_mbps"] = cfg.SQMDownloadMbps
	data["inactive_timeout"] = cfg.InactiveTimeout
	data["auto_pause_enabled"] = cfg.AutoPauseEnabled
	data["speed_limit_enabled"] = cfg.SpeedLimitEnabled
	data["global_speed_limit"] = cfg.GlobalSpeedLimit
	data["gaming_mode_enabled"] = cfg.GamingModeEnabled
	data["udp_priority_enabled"] = cfg.UDPPriorityEnabled
	data["coin_rates"] = cfg.CoinRates
	data["banner_text"] = cfg.BannerText
	data["banner_link"] = cfg.BannerLink
	data["portal_title"] = cfg.PortalTitle
	data["portal_title_color"] = cfg.PortalTitleColor
	data["portal_title_size"] = cfg.PortalTitleSize
	data["portal_subtitle"] = cfg.PortalSubtitle
	data["portal_subtitle_size"] = cfg.PortalSubtitleSize
	data["open_nat_enabled"] = cfg.OpenNATEnabled
	data["custom_ttl"] = cfg.CustomTTL
	data["banner_files"] = bannerFiles
	data["free_time_enabled"] = cfg.FreeTimeEnabled
	data["free_time_duration"] = cfg.FreeTimeDuration
	data["sound_files"] = soundFiles
	data["sound_insert_selected"] = cfg.SoundInsert
	data["sound_coin_selected"] = cfg.SoundCoin
	data["system_logs"] = logResult.Logs

	return c.JSON(data)
}

// ---------------------------------------------------------------------------
// System Stats
// ---------------------------------------------------------------------------

func getSystemStats(c *fiber.Ctx) error {
	return c.JSON(infrastructure.GetSystemStats())
}

func wsSystemStats(c *websocket.Conn) {
	for {
		if state.IsShuttingDown.Load() {
			return
		}
		if err := c.WriteJSON(infrastructure.GetSystemStats()); err != nil {
			return
		}
		time.Sleep(3 * time.Second)
	}
}

// ---------------------------------------------------------------------------
// Logs
// ---------------------------------------------------------------------------

func getLogsJSON(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)
	logType := c.Query("log_type", "")
	query := c.Query("q", "")
	return c.JSON(infrastructure.GetSystemLogs(limit, offset, logType, query))
}

func wsLogs(c *websocket.Conn) {
	f, err := os.Open("system.log")
	if err != nil {
		return
	}
	defer f.Close()

	// Seek to end of file to only tail NEW logs
	f.Seek(0, 2)

	// Tail continuously
	for {
		if state.IsShuttingDown.Load() {
			return
		}
		buf := make([]byte, 4096)
		n, err := f.Read(buf)
		if n == 0 || err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, line := range strings.Split(string(buf[:n]), "\n") {
			if m := parseLogLine(line); m != nil {
				if err := c.WriteJSON(m); err != nil {
					return
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Infrastructure devices
// ---------------------------------------------------------------------------

func getInfrastructureDevices(c *fiber.Ctx) error {
	cfg := config.Get()
	activeMacs := map[string]bool{}
	state.Users.Range(func(mac string, u *state.UserRecord) { 
		if u.Status != "new" {
			activeMacs[mac] = true 
		}
	})
	devices := infrastructure.ScanInfrastructure(activeMacs, cfg.CustomDeviceNames, cfg.CustomDeviceIPs)
	return c.JSON(fiber.Map{"devices": devices})
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

func getRestartSchedule(c *fiber.Ctx) error {
	return c.JSON(config.Get().RestartSchedule)
}

func setRestartSchedule(c *fiber.Ctx) error {
	var body struct {
		Enabled bool   `json:"enabled"`
		Time    string `json:"time"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	config.Update(func(cfg *config.AppConfig) {
		cfg.RestartSchedule = config.RestartSchedule{Enabled: body.Enabled, Time: body.Time}
	})
	config.Save()
	clientIP := c.IP()
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP),
		fmt.Sprintf("Modified restart schedule to %s (Enabled: %v)", body.Time, body.Enabled))
	return c.JSON(fiber.Map{"status": "success", "message": "Schedule updated"})
}

func getPointsConfig(c *fiber.Ctx) error {
	cfg := config.Get()
	return c.JSON(fiber.Map{
		"enabled":  cfg.PointsEnabled,
		"coin_map": cfg.CoinPointMap,
		"promos":   cfg.PointPromos,
	})
}

func savePointsConfig(c *fiber.Ctx) error {
	var body struct {
		Enabled bool                 `json:"enabled"`
		CoinMap map[string]float64   `json:"coin_map"`
		Promos  []config.PromoItem   `json:"promos"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	config.Update(func(cfg *config.AppConfig) {
		cfg.PointsEnabled = body.Enabled
		cfg.CoinPointMap = body.CoinMap
		cfg.PointPromos = body.Promos
	})
	config.Save()
	clientIP := c.IP()
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP), "Modified global points configuration")
	return c.JSON(fiber.Map{"status": "success", "message": "Points configuration saved"})
}

func parseSettingsBody(c *fiber.Ctx) (map[string]interface{}, error) {
	var rawBody map[string]interface{}
	err := c.BodyParser(&rawBody)
	return rawBody, err
}

func parseSettingInt(val interface{}, def int) int {
	str := fmt.Sprintf("%v", val)
	var v int
	if _, err := fmt.Sscan(str, &v); err != nil {
		return def
	}
	return v
}

func updateCoinSettings(c *fiber.Ctx) error {
	clientIP := c.IP()
	rawBody, err := parseSettingsBody(c)
	if err != nil { return c.Status(400).JSON(fiber.Map{"status": "error", "message": err.Error()}) }

	config.Update(func(cfg *config.AppConfig) {
		if val, ok := rawBody["coin_rates"]; ok {
			cfg.CoinRates = fmt.Sprintf("%v", val)
		}
	})
	config.Save()
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP), "Updated coin pricing settings")
	return c.JSON(fiber.Map{"status": "success"})
}

func updateNetworkSettings(c *fiber.Ctx) error {
	clientIP := c.IP()
	rawBody, err := parseSettingsBody(c)
	if err != nil { return c.Status(400).JSON(fiber.Map{"status": "error", "message": err.Error()}) }

	config.Update(func(cfg *config.AppConfig) {
		if val, ok := rawBody["sqm_enabled"]; ok {
			cfg.SQMEnabled = fmt.Sprintf("%v", val) == "on"
		}
		if val, ok := rawBody["sqm_upload_mbps"]; ok {
			cfg.SQMUploadMbps = parseSettingInt(val, cfg.SQMUploadMbps)
		}
		if val, ok := rawBody["sqm_download_mbps"]; ok {
			cfg.SQMDownloadMbps = parseSettingInt(val, cfg.SQMDownloadMbps)
		}
		if val, ok := rawBody["speed_limit_val"]; ok {
			cfg.GlobalSpeedLimit = parseSettingInt(val, cfg.GlobalSpeedLimit)
		}
		if val, ok := rawBody["speed_limit_toggle"]; ok {
			cfg.SpeedLimitEnabled = fmt.Sprintf("%v", val) == "on"
		}
		if val, ok := rawBody["gaming_mode"]; ok {
			cfg.GamingModeEnabled = fmt.Sprintf("%v", val) == "on"
		}
		if val, ok := rawBody["udp_priority"]; ok {
			cfg.UDPPriorityEnabled = fmt.Sprintf("%v", val) == "on"
		}
		if val, ok := rawBody["open_nat"]; ok {
			cfg.OpenNATEnabled = fmt.Sprintf("%v", val) == "on"
		}
		if val, ok := rawBody["custom_ttl"]; ok {
			cfg.CustomTTL = parseSettingInt(val, cfg.CustomTTL)
		}
	})
	config.Save()
	network.ReloadFirewall()
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP), "Updated network settings")
	return c.JSON(fiber.Map{"status": "success"})
}

func updateSessionSettings(c *fiber.Ctx) error {
	clientIP := c.IP()
	rawBody, err := parseSettingsBody(c)
	if err != nil { return c.Status(400).JSON(fiber.Map{"status": "error", "message": err.Error()}) }

	config.Update(func(cfg *config.AppConfig) {
		if val, ok := rawBody["inactive_timeout"]; ok {
			cfg.InactiveTimeout = parseSettingInt(val, cfg.InactiveTimeout)
		}
		if val, ok := rawBody["auto_pause"]; ok {
			cfg.AutoPauseEnabled = fmt.Sprintf("%v", val) == "on"
		}
		if val, ok := rawBody["free_time_toggle"]; ok {
			newFreeEnabled := fmt.Sprintf("%v", val) == "on"
			if newFreeEnabled != cfg.FreeTimeEnabled {
				db.ResetAllFreeClaimed()
				state.Users.Range(func(mac string, u *state.UserRecord) {
					state.Users.UpdateField(mac, func(u *state.UserRecord) { u.FreeClaimed = 0 })
				})
			}
			cfg.FreeTimeEnabled = newFreeEnabled
		}
		if val, ok := rawBody["free_time_duration"]; ok {
			cfg.FreeTimeDuration = parseSettingInt(val, cfg.FreeTimeDuration)
		}
	})
	config.Save()
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP), "Updated session limits settings")
	return c.JSON(fiber.Map{"status": "success"})
}

func updatePortalSettings(c *fiber.Ctx) error {
	clientIP := c.IP()
	rawBody, err := parseSettingsBody(c)
	if err != nil { return c.Status(400).JSON(fiber.Map{"status": "error", "message": err.Error()}) }

	config.Update(func(cfg *config.AppConfig) {
		if val, ok := rawBody["timeout"]; ok {
			cfg.SlotTimeout = parseSettingInt(val, cfg.SlotTimeout)
		}
		if val, ok := rawBody["banner_text"]; ok {
			cfg.BannerText = fmt.Sprintf("%v", val)
		}
		if val, ok := rawBody["banner_link"]; ok {
			cfg.BannerLink = fmt.Sprintf("%v", val)
		}
		if val, ok := rawBody["portal_title"]; ok {
			cfg.PortalTitle = fmt.Sprintf("%v", val)
		}
		if val, ok := rawBody["portal_title_color"]; ok {
			if color := fmt.Sprintf("%v", val); color != "" {
				cfg.PortalTitleColor = color
			}
		}
		if val, ok := rawBody["portal_title_size"]; ok {
			cfg.PortalTitleSize = parseSettingInt(val, cfg.PortalTitleSize)
		}
		if val, ok := rawBody["portal_subtitle"]; ok {
			cfg.PortalSubtitle = fmt.Sprintf("%v", val)
		}
		if val, ok := rawBody["portal_subtitle_size"]; ok {
			cfg.PortalSubtitleSize = parseSettingInt(val, cfg.PortalSubtitleSize)
		}
		if val, ok := rawBody["sound_insert"]; ok {
			cfg.SoundInsert = fmt.Sprintf("%v", val)
		}
		if val, ok := rawBody["sound_coin"]; ok {
			cfg.SoundCoin = fmt.Sprintf("%v", val)
		}
	})
	config.Save()
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP), "Updated portal settings")
	return c.JSON(fiber.Map{"status": "success"})
}

func rebootDevice(c *fiber.Ctx) error {
	clientIP := c.IP()
	logger.AuditLog("SYSTEM_REBOOT", clientIP, infrastructure.GetMACFromIP(clientIP), "Initiated manual system reboot")
	go func() {
		time.Sleep(2 * time.Second)
		infrastructure.RebootDevice()
	}()
	return c.JSON(fiber.Map{"status": "success", "message": "Rebooting now..."})
}

func resetSettings(c *fiber.Ctx) error {
	clientIP := c.IP()
	
	if err := config.ResetToDefaults(); err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to reset configuration"})
	}

	logger.AuditLog("SYSTEM_RESET", clientIP, infrastructure.GetMACFromIP(clientIP), "Performed a factory reset of system configuration settings. Rebooting...")

	go func() {
		time.Sleep(2 * time.Second)
		infrastructure.RebootDevice()
	}()

	return c.JSON(fiber.Map{"status": "success", "message": "Settings reset to defaults. Rebooting system..."})
}

func downloadBackup(c *fiber.Ctx) error {
	clientIP := c.IP()
	
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add config.json
	if configData, err := os.ReadFile("config.json"); err == nil {
		if f, err := zipWriter.Create("config.json"); err == nil {
			f.Write(configData)
		}
	}

	// Add pisowifi.db
	if dbData, err := os.ReadFile("pisowifi.db"); err == nil {
		if f, err := zipWriter.Create("pisowifi.db"); err == nil {
			f.Write(dbData)
		}
	}

	zipWriter.Close()

	logger.AuditLog("SYSTEM_BACKUP", clientIP, infrastructure.GetMACFromIP(clientIP), "Downloaded system database backup")

	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=pisowifi_backup_%s.zip", time.Now().Format("20060102_150405")))
	return c.Send(buf.Bytes())
}

func restoreBackup(c *fiber.Ctx) error {
	clientIP := c.IP()
	
	file, err := c.FormFile("backup_file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "No file uploaded"})
	}

	// Save the uploaded file to a temporary location
	tmpPath := "uploaded_backup.zip"
	if err := c.SaveFile(file, tmpPath); err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to save uploaded file"})
	}
	defer os.Remove(tmpPath)

	// Open the zip file
	r, err := zip.OpenReader(tmpPath)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid zip file"})
	}
	defer r.Close()

	var hasConfig, hasDB bool

	for _, f := range r.File {
		if f.Name == "config.json" {
			rc, err := f.Open()
			if err == nil {
				data, _ := io.ReadAll(rc)
				os.WriteFile("config.json", data, 0644)
				rc.Close()
				hasConfig = true
			}
		} else if f.Name == "pisowifi.db" {
			// CRITICAL FIX: Close the active database to detach the WAL file 
			// BEFORE overwriting the main database file.
			db.CloseDB()

			rc, err := f.Open()
			if err == nil {
				data, _ := io.ReadAll(rc)
				os.WriteFile("pisowifi.db", data, 0644)
				rc.Close()

				// Delete the WAL and SHM files to ensure SQLite doesn't try 
				// to merge old active data into the newly restored backup!
				os.Remove("pisowifi.db-wal")
				os.Remove("pisowifi.db-shm")

				hasDB = true
			}
		}
	}

	if !hasConfig && !hasDB {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Zip file does not contain valid backup files"})
	}

	logger.AuditLog("SYSTEM_RESTORE", clientIP, infrastructure.GetMACFromIP(clientIP), "Restored system database backup. Rebooting...")

	// Schedule a reboot
	go func() {
		time.Sleep(2 * time.Second)
		infrastructure.RebootDevice()
	}()

	return c.JSON(fiber.Map{"status": "success", "message": "Backup restored successfully. Rebooting system..."})
}

// ---------------------------------------------------------------------------
// Banners
// ---------------------------------------------------------------------------

func clearBanners(c *fiber.Ctx) error {
	clientIP := c.IP()
	folder := "static/banners/set"
	if entries, err := os.ReadDir(folder); err == nil {
		for _, e := range entries {
			os.Remove(filepath.Join(folder, e.Name()))
		}
	}
	config.Update(func(cfg *config.AppConfig) { cfg.BannerOrder = []string{} })
	config.Save()
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP), "Cleared all promotional banners")
	return c.JSON(fiber.Map{"status": "success"})
}

func uploadBanners(c *fiber.Ctx) error {
	clientIP := c.IP()
	os.MkdirAll("static/banners/set", 0755)
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Bad form"})
	}
	files := form.File["files"]
	for _, f := range files {
		if err := c.SaveFile(f, filepath.Join("static/banners/set", f.Filename)); err != nil {
			continue
		}
	}
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP),
		fmt.Sprintf("Uploaded %d new promotional banners", len(files)))
	return c.JSON(fiber.Map{"status": "success"})
}

func saveBannerOrder(c *fiber.Ctx) error {
	orderStr := c.FormValue("order", "[]")
	var fileList []string
	if err := json.Unmarshal([]byte(orderStr), &fileList); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	config.Update(func(cfg *config.AppConfig) { cfg.BannerOrder = fileList })
	config.Save()
	return c.JSON(fiber.Map{"status": "success"})
}

func deleteBanner(c *fiber.Ctx) error {
	filename := c.FormValue("filename", "")
	if filename != "" {
		os.Remove(filepath.Join("static/banners/set", filename))
		config.Update(func(cfg *config.AppConfig) {
			var newOrder []string
			for _, f := range cfg.BannerOrder {
				if f != filename {
					newOrder = append(newOrder, f)
				}
			}
			cfg.BannerOrder = newOrder
		})
		config.Save()
	}
	return c.JSON(fiber.Map{"status": "success"})
}

// ---------------------------------------------------------------------------
// Sounds
// ---------------------------------------------------------------------------

func uploadSound(c *fiber.Ctx) error {
	clientIP := c.IP()
	os.MkdirAll("static/sounds", 0755)
	f, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "No file"})
	}
	if err := c.SaveFile(f, filepath.Join("static/sounds", f.Filename)); err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Save error"})
	}
	logger.AuditLog("CONFIG_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP),
		fmt.Sprintf("Uploaded new sound file: %s", f.Filename))
	return c.JSON(fiber.Map{"status": "success"})
}

func deleteSound(c *fiber.Ctx) error {
	filename := c.FormValue("filename", "")
	if filename != "" {
		os.Remove(filepath.Join("static/sounds", filename))
		// Optional: we don't strictly need to update config here, but it's safe
	}
	return c.JSON(fiber.Map{"status": "success"})
}

// ---------------------------------------------------------------------------
// Vouchers
// ---------------------------------------------------------------------------

func getVouchers(c *fiber.Ctx) error {
	vouchers := db.GetAllVouchers()
	cfg := config.Get()
	return c.JSON(fiber.Map{
		"vouchers": vouchers,
		"voucher_enabled": cfg.VoucherEnabled,
		"voucher_min_time_minutes": cfg.VoucherMinTimeMinutes,
		"voucher_point_promos": cfg.VoucherPointPromos,
	})
}

func updateVoucherSettings(c *fiber.Ctx) error {
	var body struct {
		Enabled   bool    `json:"voucher_enabled"`
		MinTime   int     `json:"voucher_min_time_minutes"`
		PointPromos []config.PromoItem `json:"voucher_point_promos"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid body"})
	}

	config.Update(func(cfg *config.AppConfig) {
		cfg.VoucherEnabled = body.Enabled
		cfg.VoucherMinTimeMinutes = body.MinTime
		cfg.VoucherPointPromos = body.PointPromos
	})
	config.Save()

	return c.JSON(fiber.Map{"status": "success"})
}

// ---------------------------------------------------------------------------
// User management
// ---------------------------------------------------------------------------

func getSingleUser(c *fiber.Ctx) error {
	mac := c.Params("mac")
	user, ok := state.Users.Get(mac)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}
	leases := infrastructure.GetDhcpLeases()
	displayName, _ := infrastructure.GetVendorInfo(mac, user.IP, leases)
	salesHistory := db.GetUserSales(mac)
	for i, s := range salesHistory {
		salesHistory[i].DateStr = time.Unix(s.Timestamp, 0).Format("Jan 02, 2006 03:04 PM")
	}
	return c.JSON(fiber.Map{
		"mac":            mac,
		"user":           user,
		"time_formatted": services.FormatHumanTime(user.Time),
		"history":        salesHistory,
		"device_name":    displayName,
		"vouchers":       db.GetAllVouchersByUser(mac),
	})
}

func adminManageTime(c *fiber.Ctx) error {
	var body struct {
		MAC    string `json:"mac"`
		Amount string `json:"amount"`
		Unit   string `json:"unit"`
		Action string `json:"action"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	clientIP := c.IP()

	amount := 0
	fmt.Sscan(body.Amount, &amount)

	services.ManageUserTime(body.MAC, amount, body.Unit, body.Action)
	logger.AuditLog("TIME_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP),
		fmt.Sprintf("%s %d %s applied to target user %s", strings.ToUpper(body.Action), amount, body.Unit, body.MAC))
	return c.JSON(fiber.Map{"status": "success"})
}

func adminManagePoints(c *fiber.Ctx) error {
	var body struct {
		MAC    string `json:"mac"`
		Amount string `json:"amount"`
		Action string `json:"action"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	clientIP := c.IP()

	amount := 0.0
	fmt.Sscan(body.Amount, &amount)
	if amount < 0 {
		amount = -amount
	}

	state.Users.UpdateField(body.MAC, func(u *state.UserRecord) {
		if body.Action == "subtract" {
			u.Points -= amount
		} else {
			u.Points += amount
		}
		if u.Points < 0 {
			u.Points = 0
		}
	})
	if u, ok := state.Users.Get(body.MAC); ok {
		db.SyncUser(db.UserRecord{
			MAC: body.MAC, IP: u.IP, Time: u.Time, Status: u.Status,
			Balance: u.Balance, FreeClaimed: u.FreeClaimed, Points: u.Points,
		})
	}
	logger.AuditLog("POINTS_UPDATE", clientIP, infrastructure.GetMACFromIP(clientIP),
		fmt.Sprintf("%s %.1f points applied to target user %s", strings.ToUpper(body.Action), amount, body.MAC))
	return c.JSON(fiber.Map{"status": "success"})
}

func adminBlock(c *fiber.Ctx) error {
	var body struct{ MAC string `json:"mac"` }
	if err := c.BodyParser(&body); err != nil { return c.JSON(fiber.Map{"status":"error"}) }
	clientIP := c.IP()
	services.UpdateUserStatus(body.MAC, "blocked")
	logger.AuditLog("USER_BLOCKED", clientIP, infrastructure.GetMACFromIP(clientIP), fmt.Sprintf("Blocked access for target user %s", body.MAC))
	return c.JSON(fiber.Map{"status": "success"})
}

func adminUnblock(c *fiber.Ctx) error {
	var body struct{ MAC string `json:"mac"` }
	if err := c.BodyParser(&body); err != nil { return c.JSON(fiber.Map{"status":"error"}) }
	clientIP := c.IP()
	services.UpdateUserStatus(body.MAC, "new")
	logger.AuditLog("USER_UNBLOCKED", clientIP, infrastructure.GetMACFromIP(clientIP), fmt.Sprintf("Restored access for target user %s", body.MAC))
	return c.JSON(fiber.Map{"status": "success"})
}

func adminDeleteUser(c *fiber.Ctx) error {
	var body struct{ MAC string `json:"mac"` }
	if err := c.BodyParser(&body); err != nil { return c.JSON(fiber.Map{"status":"error"}) }
	clientIP := c.IP()
	services.DeleteUser(body.MAC)
	logger.AuditLog("USER_DELETED", clientIP, infrastructure.GetMACFromIP(clientIP), fmt.Sprintf("Deleted target user %s from system", body.MAC))
	return c.JSON(fiber.Map{"status": "success"})
}

func renameDevice(c *fiber.Ctx) error {
	var body struct {
		MAC  string `json:"mac"`
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.JSON(fiber.Map{"status": "error"})
	}
	clientIP := c.IP()
	config.Update(func(cfg *config.AppConfig) {
		if cfg.CustomDeviceNames == nil {
			cfg.CustomDeviceNames = map[string]string{}
		}
		cfg.CustomDeviceNames[body.MAC] = strings.TrimSpace(body.Name)
	})
	config.Save()
	logger.AuditLog("DEVICE_RENAMED", clientIP, infrastructure.GetMACFromIP(clientIP),
		fmt.Sprintf("Renamed device %s to '%s'", body.MAC, strings.TrimSpace(body.Name)))
	return c.JSON(fiber.Map{"status": "success"})
}

func addDevice(c *fiber.Ctx) error {
	var body struct {
		MAC  string `json:"mac"`
		Name string `json:"name"`
		IP   string `json:"ip"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid body"})
	}
	body.MAC = strings.ToLower(strings.TrimSpace(body.MAC))
	if body.MAC == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "MAC address is required"})
	}
	config.Update(func(cfg *config.AppConfig) {
		if cfg.CustomDeviceNames == nil {
			cfg.CustomDeviceNames = map[string]string{}
		}
		if cfg.CustomDeviceIPs == nil {
			cfg.CustomDeviceIPs = map[string]string{}
		}
		cfg.CustomDeviceNames[body.MAC] = strings.TrimSpace(body.Name)
		if strings.TrimSpace(body.IP) != "" {
			cfg.CustomDeviceIPs[body.MAC] = strings.TrimSpace(body.IP)
		}
	})
	config.Save()
	clientIP := c.IP()
	logger.AuditLog("DEVICE_ADDED", clientIP, infrastructure.GetMACFromIP(clientIP),
		fmt.Sprintf("Manually added device %s as '%s' with IP '%s'", body.MAC, strings.TrimSpace(body.Name), strings.TrimSpace(body.IP)))
	return c.JSON(fiber.Map{"status": "success"})
}

func removeDevice(c *fiber.Ctx) error {
	var body struct {
		MAC string `json:"mac"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid body"})
	}
	body.MAC = strings.ToLower(strings.TrimSpace(body.MAC))
	if body.MAC == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "MAC address is required"})
	}
	config.Update(func(cfg *config.AppConfig) {
		if cfg.CustomDeviceNames != nil {
			delete(cfg.CustomDeviceNames, body.MAC)
		}
		if cfg.CustomDeviceIPs != nil {
			delete(cfg.CustomDeviceIPs, body.MAC)
		}
	})
	config.Save()
	clientIP := c.IP()
	logger.AuditLog("DEVICE_REMOVED", clientIP, infrastructure.GetMACFromIP(clientIP),
		fmt.Sprintf("Removed custom device %s", body.MAC))
	return c.JSON(fiber.Map{"status": "success"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseLogLine(line string) map[string]string {
	// Format: [timestamp] [TYPE] message
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "[") {
		return nil
	}
	// Extract first bracket pair = timestamp
	end1 := strings.Index(line, "]")
	if end1 < 0 {
		return nil
	}
	rest := strings.TrimSpace(line[end1+1:])
	if !strings.HasPrefix(rest, "[") {
		// Fallback legacy: [timestamp] message
		return map[string]string{
			"timestamp": line[1:end1],
			"type":      "SYSTEM",
			"message":   rest,
		}
	}
	end2 := strings.Index(rest, "]")
	if end2 < 0 {
		return nil
	}
	return map[string]string{
		"timestamp": line[1:end1],
		"type":      rest[1:end2],
		"message":   strings.TrimSpace(rest[end2+1:]),
	}
}

func tailLines(f *os.File, n int) ([]string, error) {
	const bufSize = 64 * 1024
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return nil, nil
	}
	buf := make([]byte, bufSize)
	var collected []byte
	for pos := size; pos > 0; {
		read := int64(bufSize)
		if pos < read {
			read = pos
		}
		pos -= read
		f.Seek(pos, io.SeekStart)
		nr, _ := f.Read(buf[:read])
		collected = append(buf[:nr], collected...)
		lines := strings.Split(string(collected), "\n")
		if len(lines) >= n+1 {
			return lines[len(lines)-n:], nil
		}
	}
	return strings.Split(string(collected), "\n"), nil
}