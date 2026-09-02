package network

import (
	"encoding/json"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"pisowifi/internal/db"
	"pisowifi/internal/events"
	"pisowifi/internal/logger"
	mqttcmd "pisowifi/internal/mqtt"
	"pisowifi/internal/state"
)

func InitACKs() {
	mqttcmd.Subscribe("pisowifi/ack/block", handleAckBlock)
	mqttcmd.Subscribe("pisowifi/ack/allow", handleAckAllow)
}

type ackPayload struct {
	MAC    string `json:"mac"`
	Status string `json:"status"`
}

func handleAckBlock(_ paho.Client, msg paho.Message) {
	var p ackPayload
	if err := json.Unmarshal(msg.Payload(), &p); err != nil {
		return
	}
	mac := strings.ToLower(strings.TrimSpace(p.MAC))
	if p.Status == "success" {
		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			if u.Status == "pending_pause" || u.Status == "connected" {
				// Finalize pause time calculation
				if u.ExpiresAt > 0 {
					remaining := u.ExpiresAt - float64(time.Now().UnixNano())/1e9
					if remaining < 0 {
						remaining = 0
					}
					u.Time = int(remaining)
					u.ExpiresAt = 0
				}
				u.Status = "paused"
			}
		})
		
		if u, ok := state.Users.Get(mac); ok {
			db.SyncUser(db.UserRecord{
				MAC:         mac,
				IP:          u.IP,
				Time:        u.Time,
				Status:      u.Status,
				Balance:     u.Balance,
				FreeClaimed: u.FreeClaimed,
				Points:      u.Points,
				Hostname:    u.Hostname,
			})
			state.Manager.Send(mac, map[string]any{
				"type":           "sync",
				"status":         "paused",
				"time_remaining": u.Time,
				"balance":        u.Balance,
				"points":         u.Points,
			})
			events.Global.Broadcast("user_update", map[string]interface{}{
				"mac":    mac,
				"status": u.Status,
				"time":   u.Time,
			})
		}
		logger.SystemLog("[ACK] Router successfully blocked: " + mac)
	} else {
		logger.SystemLog("[FIREWALL] [ERROR] Router failed to block: " + mac + " (Status: " + p.Status + ")")
	}
}

func handleAckAllow(_ paho.Client, msg paho.Message) {
	var p ackPayload
	if err := json.Unmarshal(msg.Payload(), &p); err != nil {
		return
	}
	mac := strings.ToLower(strings.TrimSpace(p.MAC))
	if p.Status == "success" {
		state.Users.UpdateField(mac, func(u *state.UserRecord) {
			if u.Status == "pending_connect" || u.Status == "paused" {
				u.Status = "connected"
				u.ExpiresAt = float64(time.Now().UnixNano())/1e9 + float64(u.Time)
			}
		})

		if u, ok := state.Users.Get(mac); ok {
			db.SyncUser(db.UserRecord{
				MAC:         mac,
				IP:          u.IP,
				Time:        u.Time,
				Status:      u.Status,
				Balance:     u.Balance,
				FreeClaimed: u.FreeClaimed,
				Points:      u.Points,
				Hostname:    u.Hostname,
			})
			state.Manager.Send(mac, map[string]any{
				"type":           "sync",
				"status":         "connected",
				"time_remaining": u.Time,
				"balance":        0,
				"points":         u.Points,
			})
			events.Global.Broadcast("user_connected", map[string]interface{}{
				"mac":          mac,
				"ip":           u.IP,
				"time":         u.Time,
				"status":       "connected",
				"status_short": "c",
				"balance":      u.Balance,
				"points":       u.Points,
				"device_name":  u.Hostname,
			})
		}
		logger.SystemLog("[ACK] Router successfully allowed: " + mac)
	} else {
		logger.SystemLog("[FIREWALL] [ERROR] Router failed to allow: " + mac + " (Status: " + p.Status + ")")
	}
}
