package services

import (
	"strconv"
	"strings"

	"pisowifi/internal/config"
)

// ---------------------------------------------------------------------------
// Billing — replaces services/billing_service.py
// ---------------------------------------------------------------------------

// parseCoinRates parses "1:10,5:60,10:180,20:300" into [(denom, minutes), ...]
// sorted descending by denomination (greedy algorithm).
func parseCoinRates(ratesStr string) [][2]int {
	var rates [][2]int
	for _, part := range strings.Split(ratesStr, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) != 2 {
			continue
		}
		amt, err1 := strconv.Atoi(kv[0])
		mins, err2 := strconv.Atoi(kv[1])
		if err1 != nil || err2 != nil || amt <= 0 {
			continue
		}
		rates = append(rates, [2]int{amt, mins})
	}
	// Sort descending
	for i := 0; i < len(rates); i++ {
		for j := i + 1; j < len(rates); j++ {
			if rates[j][0] > rates[i][0] {
				rates[i], rates[j] = rates[j], rates[i]
			}
		}
	}
	return rates
}

// CalculateTimeFromBalance converts a coin balance to minutes using the
// greedy denomination algorithm — exact port of Python's billing_service.py.
func CalculateTimeFromBalance(balance int) int {
	cfg := config.Get()
	rates := parseCoinRates(cfg.CoinRates)
	if len(rates) == 0 {
		rates = [][2]int{{1, 5}}
	}

	totalMinutes := 0
	remaining := balance
	for _, r := range rates {
		denom, mins := r[0], r[1]
		count := remaining / denom
		if count > 0 {
			totalMinutes += count * mins
			remaining %= denom
		}
	}
	if remaining > 0 {
		totalMinutes += remaining * 5 // fallback: 5 min per unit
	}
	return totalMinutes
}

// CalculatePointsFromBalance converts a coin balance to loyalty points.
func CalculatePointsFromBalance(balance int) float64 {
	cfg := config.Get()
	if !cfg.PointsEnabled {
		return 0
	}

	type dv struct{ denom int; val float64 }
	var rates []dv
	for k, v := range cfg.CoinPointMap {
		d, err := strconv.Atoi(k)
		if err != nil || d <= 0 {
			continue
		}
		rates = append(rates, dv{d, v})
	}
	// Sort descending
	for i := 0; i < len(rates); i++ {
		for j := i + 1; j < len(rates); j++ {
			if rates[j].denom > rates[i].denom {
				rates[i], rates[j] = rates[j], rates[i]
			}
		}
	}

	total := 0.0
	rem := balance
	for _, r := range rates {
		count := rem / r.denom
		if count > 0 {
			total += float64(count) * r.val
			rem %= r.denom
		}
	}

	// Round to 2 decimal places
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(total, 'f', 2, 64), 64)
	return rounded
}
