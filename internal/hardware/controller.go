package hardware

/*
#cgo LDFLAGS: -lwiringPi
#include <wiringPi.h>
*/
import "C"
import (
	"fmt"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/logger"
	"pisowifi/internal/state"
)

var (
	coinPin  C.int
	relayPin C.int
)

// Setup initialises the coin input pin and the relay output pin using WiringOP (CGO).
func Setup() {
	if C.wiringPiSetupPhys() == -1 {
		logger.SystemLog("[HW] Fatal: Failed to setup wiringPi")
		return
	}

	coinPin = C.int(config.CoinPinPhys)
	relayPin = C.int(config.RelayPinPhys)

	// Set Relay as Output, default to OFF (0)
	C.pinMode(relayPin, C.OUTPUT)
	C.digitalWrite(relayPin, 0)

	// Set Coin as Input, with Pull-Up
	C.pinMode(coinPin, C.INPUT)
	C.pullUpDnControl(coinPin, C.PUD_UP)

	logger.SystemLog(fmt.Sprintf("[HW] Hardware ready (Coin: %d, Relay: %d) via WiringOP", config.CoinPinPhys, config.RelayPinPhys))
}

// TurnSlotOn powers the coin slot relay HIGH.
func TurnSlotOn() {
	C.digitalWrite(relayPin, 1)
}

// TurnSlotOff powers the relay LOW and clears the current slot user.
func TurnSlotOff() {
	C.digitalWrite(relayPin, 0)
}

// ReadPin returns the current coin signal pin value (0 = coin pulse detected, 1 = idle).
func ReadPin() int {
	return int(C.digitalRead(coinPin))
}

// WaitForPulse blocks until one or more pulses are detected and counted.
func WaitForPulse(onDetected func()) int {
	readPinFast := func() int { return int(C.digitalRead(coinPin)) }
	lastState := 1

	// SAFETY: If pin is stuck LOW (0), wait for it to clear.
	if readPinFast() == 0 {
		logger.SystemLog("   [Warning] Signal Stuck LOW. Waiting for clear...")
		timeout := time.Now()
		for readPinFast() == 0 {
			if time.Since(timeout) > 2*time.Second {
				logger.SystemLog("   [Error] Signal permanently stuck LOW. Resetting...")
				return 0
			}
			time.Sleep(10 * time.Millisecond)
		}
		logger.SystemLog("   [OK] Signal Cleared. Ready.")
	}

	// PHASE 1: Wait for coin
	for {
		if state.IsShuttingDown.Load() {
			return 0
		}

		if state.GetSlotUser() == "" {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		pinState := readPinFast()
		if pinState == 0 && lastState == 1 {
			// Coin dropped!
			if onDetected != nil {
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.SystemLog(fmt.Sprintf("Callback Error: %v", r))
						}
					}()
					onDetected()
				}()
			}
			break
		}
		lastState = pinState
		time.Sleep(10 * time.Millisecond)
	}

	// PHASE 2: Count pulses
	totalPulses := 1
	lastPulseTime := time.Now()
	lastState = 0

	// Wait for the pulse to finish (go back HIGH) with Timeout
	timeout := time.Now()
	for readPinFast() == 0 {
		if time.Since(timeout) > 500*time.Millisecond {
			logger.SystemLog("[HW] Pulse finish timeout (stuck LOW?)")
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	lastState = 1

	// Keep listening until silence
	for time.Since(lastPulseTime) < 600*time.Millisecond {
		pinState := readPinFast()
		if pinState == 0 && lastState == 1 {
			totalPulses++
			lastPulseTime = time.Now()

			timeout := time.Now()
			for readPinFast() == 0 {
				if time.Since(timeout) > 500*time.Millisecond {
					logger.SystemLog(fmt.Sprintf("[HW] Pulse %d timeout (stuck LOW?)", totalPulses))
					break
				}
				time.Sleep(1 * time.Millisecond)
			}
			pinState = 1
		}
		lastState = pinState
		time.Sleep(1 * time.Millisecond)
	}

	return totalPulses
}
