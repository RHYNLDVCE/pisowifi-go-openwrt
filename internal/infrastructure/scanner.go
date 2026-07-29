package infrastructure

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"pisowifi/internal/config"
)

// ---------------------------------------------------------------------------
// NetworkScanner — replaces infrastructure/network_scanner.py
// ---------------------------------------------------------------------------

type Device struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Vendor   string `json:"vendor"`
	IsCustom bool   `json:"is_custom"`
	IsOnline bool   `json:"is_online"`
	InARP    bool   `json:"in_arp"`
}

var leaseFiles = []string{
	"/var/lib/misc/dnsmasq.leases",
	"/var/lib/dnsmasq/dnsmasq.leases",
	"/var/lib/dhcp/dhcpd.leases",
}

// GetDhcpLeases reads DHCP lease files and returns mac→hostname map.
func GetDhcpLeases() map[string]string {
	leases := map[string]string{}
	for _, lf := range leaseFiles {
		data, err := os.ReadFile(lf)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				mac := strings.ToLower(parts[1])
				hostname := parts[3]
				if hostname != "*" {
					leases[mac] = hostname
				}
			}
		}
	}
	return leases
}

// GetVendorInfo returns a display name and whether the MAC vendor is known.
func GetVendorInfo(mac, ip string, leases map[string]string) (string, bool) {
	macClean := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(mac, ":", ""), "-", ""))
	if len(macClean) < 6 {
		return "Unknown Device", false
	}
	oui := macClean[:6]

	brand := ouiLookup(oui)
	hostname := leases[strings.ToLower(mac)]

	isKnown := brand != "Unknown"
	var display string
	switch {
	case isKnown && hostname != "":
		display = fmt.Sprintf("%s (%s)", brand, hostname)
	case isKnown:
		display = brand
	case hostname != "":
		display = hostname
	default:
		display = "Unknown Device"
	}
	return display, isKnown
}

// IsRandomMac checks the locally-administered bit (randomized MAC).
func IsRandomMac(mac string) bool {
	parts := strings.SplitN(mac, ":", 2)
	if len(parts) == 0 {
		return false
	}
	var b byte
	fmt.Sscanf(parts[0], "%x", &b)
	return (b & 0x02) != 0
}

// IsReachable pings the IP once with a 2 s timeout.
func IsReachable(ip string) bool {
	out := exec.Command("ping", "-c", "1", "-W", "2", ip)
	return out.Run() == nil
}

// ScanInfrastructure reads /proc/net/arp and pings all non-user devices in parallel.
func ScanInfrastructure(activeMacs map[string]bool, customNames map[string]string, customIPs map[string]string) []Device {
	leases := GetDhcpLeases()
	lan := config.LANInterface

	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return nil
	}

	var candidates []Device
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}
		ip, mac, iface := parts[0], strings.ToLower(parts[3]), parts[5]
		if iface != lan || mac == "00:00:00:00:00:00" {
			continue
		}
		if activeMacs[mac] {
			continue
		}
		if IsRandomMac(mac) && customNames[mac] == "" {
			continue
		}

		displayName, _ := GetVendorInfo(mac, ip, leases)
		upper := strings.ToUpper(displayName)
		phoneKeywords := []string{"NAM", "OPPO", "VIVO", "REALME", "IPHONE", "GALAXY", "XIAOMI", "POCO", "REDMI", "ANDROID"}
		isPhone := false
		for _, kw := range phoneKeywords {
			if strings.Contains(upper, kw) {
				isPhone = true
				break
			}
		}
		if isPhone && customNames[mac] == "" {
			continue
		}

		name := displayName
		if cn, ok := customNames[mac]; ok && cn != "" {
			name = cn
		}
		candidates = append(candidates, Device{
			IP:       ip,
			MAC:      mac,
			Vendor:   name,
			IsCustom: customNames[mac] != "",
			InARP:    true,
		})
	}

	// Append any custom devices that were not in the ARP table
	seenMacs := make(map[string]bool)
	for _, dev := range candidates {
		seenMacs[dev.MAC] = true
	}
	for customMac, customName := range customNames {
		if !seenMacs[customMac] {
			candidates = append(candidates, Device{
				IP:       customIPs[customMac],
				MAC:      customMac,
				Vendor:   customName,
				IsCustom: true,
				InARP:    false,
			})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Parallel ping
	var wg sync.WaitGroup
	results := make([]Device, len(candidates))
	for i, dev := range candidates {
		wg.Add(1)
		go func(idx int, d Device) {
			defer wg.Done()
			d.IsOnline = IsReachable(d.IP)
			results[idx] = d
		}(i, dev)
	}
	wg.Wait()
	return results
}

// GetMACFromIP looks up a MAC address in /proc/net/arp by IP.
func GetMACFromIP(ip string) string {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return "00:00:00:00:00:00"
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) > 3 && parts[0] == ip {
			return parts[3]
		}
	}
	return "00:00:00:00:00:00"
}

