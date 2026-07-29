package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// ---------------------------------------------------------------------------
// Hardware constants — adjust these to match your Orange Pi board wiring
// ---------------------------------------------------------------------------

// LANInterface and WANInterface are the Linux interface names for the LAN
// (the side clients connect to) and WAN (the internet uplink). These are set
// from the .env file at startup via ReloadSecrets(). Defaults match a typical
// Orange Pi bridge setup but MUST be changed to match your hardware.
var (
	LANInterface = "br0"
	WANInterface  = "eth0"

	// RouterIP is the LAN IP of the OpenWrt router.
	// Used for policy routing so HTTP replies go through the router (preserving
	// real client source IPs after removing hairpin masquerade).
	RouterIP = "10.0.0.1"
)

const (
	// sysfs GPIO numbers for your Orange Pi board.
	CoinGPIONum  = 122 // coin signal input pin for orange pi 3 lts 122
	RelayGPIONum = 121 // relay / slot power output pin for orange pi 3 lts 121

	// WiringOP Physical Pin numbers on the 40-pin header
	CoinPinPhys  = 3
	RelayPinPhys = 5

	PulseValue = 1 // coins credited per physical pulse
)

// ---------------------------------------------------------------------------
// Secrets loaded from .env (set by cmd/server/main.go via godotenv)
// ---------------------------------------------------------------------------

var (
	SecretKey     string
	AdminUsername string
	AdminPassword string

	// MQTT — connection to the OpenWrt router broker
	MQTTBroker   string
	MQTTClientID string
	MQTTUsername string
	MQTTPassword string
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ReloadSecrets reads SECRET_KEY, ADMIN_USERNAME, ADMIN_PASSWORD from the
// environment (called after godotenv.Load so .env values are in os.Getenv).
func ReloadSecrets() {
	SecretKey = getEnv("SECRET_KEY", "unsafe_default_change_me")
	AdminUsername = getEnv("ADMIN_USERNAME", "admin")
	AdminPassword = getEnv("ADMIN_PASSWORD", "admin123")
	LANInterface = getEnv("LAN_INTERFACE", "br0")
	WANInterface = getEnv("WAN_INTERFACE", "eth0")
	RouterIP = getEnv("ROUTER_IP", "10.0.0.1")

	// MQTT broker (runs on the OpenWrt router)
	MQTTBroker = getEnv("MQTT_BROKER", "tcp://10.0.0.1:1883")
	MQTTClientID = getEnv("MQTT_CLIENT_ID", "pisowifi-orangepi")
	MQTTUsername = getEnv("MQTT_USERNAME", "")
	MQTTPassword = getEnv("MQTT_PASSWORD", "")
}

// ---------------------------------------------------------------------------
// AppConfig — mirrors config.json exactly
// ---------------------------------------------------------------------------

type RestartSchedule struct {
	Enabled bool   `json:"enabled"`
	Time    string `json:"time"`
}

type PromoItem struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Cost    float64 `json:"cost"`
	Minutes int     `json:"minutes"`
}

type AppConfig struct {
	SlotTimeout            int                `json:"slot_timeout"`
	SlotExpiryTimestamp    float64            `json:"slot_expiry_timestamp"`
	SQMEnabled             bool               `json:"sqm_enabled"`
	SQMUploadMbps          int                `json:"sqm_upload_mbps"`
	SQMDownloadMbps        int                `json:"sqm_download_mbps"`
	InactiveTimeout        int                `json:"inactive_timeout"`
	AutoPauseEnabled       bool               `json:"auto_pause_enabled"`
	SpeedLimitEnabled      bool               `json:"speed_limit_enabled"`
	GlobalSpeedLimit       int                `json:"global_speed_limit"`
	GamingModeEnabled      bool               `json:"gaming_mode_enabled"`
	UDPPriorityEnabled     bool               `json:"udp_priority_enabled"`
	InactivePacketThreshold int               `json:"inactive_packet_threshold"`
	CoinRates              string             `json:"coin_rates"`
	PulseValue             int                `json:"pulse_value"`
	RestartSchedule        RestartSchedule    `json:"restart_schedule"`
	PointsEnabled          bool               `json:"points_enabled"`
	VoucherEnabled         bool               `json:"voucher_enabled"`
	OpenNATEnabled         bool               `json:"open_nat_enabled"`
	CustomTTL              int                `json:"custom_ttl"`
	CoinPointMap           map[string]float64 `json:"coin_point_map"`
	PointPromos            []PromoItem        `json:"point_promos"`
	InactiveBytesThreshold int                `json:"inactive_bytes_threshold"`
	BannerText             string             `json:"banner_text"`
	BannerLink             string             `json:"banner_link"`
	BannerOrder            []string           `json:"banner_order"`
	PortalTitle            string             `json:"portal_title"`
	PortalTitleColor       string             `json:"portal_title_color"`
	PortalTitleSize        int                `json:"portal_title_size"`
	PortalSubtitle         string             `json:"portal_subtitle"`
	PortalSubtitleSize     int                `json:"portal_subtitle_size"`
	FreeTimeEnabled        bool               `json:"free_time_enabled"`
	FreeTimeDuration       int                `json:"free_time_duration"`
	SoundInsert            string             `json:"sound_insert"`
	SoundCoin              string             `json:"sound_coin"`
	CustomDeviceNames      map[string]string  `json:"custom_device_names,omitempty"`
	CustomDeviceIPs        map[string]string  `json:"custom_device_ips,omitempty"`
	VoucherMinTimeMinutes  int                `json:"voucher_min_time_minutes"`
	VoucherPointPromos     []PromoItem        `json:"voucher_point_promos"`
}

