package state

import (
	"sync"
	"sync/atomic"

	"github.com/gofiber/websocket/v2"
)

// ---------------------------------------------------------------------------
// UserRecord — in-memory representation of a connected/known client
// ---------------------------------------------------------------------------

type UserRecord struct {
	IP              string
	Time            int     // seconds remaining
	Status          string  // "new" | "connected" | "paused" | "expired" | "blocked"
	Balance         int     // unspent coin balance
	FreeClaimed     int     // 0 or 1
	Points          float64 // loyalty points
	LastActive      float64 // Unix timestamp
	ExpiresAt       float64 // wall-clock deadline (0 = not set)
	LastByteCount   int64   // for idle detection
	LastPacketCount int64   // for idle detection
}

// ---------------------------------------------------------------------------
// UserStore — thread-safe MAC → UserRecord map
// ---------------------------------------------------------------------------

type UserStore struct {
	mu    sync.RWMutex
	users map[string]*UserRecord
}

var Users = &UserStore{users: make(map[string]*UserRecord)}

func (s *UserStore) Get(mac string) (*UserRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[mac]
	return u, ok
}

func (s *UserStore) Set(mac string, u *UserRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[mac] = u
}

func (s *UserStore) Delete(mac string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, mac)
}

// Range calls fn for each entry. fn must NOT call any UserStore method (deadlock).
func (s *UserStore) Range(fn func(mac string, u *UserRecord)) {
	s.mu.RLock()
	// Snapshot keys+pointers so fn can safely call Get/Set on individual users.
	type pair struct {
		mac string
		u   *UserRecord
	}
	pairs := make([]pair, 0, len(s.users))
	for k, v := range s.users {
		pairs = append(pairs, pair{k, v})
	}
	s.mu.RUnlock()
	for _, p := range pairs {
		fn(p.mac, p.u)
	}
}

// Snapshot returns a copy of all current MAC addresses.
func (s *UserStore) Snapshot() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	macs := make([]string, 0, len(s.users))
	for k := range s.users {
		macs = append(macs, k)
	}
	return macs
}

func (s *UserStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// UpdateField is a helper to mutate a single field under write-lock.
func (s *UserStore) UpdateField(mac string, fn func(*UserRecord)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[mac]
	if !ok {
		return false
	}
	fn(u)
	return true
}

// ---------------------------------------------------------------------------
// WsManager — thread-safe per-MAC WebSocket connection map
// ---------------------------------------------------------------------------

type wsConnWrapper struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type WsManager struct {
	mu    sync.RWMutex
	conns map[string]*wsConnWrapper
}

var Manager = &WsManager{conns: make(map[string]*wsConnWrapper)}

func (m *WsManager) Connect(mac string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conns[mac] = &wsConnWrapper{conn: conn}
}

func (m *WsManager) Disconnect(mac string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.conns, mac)
}

func (m *WsManager) Get(mac string) (*websocket.Conn, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	wrapper, ok := m.conns[mac]
	if ok {
		return wrapper.conn, true
	}
	return nil, false
}

// Send writes a JSON message to a specific MAC's WebSocket connection.
func (m *WsManager) Send(mac string, msg any) {
	m.mu.RLock()
	wrapper, ok := m.conns[mac]
	m.mu.RUnlock()
	if !ok {
		return
	}
	func() {
		defer func() { recover() }()
		wrapper.mu.Lock()
		defer wrapper.mu.Unlock()
		wrapper.conn.WriteJSON(msg)
	}()
}

// CloseAll closes every active WebSocket connection — used during shutdown.
func (m *WsManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, wrapper := range m.conns {
		func() {
			defer func() { recover() }()
			wrapper.mu.Lock()
			defer wrapper.mu.Unlock()
			wrapper.conn.Close()
		}()
	}
}

// ---------------------------------------------------------------------------
// Globals
// ---------------------------------------------------------------------------

var IsShuttingDown atomic.Bool

// CurrentSlotUser is the MAC of the device with the coin slot open (or "").
// Protected by a simple mutex because it is read/written from both HTTP handlers
// and the timer goroutine.
var (
	slotMu          sync.Mutex
	currentSlotUser string
)

func GetSlotUser() string {
	slotMu.Lock()
	defer slotMu.Unlock()
	return currentSlotUser
}

func SetSlotUser(mac string) {
	slotMu.Lock()
	defer slotMu.Unlock()
	currentSlotUser = mac
}
