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

	// workerPublishTimeout is how long the background worker waits for a single
	// PUBACK from the broker. Longer than the old 2s caller timeout because the
	// worker is not blocking any hot path — it can afford to be patient.
	workerPublishTimeout = 10 * time.Second

	// publishChSize is the capacity of the async publish channel. 256 slots is
	// enough to absorb a full burst of block+removeSpeedLimit commands for every
	// user even on a large deployment, without ever blocking the caller.
	publishChSize = 256
)

var mqttClient paho.Client

// mqttJob is a single fire-and-forget publish request queued by Publish().
type mqttJob struct {
	topic   string
	payload []byte
}

// publishCh is the channel between callers and the background worker goroutine.
// Callers enqueue and return immediately; the worker drains at broker speed.
var publishCh chan mqttJob

// Init creates and connects the MQTT client to the router broker.
// brokerURL example: "tcp://10.0.0.1:1883"
// clientID:          "pisowifi-orangepi"
func Init(brokerURL, clientID, username, password string, onConnectCb func()) {
	// Create the async publish channel and start the single worker goroutine.
	// The worker is the ONLY goroutine that calls mqttClient.Publish for
	// non-retained messages, so it serialises all outgoing commands without
	// ever blocking the caller (timer loop, session handlers, etc.).
	publishCh = make(chan mqttJob, publishChSize)
	go publishWorker()

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
	opts.SetOrderMatters(false) // process incoming messages concurrently to prevent PUBACK deadlocks

	// Last Will and Testament (LWT)
	// If this client disconnects ungracefully, the broker will automatically publish this message
	opts.SetWill("pisowifi/lwt", `{"status":"offline"}`, qos, true)

	opts.SetOnConnectHandler(func(c paho.Client) {
		logger.SystemLog("[MQTT] Connected to broker: " + brokerURL)
		
		// Clear the LWT by publishing online status in a non-blocking goroutine
		go func() {
			if token := c.Publish("pisowifi/lwt", qos, true, []byte(`{"status":"online"}`)); token.Wait() && token.Error() != nil {
				logger.SystemLog(fmt.Sprintf("[MQTT] [ERROR] Failed to publish online status: %v", token.Error()))
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
			logger.SystemLog(fmt.Sprintf("[MQTT] [ERROR] Initial connect failed: %v — will retry in background", err))
		}
	} else {
		logger.SystemLog("[MQTT] Initial connect timed out — will retry in background")
	}
}

// publishWorker is the single background goroutine that drains publishCh.
// It is the only place that calls mqttClient.Publish for non-retained messages,
// so Paho's write path is never contended by multiple goroutines at once.
// It runs until publishCh is closed (i.e. during Disconnect).
func publishWorker() {
	for job := range publishCh {
		if mqttClient == nil {
			continue
		}
		token := mqttClient.Publish(job.topic, qos, false, job.payload)
		if !token.WaitTimeout(workerPublishTimeout) {
			logger.SystemLog(fmt.Sprintf("[MQTT] [WORKER] Publish timed out: topic=%s", job.topic))
		} else if err := token.Error(); err != nil {
			logger.SystemLog(fmt.Sprintf("[MQTT] [WORKER] Publish error: topic=%s err=%v", job.topic, err))
		}
	}
}

// Publish enqueues a JSON-encoded payload for async delivery (QoS 1, non-retained).
// It returns immediately without waiting for a broker PUBACK — the background
// worker handles the actual send. If the channel is full (broker has been
// unreachable long enough to fill 256 slots) the message is dropped and logged.
// It is safe to call from any goroutine.
func Publish(topic string, payload interface{}) error {
	if mqttClient == nil {
		logger.SystemLog("[MQTT] Publish attempted before Init()")
		return fmt.Errorf("mqtt client not initialized")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mqtt marshal: %w", err)
	}

	select {
	case publishCh <- mqttJob{topic: topic, payload: data}:
		return nil
	default:
		// Channel full — broker has likely been down for an extended period.
		logger.SystemLog(fmt.Sprintf("[MQTT] [WARN] Publish channel full, dropping message: topic=%s", topic))
		return fmt.Errorf("mqtt publish channel full: %s", topic)
	}
}

// PublishRetained sends a JSON-encoded payload to the given topic with the
// Retained flag set to true. This is intentionally synchronous because it is
// only called at startup (firewall/init) or on explicit config reloads where
// the caller needs to know the message was delivered before proceeding.
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
	if !token.WaitTimeout(10 * time.Second) {
		logger.SystemLog(fmt.Sprintf("[MQTT] PublishRetained timed out: topic=%s", topic))
		return fmt.Errorf("mqtt publish timeout: %s", topic)
	}
	if err := token.Error(); err != nil {
		logger.SystemLog(fmt.Sprintf("[MQTT] [ERROR] PublishRetained error: topic=%s err=%v", topic, err))
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

// Disconnect drains any pending publish jobs, then cleanly disconnects from
// the broker during graceful shutdown.
func Disconnect() {
	if publishCh != nil {
		close(publishCh) // signal the worker to stop after draining
		publishCh = nil
	}
	if mqttClient != nil && mqttClient.IsConnected() {
		mqttClient.Disconnect(1000) // wait up to 1s to flush in-flight messages
		logger.SystemLog("[MQTT] Disconnected from broker.")
	}
}
