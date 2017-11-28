package irtt

import (
	"bytes"
	"context"
	"net"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const minOpenTimeout = 200 * time.Millisecond

const setSourceAddress = true

// nconn (network conn) is the embedded struct in conn and lconn connections. It
// adds IPVersion, socket options and some helpers to net.UDPConn.
type nconn struct {
	conn    *net.UDPConn
	ipVer   IPVersion
	ip4conn *ipv4.PacketConn
	ip6conn *ipv6.PacketConn
	dscp    int
	ttl     int
	df      DF
}

func (n *nconn) init(conn *net.UDPConn, ipVer IPVersion) {
	n.conn = conn
	n.ipVer = ipVer
	n.df = DFDefault

	// create x/net conns for socket options
	if n.ipVer&IPv4 != 0 {
		n.ip4conn = ipv4.NewPacketConn(n.conn)
	} else {
		n.ip6conn = ipv6.NewPacketConn(n.conn)
	}
}

func (n *nconn) setDSCP(dscp int) (err error) {
	if n.dscp == dscp {
		return
	}
	if n.ip4conn != nil {
		err = n.ip4conn.SetTOS(dscp)
	} else {
		err = n.ip6conn.SetTrafficClass(dscp)
	}
	if err == nil {
		n.dscp = dscp
	}
	return
}

func (n *nconn) setTTL(ttl int) (err error) {
	if n.ttl == ttl {
		return
	}
	if n.ip4conn != nil {
		err = n.ip4conn.SetTTL(ttl)
	} else {
		err = n.ip6conn.SetHopLimit(ttl)
	}
	if err == nil {
		n.ttl = ttl
	}
	return
}

func (n *nconn) setReceiveDstAddr(b bool) (err error) {
	if n.ip4conn != nil {
		err = n.ip4conn.SetControlMessage(ipv4.FlagDst, b)
	} else {
		err = n.ip6conn.SetControlMessage(ipv6.FlagDst, b)
	}
	return
}

func (n *nconn) setDF(df DF) (err error) {
	if n.df == df {
		return
	}
	err = setSockoptDF(n.conn, df)
	if err == nil {
		n.df = df
	}
	return
}

func (n *nconn) localAddr() *net.UDPAddr {
	if n.conn == nil {
		return nil
	}
	a := n.conn.LocalAddr()
	if a == nil {
		return nil
	}
	return a.(*net.UDPAddr)
}

func (n *nconn) close() error {
	return n.conn.Close()
}

// cconn is used for client connections
type cconn struct {
	*nconn
	spkt   *packet
	rpkt   *packet
	ctoken ctoken
}

func dial(ctx context.Context, cfg *Config) (*cconn, error) {
	// resolve (could support trying multiple addresses in succession)
	cfg.LocalAddress = addPort(cfg.LocalAddress, DefaultLocalPort)
	laddr, err := net.ResolveUDPAddr(cfg.IPVersion.udpNetwork(),
		cfg.LocalAddress)
	if err != nil {
		return nil, err
	}

	// add default port, if necessary, and resolve server
	cfg.RemoteAddress = addPort(cfg.RemoteAddress, DefaultPort)
	raddr, err := net.ResolveUDPAddr(cfg.IPVersion.udpNetwork(),
		cfg.RemoteAddress)
	if err != nil {
		return nil, err
	}

	// dial, using explicit network from remote address
	cfg.IPVersion = IPVersionFromUDPAddr(raddr)
	conn, err := net.DialUDP(cfg.IPVersion.udpNetwork(), laddr, raddr)
	if err != nil {
		return nil, err
	}

	// set resolved local and remote addresses back to Config
	cfg.LocalAddr = conn.LocalAddr()
	cfg.RemoteAddr = conn.RemoteAddr()
	cfg.LocalAddress = cfg.LocalAddr.String()
	cfg.RemoteAddress = cfg.RemoteAddr.String()

	// create cconn
	c := &cconn{nconn: &nconn{}}
	c.init(conn, cfg.IPVersion)

	// create send and receive packets
	cap := cfg.Length
	if cap < maxHeaderLen {
		cap = maxHeaderLen
	}
	c.spkt = newPacket(0, cap, cfg.HMACKey)
	c.rpkt = newPacket(0, cap, cfg.HMACKey)

	// open connection to server
	if err = c.open(ctx, cfg); err != nil {
		return c, err
	}

	return c, nil
}

func (c *cconn) open(ctx context.Context, cfg *Config) (err error) {
	// validate open timeouts
	for _, to := range cfg.OpenTimeouts {
		if to < minOpenTimeout {
			err = Errorf(OpenTimeoutTooShort,
				"open timeout %s must be >= %s", to, minOpenTimeout)
			return
		}
	}

	errC := make(chan error)
	params := &cfg.Params

	// start receiving open replies and drop anything else
	go func() {
		var rerr error
		defer func() {
			errC <- rerr
		}()

		for {
			_, rerr = c.receive()
			if rerr != nil {
				return
			}
			if c.rpkt.flags()&flOpen == 0 {
				continue
			}
			if rerr = c.rpkt.addFields(fopenReply, false); rerr != nil {
				return
			}
			if c.rpkt.flags()&flClose == 0 && c.rpkt.ctoken() == 0 {
				rerr = Errorf(ConnTokenZero, "received invalid zero conn token")
				return
			}
			var sp *Params
			sp, rerr = parseParams(c.rpkt.payload())
			if rerr != nil {
				return
			}
			*params = *sp
			c.ctoken = c.rpkt.ctoken()
			if c.rpkt.flags()&flClose != 0 {
				rerr = Errorf(ServerClosed, "server closed connection during open")
				c.close()
			}
			return
		}
	}()

	// start sending open requests
	defer func() {
		c.spkt.clearFlagBits(flOpen | flClose)
		c.spkt.setPayload([]byte{})
		c.spkt.setConnToken(c.ctoken)
		if err != nil {
			c.close()
		}
	}()
	c.spkt.setFlagBits(flOpen)
	if cfg.NoTest {
		c.spkt.setFlagBits(flClose)
	}
	c.spkt.setPayload(params.bytes())
	c.spkt.updateHMAC()
	var received bool
	for _, to := range cfg.OpenTimeouts {
		_, err = c.send()
		if err != nil {
			return
		}
		select {
		case <-time.After(to):
		case err = <-errC:
			received = true
			return
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
	if !received {
		defer c.nconn.close()
		err = Errorf(OpenTimeout, "no reply from server")
	}
	return
}

func (c *cconn) send() (tafter time.Time, err error) {
	var n int
	n, err = c.conn.Write(c.spkt.bytes())
	tafter = time.Now()
	if err != nil {
		return
	}
	if n < c.spkt.length() {
		err = Errorf(ShortWrite, "only %d/%d bytes were sent", n, c.spkt.length())
	}
	return
}

func (c *cconn) receive() (tafter time.Time, err error) {
	var n int
	n, err = c.conn.Read(c.rpkt.readTo())
	tafter = time.Now()
	if err != nil {
		return
	}
	if err = c.rpkt.readReset(n); err != nil {
		return
	}
	if !c.rpkt.reply() {
		err = Errorf(ExpectedReplyFlag, "reply flag not set")
		return
	}
	if c.rpkt.flags()&flClose != 0 {
		err = Errorf(ServerClosed, "server closed connection")
		c.close()
		return
	}
	return
}

func (c *cconn) remoteAddr() *net.UDPAddr {
	if c.conn == nil {
		return nil
	}
	a := c.conn.RemoteAddr()
	if a == nil {
		return nil
	}
	return a.(*net.UDPAddr)
}

func (c *cconn) close() (err error) {
	defer func() {
		err = c.nconn.close()
	}()

	// send one close packet if necessary
	if c.ctoken != 0 {
		if err = c.spkt.setFields(fcloseRequest, true); err != nil {
			return
		}
		c.spkt.setFlagBits(flClose)
		c.spkt.setConnToken(c.ctoken)
		c.spkt.updateHMAC()
		_, err = c.send()
	}
	return
}

// lconn is used for server listeners
type lconn struct {
	*nconn
	//pkt *packet
	cm4 ipv4.ControlMessage
	cm6 ipv6.ControlMessage
}

// listen creates an lconn by listening on a UDP address.
func listen(laddr *net.UDPAddr) (l *lconn, err error) {
	ipVer := IPVersionFromUDPAddr(laddr)
	var conn *net.UDPConn
	conn, err = net.ListenUDP(ipVer.udpNetwork(), laddr)
	if err != nil {
		return
	}
	l = &lconn{nconn: &nconn{}}
	l.init(conn, ipVer)
	return
}

// listenAll creates lconns on multiple addresses, with separate lconns for IPv4
// and IPv6, so that socket options can be set correctly, which is not possible
// with a dual stack conn.
func listenAll(ipVer IPVersion, addrs []string) (lconns []*lconn, err error) {
	laddrs, err := resolveListenAddrs(addrs, ipVer)
	if err != nil {
		return
	}
	lconns = make([]*lconn, 0, 16)
	for _, laddr := range laddrs {
		var l *lconn
		l, err = listen(laddr)
		if err != nil {
			return
		}
		lconns = append(lconns, l)
	}
	if len(lconns) == 0 {
		err = Errorf(NoSuitableAddressFound, "no suitable %s address found", ipVer)
		return
	}
	return
}

func (l *lconn) sendTo(pkt *packet, addr *net.UDPAddr, srcIP net.IP) (err error) {
	var n int
	if setSourceAddress && l.ip4conn != nil {
		l.cm4.Src = srcIP
		n, err = l.ip4conn.WriteTo(pkt.bytes(), &l.cm4, addr)
	} else if setSourceAddress && l.ip6conn != nil {
		l.cm6.Src = srcIP
		n, err = l.ip6conn.WriteTo(pkt.bytes(), &l.cm6, addr)
	} else {
		n, err = l.conn.WriteToUDP(pkt.bytes(), addr)
	}
	if err != nil {
		return
	}
	if n < pkt.length() {
		err = Errorf(ShortWrite, "only %d/%d bytes were sent", n, pkt.length())
	}
	return
}

func (l *lconn) receiveFrom(pkt *packet) (tafter time.Time, dstIP net.IP,
	raddr *net.UDPAddr, err error) {
	var n int
	if setSourceAddress && l.ip4conn != nil {
		var cm *ipv4.ControlMessage
		var src net.Addr
		n, cm, src, err = l.ip4conn.ReadFrom(pkt.readTo())
		if src != nil {
			raddr = src.(*net.UDPAddr)
		}
		if cm != nil {
			dstIP = cm.Dst
		}
	} else if setSourceAddress && l.ip6conn != nil {
		var cm *ipv6.ControlMessage
		var src net.Addr
		n, cm, src, err = l.ip6conn.ReadFrom(pkt.readTo())
		if src != nil {
			raddr = src.(*net.UDPAddr)
		}
		if cm != nil {
			dstIP = cm.Dst
		}
	} else {
		n, raddr, err = l.conn.ReadFromUDP(pkt.readTo())
	}
	tafter = time.Now()
	if err != nil {
		return
	}
	if err = pkt.readReset(n); err != nil {
		return
	}
	if pkt.reply() {
		err = Errorf(UnexpectedReplyFlag, "unexpected reply flag set")
		return
	}
	return
}

// parseIfaceListenAddr parses an interface listen address into an interface
// name and service. ok is false if the string does not use the syntax
// %iface:service, where :service is optional.
func parseIfaceListenAddr(addr string) (iface, service string, ok bool) {
	if !strings.HasPrefix(addr, "%") {
		return
	}
	parts := strings.Split(addr[1:], ":")
	switch len(parts) {
	case 2:
		service = parts[1]
		if len(service) == 0 {
			return
		}
		fallthrough
	case 1:
		iface = parts[0]
		if len(iface) == 0 {
			return
		}
		ok = true
		return
	}
	return
}

// resolveIfaceListenAddr resolves an interface name and service (port name
// or number) into a slice of UDP addresses.
func resolveIfaceListenAddr(ifaceName string, service string,
	ipVer IPVersion) (laddrs []*net.UDPAddr, err error) {
	// get interfaces
	var ifaces []net.Interface
	ifaces, err = net.Interfaces()
	if err != nil {
		return
	}

	// resolve service to port
	var port int
	if service != "" {
		port, err = net.LookupPort(ipVer.udpNetwork(), service)
		if err != nil {
			return
		}
	} else {
		port = DefaultPortInt
	}

	// helper to get IP and zone from interface address
	ifaceIP := func(a net.Addr) (ip net.IP, zone string, ok bool) {
		switch v := a.(type) {
		case *net.IPNet:
			{
				ip = v.IP
				ok = true
			}
		case *net.IPAddr:
			{
				ip = v.IP
				zone = v.Zone
				ok = true
			}
		}
		return
	}

	// helper to test if IP is one we can listen on
	isUsableIP := func(ip net.IP) bool {
		if IPVersionFromIP(ip)&ipVer == 0 {
			return false
		}
		if !ip.IsLinkLocalUnicast() && !ip.IsGlobalUnicast() && !ip.IsLoopback() {
			return false
		}
		return true
	}

	// get addresses
	laddrs = make([]*net.UDPAddr, 0, 16)
	ifaceFound := false
	ifaceUp := false
	for _, iface := range ifaces {
		if !glob(ifaceName, iface.Name) {
			continue
		}
		ifaceFound = true
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		ifaceUp = true
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, a := range ifaceAddrs {
			ip, zone, ok := ifaceIP(a)
			if ok && isUsableIP(ip) {
				if ip.IsLinkLocalUnicast() && zone == "" {
					zone = iface.Name
				}
				udpAddr := &net.UDPAddr{IP: ip, Port: port, Zone: zone}
				laddrs = append(laddrs, udpAddr)
			}
		}
	}

	if !ifaceFound {
		err = Errorf(NoMatchingInterfaces, "%s does not match any interfaces", ifaceName)
	} else if !ifaceUp {
		err = Errorf(NoMatchingInterfacesUp, "no interfaces matching %s are up", ifaceName)
	}

	return
}

// resolveListenAddr resolves a listen address string into a slice of UDP
// addresses.
func resolveListenAddr(addr string, ipVer IPVersion) (laddrs []*net.UDPAddr,
	err error) {
	laddrs = make([]*net.UDPAddr, 0, 2)
	for _, v := range ipVer.Separate() {
		addr = addPort(addr, DefaultPort)
		laddr, err := net.ResolveUDPAddr(v.udpNetwork(), addr)
		if err != nil {
			continue
		}
		if laddr.IP == nil {
			laddr.IP = v.ZeroIP()
		}
		laddrs = append(laddrs, laddr)
	}
	return
}

// resolveListenAddrs resolves a slice of listen address strings into a slice
// of UDP addresses.
func resolveListenAddrs(addrs []string, ipVer IPVersion) (laddrs []*net.UDPAddr,
	err error) {
	// resolve addresses
	laddrs = make([]*net.UDPAddr, 0, 16)
	for _, addr := range addrs {
		var la []*net.UDPAddr
		iface, service, ok := parseIfaceListenAddr(addr)
		if ok {
			la, err = resolveIfaceListenAddr(iface, service, ipVer)
		} else {
			la, err = resolveListenAddr(addr, ipVer)
		}
		if err != nil {
			return
		}
		laddrs = append(laddrs, la...)
	}
	// sort addresses
	sort.Slice(laddrs, func(i, j int) bool {
		if bytes.Compare(laddrs[i].IP, laddrs[j].IP) < 0 {
			return true
		}
		if laddrs[i].Port < laddrs[j].Port {
			return true
		}
		return laddrs[i].Zone < laddrs[j].Zone
	})
	// remove duplicates
	udpAddrsEqual := func(a *net.UDPAddr, b *net.UDPAddr) bool {
		if !a.IP.Equal(b.IP) {
			return false
		}
		if a.Port != b.Port {
			return false
		}
		return a.Zone == b.Zone
	}
	for i := 1; i < len(laddrs); i++ {
		if udpAddrsEqual(laddrs[i], laddrs[i-1]) {
			laddrs = append(laddrs[:i], laddrs[i+1:]...)
			i--
		}
	}
	// check for combination of specified and unspecified IP addresses
	m := make(map[int]int)
	for _, la := range laddrs {
		if la.IP.IsUnspecified() {
			m[la.Port] = m[la.Port] | 1
		} else {
			m[la.Port] = m[la.Port] | 2
		}
	}
	for k, v := range m {
		if v > 2 {
			err = Errorf(UnspecifiedWithSpecifiedAddresses,
				"invalid combination of unspecified and specified IP addresses port %d", k)
			break
		}
	}
	return
}
