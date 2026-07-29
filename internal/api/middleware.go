package api

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// LoginRateLimiter — replaces the Python LoginLimiter class in auth.py
// ---------------------------------------------------------------------------

const (
	maxAttempts   = 5
	windowSeconds = 300 // 5 minutes
)

type LoginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]int64 // ip → list of Unix timestamps
}

var Limiter = &LoginRateLimiter{
	attempts: make(map[string][]int64),
}

// IsBlocked returns true if the IP has exceeded max attempts in the window.
func (l *LoginRateLimiter) IsBlocked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().Unix()
	attempts := l.attempts[ip]
	valid := attempts[:0]
	for _, t := range attempts {
		if now-t < windowSeconds {
			valid = append(valid, t)
		}
	}
	l.attempts[ip] = valid
	return len(valid) >= maxAttempts
}

// RecordAttempt adds a failed login timestamp for the IP.
func (l *LoginRateLimiter) RecordAttempt(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempts[ip] = append(l.attempts[ip], time.Now().Unix())
}

// Clear removes all failed attempts for the IP (on successful login).
func (l *LoginRateLimiter) Clear(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}
