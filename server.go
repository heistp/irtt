package irtt

import (
	"encoding/binary"
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

// number of sconns to check to remove on each add (two seems to be the least
// aggresive number where the map size still levels off over time, but I use 5
// to clean up unused sconns more quickly)
const checkExpiredCount = 5

// initial capacity for sconns map
const sconnsInitSize = 32

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

	// send ServerStop event
	if s.Handler != nil {
		s.Handler.OnEvent(Eventf(ServerStop, nil, nil,
			"stopped IRTT server"))
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
			l.eventf(ListenerError, nil, "error for listener on %s (%s)",
				l.conn.localAddr(), err)
		} else {
			l.eventf(ListenerStop, nil, "stopped listener on %s",
				l.conn.localAddr())
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

func (l *listener) readAndReply() (err error) {
	p := l.pktPool.new()
	for {
		if err = l.readOneAndReply(p); err != nil {
			if l.isFatalError(err) {
				return
			}
			l.eventf(Drop, p.raddr, "%s", err.Error())
		}
	}
}

func (l *listener) readOneAndReply(p *packet) (err error) {
	// read a packet
	if err = l.conn.receive(p); err != nil {
		return
	}

	// set source IP from received destination, if necessary
	if l.SetSrcIP {
		p.srcIP = p.dstIP
	}

	// handle open
	if p.flags()&flOpen != 0 {
		// serial: call accept to create sconn
		// concurrent: create new goroutine and send packet to its channel
		err = accept(l, p)
		return
	}

	// serial: find sconn by ctoken and call its serve method
	// concurrent: find sconn by ctoken and send packet to its channel

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
	if l.MaxLength > 0 && p.length() > l.MaxLength {
		l.eventf(DropTooLarge, p.raddr,
			"request too large (%d > %d)", p.length(), l.MaxLength)
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
			return
		}
	}

	// set packet dscp value
	if l.AllowDSCP && l.conn.dscpSupport {
		p.dscp = sc.params.DSCP
	}

	// initialize test packet
	p.setLen(0)

	// set received stats
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

	// set timestamps
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

	// set length
	p.setLen(sc.params.Length)

	// simulate dropped packets, if necessary
	if serverDropsPercent > 0 && rand.Float32() < serverDropsPercent {
		return
	}

	// simulate duplicates, if necessary
	if serverDupsPercent > 0 {
		for rand.Float32() < serverDupsPercent {
			if err = l.conn.send(p); err != nil {
				return
			}
		}
	}

	// send response
	err = l.conn.send(p)

	return
}

func (l *listener) isFatalError(err error) (fatal bool) {
	if nerr, ok := err.(net.Error); ok {
		fatal = !nerr.Temporary()
	}
	return
}

func (l *listener) restrictParams(pkt *packet, p *Params) {
	if l.MaxDuration > 0 && p.Duration > l.MaxDuration {
		p.Duration = l.MaxDuration
	}
	if l.MinInterval > 0 && p.Interval < l.MinInterval {
		p.Interval = l.MinInterval
	}
	if l.MaxLength > 0 && p.Length > l.MaxLength {
		p.Length = l.MaxLength
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

// pktPool pools packets to reduce per-packet heap allocations
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

// connmgr manages server connections
type connmgr struct {
	*ServerConfig
	ref    func(bool)
	sconns map[ctoken]*sconn
}

func newConnMgr(cfg *ServerConfig, ref func(bool)) *connmgr {
	return &connmgr{
		ServerConfig: cfg,
		ref:          ref,
		sconns:       make(map[ctoken]*sconn, sconnsInitSize),
	}
}

func (cm *connmgr) put(sc *sconn) {
	cm.removeSomeExpired()
	ct := cm.newCtoken()
	sc.ctoken = ct
	cm.sconns[ct] = sc
	cm.ref(true)
}

func (cm *connmgr) get(p *packet) (sconn *sconn,
	exists bool, addrOk bool, intervalOk bool) {
	ct := p.ctoken()
	sc := cm.sconns[ct]
	if sc == nil {
		return
	}
	exists = true
	if sc.expired() {
		delete(cm.sconns, ct)
		return
	}
	if !udpAddrsEqual(p.raddr, &sc.raddr) {
		return
	}
	addrOk = true
	now := time.Now()
	if sc.firstUsed.IsZero() {
		sc.firstUsed = now
	}
	if cm.MinInterval > 0 {
		if !sc.lastUsed.IsZero() {
			earned := float64(now.Sub(sc.lastUsed)) / float64(cm.MinInterval)
			sc.packetBucket += earned
			if sc.packetBucket > float64(cm.PacketBurst) {
				sc.packetBucket = float64(cm.PacketBurst)
			}
		}
		if sc.packetBucket >= 1 {
			sc.packetBucket--
			intervalOk = true
		}
	} else {
		intervalOk = true
	}
	// slide received seqno window
	seqno := p.seqno()
	sinceLastSeqno := seqno - sc.lastSeqno
	if sinceLastSeqno > 0 {
		sc.receivedWindow <<= sinceLastSeqno
	}
	if sinceLastSeqno >= 0 { // new, duplicate or first packet
		sc.receivedWindow |= 0x1
		sc.rwinValid = true
	} else { // late packet
		sc.receivedWindow |= (0x1 << -sinceLastSeqno)
		sc.rwinValid = false
	}
	// update received count
	sc.receivedCount++
	// update seqno and last used times
	sc.lastSeqno = seqno
	sc.lastUsed = now
	sconn = sc
	return
}

func (cm *connmgr) remove(ct ctoken) (sc *sconn) {
	var ok bool
	if sc, ok = cm.sconns[ct]; ok {
		delete(cm.sconns, ct)
		cm.ref(false)
	}
	return
}

// removeSomeExpired checks checkExpiredCount sconns for expiration and removes
// them if expired. Yes, I know, I'm depending on Go's random map iteration,
// which per the language spec, I should not depend on. That said, this makes
// for a highly CPU efficient way to eventually clean up expired sconns, and
// because the Go team very intentionally made map order traversal random for a
// good reason, I don't think that's going to change any time soon.
func (cm *connmgr) removeSomeExpired() {
	for i := 0; i < checkExpiredCount; i++ {
		for ct, sc := range cm.sconns {
			if sc.expired() {
				delete(cm.sconns, ct)
			}
			break
		}
	}
}

func (cm *connmgr) newCtoken() ctoken {
	var ct ctoken
	b := make([]byte, 8)
	for {
		rand.Read(b)
		ct = ctoken(binary.LittleEndian.Uint64(b))
		if _, ok := cm.sconns[ct]; !ok {
			break
		}
	}
	return ct
}
