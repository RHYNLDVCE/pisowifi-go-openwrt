package network

// firewall.go — Orange Pi side of the firewall adapter.
//
// In the new split architecture:
//   - All nftables / tc / conntrack commands run on the OpenWrt ROUTER.
//   - The Orange Pi tells the router what to do via MQTT.
//   - This file keeps the same public API (AllowUser, BlockUser, etc.) so
//     the rest of the codebase (services/session.go, services/admin.go, …)
//     does NOT need to change.
//
// GetAllTraffic() reads traffic counters from the router via an MQTT reply
// published on pisowifi/traffic/response. For now it returns nil if no data
// has arrived yet (non-blocking).

import (
	"encoding/json"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"pisowifi/internal/config"
	"pisowifi/internal/logger"
	mqttcmd "pisowifi/internal/mqtt"
	"pisowifi/internal/state"
)

// ---------------------------------------------------------------------------
// Traffic counter cache (populated by the router via MQTT)
// ---------------------------------------------------------------------------

var (
	trafficMu    sync.RWMutex
	trafficCache map[string][2]int64 // mac → [bytes, packets]
)

func init() {
	trafficCache = make(map[string][2]int64)
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

// InitFirewall sends the full nftables init payload to the router over MQTT.
// Called once at startup (after mqtt.Init).
func InitFirewall() {
	logger.SystemLog("[FIREWALL] Sending firewall init to router via MQTT...")
	cfg := config.Get()
	payload := mqttcmd.NewFirewallPayload(
		config.LANInterface,
		config.WANInterface,
		cfg.CustomTTL,
		cfg.UDPPriorityEnabled,
		cfg.OpenNATEnabled,
		cfg.SQMEnabled,
		cfg.SQMUploadMbps,
		cfg.SQMDownloadMbps,
	)
	if err := mqttcmd.InitFirewall(payload); err != nil {
		logger.SystemLog("[FIREWALL] Failed to send init command: " + err.Error())
	} else {
		logger.SystemLog("[FIREWALL] Init command sent to router.")
	}

	// Subscribe to traffic stats published by the router
	subscribeTraffic()

	// Subscribe to pisowifi/arp and request a full MAC→IP dump from the router.
	// This must run AFTER mqtt.Init() so the subscription takes effect immediately.
	InitARPCache()
}

// ReloadFirewall sends a reload command to the router (e.g. after config change)
// and then re-allows all currently connected users.
func ReloadFirewall() {
	logger.SystemLog("[FIREWALL] Sending firewall reload to router via MQTT...")
	cfg := config.Get()
	payload := mqttcmd.NewFirewallPayload(
		config.LANInterface,
		config.WANInterface,
		cfg.CustomTTL,
		cfg.UDPPriorityEnabled,
		cfg.OpenNATEnabled,
		cfg.SQMEnabled,
		cfg.SQMUploadMbps,
		cfg.SQMDownloadMbps,
	)
	if err := mqttcmd.ReloadFirewall(payload); err != nil {
		logger.SystemLog("[FIREWALL] Failed to send reload command: " + err.Error())
		return
	}

	// Re-allow all connected users after the router reloads its ruleset
	time.Sleep(500 * time.Millisecond) // give the router a moment to apply rules
	state.Users.Range(func(mac string, u *state.UserRecord) {
		if u.Status == "connected" && u.IP != "" {
			AllowUser(mac, u.IP)
		}
	})
	logger.SystemLog("[FIREWALL] Reload complete. Active sessions restored.")
}

// ---------------------------------------------------------------------------
// Allow / Block — published to router, no local nft
// ---------------------------------------------------------------------------

// AllowUser tells the router to add the MAC to the authorized_users nftables set
// and (if IP is known) apply the configured speed limit via tc.
func AllowUser(mac, ip string) {
	if err := mqttcmd.AllowUser(mac, ip); err != nil {
		logger.SystemLog("[FIREWALL] AllowUser MQTT error for " + mac + ": " + err.Error())
	}
	if ip != "" {
		ApplySpeedLimit(ip)
	}
}

// BlockUser tells the router to remove the MAC from authorized_users and
// flush conntrack entries for that IP. Also removes tc speed limit class.
func BlockUser(mac, ip string) {
	resolvedIP := ip
	if resolvedIP == "" {
		// Fall back to the MQTT-driven ARP cache (populated by the router's
		// DHCP hook). This is the correct source — the OrangePi's own ARP
		// table never sees client traffic.
		resolvedIP = GetIPByMAC(mac)
	}
	if err := mqttcmd.BlockUser(mac, resolvedIP); err != nil {
		logger.SystemLog("[FIREWALL] BlockUser MQTT error for " + mac + ": " + err.Error())
	}
	if resolvedIP != "" {
		RemoveSpeedLimit(resolvedIP)
	}
}

// ---------------------------------------------------------------------------
// Speed limits — published to router, tc runs there
// ---------------------------------------------------------------------------

// ApplySpeedLimit tells the router to apply an HTB tc class for the IP.
func ApplySpeedLimit(ip string) {
	cfg := config.Get()
	if !cfg.SpeedLimitEnabled || ip == "" {
		return
	}
	if err := mqttcmd.ApplySpeedLimit(ip, cfg.GlobalSpeedLimit); err != nil {
		logger.SystemLog("[FIREWALL] ApplySpeedLimit MQTT error for " + ip + ": " + err.Error())
	}
}

// RemoveSpeedLimit tells the router to remove the tc class for the IP.
func RemoveSpeedLimit(ip string) {
	if ip == "" {
		return
	}
	if err := mqttcmd.RemoveSpeedLimit(ip); err != nil {
		logger.SystemLog("[FIREWALL] RemoveSpeedLimit MQTT error for " + ip + ": " + err.Error())
	}
}

// RefreshAllLimits re-applies speed limits for all connected users.
// Called when global speed limit setting changes.
func RefreshAllLimits() {
	state.Users.Range(func(mac string, u *state.UserRecord) {
		if u.Status == "connected" && u.IP != "" {
			RemoveSpeedLimit(u.IP)
			ApplySpeedLimit(u.IP)
		}
	})
}

// ---------------------------------------------------------------------------
// Traffic accounting — data pushed from router via MQTT
// ---------------------------------------------------------------------------

// trafficPayload matches what the router publishes on pisowifi/traffic.
type trafficPayload struct {
	MAC     string `json:"mac"`
	BytesRX int64  `json:"bytes_rx"`
	BytesTX int64  `json:"bytes_tx"`
	Packets int64  `json:"packets"`
}

func subscribeTraffic() {
	mqttcmd.Subscribe("pisowifi/traffic", func(_ paho.Client, msg paho.Message) {
		var p trafficPayload
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			return
		}
		trafficMu.Lock()
		trafficCache[p.MAC] = [2]int64{p.BytesRX + p.BytesTX, p.Packets}
		trafficMu.Unlock()
	})
}

// GetAllTraffic returns the latest traffic counters pushed by the router.
// Returns nil if no data has been received yet.
func GetAllTraffic() map[string][2]int64 {
	trafficMu.RLock()
	defer trafficMu.RUnlock()
	if len(trafficCache) == 0 {
		return nil
	}
	// Return a snapshot copy
	out := make(map[string][2]int64, len(trafficCache))
	for k, v := range trafficCache {
		out[k] = v
	}
	return out
}

