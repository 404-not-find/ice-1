package ice

import (
	"net"

	"github.com/gortc/ice/gather"
	"github.com/gortc/ice/internal"
)

// See Deprecating Site Local Addresses [RFC3879]
var siteLocalIPv6 = internal.MustParseNet("FEC0::/10")

// IsHostIPValid reports whether ip is valid as host address ip.
func IsHostIPValid(ip net.IP, ipv6Only bool) bool {
	var (
		v4 = ip.To4() != nil
		v6 = !v4
	)
	if v6 && ip.To16() == nil {
		return false
	}
	if v4 && ipv6Only {
		// IPv4-mapped IPv6 addresses SHOULD NOT be included in the address
		// candidates unless the application using ICE does not support IPv4
		// (i.e., it is an IPv6-only application [RFC4038]).
		return false
	}
	if ip.IsLoopback() {
		// Addresses from a loopback interface MUST NOT be included in the
		// candidate addresses.
		return false
	}
	if siteLocalIPv6.Contains(ip) {
		// Deprecated IPv4-compatible IPv6 addresses [RFC4291] and IPv6 site-
		// local unicast addresses [RFC3879] MUST NOT be included in the
		// address candidates.
		return false
	}
	if ip.IsLinkLocalUnicast() && v6 {
		// When host candidates corresponding to an IPv6 address generated
		// using a mechanism that prevents location tracking are gathered, then
		// host candidates corresponding to IPv6 link-local addresses [RFC4291]
		// MUST NOT be gathered.
		return false
	}
	return true
}

// HostAddr wraps IP of host interface and local preference.
type HostAddr struct {
	IP              net.IP
	LocalPreference int
}

func processDualStack(all, v4, v6 []gather.Addr) []HostAddr {
	var (
		v6InARow int
	)
	// TODO(ar): Simplify.
	nHi := (len(v6) + len(v4)) / len(v4)
	hostAddrs := make([]HostAddr, 0, len(all))
	for i := 0; i < len(all); i++ {
		useV6 := true
		if v6InARow >= nHi {
			v6InARow = 0
			useV6 = false
		}
		if useV6 && len(v6) > 0 {
			v6InARow++
			hostAddrs = append(hostAddrs, HostAddr{
				IP:              v6[0].IP,
				LocalPreference: len(all) - i,
			})
			v6 = v6[1:]
		} else if len(v4) > 0 {
			hostAddrs = append(hostAddrs, HostAddr{
				IP:              v4[0].IP,
				LocalPreference: len(all) - i,
			})
			v4 = v4[1:]
		}
	}
	return hostAddrs
}

func isV6Only(addrs []gather.Addr) bool {
	v6Only := true
	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			v6Only = false
			break
		}
	}
	return v6Only
}

// HostAddresses returns valid host addresses from gathered addresses with
// calculated local preference.
//
// When gathered addresses are only IPv6, the host is considered ipv6-only.
// When there are both IPv6 and IPv4 addresses, the RFC 8421 is used to
// calculate local preferences.
func HostAddresses(gathered []gather.Addr) ([]HostAddr, error) {
	if len(gathered) == 0 {
		return []HostAddr{}, nil
	}
	var (
		v6Only    = isV6Only(gathered)
		validOnly = make([]gather.Addr, 0, len(gathered))
	)
	for _, addr := range gathered {
		if !IsHostIPValid(addr.IP, v6Only) {
			continue
		}
		validOnly = append(validOnly, addr)
	}
	if len(validOnly) == 0 {
		return []HostAddr{}, nil
	}
	var (
		v6Addrs, v4Addrs []gather.Addr
	)
	for _, addr := range validOnly {
		if addr.IP.To4() == nil {
			v6Addrs = append(v6Addrs, addr)
		} else {
			v4Addrs = append(v4Addrs, addr)
		}
	}
	if len(v4Addrs) == 0 || len(v6Addrs) == 0 {
		// Single-stack, but multi-homed.
		hostAddrs := make([]HostAddr, 0, len(validOnly))
		for i, a := range validOnly {
			hostAddrs = append(hostAddrs, HostAddr{
				IP:              a.IP,
				LocalPreference: len(validOnly) - i,
			})
		}
		return hostAddrs, nil
	}
	// Dual-stack calculation as defined in RFC 8421.
	return processDualStack(validOnly, v4Addrs, v6Addrs), nil
}
