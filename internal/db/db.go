package db

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/logger"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/pbkdf2"
)

const dbFile = "pisowifi.db"

var db *sql.DB

// UserRecord mirrors the users table row.
type UserRecord struct {
	MAC         string
	IP          string
	Time        int
	Status      string
	Balance     int
	FreeClaimed int
	Points      float64
}

// SaleRecord is a single coin-insertion history row.
type SaleRecord struct {
	Amount    int
	Timestamp int64
	DateStr   string // filled by API layer
}

// VoucherRecord mirrors the vouchers table row.
type VoucherRecord struct {
	Code      string  `json:"code"`
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	Status    string  `json:"status"`
	CreatedBy string  `json:"created_by"`
	UsedBy    string  `json:"used_by"`
	CreatedAt int64   `json:"created_at"`
	UsedAt    int64   `json:"used_at"`
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

// InitDB opens the SQLite database, enables WAL, and creates all tables.
// Also seeds the default admin from config.
func InitDB() {
	var err error
	db, err = sql.Open("sqlite3", dbFile+"?_timeout=30000&_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		panic(fmt.Sprintf("[DB] Cannot open database: %v", err))
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer

	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL")

	mustExec(`CREATE TABLE IF NOT EXISTS users (
		mac          TEXT PRIMARY KEY,
		ip           TEXT,
		time_remaining INTEGER,
		status       TEXT,
		last_updated INTEGER,
		balance      INTEGER DEFAULT 0,
		free_claimed INTEGER DEFAULT 0,
		points       REAL    DEFAULT 0
	)`)

	mustExec(`CREATE TABLE IF NOT EXISTS sales (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		mac       TEXT,
		amount    INTEGER,
		timestamp INTEGER
	)`)

	mustExec(`CREATE TABLE IF NOT EXISTS admins (
		username      TEXT PRIMARY KEY,
		password_hash TEXT
	)`)

	mustExec(`CREATE TABLE IF NOT EXISTS vouchers (
		code       TEXT PRIMARY KEY,
		type       TEXT,
		value      REAL,
		status     TEXT,
		created_by TEXT,
		used_by    TEXT,
		created_at INTEGER,
		used_at    INTEGER
	)`)

	// Safe migrations — ignore errors if column already exists
	safeExec("ALTER TABLE users ADD COLUMN balance INTEGER DEFAULT 0")
	safeExec("ALTER TABLE users ADD COLUMN free_claimed INTEGER DEFAULT 0")
	safeExec("ALTER TABLE users ADD COLUMN points REAL DEFAULT 0")

	// Seed default admin (fresh password hash)
	seedAdmin()
}

// CloseDB gracefully closes the database connection.
// In WAL mode, this forces SQLite to flush all pending data from the -wal file
// directly into the main .db file, keeping backups clean and safe.
func CloseDB() {
	if db != nil {
		db.Close()
	}
}

func mustExec(query string) {
	if _, err := db.Exec(query); err != nil {
		panic(fmt.Sprintf("[DB] Fatal: %v — query: %s", err, query))
	}
}

func safeExec(query string) {
	_, _ = db.Exec(query)
}

// seedAdmin inserts the admin from .env if not already present.
// On a fresh DB (or after DELETE from admins) this will always run.
// Q2 answer: we always reset — hash is derived from the .env password.
func seedAdmin() {
	hash, err := hashPassword(config.AdminPassword)
	if err != nil {
		fmt.Printf("[DB] Could not hash admin password: %v\n", err)
		return
	}

	_, err = db.Exec(
		"INSERT OR REPLACE INTO admins (username, password_hash) VALUES (?, ?)",
		config.AdminUsername, hash,
	)
	if err != nil {
		fmt.Printf("[DB] Could not seed admin: %v\n", err)
		return
	}
	fmt.Printf("[DB] Admin '%s' seeded/reset from .env\n", config.AdminUsername)
}

// ---------------------------------------------------------------------------
// Password helpers — PBKDF2-SHA256 compatible with passlib's format
// Format: $pbkdf2-sha256$<iterations>$<b64salt>$<b64hash>
// ---------------------------------------------------------------------------

const (
	pbkdf2Iterations = 29000
	pbkdf2KeyLen     = 32
)

func hashPassword(password string) (string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := base64.RawStdEncoding.EncodeToString(saltBytes)
	key := pbkdf2.Key([]byte(password), saltBytes, pbkdf2Iterations, pbkdf2KeyLen, sha256.New)
	hash := base64.RawStdEncoding.EncodeToString(key)
	return fmt.Sprintf("$pbkdf2-sha256$%d$%s$%s", pbkdf2Iterations, salt, hash), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	// parts: ["", "pbkdf2-sha256", "<iter>", "<salt>", "<hash>"]
	if len(parts) != 5 || parts[1] != "pbkdf2-sha256" {
		return false
	}
	iters, err := strconv.Atoi(parts[2])
	if err != nil {
		return false
	}
	saltBytes, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	derived := pbkdf2.Key([]byte(password), saltBytes, iters, len(storedHash), sha256.New)
	return constantTimeEqual(derived, storedHash)
}

func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

func VerifyAdmin(username, password string) bool {
	var hash string
	err := db.QueryRow("SELECT password_hash FROM admins WHERE username=?", username).Scan(&hash)
	if err != nil {
		return false
	}
	return verifyPassword(password, hash)
}

// ---------------------------------------------------------------------------
// User CRUD
// ---------------------------------------------------------------------------

func LoadUsers() map[string]UserRecord {
	rows, err := db.Query("SELECT mac, ip, time_remaining, status, balance, free_claimed, points FROM users")
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] LoadUsers: %v", err))
		return map[string]UserRecord{}
	}
	defer rows.Close()

	result := make(map[string]UserRecord)
	for rows.Next() {
		var u UserRecord
		if err := rows.Scan(&u.MAC, &u.IP, &u.Time, &u.Status, &u.Balance, &u.FreeClaimed, &u.Points); err != nil {
			continue
		}
		result[u.MAC] = u
	}
	return result
}

