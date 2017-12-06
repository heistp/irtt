package irtt

import (
	"math/rand"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// settings for testing
const serverDupsPercent = 0
const serverDropsPercent = 0

// Server is the irtt server.
type Server struct {
	*ServerConfig
	start       time.Time
	connRefs    int
	connRefMtx  sync.Mutex
	shutdown    bool
	shutdownMtx sync.Mutex
	shutdownC   chan struct{}
}

// NewServer returns a new server.
func NewServer(cfg *ServerConfig) *Server {
	return &Server{
		ServerConfig: cfg,
		shutdownC:    make(chan struct{}),
	}
}

// ListenAndServe creates listeners for all requested addresses and serves
// requests indefinitely. It exits after the listeners have exited. Errors for
// individual listeners may be handled with a ServerHandler, and will not be
// returned from this method.
func (s *Server) ListenAndServe() error {
	// start is the base time that monotonic timestamp values are from
	s.start = time.Now()

	// send ServerStart event
	if s.Handler != nil {
		s.Handler.OnEvent(Eventf(ServerStart, nil, nil,
			"starting IRTT server version %s", Version))
	}

	// make listeners
	listeners, err := s.makeListeners()
	if err != nil {
		return err
	}

	// start listeners
	errC := make(chan error)
	for _, l := range listeners {
		// send ListenerStart event
		l.eventf(ListenerStart, nil, "starting %s listener on %s", l.conn.ipVer,
			l.conn.localAddr())

		go l.listenAndServe(errC)
	}

	// disable GC, if requested
	if s.GCMode == GCOff {
		debug.SetGCPercent(-1)
	}

	// wait on shutdown chan
	go func() {
		<-s.shutdownC
		for _, l := range listeners {
			l.shutdown()
		}
	}()

	// wait for all listeners, and out of an abundance of caution, shut down
	// all other listeners if any one of them fails
	for i := 0; i < len(listeners); i++ {
		if err := <-errC; err != nil {
			s.Shutdown()
		}
	}

	return nil
}

// Shutdown stops the Server. After this call, the Server may no longer be used.
func (s *Server) Shutdown() {
	s.shutdownMtx.Lock()
	defer s.shutdownMtx.Unlock()
	if !s.shutdown {
		close(s.shutdownC)
		s.shutdown = true
	}
}

func (s *Server) makeListeners() ([]*listener, error) {
	lconns, err := listenAll(s.IPVersion, s.Addrs, s.SetSrcIP)
	if err != nil {
		return nil, err
	}
	ls := make([]*listener, 0, len(lconns))
	for _, lconn := range lconns {
		ls = append(ls, newListener(s.ServerConfig, lconn, s.connRef))
	}
	return ls, nil
}

func (s *Server) connRef(b bool) {
	s.connRefMtx.Lock()
	defer s.connRefMtx.Unlock()
	if b {
		s.connRefs++
		if s.connRefs == 1 {
			runtime.GC()
			if s.GCMode == GCIdle {
				debug.SetGCPercent(-1)
			}
		}
	} else {
		s.connRefs--
		if s.connRefs == 0 {
			if s.GCMode == GCIdle {
				debug.SetGCPercent(100)
			}
			runtime.GC()
		}
	}
}

// listener is a server listener.
type listener struct {
	*ServerConfig
	conn      *lconn
	pktPool   *pktPool
	cmgr      *connmgr
	closed    bool
	closedMtx sync.Mutex
}

func newListener(cfg *ServerConfig, lc *lconn, cref func(bool)) *listener {
	cap, _ := detectMTU(lc.localAddr().IP)

	pp := newPacketPool(func() *packet {
		return newPacket(0, cap, cfg.HMACKey)
	}, 16)

	return &listener{
		ServerConfig: cfg,
		conn:         lc,
		pktPool:      pp,
		cmgr:         newConnMgr(cfg, cref),
	}
}

func (l *listener) listenAndServe(errC chan<- error) (err error) {
	// always return error to channel
	defer func() {
		errC <- err
	}()

	// always close conn
	defer func() {
		l.conn.close()
	}()

	// always log error or stoppage
	defer func() {
		if err != nil {
			l.eventf(ListenerError, nil, "listener shut down due to error (%s)", err)
		} else {
			l.eventf(ListenerStop, nil, "listener stopped")
		}
	}()

	// lock to thread
	if l.ThreadLock {
		runtime.LockOSThread()
	}

	// set TTL
	if l.TTL != 0 {
		err = l.conn.setTTL(l.TTL)
		if err != nil {
			return
		}
	}

	// warn if DSCP not supported
	if l.AllowDSCP && !l.conn.dscpSupport {
		l.eventf(NoDSCPSupport, nil, "[%s] no %s DSCP support available (%s)",
			l.conn.localAddr(), l.conn.ipVer, l.conn.dscpError)
	}

	// enable receipt of destination IP
	if l.SetSrcIP && l.conn.localAddr().IP.IsUnspecified() {
		if rdsterr := l.conn.setReceiveDstAddr(true); rdsterr != nil {
			l.eventf(NoReceiveDstAddrSupport, nil,
				"no support for determining packet destination address (%s)", rdsterr)
			if err := l.warnOnMultipleAddresses(); err != nil {
				return err
			}
		}
	}

	if l.Concurrent {
		err = l.readAndReplyConcurrent()
	} else {
		err = l.readAndReply()
	}
	if l.isClosed() {
		err = nil
	}
	return
}

func (l *listener) readAndReplyConcurrent() (err error) {
	panic("not implemented")
}

func (l *listener) readAndReply() error {
	p := l.pktPool.new()
	for {
		if fatal, err := l.readOneAndReply(p); fatal {
			return err
		}
	}
}

func (l *listener) readOneAndReply(p *packet) (fatal bool, err error) {
	// read a packet
	err = l.conn.receive(p)
	if err != nil {
		if _, ok := err.(*Error); ok {
			l.eventf(Drop, p.raddr, "%s", err.Error())
		} else {
			fatal = true
		}
		return
	}

	// set source IP from received destination, if necessary
	if l.SetSrcIP {
		p.srcIP = p.dstIP
	}

	// handle open
	if p.flags()&flOpen != 0 {
		var params *Params
		params, err = parseParams(p.payload())
		if err != nil {
			l.eventf(DropUnparseableParams, p.raddr,
				"unparseable negotiation parameters: %s", err.Error())
			return
		}
		l.restrictParams(p, params)
		sc := l.cmgr.new(p.raddr, params, p.flags()&flClose != 0)
		if p.flags()&flClose == 0 {
			l.eventf(NewConn, p.raddr, "new connection, token=%016x",
				sc.ctoken)
			p.setConnToken(sc.ctoken)
		} else {
			l.eventf(OpenClose, p.raddr, "open-close")
			p.setConnToken(0)
		}
		p.setReply(true)
		p.setPayload(params.bytes())
		if err = l.sendPacket(p, sc, false); err != nil {
			fatal = true
		}
		return
	}

	// handle close
	if p.flags()&flClose != 0 {
		if !l.addFields(p, fcloseRequest) {
			return
		}
		sc := l.cmgr.remove(p.ctoken())
		if sc == nil {
			l.eventf(DropInvalidConnToken, p.raddr,
				"close for invalid conn token %016x", p.ctoken())
			return
		}
		// check remote address
		if !udpAddrsEqual(p.raddr, &sc.raddr) {
			l.eventf(DropAddressMismatch, p.raddr,
				"drop close due to address mismatch (expected %s for %016x)",
				&sc.raddr, p.ctoken())
			return
		}
		l.eventf(CloseConn, p.raddr, "close connection, token=%016x", sc.ctoken)
		return
	}

	// handle echo request
	if !l.addFields(p, fechoRequest) {
		return
	}

	// check conn, token and address
	sc, exists, addrOk, intervalOk := l.cmgr.get(p)
	if !exists {
		l.eventf(DropInvalidConnToken, p.raddr,
			"request for invalid conn token %016x", p.ctoken())
		return
	}
	if !addrOk {
		l.eventf(DropAddressMismatch, p.raddr,
			"drop request due to address mismatch (expected %s for %016x)",
			&sc.raddr, p.ctoken())
		return
	}
	if !intervalOk {
		l.eventf(DropShortInterval, p.raddr,
			"drop request due to short interval")
		return
	}

	// set reply flag
	p.setReply(true)

	// check if max test duration exceeded (but still return packet)
	if l.MaxDuration > 0 && time.Since(sc.firstUsed) > l.MaxDuration {
		l.eventf(DurationLimitExceeded, p.raddr,
			"closing connection due to duration limit exceeded")
		l.cmgr.remove(p.ctoken())
		p.setFlagBits(flClose)
	}

	// fill payload
	if l.Filler != nil {
		err = p.readPayload(l.Filler)
		if err != nil {
			fatal = true
			return
		}
	}

	// send response
	if err = l.sendPacket(p, sc, true); err != nil {
		fatal = true
	}

	return
}

// sendPacket sends a packet, locking and setting socket options as necessary.
func (l *listener) sendPacket(p *packet, sc *sconn, testPacket bool) (err error) {
	// for test packets, add stats and timestamps according to conn params
	if testPacket {
		if l.AllowDSCP && l.conn.dscpSupport {
			p.dscp = sc.params.DSCP
		}
		p.setLen(0)
		if sc.params.ReceivedStats&ReceivedStatsCount != 0 {
			p.setReceivedCount(sc.receivedCount)
		}
		if sc.params.ReceivedStats&ReceivedStatsWindow != 0 {
			if sc.rwinValid {
				p.setReceivedWindow(sc.receivedWindow)
			} else {
				p.setReceivedWindow(0)
			}
		}
		at := sc.params.StampAt
		cl := sc.params.Clock
		if at != AtNone {
			var rt Time
			var st Time
			if at == AtMidpoint {
				mt := midpoint(p.trcvd, time.Now())
				rt = newTime(mt, cl)
				st = newTime(mt, cl)
			} else {
				if at&AtReceive != 0 {
					rt = newTime(p.trcvd, cl)
				}
				if at&AtSend != 0 {
					st = newTime(time.Now(), cl)
				}
			}
			p.setTimestamp(Timestamp{rt, st})
		} else {
			p.removeTimestamps()
		}
		p.setLen(sc.params.Length)
	}

	// calculate HMAC
	p.updateHMAC()

	// simulate dropped packets, if necessary
	if serverDropsPercent > 0 && rand.Float32() < serverDropsPercent {
		return
	}

	// simulate duplicates, if necessary
	if serverDupsPercent > 0 {
		for rand.Float32() < serverDupsPercent {
			err = l.conn.send(p)
			if err != nil {
				return
			}
		}
	}

	err = l.conn.send(p)

	return
}

func (l *listener) restrictParams(pkt *packet, p *Params) {
	if l.MaxDuration > 0 && p.Duration > l.MaxDuration {
		p.Duration = l.MaxDuration
	}
	if l.MinInterval > 0 && p.Interval < l.MinInterval {
		p.Interval = l.MinInterval
	}
	if p.Length > pkt.capacity() {
		p.Length = pkt.capacity()
	}
	p.StampAt = l.AllowStamp.Restrict(p.StampAt)
	if !l.AllowDSCP || !l.conn.dscpSupport {
		p.DSCP = 0
	}
}

func (l *listener) addFields(p *packet, fidxs []fidx) bool {
	if err := p.addFields(fidxs, false); err != nil {
		if _, ok := err.(*Error); ok {
			l.eventf(Drop, p.raddr, "%s", err.Error())
		}
		return false
	}
	return true
}

func (l *listener) warnOnMultipleAddresses() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	n := 0
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.IsGlobalUnicast() {
					n++
				}
			case *net.IPAddr:
				if v.IP.IsGlobalUnicast() {
					n++
				}
			}
		}
	}
	if n > 1 {
		l.eventf(MultipleAddresses, nil, "warning: multiple IP addresses, "+
			"all bind addresses should be explicitly specified with -b or "+
			"clients may not be able to connect")
	}
	return nil
}

