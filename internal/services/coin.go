package services

import (
	"fmt"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/logger"
	"pisowifi/internal/state"
)

// ---------------------------------------------------------------------------
// CoinService — replaces services/coin_service.py
// ---------------------------------------------------------------------------

// NotifyCounting sends a "hardware is counting" signal to the user's portal WebSocket.
func NotifyCounting(mac string) {
	if mac == "" {
		return
	}
	state.Manager.Send(mac, map[string]any{
		"type":        "coin_counting",
		"is_counting": true,
	})
}

// NotifyDoneCounting tells the portal the counting phase is finished.
func NotifyDoneCounting(mac string) {
	if mac == "" {
		return
	}
	state.Manager.Send(mac, map[string]any{
		"type":        "coin_counting",
		"is_counting": false,
	})
}

// ProcessCoin credits `pulses * pulse_value` to the user's balance and
// persists to DB. Mirrors CoinService.process_coin exactly.
func ProcessCoin(pulses int, mac string) {
	if pulses <= 0 || mac == "" {
		logger.SystemLog("[WARNING] Coin processed but no valid MAC address found.")
		return
	}

	user, ok := state.Users.Get(mac)
	if !ok {
		logger.SystemLog("[WARNING] Coin processed but MAC not in user store.")
		return
	}

	if user.Status == "blocked" {
		return
	}

	// Extend portal slot timeout while the user is dropping coins
	cfg := config.Get()
	config.Update(func(c *config.AppConfig) {
		c.SlotExpiryTimestamp = float64(time.Now().Unix()) + float64(c.SlotTimeout)
	})

	pulseValue := cfg.PulseValue
	if pulseValue <= 0 {
		pulseValue = 1
	}
	amount := pulses * pulseValue

	state.Users.UpdateField(mac, func(u *state.UserRecord) {
		u.Balance += amount
		u.LastActive = float64(time.Now().UnixNano()) / 1e9
	})

	// Persist
	db.SyncUser(db.UserRecord{
		MAC: mac, IP: user.IP, Time: user.Time, Status: user.Status,
		Balance: user.Balance, FreeClaimed: user.FreeClaimed, Points: user.Points,
	})
	db.AddSale(mac, amount)

	logger.SystemLog(fmt.Sprintf("[COIN_SUCCESS] Credited %d to %s. Balance: %d", amount, mac, user.Balance))

	// Re-read updated balance for WS messages
	updated, _ := state.Users.Get(mac)
	bal := 0
	pts := 0.0
	trem := 0
	st := ""
	if updated != nil {
		bal = updated.Balance
		pts = updated.Points
		trem = updated.Time
		st = updated.Status
	}

	state.Manager.Send(mac, map[string]any{
		"type":         "coin_inserted",
		"inserted":     amount,
		"balance":      bal,
		"points":       pts,
		"slot_seconds": cfg.SlotTimeout,
		"pulse_value":  pulseValue,
	})
	state.Manager.Send(mac, map[string]any{
		"type":           "sync",
		"balance":        bal,
		"time_remaining": trem,
		"points":         pts,
		"status":         st,
	})
}
