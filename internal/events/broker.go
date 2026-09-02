package events

import (
	"fmt"
	"strings"
	"sync"
)

// AdminEvent represents a real-time event sent to the admin dashboard
type AdminEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// SSEBroker manages real-time Server-Sent Events subscribers
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[chan AdminEvent]struct{}
}

// Global is the singleton broker for all admin real-time events
var Global = &SSEBroker{
	clients: make(map[chan AdminEvent]struct{}),
}

// Subscribe registers a new client channel for SSE broadcasts
func (b *SSEBroker) Subscribe() chan AdminEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan AdminEvent, 64)
	b.clients[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a client channel and closes it
func (b *SSEBroker) Unsubscribe(ch chan AdminEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.clients[ch]; ok {
		delete(b.clients, ch)
		close(ch)
	}
}

// Broadcast sends an event to all connected admin SSE clients non-blockingly
func (b *SSEBroker) Broadcast(event string, data interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.clients) == 0 {
		return
	}

	msg := AdminEvent{Event: event, Data: data}
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// Non-blocking: drop if buffer full to protect server
		}
	}
}

// ClientCount returns the current number of active SSE listeners
func (b *SSEBroker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// FormatHumanTime formats seconds into human-readable strings like "1h 30m 15s"
func FormatHumanTime(seconds int) string {
	if seconds <= 0 {
		return "0s"
	}
	y := seconds / 31536000
	mo := (seconds % 31536000) / 2592000
	d := (seconds % 2592000) / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	var parts []string
	if y > 0 {
		parts = append(parts, fmt.Sprintf("%dy", y))
	}
	if mo > 0 {
		parts = append(parts, fmt.Sprintf("%dmo", mo))
	}
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if y == 0 && mo == 0 && d == 0 {
		if h > 0 {
			parts = append(parts, fmt.Sprintf("%dh", h))
		}
		if m > 0 {
			parts = append(parts, fmt.Sprintf("%dm", m))
		}
		parts = append(parts, fmt.Sprintf("%ds", s))
	} else {
		parts = append(parts, fmt.Sprintf("%dh %dm", h, m))
	}
	return strings.Join(parts, " ")
}
