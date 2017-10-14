// +build openbsd freebsd

package irtt

import (
	"net"

	"golang.org/x/sys/unix"
)

func setSockoptDF(conn *net.UDPConn, df DF) error {
	var value int
	switch df {
	case DFDefault:
		value = 0
	case DFTrue:
		value = 1
	case DFFalse:
		value = 0
	}
	return setSockoptInt(conn, unix.IPPROTO_IP, unix.IP_DF, value)
}
