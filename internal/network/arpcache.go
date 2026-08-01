package network

// arpcache.go — Orange Pi side MAC ↔ IP cache.
//
// The router is the source of truth for MAC→IP mappings (it runs DHCP and
// holds the ARP table). This package subscribes to the MQTT topic
// "pisowifi/arp" published by the router's dnsmasq DHCP hook and the
// subscriber script's dump_arp_table() function.
//
// On startup (or reconnect) the Orange Pi publishes "pisowifi/arp/request"
// which triggers the router to dump its full ARP table as individual
// retained "pisowifi/arp" messages.
//
// Other packages should call:
//   network.GetIPByMAC(mac) string   — look up IP for a known MAC
//   network.GetMACByIP(ip)  string   — reverse lookup (for logging)

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"pisowifi/internal/logger"
	mqttcmd "pisowifi/internal/mqtt"
	"pisowifi/internal/state"
)

// ---------------------------------------------------------------------------
// Cache
// ---------------------------------------------------------------------------

type arpEntry struct {
	MAC string
	IP  string
}

var (
	arpMu        sync.RWMutex
	macToIP      = make(map[string]string) // lowercase MAC → IP
	ipToMAC      = make(map[string]string) // IP → lowercase MAC
	macToHost    = make(map[string]string) // lowercase MAC → Hostname
)

// ---------------------------------------------------------------------------
// Init — called once from InitFirewall after MQTT is ready
// ---------------------------------------------------------------------------

// InitARPCache subscribes to pisowifi/arp and immediately requests a full
// ARP table dump from the router.
func InitARPCache() {
	mqttcmd.Subscribe("pisowifi/arp", handleARPMessage)
	logger.SystemLog("[ARP] Subscribed to pisowifi/arp — requesting full table dump...")
	RequestARPSync()
}

var (
	lastSyncTime time.Time
	syncMu       sync.Mutex
)

// RequestARPSync publishes pisowifi/arp/request so the router dumps its
// current ARP table. Rate limited to once every 10 seconds.
func RequestARPSync() {
	syncMu.Lock()
	if time.Since(lastSyncTime) < 10*time.Second {
		syncMu.Unlock()
		return
	}
	lastSyncTime = time.Now()
	syncMu.Unlock()

	if err := mqttcmd.PublishRetained("pisowifi/arp/request", map[string]string{"action": "dump"}); err != nil {
		logger.SystemLog("[ARP] [ERROR] Failed to publish arp/request: " + err.Error())
	}
}

// ---------------------------------------------------------------------------
// MQTT message handler
// ---------------------------------------------------------------------------

// arpPayload mirrors what dhcp_hook.sh and dump_arp_table() publish.
type arpPayload struct {
	MAC      string `json:"mac"`
	IP       string `json:"ip"`
	Action   string `json:"action"` // "add" | "del"
	Hostname string `json:"hostname,omitempty"`
}

func handleARPMessage(_ paho.Client, msg paho.Message) {
	var p arpPayload
	if err := json.Unmarshal(msg.Payload(), &p); err != nil {
		return
	}
	mac := strings.ToLower(strings.TrimSpace(p.MAC))
	ip := strings.TrimSpace(p.IP)
	if mac == "" || ip == "" {
		return
	}

	arpMu.Lock()
	defer arpMu.Unlock()

	switch p.Action {
	case "add", "": // "old" dnsmasq events are remapped to "add" in the hook
		// Remove any stale reverse mapping for the old IP of this MAC
		if oldIP, ok := macToIP[mac]; ok && oldIP != ip {
			delete(ipToMAC, oldIP)
		}
		macToIP[mac] = ip
		ipToMAC[ip] = mac
		if p.Hostname != "" {
			macToHost[mac] = p.Hostname
			// Push it directly to the session state so it can be saved to SQLite permanently
			state.Users.UpdateField(mac, func(u *state.UserRecord) {
				if u.Hostname == "" || u.Hostname != p.Hostname {
					u.Hostname = p.Hostname
				}
			})
		}
		// logger.SystemLog("[ARP] Learned: " + mac + " → " + ip + " (" + p.Hostname + ")")

	case "del":
		if oldIP, ok := macToIP[mac]; ok {
			delete(ipToMAC, oldIP)
		}
		delete(macToIP, mac)
		// We intentionally DO NOT delete macToHost[mac] here!
		// If a user's phone goes to sleep, we still want the Admin UI to know their hostname.
		// logger.SystemLog("[ARP] Removed IP mapping: " + mac + " (was " + ip + ")")
	}
}

// ---------------------------------------------------------------------------
// Public lookups
// ---------------------------------------------------------------------------

// GetIPByMAC returns the last-known IP for a MAC address, or "" if unknown.
func GetIPByMAC(mac string) string {
	arpMu.RLock()
	defer arpMu.RUnlock()
	return macToIP[strings.ToLower(mac)]
}

// GetMACByIP looks up the MAC for an IP. Returns empty string if unknown.
func GetMACByIP(ip string) string {
	arpMu.RLock()
	defer arpMu.RUnlock()
	return ipToMAC[ip]
}

// GetHostnameByMAC looks up the DHCP hostname for a MAC. Returns empty string if unknown.
func GetHostnameByMAC(mac string) string {
	mac = strings.ToLower(mac)
	arpMu.RLock()
	defer arpMu.RUnlock()
	return macToHost[mac]
}

// Snapshot returns a copy of the full MAC→IP table (for diagnostics/admin API).
func ARPSnapshot() map[string]string {
	arpMu.RLock()
	defer arpMu.RUnlock()
	out := make(map[string]string, len(macToIP))
	for k, v := range macToIP {
		out[k] = v
	}
	return out
}
