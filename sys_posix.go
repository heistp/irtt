// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package irtt

import (
	"net"

	"golang.org/x/sys/unix"
)

/*
// old syscall code, not used with golang.org/x/net

func setSockoptTOS(conn *net.UDPConn, tos int) error {
	return setSockoptInt(conn, unix.IPPROTO_IP, unix.IP_TOS, tos)
}

func setSockoptTrafficClass(conn *net.UDPConn, tclass int) error {
	return setSockoptInt(conn, unix.IPPROTO_IPV6, unix.IPV6_TCLASS, tclass)
}

func setSockoptTTL(conn *net.UDPConn, ttl int) error {
	return setSockoptInt(conn, unix.IPPROTO_IP, unix.IP_TTL, ttl)
}
*/

func setSockoptInt(conn *net.UDPConn, level int, opt int, value int) error {
	cfile, err := conn.File()
	if err != nil {
		return err
	}
	defer cfile.Close()
	fd := int(cfile.Fd())
	err = unix.SetsockoptInt(fd, level, opt, value)
	if err != nil {
		return err
	}
	return unix.SetNonblock(fd, true)
}