func SyncUser(u UserRecord) {
	_, err := db.Exec(
		`INSERT OR REPLACE INTO users
		 (mac, ip, time_remaining, status, last_updated, balance, free_claimed, points)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.MAC, u.IP, u.Time, u.Status, time.Now().Unix(), u.Balance, u.FreeClaimed, u.Points,
	)
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] SyncUser %s: %v", u.MAC, err))
	}
}

func SyncMultipleUsers(users []UserRecord) {
	if len(users) == 0 {
		return
	}
	tx, err := db.Begin()
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] SyncMultiple begin: %v", err))
		return
	}
	stmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO users
		 (mac, ip, time_remaining, status, last_updated, balance, free_claimed, points)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		tx.Rollback()
		return
	}
	now := time.Now().Unix()
	for _, u := range users {
		_, _ = stmt.Exec(u.MAC, u.IP, u.Time, u.Status, now, u.Balance, u.FreeClaimed, u.Points)
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] SyncMultiple commit: %v", err))
	}
}

func DeleteUser(mac string) {
	_, err := db.Exec("DELETE FROM users WHERE mac=?", mac)
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] DeleteUser %s: %v", mac, err))
	}
}

func ResetAllFreeClaimed() {
	_, _ = db.Exec("UPDATE users SET free_claimed = 0")
}

// ---------------------------------------------------------------------------
// Sales
// ---------------------------------------------------------------------------

func AddSale(mac string, amount int) {
	_, err := db.Exec("INSERT INTO sales (mac, amount, timestamp) VALUES (?, ?, ?)",
		mac, amount, time.Now().Unix())
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] AddSale: %v", err))
	}
}

func GetTotalSales() int {
	return sumQuery("SELECT SUM(amount) FROM sales")
}

func GetSalesSince(ts int64) int {
	return sumQuery("SELECT SUM(amount) FROM sales WHERE timestamp >= ?", ts)
}

func GetSalesRange(start, end int64) int {
	return sumQuery("SELECT SUM(amount) FROM sales WHERE timestamp >= ? AND timestamp < ?", start, end)
}

func GetUserSales(mac string) []SaleRecord {
	rows, err := db.Query(
		"SELECT amount, timestamp FROM sales WHERE mac=? ORDER BY timestamp DESC", mac)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []SaleRecord
	for rows.Next() {
		var s SaleRecord
		rows.Scan(&s.Amount, &s.Timestamp)
		result = append(result, s)
	}
	return result
}

func sumQuery(query string, args ...any) int {
	var total sql.NullInt64
	db.QueryRow(query, args...).Scan(&total)
	if total.Valid {
		return int(total.Int64)
	}
	return 0
}

// ---------------------------------------------------------------------------
// Vouchers
// ---------------------------------------------------------------------------

func CreateVoucher(v *VoucherRecord) error {
	_, err := db.Exec(
		`INSERT INTO vouchers (code, type, value, status, created_by, used_by, created_at, used_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		v.Code, v.Type, v.Value, v.Status, v.CreatedBy, v.UsedBy, v.CreatedAt, v.UsedAt,
	)
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] CreateVoucher: %v", err))
	}
	return err
}

