// +build linux

package irtt

import (
	"net"

	"golang.org/x/sys/unix"
)

func setSockoptDF(conn *net.UDPConn, df DF) error {
	var value int
	switch df {
	case DFDefault:
		value = unix.IP_PMTUDISC_WANT
	case DFTrue:
		value = unix.IP_PMTUDISC_DO
	case DFFalse:
		value = unix.IP_PMTUDISC_DONT
	}
	return setSockoptInt(conn, unix.IPPROTO_IP, unix.IP_MTU_DISCOVER, value)
}
