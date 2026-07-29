package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// rotatingWriter implements a simple size-limited log rotation.
// We keep it dependency-free to avoid CGO issues.
type rotatingWriter struct {
	filename  string
	maxBytes  int64
	backups   int
	file      *os.File
	written   int64
}

func newRotatingWriter(filename string, maxBytes int64, backups int) (*rotatingWriter, error) {
	rw := &rotatingWriter{filename: filename, maxBytes: maxBytes, backups: backups}
	if err := rw.openFile(); err != nil {
		return nil, err
	}
	return rw, nil
}

func (rw *rotatingWriter) openFile() error {
	f, err := os.OpenFile(rw.filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	info, _ := f.Stat()
	if info != nil {
		rw.written = info.Size()
	}
	rw.file = f
	return nil
}

func (rw *rotatingWriter) Write(p []byte) (n int, err error) {
	if rw.written+int64(len(p)) > rw.maxBytes {
		rw.rotate()
	}
	n, err = rw.file.Write(p)
	rw.written += int64(n)
	return
}

func (rw *rotatingWriter) rotate() {
	rw.file.Close()
	for i := rw.backups - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", rw.filename, i)
		dst := fmt.Sprintf("%s.%d", rw.filename, i+1)
		_ = os.Rename(src, dst)
	}
	_ = os.Rename(rw.filename, rw.filename+".1")
	_ = rw.openFile()
}

// ---------------------------------------------------------------------------
// Package-level logger setup
// ---------------------------------------------------------------------------

var (
	infoLog  *log.Logger
	stdLog   *log.Logger
)

// Init sets up the global logger. Call once from main.
func Init() {
	rw, err := newRotatingWriter("system.log", 5*1024*1024, 3)
	if err != nil {
		// Fallback: log only to stdout
		infoLog = log.New(os.Stdout, "", 0)
	} else {
		infoLog = log.New(io.MultiWriter(rw, os.Stdout), "", 0)
	}
	stdLog = log.New(os.Stdout, "", 0)
}

func now() string {
	return time.Now().Format("2006-01-02 03:04:05 PM")
}

// SystemLog logs portal/coin/slot events.
// Format: [timestamp] [TYPE] message — matches what the admin dashboard JS parses.
func SystemLog(msg string) {
	if infoLog == nil {
		fmt.Println(msg)
		return
	}
	infoLog.Printf("[%s] %s", now(), msg)
}

// AuditLog logs administrator actions.
// Format: [timestamp] [ADMIN_AUDIT] [ip | mac] action: details
func AuditLog(action, ip, mac, details string) {
	if infoLog == nil {
		fmt.Printf("[ADMIN_AUDIT] [%s | %s] %s: %s\n", ip, mac, action, details)
		return
	}
	infoLog.Printf("[%s] [ADMIN_AUDIT] [%s | %s] %s: %s", now(), ip, mac, action, details)
}