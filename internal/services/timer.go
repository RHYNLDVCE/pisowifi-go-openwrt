package services

import (
	"fmt"
	"os/exec"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/hardware"
	"pisowifi/internal/logger"
	"pisowifi/internal/network"
	"pisowifi/internal/state"
)

// ---------------------------------------------------------------------------
// TimerService — replaces services/timer_service.py
// ---------------------------------------------------------------------------

var rebootTriggered bool

// CheckRebootSchedule runs every 5 ticks (~5 s). Triggers a system reboot
// at the configured time (HH:MM) if enabled.
func CheckRebootSchedule() {
	cfg := config.Get()
	if !cfg.RestartSchedule.Enabled {
		rebootTriggered = false
		return
	}

	currentTime := time.Now().Format("15:04")
	target := cfg.RestartSchedule.Time

	if currentTime == target && !rebootTriggered {
		rebootTriggered = true

		// Notify all portal users
		state.Users.Range(func(mac string, _ *state.UserRecord) {
			state.Manager.Send(mac, map[string]any{
				"type":    "system_message",
				"message": "System is restarting. Please wait...",
			})
		})

		time.Sleep(3 * time.Second)

		// Save all users to DB
		state.Users.Range(func(mac string, u *state.UserRecord) {
			if u.Status != "new" {
				db.SyncUser(toDBRecord(mac, u))
			}
		})

		exec.Command("sync").Run()
		time.Sleep(2 * time.Second)
		exec.Command("sudo", "systemctl", "reboot").Run()
	}

	if currentTime != target {
		rebootTriggered = false
	}
}

// TickUsers decrements connected users via wall-clock expiry, sends
// periodic WS syncs, and batch-writes to DB every 30 ticks.
func TickUsers(ticks int) {
	now := float64(time.Now().UnixNano()) / 1e9
	var toSync []db.UserRecord

	state.Users.Range(func(mac string, u *state.UserRecord) {
		if u.Status == "connected" {
			if u.ExpiresAt == 0 {
				// Safety: reconstruct deadline
				state.Users.UpdateField(mac, func(u *state.UserRecord) {
					u.ExpiresAt = now + float64(u.Time)
				})
				u, _ = state.Users.Get(mac)
			}

			timeLeft := u.ExpiresAt - now
			newTime := int(timeLeft)
			if newTime < 0 {
				newTime = 0
			}

			state.Users.UpdateField(mac, func(u *state.UserRecord) {
				u.Time = newTime
			})

			if timeLeft <= 0 {
				state.Users.UpdateField(mac, func(u *state.UserRecord) {
					u.Time = 0
					u.Status = "expired"
					u.ExpiresAt = 0
				})
				logger.SystemLog(fmt.Sprintf("[TIMER] User %s (IP: %s) out of time. Disconnecting...", mac, u.IP))
				network.BlockUser(mac, u.IP)
				if updated, ok := state.Users.Get(mac); ok {
					toSync = append(toSync, toDBRecord(mac, updated))
				}
				return
			}
		}

		// Batch DB sync every 30 ticks
		if ticks >= 30 && u.Status == "connected" {
			if fresh, ok := state.Users.Get(mac); ok {
				toSync = append(toSync, toDBRecord(mac, fresh))
			}
		}

		// WS sync every 5 ticks
		if ticks%5 == 0 {
			if fresh, ok := state.Users.Get(mac); ok {
				state.Manager.Send(mac, map[string]any{
					"type":           "sync",
					"time_remaining": fresh.Time,
					"status":         fresh.Status,
					"balance":        fresh.Balance,
					"points":         fresh.Points,
				})
			}
		}
	})

	if len(toSync) > 0 {
		db.SyncMultipleUsers(toSync)
	}
}

// CheckSlotExpiry closes the coin slot if its timer has elapsed.
func CheckSlotExpiry() {
	slotUser := state.GetSlotUser()
	if slotUser == "" {
		return
	}
	cfg := config.Get()
	if float64(time.Now().Unix()) >= cfg.SlotExpiryTimestamp {
		state.Manager.Send(slotUser, map[string]any{"type": "slot_closed"})
		hardware.TurnSlotOff()
		state.SetSlotUser("")
	}
}
