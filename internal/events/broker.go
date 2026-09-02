package events

import (
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
