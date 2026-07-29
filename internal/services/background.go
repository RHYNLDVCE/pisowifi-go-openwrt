package services

import (
	"fmt"
	"sync"
	"time"

	"pisowifi/internal/hardware"
	"pisowifi/internal/logger"
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
	Wg.Add(3)
	go coinListener()
	go timeManager()
	go connectivityMonitor()
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
				logger.SystemLog(fmt.Sprintf("[COIN_INSERT] %d pulse(s) by Device: %s", coinValue, userLog))

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

// connectivityMonitor checks idle users every 15 seconds.
// Mirrors _connectivity_monitor() from background.py.
func connectivityMonitor() {
	defer Wg.Done()
	logger.SystemLog("Connectivity Monitor STARTED.")
	for {
		if state.IsShuttingDown.Load() {
			return
		}
		time.Sleep(15 * time.Second)

		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.SystemLog(fmt.Sprintf("CRITICAL ERROR in Monitor loop: %v", r))
				}
			}()
			EvaluateAllConnections()
		}()
	}
}
