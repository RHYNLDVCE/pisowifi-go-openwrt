package infrastructure

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"pisowifi/internal/config"
)

// ---------------------------------------------------------------------------
// SystemOps — replaces infrastructure/system_ops.py
// ---------------------------------------------------------------------------

type SystemStats struct {
	CPU        float64            `json:"cpu"`
	Temp       interface{}        `json:"temp"` // float64 or "N/A"
	RAM        float64            `json:"ram"`
	RAMUsed    float64            `json:"ram_used"`
	RAMTotal   float64            `json:"ram_total"`
	Disk       float64            `json:"disk"`
	DiskFree   float64            `json:"disk_free"`
	Uptime     string             `json:"uptime"`
	IPs        string             `json:"ips"`
	WANIface   string             `json:"wan_iface"`
	WANRXTotal uint64             `json:"wan_rx_total"`
	WANTXTotal uint64             `json:"wan_tx_total"`
	LANIface   string             `json:"lan_iface"`
	LANRXTotal uint64             `json:"lan_rx_total"`
	LANTXTotal uint64             `json:"lan_tx_total"`
	Interfaces map[string]ifStats `json:"interfaces"`
}

type ifStats struct {
	RXBytes uint64 `json:"rx_bytes"`
	TXBytes uint64 `json:"tx_bytes"`
}

func GetSystemStats() SystemStats {
	var s SystemStats
	s.Interfaces = make(map[string]ifStats)

	// CPU (simple /proc/stat 100ms delta)
	s.CPU = getCPUPercent()

	// Temperature
	if data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		var raw int
		fmt.Sscan(strings.TrimSpace(string(data)), &raw)
		s.Temp = math.Round(float64(raw)/1000*10) / 10
	} else {
		s.Temp = "N/A"
	}

	// RAM from /proc/meminfo
	s.RAM, s.RAMUsed, s.RAMTotal = getMemInfo()

	// Disk
	s.Disk, s.DiskFree = getDiskInfo("/")

	// Uptime
	s.Uptime = getUptime()

	// IPs
	s.IPs = getLocalIPs()

	// Network interfaces from /proc/net/dev
	ifaces := getNetStats()
	wan := config.WANInterface
	s.WANIface = wan
	if ws, ok := ifaces[wan]; ok {
		s.WANRXTotal = ws.RXBytes
		s.WANTXTotal = ws.TXBytes
	}

	lan := config.LANInterface
	s.LANIface = lan
	if ls, ok := ifaces[lan]; ok {
		s.LANRXTotal = ls.RXBytes
		s.LANTXTotal = ls.TXBytes
	}
	for name, stats := range ifaces {
		if name == "lo" || strings.HasPrefix(name, "br") || strings.HasPrefix(name, "wlan") {
			continue
		}
		s.Interfaces[name] = stats
	}

	return s
}

func RebootDevice() {
	exec.Command("sudo", "reboot").Run()
}

// ---------------------------------------------------------------------------
// Log helpers
// ---------------------------------------------------------------------------

var logPattern = regexp.MustCompile(`\[(.+?)\] \[(.+?)\] (.*)`)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Message   string `json:"message"`
}

type LogResult struct {
	Logs   []LogEntry `json:"logs"`
	Total  int        `json:"total"`
	Offset int        `json:"offset"`
	Limit  int        `json:"limit"`
}

var typeMap = map[string][]string{
	"COIN":     {"COIN_INSERT", "COIN_SUCCESS"},
	"PORTAL":   {"PORTAL_EVENT"},
	"ADMIN":    {"ADMIN_AUDIT"},
	"SECURITY": {"SECURITY_ALERT", "CRITICAL"},
}

func GetSystemLogs(limit, offset int, logType, query string) LogResult {
	f, err := os.Open("system.log")
	if err != nil {
		return LogResult{Logs: []LogEntry{}, Limit: limit}
	}
	defer f.Close()

	var parsed []LogEntry
	scanner := bufio.NewScanner(f)
	
	lowerQuery := ""
	if query != "" {
		lowerQuery = strings.ToLower(query)
	}

	for scanner.Scan() {
		line := scanner.Text()
		
		// If query is provided, skip lines that don't match
		if lowerQuery != "" && !strings.Contains(strings.ToLower(line), lowerQuery) {
			continue
		}

		if m := logPattern.FindStringSubmatch(line); m != nil {
			parsed = append(parsed, LogEntry{Timestamp: m[1], Type: m[2], Message: m[3]})
		} else if strings.HasPrefix(line, "[") {
			idx := strings.Index(line, "]")
			if idx > 0 {
				parsed = append(parsed, LogEntry{Timestamp: line[1:idx], Type: "SYSTEM", Message: strings.TrimSpace(line[idx+1:])})
			}
		}
	}

	// Filter by type
	if logType != "" && strings.ToUpper(logType) != "ALL" {
		cat := strings.ToUpper(logType)
		if allowed, ok := typeMap[cat]; ok {
			allowedSet := map[string]bool{}
			for _, t := range allowed {
				allowedSet[t] = true
			}
			var filtered []LogEntry
			for _, e := range parsed {
				if allowedSet[e.Type] {
					filtered = append(filtered, e)
				}
			}
			parsed = filtered
		} else {
			// SYSTEM = everything NOT in known categories
			known := map[string]bool{}
			for _, types := range typeMap {
				for _, t := range types {
					known[t] = true
				}
			}
			var filtered []LogEntry
			for _, e := range parsed {
				if !known[e.Type] {
					filtered = append(filtered, e)
				}
			}
			parsed = filtered
		}
	}

	// Reverse (newest first)
	for i, j := 0, len(parsed)-1; i < j; i, j = i+1, j-1 {
		parsed[i], parsed[j] = parsed[j], parsed[i]
	}

	total := len(parsed)
	end := offset + limit
	if end > total {
		end = total
	}
	sliced := parsed[offset:end]
	if sliced == nil {
		sliced = []LogEntry{}
	}
	return LogResult{Logs: sliced, Total: total, Offset: offset, Limit: limit}
}

