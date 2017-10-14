// +build !linux,!openbsd,!freebsd

package irtt

import (
	"net"
)

func setSockoptDF(conn *net.UDPConn, df DF) error {
	return Errorf(DFNotSupported, "DF sockopt not supported")
}
