package mqtt

// commands.go — high-level MQTT commands that the Orange Pi sends to the router.
//
// Each function maps to a topic the router's subscriber script listens on.
// The router then applies nftables / tc / conntrack rules locally.
//
// Topic layout:
//   pisowifi/allow              → add MAC to authorized_users set
//   pisowifi/block              → remove MAC, flush conntrack
//   pisowifi/speed_limit/apply  → apply tc HTB class for IP
//   pisowifi/speed_limit/remove → remove tc class for IP
//   pisowifi/firewall/init      → full nftables ruleset init
//   pisowifi/firewall/reload    → reload rules + restore sessions

import "fmt"

// ---------------------------------------------------------------------------
// Payload types — must match what the router subscriber script expects
// ---------------------------------------------------------------------------

type macIPPayload struct {
	MAC string `json:"mac"`
	IP  string `json:"ip,omitempty"`
}

type speedPayload struct {
	IP   string `json:"ip"`
	Mbps int    `json:"mbps"`
}

type firewallInitPayload struct {
	LAN                string `json:"lan"`
	WAN                string `json:"wan"`
	CustomTTL          int    `json:"custom_ttl"`
	UDPPriority        bool   `json:"udp_priority"`
	OpenNAT            bool   `json:"open_nat"`
	SQMEnabled               bool   `json:"sqm_enabled"`
	SQMUploadMbps            int    `json:"sqm_upload_mbps"`
	SQMDownloadMbps          int    `json:"sqm_download_mbps"`
	AutoPauseEnabled         bool   `json:"auto_pause_enabled"`
	InactiveTimeout          int    `json:"inactive_timeout"`
	InactiveBytesThreshold   int    `json:"inactive_bytes_threshold"`
	InactivePacketsThreshold int    `json:"inactive_packets_threshold"`
}

// ---------------------------------------------------------------------------
// User allow / block
// ---------------------------------------------------------------------------

// AllowUser tells the router to add the MAC to the nftables authorized_users set.
func AllowUser(mac, ip string) error {
	return Publish("pisowifi/allow", macIPPayload{MAC: mac, IP: ip})
}

// BlockUser tells the router to remove the MAC from authorized_users and flush conntrack.
func BlockUser(mac, ip string) error {
	return Publish("pisowifi/block", macIPPayload{MAC: mac, IP: ip})
}

// ---------------------------------------------------------------------------
// Speed limits (tc HTB)
// ---------------------------------------------------------------------------

// ApplySpeedLimit tells the router to create/replace an HTB class for the IP.
func ApplySpeedLimit(ip string, mbps int) error {
	if ip == "" || mbps <= 0 {
		return fmt.Errorf("ApplySpeedLimit: invalid args ip=%q mbps=%d", ip, mbps)
	}
	return Publish("pisowifi/speed_limit/apply", speedPayload{IP: ip, Mbps: mbps})
}

// RemoveSpeedLimit tells the router to remove the HTB class for the IP.
func RemoveSpeedLimit(ip string) error {
	if ip == "" {
		return nil
	}
	return Publish("pisowifi/speed_limit/remove", macIPPayload{IP: ip})
}

// ---------------------------------------------------------------------------
// Firewall lifecycle
// ---------------------------------------------------------------------------

// InitFirewall tells the router to run the full nftables init sequence.
// Called once at Orange Pi startup.
func InitFirewall(p firewallInitPayload) error {
	return Publish("pisowifi/firewall/init", p)
}

// ReloadFirewall tells the router to reload nftables rules.
// Called after config changes (e.g. speed limit toggle, SQM toggle).
func ReloadFirewall(p firewallInitPayload) error {
	return Publish("pisowifi/firewall/reload", p)
}

// NewFirewallPayload is a convenience constructor for building the payload
// from config values.
func NewFirewallPayload(lan, wan string, customTTL int, udpPriority, openNAT, sqmEnabled bool, sqmUp, sqmDown int, autoPause bool, inactiveTimeout, inactiveBytes, inactivePkts int) firewallInitPayload {
	return firewallInitPayload{
		LAN:                      lan,
		WAN:                      wan,
		CustomTTL:                customTTL,
		UDPPriority:              udpPriority,
		OpenNAT:                  openNAT,
		SQMEnabled:               sqmEnabled,
		SQMUploadMbps:            sqmUp,
		SQMDownloadMbps:          sqmDown,
		AutoPauseEnabled:         autoPause,
		InactiveTimeout:          inactiveTimeout,
		InactiveBytesThreshold:   inactiveBytes,
		InactivePacketsThreshold: inactivePkts,
	}
}
