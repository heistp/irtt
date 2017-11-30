package irtt

import (
	"math/rand"
	"net"
	"runtime"
	"sync"
	"time"
)

// settings for testing
const serverDupsPercent = 0
const serverDropsPercent = 0

// the grace period after the max duration is up when the conn is closed.
const durationGraceTime = 10 * time.Second

// Server is the irtt server.
type Server struct {
	Addrs           []string
	HMACKey         []byte
	MaxDuration     time.Duration
	MinInterval     time.Duration
	MaxLength       int
	PacketBurst     int
	Filler          Filler
	AllowStamp      AllowStamp
	TTL             int
	IPVersion       IPVersion
	Handler         Handler
	EventMask       EventCode
	SetSourceIP     bool
	ThreadLock      bool
	hardMaxDuration time.Duration
	start           time.Time
	shutdown        bool
	shutdownMtx     sync.Mutex
	shutdownC       chan struct{}
}

// NewServer creates a new server.
func NewServer() *Server {
	return &Server{
		Addrs:       DefaultBindAddrs,
		MaxDuration: DefaultMaxDuration,
		MinInterval: DefaultMinInterval,
		MaxLength:   DefaultMaxLength,
		PacketBurst: DefaultPacketBurst,
		Filler:      DefaultServerFiller,
		AllowStamp:  DefaultAllowStamp,
		TTL:         DefaultTTL,
		IPVersion:   DefaultIPVersion,
		EventMask:   AllEvents,
		SetSourceIP: DefaultSetSrcIP,
		ThreadLock:  DefaultThreadLock,
		shutdownC:   make(chan struct{}),
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
	s.eventf(ServerStart, "starting IRTT server version %s", Version)

	// set max duration
	if s.MaxDuration > 0 {
		s.hardMaxDuration = s.MaxDuration + durationGraceTime
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
		l.eventf(ListenerStart, "starting %s listener on %s", l.conn.ipVer,
			l.conn.localAddr())

		go l.listenAndServe(errC)
	}

	// wait for all listeners, and out of an abundance of caution, shut down
	// all other listeners if any one of them fails
	for i := 0; i < len(listeners); {
		select {
		case err := <-errC:
			if err != nil {
				s.Shutdown()
			}
			i++
		case <-s.shutdownC:
			for _, l := range listeners {
				l.shutdown()
			}
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

func (s *Server) warnOnMultipleAddresses() error {
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
		s.eventf(MultipleAddresses, "warning: multiple IP addresses, all bind addresses "+
			"should be explicitly specified with -b or clients may not be able to connect")
	}
	return nil
}

func (s *Server) makeListeners() ([]*listener, error) {
	lconns, err := listenAll(s.IPVersion, s.Addrs, s.SetSourceIP)
	if err != nil {
		return nil, err
	}
	ls := make([]*listener, 0, len(lconns))
	for _, lconn := range lconns {
		ls = append(ls, newListener(s, lconn, newConnMgr(s.PacketBurst, s.MinInterval)))
	}
	return ls, nil
}

func (s *Server) eventf(code EventCode, format string, args ...interface{}) {
	if s.Handler != nil && s.EventMask&code != 0 {
		s.Handler.OnEvent(Eventf(code, nil, nil, format, args...))
	}
}

// listener is a server listener.
type listener struct {
	*Server
	conn        *lconn
	pktPool     *pktPool
	cmgr        *connmgr
	raddr       *net.UDPAddr
	mtx         sync.Mutex
	dscp        int
	dscpSupport bool
	closed      bool
	closedMtx   sync.Mutex
}

func newListener(s *Server, lc *lconn, cmgr *connmgr) *listener {
	/*
		// this was the old logic for packet max length calculation- necessary?
		var cap int
		if l.MaxLength == 0 {
			cap, _ = detectMTU(l.conn.localAddr().IP)
		} else if l.MaxLength < maxHeaderLen {
			// this could actually be down to the minimum test packet size, but is
			// not worth that effort now
			cap = maxHeaderLen
		} else {
			cap = l.MaxLength
		}
		p := newPacket(0, cap, l.HMACKey)
	*/

	pp := newPacketPool(func() *packet {
		return newPacket(0, s.MaxLength, s.HMACKey)
	}, 16)

	return &listener{
		Server:  s,
		conn:    lc,
		pktPool: pp,
		cmgr:    newConnMgr(s.PacketBurst, s.MinInterval),
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
			l.eventf(ListenerError, "listener shut down due to error (%s)", err)
		} else {
			l.eventf(ListenerStop, "listener stopped")
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

	// determine if we can set DSCP
	de1 := l.conn.setDSCP(1)
	de0 := l.conn.setDSCP(0)
	if de1 != nil || de0 != nil {
		l.eventf(NoDSCPSupport, "no DSCP support available (%s)", de1.Error())
	} else {
		l.dscpSupport = true
	}

	// enable receipt of destination IP
	if rdsterr := l.conn.setReceiveDstAddr(true); rdsterr != nil {
		l.eventf(NoReceiveDstAddrSupport,
			"no support for determining packet destination address (%s)", rdsterr)
		if l.conn.localAddr().IP.IsUnspecified() {
			if err := l.warnOnMultipleAddresses(); err != nil {
				return err
			}
		}
	}

	err = l.readAndReply()
	if l.isClosed() {
		err = nil
	}
	return
}

func (l *listener) readAndReply() (err error) {
	for {
		p := l.pktPool.get()
		var fatal bool
		fatal, err = l.readOneAndReply(p)
		if fatal {
			return
		}
	}
}

func (l *listener) readOneAndReply(p *packet) (fatal bool, err error) {
	defer l.pktPool.put(p)

	// read a packet
	var trecv time.Time
	var dstIP net.IP
	trecv, dstIP, l.raddr, err = l.conn.receiveFrom(p)
	if err != nil {
		if e, ok := err.(*Error); ok {
			l.eventf(dropCode(e.Code), "[%s] %s", l.raddr, err.Error())
		} else {
			fatal = true
		}
		return
	}

	// handle open
	if p.flags()&flOpen != 0 {
		var params *Params
		params, err = parseParams(p.payload())
		if err != nil {
			l.eventf(DropUnparseableParams,
				"[%s] unparseable negotiation parameters: %s",
				l.raddr, err.Error())
			return
		}
		l.restrictParams(p, params)
		sc := l.cmgr.newConn(l.raddr, params, p.flags()&flClose != 0)
		if p.flags()&flClose == 0 {
			l.eventf(NewConn, "[%s] new connection, token=%016x",
				l.raddr, sc.ctoken)
			p.setConnToken(sc.ctoken)
		} else {
			l.eventf(OpenClose, "[%s] open-close", l.raddr)
			p.setConnToken(0)
		}
		p.setReply(true)
		p.setPayload(params.bytes())
		if err = l.sendPacket(p, trecv, dstIP, sc, false); err != nil {
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
			l.eventf(DropInvalidConnToken, "[%s] close for invalid conn token %016x",
				l.raddr, p.ctoken())
			return
		}
		// check remote address
		if !udpAddrsEqual(l.raddr, &sc.raddr) {
			l.eventf(DropAddressMismatch,
				"[%s] drop close due to address mismatch (expected %s for %016x)",
				l.raddr, &sc.raddr, p.ctoken())
			return
		}
		l.eventf(CloseConn, "[%s] close connection, token=%016x", l.raddr, sc.ctoken)
		return
	}

	// handle echo request
	if !l.addFields(p, fechoRequest) {
		return
	}

	// check conn, token and address
	sc, exists, addrOk, intervalOk := l.cmgr.conn(p, l.raddr)
	if !exists {
		l.eventf(DropInvalidConnToken, "[%s] request for invalid conn token %016x",
			l.raddr, p.ctoken())
		return
	}
	if !addrOk {
		l.eventf(DropAddressMismatch,
			"[%s] drop request due to address mismatch (expected %s for %016x)", l.raddr,
			&sc.raddr, p.ctoken())
		return
	}
	if !intervalOk {
		l.eventf(DropShortInterval,
			"[%s] drop request due to short interval", l.raddr)
		return
	}

	// set reply flag
	p.setReply(true)

	// check if max test duration exceeded (but still return packet)
	if l.hardMaxDuration > 0 && time.Since(sc.firstUsed) > l.hardMaxDuration {
		l.eventf(DurationLimitExceeded,
			"[%s] closing connection due to duration limit exceeded", l.raddr)
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
	if err = l.sendPacket(p, trecv, dstIP, sc, true); err != nil {
		fatal = true
	}

	return
}

// sendPacket sends a packet, locking and setting socket options as necessary.
func (l *listener) sendPacket(p *packet, trecv time.Time, srcIP net.IP,
	sc *sconn, testPacket bool) (err error) {
	// set socket options
	if l.dscpSupport {
		l.conn.setDSCP(sc.params.DSCP)
	}

	// for test packets, add stats and timestamps according to conn params
	if testPacket {
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
				mt := midpoint(trecv, time.Now())
				rt = newTime(mt, cl)
				st = newTime(mt, cl)
			} else {
				if at&AtReceive != 0 {
					rt = newTime(trecv, cl)
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

	p.updateHMAC()

	// simulate dropped packets, if necessary
	if serverDropsPercent > 0 && rand.Float32() < serverDropsPercent {
		return
	}

	// simulate duplicates, if necessary
	if serverDupsPercent > 0 {
		for rand.Float32() < serverDupsPercent {
			err = l.conn.sendTo(p, l.raddr, srcIP)
			if err != nil {
				return
			}
		}
	}

	err = l.conn.sendTo(p, l.raddr, srcIP)

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
	if !l.dscpSupport {
		p.DSCP = 0
	}
}

func (l *listener) addFields(pkt *packet, fidxs []fidx) bool {
	if err := pkt.addFields(fidxs, false); err != nil {
		if e, ok := err.(*Error); ok {
			l.eventf(dropCode(e.Code), err.Error())
		}
		return false
	}
	return true
}

func (l *listener) eventf(code EventCode, format string, args ...interface{}) {
	if l.Handler != nil && l.EventMask&code != 0 {
		l.Handler.OnEvent(Eventf(code, l.conn.localAddr(), l.raddr, format, args...))
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
	//mtx  sync.Mutex
	new func() *packet
}

func newPacketPool(new func() *packet, cap int) *pktPool {
	pp := &pktPool{
		pool: make([]*packet, 0, cap),
		new:  new,
	}
	return pp
}

func (po *pktPool) get() *packet {
	//po.mtx.Lock()
	//defer po.mtx.Unlock()
	l := len(po.pool)
	if l == 0 {
		return po.new()
	}
	p := po.pool[l-1]
	po.pool = po.pool[:l-1]
	return p
}

func (po *pktPool) put(p *packet) {
	//po.mtx.Lock()
	//defer po.mtx.Unlock()
	po.pool = append(po.pool, p)
}
