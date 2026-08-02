package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"pisowifi/internal/hardware"
	"pisowifi/internal/logger"
	"pisowifi/internal/mqtt"
	"pisowifi/internal/state"
)

// Wg tracks the background goroutines for graceful shutdown
var Wg sync.WaitGroup

// ---------------------------------------------------------------------------
// Background — replaces services/background.py
// Launches three goroutines: coin listener, timer, connectivity monitor.
// ---------------------------------------------------------------------------

// StartBackgroundTasks launches all three daemon goroutines.
func StartBackgroundTasks() {
	Wg.Add(2)
	go coinListener()
	go timeManager()

	// Subscribe to auto-pause events from the router
	mqtt.Subscribe("pisowifi/pause_user", func(_ paho.Client, msg paho.Message) {
		var p struct {
			MAC string `json:"mac"`
		}
		if err := json.Unmarshal(msg.Payload(), &p); err == nil && p.MAC != "" {
			logger.SystemLog(fmt.Sprintf("[AUTO-PAUSE] Router reported user %s is inactive.", p.MAC))
			PauseUser(p.MAC)
		}
	})

	// Subscribe to router system stats replies
	mqtt.Subscribe("pisowifi/router/stats/reply", func(_ paho.Client, msg paho.Message) {
		var stats map[string]interface{}
		if err := json.Unmarshal(msg.Payload(), &stats); err == nil {
			state.RouterStatsMu.Lock()
			state.RouterStats = stats
			state.RouterStatsMu.Unlock()
		}
	})
}

// coinListener polls the coin GPIO and credits users on pulse detection.
// Mirrors _coin_listener() from background.py.
func coinListener() {
	defer Wg.Done()
	logger.SystemLog("Coin Listener STARTED (Polling Mode).")
	for {
		if state.IsShuttingDown.Load() {
			return
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.SystemLog(fmt.Sprintf("CRITICAL ERROR in Coin loop: %v", r))
					time.Sleep(time.Second)
				}
			}()

			// Capture the active user at the moment the first pulse is detected
			var activeMac string

			onFirstPulse := func() {
				activeMac = state.GetSlotUser()
				NotifyCounting(activeMac)
			}

			coinValue := hardware.WaitForPulse(onFirstPulse)

			if coinValue > 0 {
				mac := activeMac
				userLog := mac
				if mac == "" {
					userLog = "Unknown_Device"
				}
				logger.SystemLog(fmt.Sprintf("[COIN-INSERT] %d pulse(s) by Device: %s", coinValue, userLog))

				if mac != "" {
					ProcessCoin(coinValue, mac)
					NotifyDoneCounting(mac)
				}
			}

			time.Sleep(100 * time.Millisecond)
		}()
	}
}

// timeManager ticks every 1 second, managing user time and scheduled tasks.
// Mirrors _time_manager() from background.py.
func timeManager() {
	defer Wg.Done()
	logger.SystemLog("Time Manager & Scheduler Started...")
	ticks := 0
	for {
		if state.IsShuttingDown.Load() {
			return
		}
		time.Sleep(time.Second)
		ticks++

		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.SystemLog(fmt.Sprintf("CRITICAL ERROR in Timer loop: %v", r))
				}
			}()

			if ticks%5 == 0 {
				CheckRebootSchedule()
			}
			TickUsers(ticks)
			CheckSlotExpiry()

			if ticks >= 30 {
				ticks = 0
			}
		}()
	}
}

