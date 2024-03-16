package irtt

import (
	"encoding/json"
	"fmt"
	"net"
)

// IPVersion is an IP version, or dual stack for IPv4 and IPv6.
type IPVersion int

// IPVersion constants.
const (
	IPv4 IPVersion = 1 << iota
	IPv6
	DualStack = IPv4 | IPv6
)

// IPVersionFromBooleans returns an IPVersion from booleans. If both ipv4 and
// ipv6 are true, DualStack is returned. If neither are true, the value of dfl
// is returned.
func IPVersionFromBooleans(ipv4 bool, ipv6 bool, dfl IPVersion) IPVersion {
	if ipv4 {
		if ipv6 {
			return DualStack
		}
		return IPv4
	}
	if ipv6 {
		return IPv6
	}
	return dfl
}

// IPVersionFromIP returns an IPVersion from a net.IP.
func IPVersionFromIP(ip net.IP) IPVersion {
	if ip.To4() != nil {
		return IPv4
	}
	return IPv6
}

// IPVersionFromUDPAddr returns an IPVersion from a net.UDPAddr.
func IPVersionFromUDPAddr(addr *net.UDPAddr) IPVersion {
	return IPVersionFromIP(addr.IP)
}

var udpNets = [...]string{"udp4", "udp6", "udp"}

func (v IPVersion) udpNetwork() string {
	if int(v-1) < 0 || int(v-1) > len(udpNets) {
		return fmt.Sprintf("IPVersion.udpNetwork:%d", v)
	}
	return udpNets[v-1]
}

// 28 == 20 (min IPv4 header) + 8 (UDP header)
// 48 == 40 (min IPv4 header) + 8 (UDP header)
//var muhs = [...]int{28, 48, 28}

// func (v IPVersion) minUDPHeaderSize() int {
// 	return muhs[v-1]
// }

var ipvs = [...]string{"IPv4", "IPv6", "IPv4+6"}

func (v IPVersion) String() string {
	if int(v-1) < 0 || int(v-1) > len(ipvs) {
		return fmt.Sprintf("IPVersion:%d", v)
	}
	return ipvs[v-1]
}

//var ipvi = [...]int{4, 6, 46}

// Separate returns a slice of IPVersions, separating DualStack into IPv4 and
// IPv6 if necessary.
func (v IPVersion) Separate() []IPVersion {
	if v == IPv4 {
		return []IPVersion{IPv4}
	}
	if v == IPv6 {
		return []IPVersion{IPv6}
	}
	return []IPVersion{IPv4, IPv6}
}

// ZeroIP returns the zero IP for the IPVersion (net.IPv4zero for IPv4 and
// otherwise net.IPv6zero).
func (v IPVersion) ZeroIP() net.IP {
	if v == IPv4 {
		return net.IPv4zero
	}
	return net.IPv6zero
}

// MarshalJSON implements the json.Marshaler interface.
func (v IPVersion) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

// addPort adds the default port to a string, if the string does not
// already contain a port.
func addPort(hostport, port string) string {
	if _, _, err := net.SplitHostPort(hostport); err != nil {
		// JoinHostPort doesn't seem to work with IPv6 addresses with [], so I
		// join manually.
		return fmt.Sprintf("%s:%s", hostport, port)
	}
	return hostport
}

// udpAddrsEqual returns true if all fields of the passed in UDP addresses are
// equal.
func udpAddrsEqual(a1 *net.UDPAddr, a2 *net.UDPAddr) bool {
	if !a1.IP.Equal(a2.IP) {
		return false
	}
	if a1.Port != a2.Port {
		return false
	}
	return a1.Zone == a2.Zone
}