func (l *listener) eventf(code Code, raddr *net.UDPAddr, format string,
	detail ...interface{}) {
	if l.Handler != nil {
		l.Handler.OnEvent(Eventf(code, l.conn.localAddr(), raddr, format, detail...))
	}
}

func (l *listener) isClosed() bool {
	l.closedMtx.Lock()
	defer l.closedMtx.Unlock()
	return l.closed
}

func (l *listener) shutdown() {
	l.closedMtx.Lock()
	defer l.closedMtx.Unlock()
	if !l.closed {
		if l.conn != nil {
			l.conn.close()
		}
		l.closed = true
	}
}

type pktPool struct {
	pool []*packet
	mtx  sync.Mutex
	new  func() *packet
}

func newPacketPool(new func() *packet, cap int) *pktPool {
	pp := &pktPool{
		pool: make([]*packet, 0, cap),
		new:  new,
	}
	return pp
}

func (po *pktPool) get() *packet {
	po.mtx.Lock()
	defer po.mtx.Unlock()
	l := len(po.pool)
	if l == 0 {
		return po.new()
	}
	p := po.pool[l-1]
	po.pool = po.pool[:l-1]
	return p
}

func (po *pktPool) put(p *packet) {
	po.mtx.Lock()
	defer po.mtx.Unlock()
	po.pool = append(po.pool, p)
}