// Defaults mirror state.py defaults
var defaultConfig = AppConfig{
	SlotTimeout:             30,
	SlotExpiryTimestamp:     0,
	SQMEnabled:              false,
	SQMUploadMbps:           100,
	SQMDownloadMbps:         100,
	InactiveTimeout:         300,
	AutoPauseEnabled:        false,
	SpeedLimitEnabled:       false,
	GlobalSpeedLimit:        5,
	GamingModeEnabled:       false,
	UDPPriorityEnabled:      false,
	InactivePacketThreshold: 5,
	CoinRates:               "1:10,5:60,10:180,20:300",
	PulseValue:              1,
	RestartSchedule:         RestartSchedule{Enabled: false, Time: "03:00"},
	PointsEnabled:           false,
	VoucherEnabled:          false,
	OpenNATEnabled:          false,
	CustomTTL:               1,
	CoinPointMap:            map[string]float64{"1": 0.5, "5": 1, "10": 3, "20": 5},
	PointPromos:             []PromoItem{{ID: 1, Name: "3 Hours Free", Cost: 20, Minutes: 180}},
	InactiveBytesThreshold:  500,
	BannerText:              "",
	BannerLink:              "",
	BannerOrder:             []string{},
	PortalTitle:             "PISOWIFI",
	PortalTitleColor:        "#fde68a",
	PortalTitleSize:         27,
	PortalSubtitle:          "Premium internet connectevity",
	PortalSubtitleSize:      15,
	FreeTimeEnabled:         false,
	FreeTimeDuration:        5,
	SoundInsert:             "insert_coin_sound.mp3",
	SoundCoin:               "coin-recieved.mp3",
	CustomDeviceNames:       map[string]string{},
	CustomDeviceIPs:         map[string]string{},
	VoucherMinTimeMinutes:   5,
	VoucherPointPromos:      []PromoItem{{ID: 1, Name: "Convert 50 Points", Cost: 50, Minutes: 30}},
}

const configFile = "config.json"

var (
	mu      sync.RWMutex
	current AppConfig
)

// Get returns a copy of the current config (safe to read without holding the lock).
func Get() AppConfig {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Update merges fields into the current config. Caller must call Save() afterward.
func Update(fn func(*AppConfig)) {
	mu.Lock()
	defer mu.Unlock()
	fn(&current)
}

// Load reads config.json into memory, merging over defaults.
func Load() {
	mu.Lock()
	defer mu.Unlock()
	current = defaultConfig

	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println("[Config] config.json not found, using defaults.")
		return
	}
	if err := json.Unmarshal(data, &current); err != nil {
		fmt.Printf("[Config] config.json corrupt: %v — using defaults.\n", err)
		_ = os.Rename(configFile, configFile+".corrupt")
		current = defaultConfig
	}
	// Ensure maps are never nil
	if current.CoinPointMap == nil {
		current.CoinPointMap = defaultConfig.CoinPointMap
	}
	if current.CustomDeviceNames == nil {
		current.CustomDeviceNames = map[string]string{}
	}
	if current.CustomDeviceIPs == nil {
		current.CustomDeviceIPs = map[string]string{}
	}
	fmt.Println("[Config] Loaded from config.json.")
}

// ResetToDefaults resets the current configuration in memory to the defaults and saves it to disk.
func ResetToDefaults() error {
	mu.Lock()
	current = defaultConfig
	mu.Unlock()
	return Save()
}

// Save atomically writes the current config to config.json.
func Save() error {
	mu.RLock()
	data, err := json.MarshalIndent(current, "", "    ")
	mu.RUnlock()
	if err != nil {
		return fmt.Errorf("config marshal: %w", err)
	}

	tmp := configFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("config create tmp: %w", err)
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmp, configFile)
}
