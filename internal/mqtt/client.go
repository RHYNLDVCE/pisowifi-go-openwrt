package mqtt

// client.go — persistent MQTT client connecting the Orange Pi to the
// OpenWrt router's Mosquitto broker.
//
// The Orange Pi NEVER enforces firewall rules locally. It only publishes
// commands here; the router-side subscriber script applies them.

import (
	"encoding/json"
	"fmt"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"pisowifi/internal/logger"
	"pisowifi/internal/state"
)

const (
	// QoS 1 — at-least-once delivery. Good enough for allow/block commands;
	// if the broker is momentarily down the publish will fail and log an error.
	qos = 1

	// connectTimeout is how long we wait for the initial broker connection.
	connectTimeout = 5 * time.Second

	// reconnectWait is the time between automatic reconnect attempts.
	reconnectWait = 5 * time.Second
)

var mqttClient paho.Client

// Init creates and connects the MQTT client to the router broker.
// brokerURL example: "tcp://10.0.0.1:1883"
// clientID:          "pisowifi-orangepi"
func Init(brokerURL, clientID, username, password string, onConnectCb func()) {
	opts := paho.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	// Username/password auth is not used — access is secured by a firewall rule
	// on the router that only allows the Orange Pi (10.0.0.2) to reach port 1883.

	// Fast KeepAlive & PingTimeout so connection drop is detected in ~3 seconds
	opts.SetKeepAlive(3 * time.Second)
	opts.SetPingTimeout(2 * time.Second)

	// Auto-reconnect is handled by paho natively
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(reconnectWait)
	opts.SetCleanSession(false) // persist QoS 1 subscriptions across reconnects

	// Last Will and Testament (LWT)
	// If this client disconnects ungracefully, the broker will automatically publish this message
	opts.SetWill("pisowifi/lwt", `{"status":"offline"}`, qos, true)

	opts.SetOnConnectHandler(func(c paho.Client) {
		logger.SystemLog("[MQTT] Connected to broker: " + brokerURL)
		
		// Clear the LWT by publishing online status in a non-blocking goroutine
		go func() {
			if token := c.Publish("pisowifi/lwt", qos, true, []byte(`{"status":"online"}`)); token.Wait() && token.Error() != nil {
				logger.SystemLog(fmt.Sprintf("[MQTT] Failed to publish online status: %v", token.Error()))
			}
			
			if onConnectCb != nil {
				onConnectCb()
			}
		}()
	})
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		logger.SystemLog(fmt.Sprintf("[MQTT] Connection to router lost: %v — freezing active user timers...", err))
		now := float64(time.Now().UnixNano()) / 1e9
		state.Users.Range(func(mac string, u *state.UserRecord) {
			if u.Status == "connected" && u.ExpiresAt > 0 {
				rem := u.ExpiresAt - now
				if rem < 0 {
					rem = 0
				}
				state.Users.UpdateField(mac, func(ur *state.UserRecord) {
					ur.Time = int(rem)
					ur.ExpiresAt = 0
				})
			}
		})
	})
	opts.SetReconnectingHandler(func(_ paho.Client, _ *paho.ClientOptions) {
		logger.SystemLog("[MQTT] Attempting to reconnect to broker...")
	})

	mqttClient = paho.NewClient(opts)

	token := mqttClient.Connect()
	if token.WaitTimeout(connectTimeout) {
		if err := token.Error(); err != nil {
			logger.SystemLog(fmt.Sprintf("[MQTT] Initial connect failed: %v — will retry in background", err))
		}
	} else {
		logger.SystemLog("[MQTT] Initial connect timed out — will retry in background")
	}
}

// Publish sends a JSON-encoded payload to the given topic (QoS 1, non-retained).
// It is safe to call from any goroutine. If the client is not connected the
// message is dropped and an error is logged — no fallback, by design.
func Publish(topic string, payload interface{}) error {
	if mqttClient == nil {
		logger.SystemLog("[MQTT] Publish attempted before Init()")
		return fmt.Errorf("mqtt client not initialized")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mqtt marshal: %w", err)
	}

	token := mqttClient.Publish(topic, qos, false, data)
	// WaitTimeout returns false on timeout, true otherwise
	if !token.WaitTimeout(2 * time.Second) {
		logger.SystemLog(fmt.Sprintf("[MQTT] Publish timed out: topic=%s", topic))
		return fmt.Errorf("mqtt publish timeout: %s", topic)
	}
	if err := token.Error(); err != nil {
		logger.SystemLog(fmt.Sprintf("[MQTT] Publish error: topic=%s err=%v", topic, err))
		return err
	}
	return nil
}

// PublishRetained sends a JSON-encoded payload to the given topic with the Retained flag set to true.
// This ensures that late subscribers (like the router script on a slow boot) will still receive the message.
func PublishRetained(topic string, payload interface{}) error {
	if mqttClient == nil {
		logger.SystemLog("[MQTT] PublishRetained attempted before Init()")
		return fmt.Errorf("mqtt client not initialized")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mqtt marshal: %w", err)
	}

	token := mqttClient.Publish(topic, qos, true, data)
	if !token.WaitTimeout(2 * time.Second) {
		logger.SystemLog(fmt.Sprintf("[MQTT] PublishRetained timed out: topic=%s", topic))
		return fmt.Errorf("mqtt publish timeout: %s", topic)
	}
	if err := token.Error(); err != nil {
		logger.SystemLog(fmt.Sprintf("[MQTT] PublishRetained error: topic=%s err=%v", topic, err))
		return err
	}
	return nil
}

// Subscribe registers a handler for the given topic pattern (QoS 1).
// Used for receiving ACKs / traffic stats from the router.
func Subscribe(topic string, handler paho.MessageHandler) {
	if mqttClient == nil {
		return
	}
	mqttClient.Subscribe(topic, qos, handler)
}

// IsConnected returns true if the client currently has an active connection.
func IsConnected() bool {
	return mqttClient != nil && mqttClient.IsConnected()
}

// Disconnect cleanly disconnects from the broker during graceful shutdown.
func Disconnect() {
	if mqttClient != nil && mqttClient.IsConnected() {
		mqttClient.Disconnect(500) // wait up to 500 ms to flush
		logger.SystemLog("[MQTT] Disconnected from broker.")
	}
}