// GetLocalHostname tries a reverse DNS lookup for an IP.
func GetLocalHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

// ---------------------------------------------------------------------------
// OUI vendor lookup table (ported from network_scanner.py)
// ---------------------------------------------------------------------------

func ouiLookup(oui string) string {
	vendors := map[string]string{
		// Comfast
		"200DB0": "Comfast", "40A5EF": "Comfast", "E0E1A9": "Comfast", "8C3D16": "Comfast", "00E04C": "Comfast",
		// TP-Link (subset — full list below)
		"18D6C7": "TP-Link", "CC32E5": "TP-Link", "003192": "TP-Link", "14CC20": "TP-Link",
		"50C7BF": "TP-Link", "8416F9": "TP-Link", "C025E9": "TP-Link", "E848B8": "TP-Link",
		"000AEB": "TP-Link", "001478": "TP-Link", "0019E0": "TP-Link", "001D0F": "TP-Link",
		"002127": "TP-Link", "0023CD": "TP-Link", "002586": "TP-Link", "002719": "TP-Link",
		"04F9F8": "TP-Link", "081F71": "TP-Link", "0C4B54": "TP-Link", "10FEED": "TP-Link",
		"147590": "TP-Link", "14CF92": "TP-Link", "18A6F7": "TP-Link", "1C3BF3": "TP-Link",
		"206BE7": "TP-Link", "20DCE6": "TP-Link", "246968": "TP-Link", "282CB2": "TP-Link",
		"30B5C2": "TP-Link", "349672": "TP-Link", "34E894": "TP-Link", "388345": "TP-Link",
		"3C46D8": "TP-Link", "40169F": "TP-Link", "44B32D": "TP-Link", "480EEC": "TP-Link",
		"503EAA": "TP-Link", "50BD5F": "TP-Link", "54E6FC": "TP-Link", "584120": "TP-Link",
		"60E327": "TP-Link", "6466B3": "TP-Link", "704F57": "TP-Link", "7405A5": "TP-Link",
		"74DA88": "TP-Link", "7844FD": "TP-Link", "7C8BCA": "TP-Link", "808917": "TP-Link",
		"882593": "TP-Link", "8C210A": "TP-Link", "90F652": "TP-Link", "940C6D": "TP-Link",
		"984827": "TP-Link", "98DED0": "TP-Link", "A0F3C1": "TP-Link", "A42BB0": "TP-Link",
		"AC84C6": "TP-Link", "B0487A": "TP-Link", "B0BE76": "TP-Link", "B8F883": "TP-Link",
		"C04A00": "TP-Link", "C46E1F": "TP-Link", "CC3429": "TP-Link", "D4016D": "TP-Link",
		"D807B6": "TP-Link", "DC0077": "TP-Link", "E005C5": "TP-Link", "E4C32A": "TP-Link",
		"EC086B": "TP-Link", "F4F26D": "TP-Link", "F81A67": "TP-Link", "FC70F4": "TP-Link",
		// Tenda
		"0495E6": "Tenda", "0840F3": "Tenda", "500FF5": "Tenda", "502B73": "Tenda",
		"CC2D21": "Tenda", "C83A35": "Tenda", "0050FC": "Tenda",
		// Huawei
		"001882": "Huawei", "00E0FC": "Huawei", "4846F1": "Huawei",
		// ZTE
		"0015EB": "ZTE", "001E73": "ZTE", "D0DD7C": "ZTE",
		// FiberHome
		"286ED4": "FiberHome", "807D14": "FiberHome",
	}
	if v, ok := vendors[oui]; ok {
		return v
	}
	return "Unknown"
}