// ---------------------------------------------------------------------------
// Banner / Sound helpers
// ---------------------------------------------------------------------------

func GetBanners(order []string) []string {
	dir := "static/banners/set"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	actualFiles := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			actualFiles[e.Name()] = true
		}
	}
	var result []string
	for _, f := range order {
		if actualFiles[f] {
			result = append(result, f)
		}
	}
	for f := range actualFiles {
		found := false
		for _, r := range result {
			if r == f {
				found = true
				break
			}
		}
		if !found {
			result = append(result, f)
		}
	}
	return result
}

func GetSounds() []string {
	entries, err := os.ReadDir("static/sounds")
	if err != nil {
		return nil
	}
	var sounds []string
	for _, e := range entries {
		name := e.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".mp3") || strings.HasSuffix(lower, ".wav") || strings.HasSuffix(lower, ".ogg") {
			sounds = append(sounds, name)
		}
	}
	return sounds
}

func GetBannerURLs() []string {
	dir := "static/banners/set"
	images := []string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return images
	}
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
			images = append(images, fmt.Sprintf("/static/banners/set/%s", e.Name()))
		}
	}
	return images
}

// ---------------------------------------------------------------------------
// /proc helpers (replaces psutil)
// ---------------------------------------------------------------------------

var (
	lastCPUTotal, lastCPUIdle uint64
	cpuMu                     sync.Mutex
)

func getCPUPercent() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	line := strings.SplitN(string(data), "\n", 2)[0]
	var cpu string
	var user, nice, system, idle, iowait, irq, softirq uint64
	fmt.Sscanf(line, "%s %d %d %d %d %d %d %d", &cpu, &user, &nice, &system, &idle, &iowait, &irq, &softirq)
	total := user + nice + system + idle + iowait + irq + softirq

	cpuMu.Lock()
	prevTotal, prevIdle := lastCPUTotal, lastCPUIdle
	lastCPUTotal, lastCPUIdle = total, idle
	cpuMu.Unlock()

	if prevTotal == 0 {
		return 0
	}
	deltaTotal := float64(total - prevTotal)
	deltaIdle := float64(idle - prevIdle)
	if deltaTotal == 0 {
		return 0
	}
	return math.Round((1-deltaIdle/deltaTotal)*100*10) / 10
}

func getMemInfo() (percent, usedGB, totalGB float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}
	var totalKB, availKB uint64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscan(strings.Fields(line)[1], &totalKB)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscan(strings.Fields(line)[1], &availKB)
		}
	}
	if totalKB == 0 {
		return
	}
	usedKB := totalKB - availKB
	totalGB = math.Round(float64(totalKB)/1024/1024*100) / 100
	usedGB = math.Round(float64(usedKB)/1024/1024*100) / 100
	percent = math.Round(float64(usedKB)/float64(totalKB)*100*10) / 10
	return
}

func getDiskInfo(path string) (percent, freeGB float64) {
	// Use df command — statvfs syscall requires cgo on some targets
	out, err := exec.Command("df", "-B1", path).Output()
	if err != nil {
		return
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return
	}
	var total, used, free uint64
	fmt.Sscan(fields[1], &total)
	fmt.Sscan(fields[2], &used)
	fmt.Sscan(fields[3], &free)
	if total > 0 {
		percent = math.Round(float64(used)/float64(total)*100*10) / 10
	}
	freeGB = math.Round(float64(free)/1024/1024/1024*100) / 100
	return
}

func getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "Unknown"
	}
	var secs float64
	fmt.Sscan(strings.Fields(string(data))[0], &secs)
	d := int(secs) / 86400
	h := (int(secs) % 86400) / 3600
	m := (int(secs) % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	} else if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%d min", m)
}

func getLocalIPs() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	var ips []string
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "lo") {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return strings.Join(ips, "\n")
}

func getNetStats() map[string]ifStats {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil
	}
	result := map[string]ifStats{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[2:] { // skip header lines
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:idx])
		fields := strings.Fields(line[idx+1:])
		if len(fields) < 9 {
			continue
		}
		var rx, tx uint64
		fmt.Sscan(fields[0], &rx)
		fmt.Sscan(fields[8], &tx)
		result[name] = ifStats{RXBytes: rx, TXBytes: tx}
	}
	return result
}

// Ensure getCPUPercent is called once at startup to seed the delta
func init() {
	getCPUPercent()
	time.Sleep(100 * time.Millisecond)
	_ = runtime.NumCPU() // suppress unused import
}
