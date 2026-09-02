package services

import (
	"fmt"
	"strings"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/events"
	"pisowifi/internal/hardware"
	"pisowifi/internal/logger"
	"pisowifi/internal/network"
	"pisowifi/internal/state"
)

// ---------------------------------------------------------------------------
// SessionService — replaces services/session_service.py
// ---------------------------------------------------------------------------

// ConnectUser converts the user's stored balance to time, allows them through
// the firewall, and sends a WebSocket sync. Returns "success", "fail", or "blocked".
func ConnectUser(mac string) string {
	user, ok := state.Users.Get(mac)
	if !ok {
		return "fail"
	}
	if user.Status == "blocked" {
		return "blocked"
	}

	// Convert any pending balance to time + points
	if user.Balance > 0 {
		addedMinutes := CalculateTimeFromBalance(user.Balance)
		cfg := config.Get()

		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			u.Time += addedMinutes * 60
			if cfg.PointsEnabled {
				earned := CalculatePointsFromBalance(u.Balance)
				u.Points = RoundFloat(u.Points+earned, 2)
			}
			u.Balance = 0
		})

		// Re-read to get fresh values for DB write
		if u, ok := state.Users.Get(mac); ok {
			db.SyncUser(toDBRecord(mac, u))
		}
	}

	user, _ = state.Users.Get(mac)
	if user.Time <= 0 {
		return "fail"
	}

	// Connect
	state.Users.UpdateField(mac, func(u *state.UserRecord) {
		u.Status = "pending_connect"
		u.LastActive = float64(time.Now().UnixNano()) / 1e9
	})

	network.AllowUser(mac, user.IP)

	// Set a 10s timeout to revert if ACK doesn't arrive
	go func() {
		time.Sleep(10 * time.Second)
		if u, ok := state.Users.Get(mac); ok && u.Status == "pending_connect" {
			state.Users.UpdateField(mac, func(ur *state.UserRecord) {
				ur.Status = "paused"
			})
			logger.SystemLog(fmt.Sprintf("[SESSION] [ERROR] Router timeout for MAC %s: failed to connect", mac))
			state.Manager.Send(mac, map[string]any{
				"type":           "sync",
				"status":         "paused",
				"error":          "Router timeout: failed to connect",
				"time_remaining": u.Time,
			})
		}
	}()

	// Close slot if this user had it open
	if state.GetSlotUser() == mac {
		hardware.TurnSlotOff()
		state.SetSlotUser("")
	}

	if u, ok := state.Users.Get(mac); ok {
		db.SyncUser(toDBRecord(mac, u))
	}

	if u, ok := state.Users.Get(mac); ok {
		state.Manager.Send(mac, map[string]any{
			"type":           "sync",
			"status":         "pending_connect",
			"time_remaining": u.Time,
			"balance":        0,
			"points":         u.Points,
		})
		events.Global.Broadcast("user_update", map[string]interface{}{
			"mac":            mac,
			"ip":             u.IP,
			"time":           u.Time,
			"time_formatted": FormatHumanTime(u.Time),
			"status":         "pending_connect",
			"balance":        0,
			"points":         u.Points,
			"device_name":    u.Hostname,
			"active_users":   CountActiveUsers(),
		})
	}
	return "success"
}

// PauseUser pauses a connected user — blocks firewall and saves state.
func PauseUser(mac string) string {
	user, ok := state.Users.Get(mac)
	if !ok {
		return "fail"
	}
	if user.Status == "blocked" {
		return "fail"
	}
	if user.Status != "connected" {
		return "fail"
	}

	// Snapshot true remaining seconds from deadline
	state.Users.UpdateField(mac, func(u *state.UserRecord) {
		u.Status = "pending_pause"
	})

	network.BlockUser(mac, user.IP)

	// Set a 10s timeout to revert if ACK doesn't arrive
	go func() {
		time.Sleep(10 * time.Second)
		if u, ok := state.Users.Get(mac); ok && u.Status == "pending_pause" {
			state.Users.UpdateField(mac, func(ur *state.UserRecord) {
				ur.Status = "connected"
			})
			logger.SystemLog(fmt.Sprintf("[SESSION] [ERROR] Router timeout for MAC %s: failed to pause", mac))
			state.Manager.Send(mac, map[string]any{
				"type":           "sync",
				"status":         "connected",
				"error":          "Router timeout: failed to pause",
				"time_remaining": u.Time,
			})
		}
	}()

	if u, ok := state.Users.Get(mac); ok {
		db.SyncUser(toDBRecord(mac, u))
		state.Manager.Send(mac, map[string]any{
			"type":           "sync",
			"status":         "pending_pause",
			"time_remaining": u.Time,
			"balance":        u.Balance,
			"points":         u.Points,
		})
		events.Global.Broadcast("user_update", map[string]interface{}{
			"mac":            mac,
			"status":         "paused",
			"time":           u.Time,
			"time_formatted": FormatHumanTime(u.Time),
			"active_users":   CountActiveUsers(),
		})
	}
	return "success"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toDBRecord(mac string, u *state.UserRecord) db.UserRecord {
	return db.UserRecord{
		MAC:         mac,
		IP:          u.IP,
		Time:        u.Time,
		Status:      u.Status,
		Balance:     u.Balance,
		FreeClaimed: u.FreeClaimed,
		Points:      u.Points,
		Hostname:    u.Hostname,
	}
}

func RoundFloat(val float64, precision int) float64 {
	// Simple rounding to N decimal places
	p := 1.0
	for i := 0; i < precision; i++ {
		p *= 10
	}
	return float64(int(val*p+0.5)) / p
}

func FormatHumanTime(seconds int) string {
	if seconds <= 0 {
		return "0s"
	}
	y := seconds / 31536000
	mo := (seconds % 31536000) / 2592000
	d := (seconds % 2592000) / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	var parts []string
	if y > 0 {
		parts = append(parts, fmt.Sprintf("%dy", y))
	}
	if mo > 0 {
		parts = append(parts, fmt.Sprintf("%dmo", mo))
	}
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if y == 0 && mo == 0 && d == 0 {
		if h > 0 {
			parts = append(parts, fmt.Sprintf("%dh", h))
		}
		if m > 0 {
			parts = append(parts, fmt.Sprintf("%dm", m))
		}
		parts = append(parts, fmt.Sprintf("%ds", s))
	} else {
		parts = append(parts, fmt.Sprintf("%dh %dm", h, m))
	}
	return strings.Join(parts, " ")
}
