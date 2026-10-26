package system_setting

import (
	"net"
	"strings"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
)

const (
	// IPBlacklistEnabledOption enables inbound client IP blocking.
	IPBlacklistEnabledOption = "ip_blacklist_setting.enabled"
	// IPBlacklistListOption stores a JSON array of banned client IPs and
	// CIDR ranges.
	IPBlacklistListOption = "ip_blacklist_setting.ip_list"
)

type ipBlacklistMatcher struct {
	raw  string
	nets []*net.IPNet
}

var ipBlacklistMatcherCache atomic.Pointer[ipBlacklistMatcher]

// ParseIPBlacklistJSON parses a JSON array of IP addresses and CIDR ranges.
// It returns every valid network plus the entries that could not be parsed.
func ParseIPBlacklistJSON(raw string) ([]*net.IPNet, []string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil, nil
	}
	var entries []string
	if err := common.UnmarshalJsonStr(raw, &entries); err != nil {
		return nil, nil, err
	}
	nets := make([]*net.IPNet, 0, len(entries))
	invalid := make([]string, 0)
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		network, ok := parseIPBlacklistEntry(entry)
		if !ok {
			invalid = append(invalid, entry)
			continue
		}
		nets = append(nets, network)
	}
	return nets, invalid, nil
}

func parseIPBlacklistEntry(entry string) (*net.IPNet, bool) {
	if strings.Contains(entry, "/") {
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, false
		}
		return network, true
	}
	ip := net.ParseIP(entry)
	if ip == nil {
		return nil, false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return &net.IPNet{IP: ipv4, Mask: net.CIDRMask(32, 32)}, true
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}, true
}

// GetIPBlacklistEntries returns the configured blacklist entries as stored.
func GetIPBlacklistEntries() ([]string, error) {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[IPBlacklistListOption]
	common.OptionMapRWMutex.RUnlock()
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var entries []string
	if err := common.UnmarshalJsonStr(raw, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// IsIPBlacklisted reports whether the inbound client IP is blocked. A disabled
// or malformed list never blocks traffic.
func IsIPBlacklisted(ip net.IP) bool {
	if ip == nil {
		return false
	}
	common.OptionMapRWMutex.RLock()
	enabled := common.OptionMap[IPBlacklistEnabledOption] == "true"
	raw := common.OptionMap[IPBlacklistListOption]
	common.OptionMapRWMutex.RUnlock()
	if !enabled {
		return false
	}
	for _, network := range compiledIPBlacklist(raw) {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func compiledIPBlacklist(raw string) []*net.IPNet {
	if cached := ipBlacklistMatcherCache.Load(); cached != nil && cached.raw == raw {
		return cached.nets
	}
	nets, _, err := ParseIPBlacklistJSON(raw)
	if err != nil {
		return nil
	}
	ipBlacklistMatcherCache.Store(&ipBlacklistMatcher{raw: raw, nets: nets})
	return nets
}
