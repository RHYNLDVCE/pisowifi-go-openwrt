package services

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/logger"
	"pisowifi/internal/network"
	"pisowifi/internal/state"
)

// GenerateVoucherCode creates a random 8-character code like "8XY2P9KL"
func GenerateVoucherCode() string {
	b := make([]byte, 5)
	rand.Read(b)
	str := base32.StdEncoding.EncodeToString(b)
	str = strings.ReplaceAll(str, "=", "")
	return str[:8]
}

// GenerateVoucher creates a new voucher from the user's balance.
func GenerateVoucher(mac, vType string, value float64) (string, error) {
	cfg := config.Get()

	// 1. Validate inputs
	if vType != "time" && vType != "points" {
		return "", fmt.Errorf("invalid voucher type")
	}

	if vType == "time" && value < float64(cfg.VoucherMinTimeMinutes) {
		return "", fmt.Errorf("amount below minimum limit (%d mins)", cfg.VoucherMinTimeMinutes)
	}

	var selectedPromo *config.PromoItem
	if vType == "points" {
		promoID := int64(value)
		for _, p := range cfg.VoucherPointPromos {
			if p.ID == promoID {
				// Make a copy so we can take its address or just assign
				sp := p
				selectedPromo = &sp
				break
			}
		}
		if selectedPromo == nil {
			return "", fmt.Errorf("invalid promo selection")
		}
	}

	// 2. Check user balance
	user, ok := state.Users.Get(mac)
	if !ok {
		return "", fmt.Errorf("user not found")
	}

	// 3. Deduct balance
	if vType == "time" {
		secondsToDeduct := int(value * 60)
		if user.Time < secondsToDeduct {
			return "", fmt.Errorf("insufficient time balance")
		}
		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			u.Time -= secondsToDeduct
			if u.Time < 0 {
				u.Time = 0
			}
			if u.Status == "connected" {
				u.ExpiresAt = float64(time.Now().UnixNano())/1e9 + float64(u.Time)
			}
			if u.Time == 0 && u.Status == "connected" {
				u.Status = "expired"
				u.ExpiresAt = 0
				network.BlockUser(mac, u.IP)
			}
		})
	} else if vType == "points" {
		if user.Points < selectedPromo.Cost {
			return "", fmt.Errorf("insufficient points balance")
		}
		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			u.Points -= selectedPromo.Cost
		})
		
		// A points voucher is actually just a time voucher now.
		vType = "time"
		value = float64(selectedPromo.Minutes)
	}

	// Update DB to reflect the new balance
	user, _ = state.Users.Get(mac)
	db.SyncUser(db.UserRecord{
		MAC:         mac,
		IP:          user.IP,
		Time:        user.Time,
		Status:      user.Status,
		Balance:     user.Balance,
		FreeClaimed: user.FreeClaimed,
		Points:      user.Points,
	})

	// 4. Create voucher
	var code string
	for {
		code = GenerateVoucherCode()
		if _, err := db.GetVoucher(code); err != nil {
			break // code doesn't exist
		}
	}

	v := &db.VoucherRecord{
		Code:      code,
		Type:      vType,
		Value:     value,
		Status:    "active",
		CreatedBy: mac,
		UsedBy:    "",
		CreatedAt: time.Now().Unix(),
		UsedAt:    0,
	}

	err := db.CreateVoucher(v)
	if err != nil {
		return "", fmt.Errorf("database error")
	}

	state.Manager.Send(mac, map[string]any{
		"type":           "sync",
		"time_remaining": user.Time,
		"status":         user.Status,
		"balance":        user.Balance,
		"points":         user.Points,
	})

	logger.SystemLog(fmt.Sprintf("[VOUCHER] Created: %s | Type: %s | Value: %.2f | By: %s", code, vType, value, mac))
	return code, nil
}

// RedeemVoucher marks a voucher as used and credits the user.
func RedeemVoucher(mac, code string) error {
	// 1. Get voucher
	v, err := db.GetVoucher(code)
	if err != nil {
		return fmt.Errorf("invalid voucher code")
	}

	if v.Status != "active" {
		return fmt.Errorf("voucher already used")
	}

	// 2. Ensure user exists
	if _, ok := state.Users.Get(mac); !ok {
		state.Users.Set(mac, &state.UserRecord{
			Status: "new", Time: 0, Balance: 0, FreeClaimed: 0, Points: 0,
		})
	}

	// 3. Credit the user
	if v.Type == "time" {
		secondsToAdd := int(v.Value * 60)
		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			u.Time += secondsToAdd
			if u.Time > 0 && (u.Status == "expired" || u.Status == "new" || u.Status == "") {
				u.Status = "connected"
				u.ExpiresAt = float64(time.Now().UnixNano())/1e9 + float64(u.Time)
				network.AllowUser(mac, u.IP)
			} else if u.Status == "connected" {
				u.ExpiresAt = float64(time.Now().UnixNano())/1e9 + float64(u.Time)
			}
		})
	} else if v.Type == "points" {
		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			u.Points += v.Value
		})
	}

	// 4. Update DB user
	user, _ := state.Users.Get(mac)
	db.SyncUser(db.UserRecord{
		MAC:         mac,
		IP:          user.IP,
		Time:        user.Time,
		Status:      user.Status,
		Balance:     user.Balance,
		FreeClaimed: user.FreeClaimed,
		Points:      user.Points,
	})

	// 5. Mark voucher used
	err = db.MarkVoucherUsed(code, mac)
	if err != nil {
		return fmt.Errorf("database error")
	}

	state.Manager.Send(mac, map[string]any{
		"type":           "sync",
		"time_remaining": user.Time,
		"status":         user.Status,
		"balance":        user.Balance,
		"points":         user.Points,
	})

	logger.SystemLog(fmt.Sprintf("[VOUCHER] Redeemed: %s | Type: %s | Value: %.2f | By: %s", code, v.Type, v.Value, mac))
	return nil
}
