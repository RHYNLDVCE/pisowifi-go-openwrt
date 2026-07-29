package services

import (
	"math"
	"sort"
	"strings"
	"time"

	"pisowifi/internal/config"
	"pisowifi/internal/db"
	"pisowifi/internal/infrastructure"
	"pisowifi/internal/network"
	"pisowifi/internal/state"
)

// ---------------------------------------------------------------------------
// AdminService — replaces services/admin_service.py
// ---------------------------------------------------------------------------

type DailyStat struct {
	Date  string `json:"date"`
	Total int    `json:"total"`
}

type DashboardStats struct {
	Total     int         `json:"total"`
	Yesterday int         `json:"yesterday"`
	Daily     int         `json:"daily"`
	Weekly    int         `json:"weekly"`
	Monthly   int         `json:"monthly"`
	LastMonth int         `json:"last_month"`
	Yearly    int         `json:"yearly"`
	ChartData []DailyStat `json:"chart_data"`
}

func GetDashboardStats() DashboardStats {
	now := time.Now()

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	// Calculate offset for Monday as the start of the week
	// time.Weekday(): Sunday=0, Monday=1, ..., Saturday=6
	// We want an offset where Monday=0, Tuesday=1, ..., Sunday=6
	offset := (int(now.Weekday()) + 6) % 7
	startOfWeek := startOfDay.AddDate(0, 0, -offset)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfLastMonth := startOfMonth.AddDate(0, -1, 0)
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	yesterday := startOfDay.AddDate(0, 0, -1)

	var chartData []DailyStat
	for i := 6; i >= 0; i-- {
		dayStart := startOfDay.AddDate(0, 0, -i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		dayTotal := db.GetSalesRange(dayStart.Unix(), dayEnd.Unix())
		chartData = append(chartData, DailyStat{
			Date:  dayStart.Format("Mon"),
			Total: dayTotal,
		})
	}

	return DashboardStats{
		Total:     db.GetTotalSales(),
		Yesterday: db.GetSalesRange(yesterday.Unix(), startOfDay.Unix()),
		Daily:     db.GetSalesSince(startOfDay.Unix()),
		Weekly:    db.GetSalesSince(startOfWeek.Unix()),
		Monthly:   db.GetSalesSince(startOfMonth.Unix()),
		LastMonth: db.GetSalesRange(startOfLastMonth.Unix(), startOfMonth.Unix()),
		Yearly:    db.GetSalesSince(startOfYear.Unix()),
		ChartData: chartData,
	}
}

// ManageUserTime adds or subtracts time for a user. Mirrors admin_service.py.
func ManageUserTime(mac string, amount int, unit string, action string) {
	seconds := amount * 60
	if unit == "hours" {
		seconds = amount * 3600
	}
	if seconds < 0 {
		seconds = -seconds
	}

	state.Users.UpdateField(mac, func(u *state.UserRecord) {
		if action == "subtract" {
			u.Time -= seconds
		} else {
			u.Time += seconds
		}
		if u.Time < 0 {
			u.Time = 0
		}

		if u.Time == 0 && u.Status == "connected" {
			u.Status = "expired"
			u.ExpiresAt = 0
			network.BlockUser(mac, u.IP)
		} else if action == "add" && u.Time > 0 && u.Status == "expired" {
			u.Status = "connected"
			u.ExpiresAt = float64(time.Now().UnixNano())/1e9 + float64(u.Time)
			network.AllowUser(mac, u.IP)
		}

		if u.Status == "connected" {
			u.ExpiresAt = float64(time.Now().UnixNano())/1e9 + float64(u.Time)
		}
	})

	if u, ok := state.Users.Get(mac); ok {
		db.SyncUser(toDBRecord(mac, u))
	}
}

// UpdateUserStatus sets a user's status (block/unblock).
func UpdateUserStatus(mac, newStatus string) {
	state.Users.UpdateField(mac, func(u *state.UserRecord) {
		u.Status = newStatus
		if newStatus == "blocked" {
			network.BlockUser(mac, u.IP)
		}
	})
	if u, ok := state.Users.Get(mac); ok {
		db.SyncUser(toDBRecord(mac, u))
	}
}

// DeleteUser removes a user from the firewall, state, and DB.
func DeleteUser(mac string) {
	if u, ok := state.Users.Get(mac); ok {
		network.BlockUser(mac, u.IP)
	}
	state.Users.Delete(mac)
	db.DeleteUser(mac)
}

// GetPaginatedUsers retrieves and formats users for the dashboard
func GetPaginatedUsers(search string, page int, sortBy string, itemsPerPage int) map[string]interface{} {
	cfg := config.Get()
	stats := GetDashboardStats()
	leases := infrastructure.GetDhcpLeases()
	customNames := cfg.CustomDeviceNames

	type enriched struct {
		MAC  string
		Data *state.UserRecord
		Name string
	}

	var users []enriched
	state.Users.Range(func(mac string, u *state.UserRecord) {
		name, _ := infrastructure.GetVendorInfo(mac, u.IP, leases)
		if customNames[mac] != "" {
			name = customNames[mac]
		}
		users = append(users, enriched{MAC: mac, Data: u, Name: name})
	})

	if search != "" {
		lower := strings.ToLower(search)
		var filtered []enriched
		for _, u := range users {
			if strings.Contains(strings.ToLower(u.MAC), lower) ||
				strings.Contains(strings.ToLower(u.Data.IP), lower) ||
				strings.Contains(strings.ToLower(u.Name), lower) {
				filtered = append(filtered, u)
			}
		}
		users = filtered
	}

	sort.Slice(users, func(i, j int) bool {
		if sortBy == "time" {
			return users[i].Data.Time > users[j].Data.Time
		}
		if sortBy == "points" {
			return users[i].Data.Points > users[j].Data.Points
		}

		rank := func(s string) int {
			s = strings.ToLower(s)
			if s == "connected" {
				return 1
			} else if s == "paused" {
				return 2
			} else if s == "expired" {
				return 3
			} else if s == "new" {
				return 4
			}
			return 5
		}
		ri, rj := rank(users[i].Data.Status), rank(users[j].Data.Status)
		if ri != rj {
			return ri < rj
		}
		return users[i].Data.Time > users[j].Data.Time
	})

	totalFiltered := len(users)
	totalPages := int(math.Ceil(float64(totalFiltered) / float64(itemsPerPage)))
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage
	if end > totalFiltered {
		end = totalFiltered
	}
	paginatedUsers := users[start:end]

	usersMap := make(map[string]interface{})
	for _, u := range paginatedUsers {
		shortStatus := ""
		if len(u.Data.Status) > 0 {
			shortStatus = string(u.Data.Status[0])
		}
		usersMap[u.MAC] = map[string]interface{}{
			"ip": u.Data.IP, "time": u.Data.Time, "status": u.Data.Status,
			"balance": u.Data.Balance, "points": u.Data.Points,
			"free_claimed": u.Data.FreeClaimed, "device_name": u.Name,
			"time_formatted": FormatHumanTime(u.Data.Time),
			"status_short":   shortStatus,
		}
	}

	activeMacs := map[string]bool{}
	state.Users.Range(func(mac string, u *state.UserRecord) {
		if u.Status != "new" {
			activeMacs[mac] = true
		}
	})
	devices := infrastructure.ScanInfrastructure(activeMacs, customNames, cfg.CustomDeviceIPs)

	activeCount := 0
	state.Users.Range(func(_ string, u *state.UserRecord) {
		if u.Status == "connected" {
			activeCount++
		}
	})

	return map[string]interface{}{
		"users":          usersMap,
		"devices":        devices,
		"current_page":   page,
		"total_pages":    totalPages,
		"search_query":   search,
		"active_users":   activeCount,
		"total_users":    state.Users.Count(),
		"total_filtered": totalFiltered,
		"stats":          stats,
	}
}