func GetVoucher(code string) (*VoucherRecord, error) {
	var v VoucherRecord
	err := db.QueryRow(
		`SELECT code, type, value, status, created_by, used_by, created_at, used_at
		 FROM vouchers WHERE code = ?`, code,
	).Scan(&v.Code, &v.Type, &v.Value, &v.Status, &v.CreatedBy, &v.UsedBy, &v.CreatedAt, &v.UsedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func MarkVoucherUsed(code, usedBy string) error {
	now := time.Now().Unix()
	_, err := db.Exec(
		`UPDATE vouchers SET status = 'used', used_by = ?, used_at = ? WHERE code = ?`,
		usedBy, now, code,
	)
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] MarkVoucherUsed: %v", err))
	}
	return err
}

func GetAllVouchers() []VoucherRecord {
	rows, err := db.Query(
		`SELECT code, type, value, status, created_by, used_by, created_at, used_at
		 FROM vouchers ORDER BY created_at DESC`,
	)
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] GetAllVouchers: %v", err))
		return []VoucherRecord{}
	}
	defer rows.Close()

	var result []VoucherRecord
	for rows.Next() {
		var v VoucherRecord
		err := rows.Scan(&v.Code, &v.Type, &v.Value, &v.Status, &v.CreatedBy, &v.UsedBy, &v.CreatedAt, &v.UsedAt)
		if err == nil {
			result = append(result, v)
		}
	}
	return result
}

func GetActiveVouchersByUser(mac string) []VoucherRecord {
	rows, err := db.Query(
		`SELECT code, type, value, status, created_by, used_by, created_at, used_at
		 FROM vouchers WHERE created_by = ? AND status = 'active' ORDER BY created_at DESC`,
		mac,
	)
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] GetActiveVouchersByUser: %v", err))
		return []VoucherRecord{}
	}
	defer rows.Close()

	var result []VoucherRecord
	for rows.Next() {
		var v VoucherRecord
		err := rows.Scan(&v.Code, &v.Type, &v.Value, &v.Status, &v.CreatedBy, &v.UsedBy, &v.CreatedAt, &v.UsedAt)
		if err == nil {
			result = append(result, v)
		}
	}
	return result
}

func GetAllVouchersByUser(mac string) []VoucherRecord {
	rows, err := db.Query(
		`SELECT code, type, value, status, created_by, used_by, created_at, used_at
		 FROM vouchers WHERE created_by = ? ORDER BY created_at DESC`,
		mac,
	)
	if err != nil {
		logger.SystemLog(fmt.Sprintf("[DB ERROR] GetAllVouchersByUser: %v", err))
		return []VoucherRecord{}
	}
	defer rows.Close()

	var result []VoucherRecord
	for rows.Next() {
		var v VoucherRecord
		err := rows.Scan(&v.Code, &v.Type, &v.Value, &v.Status, &v.CreatedBy, &v.UsedBy, &v.CreatedAt, &v.UsedAt)
		if err == nil {
			result = append(result, v)
		}
	}
	return result
}
