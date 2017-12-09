package irtt

import (
	"fmt"
	"net"
)

// detectMTU autodetects and returns the MTU either for the interface associated
// with the specified IP, or if the ip parameter is nil, the max MTU of all
// interfaces, and if it cannot be determined, a fallback default.
func detectMTU(ip net.IP) (int, string) {
	if ip != nil {
		iface, err := interfaceByIP(ip)
		if err != nil || iface == nil {
			return maxOrDefaultMTU()
		}
		return iface.MTU, iface.Name
	}
	return maxOrDefaultMTU()
}

// maxOrDefaultMTU returns the maximum MTU for all interfaces, or the
// compiled-in default if it could not be determined.
func maxOrDefaultMTU() (int, string) {
	mtu, _, err := largestMTU(false)
	msg := "all"
	if err != nil || mtu < minValidMTU {
		msg = fmt.Sprintf("fallback (%s)", err)
		mtu = maxMTU
	} else if mtu < minValidMTU {
		msg = fmt.Sprintf("fallback (MTU %d too small)", mtu)
		mtu = maxMTU
	}
	return mtu, msg
}

// largestMTU queries all interfaces and returns the largest MTU. If the up
// parameter is true, only interfaces that are up are considered.
func largestMTU(up bool) (lmtu int, ifaces []string, err error) {
	ifcs, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, iface := range ifcs {
		ifaces = append(ifaces, iface.Name)
		if (!up || ((iface.Flags & net.FlagUp) != 0)) && iface.MTU > lmtu {
			lmtu = iface.MTU
		}
	}
	return
}

// interfaceByIP returns the first interface whose network contains the given
// IP address. An interface of nil is returned if no matching interface is
// found.
func interfaceByIP(ip net.IP) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		// I've only ever seen *net.IPNet returned by the Addrs() method,
		//but I'll test for *net.IPAddr just in case.
		for _, a := range addrs {
			switch ipv := a.(type) {
			case *net.IPNet:
				if ipv.IP.Equal(ip) {
					return &iface, nil
				}
			case *net.IPAddr:
				if ipv.IP.Equal(ip) {
					return &iface, nil
				}
			}
		}
	}
	return nil, nil
}
