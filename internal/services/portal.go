package services

import (
	"fmt"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/hardware"
	"pisowifi/internal/logger"
	"pisowifi/internal/network"
	"pisowifi/internal/state"
)

// EnableSlot tries to assign the hardware coin slot to the specified MAC address.
func EnableSlot(mac string) (string, int, int, float64, int) {
	cfg := config.Get()
	user, ok := state.Users.Get(mac)
	if ok && user.Status == "blocked" {
		return "blocked", 0, 0, 0, 0
	}

	slotUser := state.GetSlotUser()
	if slotUser == "" || slotUser == mac {
		state.SetSlotUser(mac)
		hardware.TurnSlotOn()
		config.Update(func(cfg *config.AppConfig) {
			cfg.SlotExpiryTimestamp = float64(time.Now().Unix()) + float64(cfg.SlotTimeout)
		})
		logger.SystemLog(fmt.Sprintf("[PORTAL_EVENT] SLOT OPENED by Device: %s", mac))

		bal := 0
		pts := 0.0
		trem := 0
		if user != nil {
			bal = user.Balance
			pts = user.Points
			trem = user.Time
		}

		state.Manager.Send(mac, map[string]interface{}{
			"type":           "slot_opened",
			"slot_seconds":   cfg.SlotTimeout,
			"balance":        bal,
			"points":         pts,
			"coin_rates":     cfg.CoinRates,
			"time_remaining": trem,
		})
		return "success", cfg.SlotTimeout, bal, pts, trem
	}
	return "busy", 0, 0, 0, 0
}

// CancelSlot closes the hardware coin slot if it belongs to the given MAC address.
func CancelSlot(mac string) bool {
	if state.GetSlotUser() == mac {
		hardware.TurnSlotOff()
		state.SetSlotUser("")
		config.Update(func(cfg *config.AppConfig) {
			cfg.SlotExpiryTimestamp = 0
		})
		return true
	}
	return false
}

// ClaimFreeTime awards free time to the specified MAC if allowed.
func ClaimFreeTime(mac string) string {
	cfg := config.Get()
	if !cfg.FreeTimeEnabled {
		return "disabled"
	}

	user, ok := state.Users.Get(mac)
	if !ok {
		return "error"
	}
	if user.FreeClaimed == 1 {
		return "already_claimed"
	}

	duration := cfg.FreeTimeDuration
	now := float64(time.Now().UnixNano()) / 1e9

	state.Users.UpdateField(mac, func(u *state.UserRecord) {
		u.Time += duration * 60
		u.FreeClaimed = 1
		u.Status = "connected"
		u.LastActive = now
		u.ExpiresAt = now + float64(u.Time)
	})

	network.AllowUser(mac, user.IP)
	if state.GetSlotUser() == mac {
		hardware.TurnSlotOff()
		state.SetSlotUser("")
	}

	if fresh, ok := state.Users.Get(mac); ok {
		db.SyncUser(db.UserRecord{
			MAC: mac, IP: fresh.IP, Time: fresh.Time, Status: fresh.Status,
			Balance: fresh.Balance, FreeClaimed: fresh.FreeClaimed, Points: fresh.Points,
		})
		logger.SystemLog(fmt.Sprintf("[%s | %s] Claimed %d mins of Free Time.", user.IP, mac, duration))
		state.Manager.Send(mac, map[string]interface{}{
			"type": "sync", "status": "connected", "time_remaining": fresh.Time,
			"balance": 0, "points": fresh.Points,
		})
	}
	return "success"
}

// RedeemPoints deducts points and grants time based on the promo ID.
func RedeemPoints(mac string, ip string, promoID int64) (string, string) {
	cfg := config.Get()
	if mac == "" {
		return "error", "User not found"
	}
	if !cfg.PointsEnabled {
		return "error", "Rewards disabled."
	}

	user, ok := state.Users.Get(mac)
	if !ok {
		return "error", "User not found"
	}

	var promo *config.PromoItem
	for _, p := range cfg.PointPromos {
		if p.ID == promoID {
			pp := p
			promo = &pp
			break
		}
	}
	if promo == nil {
		return "error", "Invalid Promo"
	}
	if user.Points < promo.Cost {
		return "error", "Not enough points"
	}

	now := float64(time.Now().UnixNano()) / 1e9
	state.Users.UpdateField(mac, func(u *state.UserRecord) {
		u.Points = RoundFloat(u.Points-promo.Cost, 2)
		u.Time += promo.Minutes * 60
		u.Status = "connected"
		u.LastActive = now
		u.ExpiresAt = now + float64(u.Time)
	})

	network.AllowUser(mac, user.IP)

	if fresh, ok := state.Users.Get(mac); ok {
		db.SyncUser(db.UserRecord{
			MAC: mac, IP: fresh.IP, Time: fresh.Time, Status: fresh.Status,
			Balance: fresh.Balance, FreeClaimed: fresh.FreeClaimed, Points: fresh.Points,
		})
		logger.SystemLog(fmt.Sprintf("[%s | %s] Redeemed '%s' for %.1f points.", ip, mac, promo.Name, promo.Cost))
		state.Manager.Send(mac, map[string]interface{}{
			"type": "sync", "status": "connected",
			"time_remaining": fresh.Time, "points": fresh.Points,
		})
	}
	return "success", fmt.Sprintf("Successfully redeemed: %s", promo.Name)
}
