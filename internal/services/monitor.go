package services

import (
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/network"
	"pisowifi/internal/state"
)

// ---------------------------------------------------------------------------
// NetworkMonitorService — replaces services/network_monitor.py
// ---------------------------------------------------------------------------

// EvaluateAllConnections checks each connected user's traffic counters.
// If a user has been idle for longer than inactive_timeout, they are auto-paused.
func EvaluateAllConnections() {
	cfg := config.Get()
	if !cfg.AutoPauseEnabled {
		return
	}

	timeoutLimit := float64(cfg.InactiveTimeout)
	packetLimit := int64(cfg.InactivePacketThreshold)
	bytesLimit := int64(cfg.InactiveBytesThreshold)
	now := float64(time.Now().UnixNano()) / 1e9

	allTraffic := network.GetAllTraffic() // map[mac][2]int64{bytes, packets}

	state.Users.Range(func(mac string, u *state.UserRecord) {
		if u.Status != "connected" {
			return
		}

		curr, hasStat := allTraffic[mac]
		currBytes := int64(0)
		currPackets := int64(0)
		if hasStat {
			currBytes = curr[0]
			currPackets = curr[1]
		}

		prevBytes := u.LastByteCount
		prevPackets := u.LastPacketCount

		// Baseline init — first time we see traffic
		if prevBytes == 0 && currBytes > 0 {
			state.Users.UpdateField(mac, func(u *state.UserRecord) {
				u.LastByteCount = currBytes
				u.LastPacketCount = currPackets
				u.LastActive = now
			})
			return
		}

		diffBytes := currBytes - prevBytes
		diffPackets := currPackets - prevPackets
		if diffBytes < 0 {
			diffBytes = 0
		}
		if diffPackets < 0 {
			diffPackets = 0
		}

		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			u.LastByteCount = currBytes
			u.LastPacketCount = currPackets
		})

		isActive := (diffBytes > bytesLimit) || (diffPackets >= packetLimit)

		if isActive {
			state.Users.UpdateField(mac, func(u *state.UserRecord) {
				u.LastActive = now
			})
			return
		}

		// Check idle time
		idleTime := now - u.LastActive
		if idleTime > timeoutLimit {
			state.Users.UpdateField(mac, func(u *state.UserRecord) {
				u.Status = "paused"
			})
			network.BlockUser(mac, u.IP)
			if fresh, ok := state.Users.Get(mac); ok {
				db.SyncUser(toDBRecord(mac, fresh))
			}
			state.Manager.Send(mac, map[string]any{
				"type":           "sync",
				"status":         "paused",
				"time_remaining": u.Time,
			})
		}
	})
}
