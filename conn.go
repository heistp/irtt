package irtt

import (
	"context"
	"net"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const minOpenTimeout = 200 * time.Millisecond

// nconn (network conn) is the embedded struct in conn and lconn connections. It
// adds IPVersion, socket options and some helpers to net.UDPConn.
type nconn struct {
	conn    *net.UDPConn
	ipVer   IPVersion
	ip4conn *ipv4.Conn
	ip6conn *ipv6.Conn
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
		n.ip4conn = ipv4.NewConn(n.conn)
	} else {
		n.ip6conn = ipv6.NewConn(n.conn)
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

// lconn is used in the lconn
type lconn struct {
	*nconn
	pkt *packet
}

// listen creates an lconn by listening on a UDP address. It takes an IPVersion
// because the UDPAddr returned by ResolveUDPAddr has the network "udp", as
// opposed to either "udp4" or "udp6", even when an explicit network was
// requested.
func listen(ipVer IPVersion, laddr *net.UDPAddr, maxLen int,
	hmacKey []byte) (l *lconn, err error) {
	var conn *net.UDPConn
	conn, err = net.ListenUDP(ipVer.udpNetwork(), laddr)
	if err != nil {
		return
	}
	l = &lconn{nconn: &nconn{}}
	l.init(conn, ipVer)
	var cap int
	if maxLen == 0 {
		cap, _ = detectMTU(l.localAddr().IP)
	} else if maxLen < maxHeaderLen {
		// TODO this could actually be down to the minimum test packet size
		cap = maxHeaderLen
	} else {
		cap = maxLen
	}
	l.pkt = newPacket(0, cap, hmacKey)
	return
}

// listenAll creates lconns on multiple addresses, with separate lconns for IPv4
// and IPv6, so that socket options can be set correctly, which is not possible
// with a dual stack conn.
func listenAll(ipVer IPVersion, addrs []string, maxPacketLen int,
	hmacKey []byte) ([]*lconn, error) {
	lconns := make([]*lconn, 0, 16)

	for _, addr := range addrs {
		addr := addPort(addr, DefaultPort)
		for _, v := range ipVer.Separate() {
			laddr, err := net.ResolveUDPAddr(v.udpNetwork(), addr)
			if err != nil {
				continue
			}
			l, err := listen(v, laddr, maxPacketLen, hmacKey)
			if err != nil {
				return nil, err
			}
			lconns = append(lconns, l)
		}
	}

	if len(lconns) == 0 {
		return nil, Errorf(NoSuitableAddressFound,
			"no suitable %s address found", ipVer)
	}

	return lconns, nil
}

func (l *lconn) sendTo(addr *net.UDPAddr) (err error) {
	var n int
	n, err = l.conn.WriteToUDP(l.pkt.bytes(), addr)
	if err != nil {
		return
	}
	if n < l.pkt.length() {
		err = Errorf(ShortWrite, "only %d/%d bytes were sent", n, l.pkt.length())
	}
	return
}

func (l *lconn) receiveFrom() (tafter time.Time, raddr *net.UDPAddr, err error) {
	var n int
	n, raddr, err = l.conn.ReadFromUDP(l.pkt.readTo())
	tafter = time.Now()
	if err != nil {
		return
	}
	if err != nil {
		return
	}
	if err = l.pkt.readReset(n); err != nil {
		return
	}
	if l.pkt.reply() {
		err = Errorf(UnexpectedReplyFlag, "unexpected reply flag set")
		return
	}
	return
}
