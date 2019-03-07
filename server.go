package irtt

import (
	"encoding/binary"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"time"
)

// Server is the irtt server.
type Server struct {
	*ServerConfig
	start       time.Time
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
	lconns, err := listenAll(s.IPVersion, s.Addrs, s.SetSrcIP, s.TimeSource)
	if err != nil {
		return nil, err
	}
	ls := make([]*listener, 0, len(lconns))
	for _, lconn := range lconns {
		ls = append(ls, newListener(s.ServerConfig, lconn))
	}
	return ls, nil
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

func newListener(cfg *ServerConfig, lc *lconn) *listener {
	cap, _ := detectMTU(lc.localAddr().IP)

	pp := newPacketPool(func() *packet {
		return newPacket(0, cap, cfg.HMACKey)
	}, 16)

	return &listener{
		ServerConfig: cfg,
		conn:         lc,
		pktPool:      pp,
		cmgr:         newConnMgr(cfg),
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

	err = l.readAndReply()
	if l.isClosed() {
		err = nil
	}
	return
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

	// handle open
	if p.flags()&flOpen != 0 {
		_, err = accept(l, p)
		return
	}

	// handle packet for sconn
	if err = p.addFields(fRequest, false); err != nil {
		return
	}
	ct := p.ctoken()
	sc := l.cmgr.get(ct)
	if sc == nil {
		err = Errorf(InvalidConnToken, "invalid conn token %016x", ct)
		return
	}
	_, err = sc.serve(p)
	return
}

func (l *listener) eventf(code Code, raddr *net.UDPAddr, format string,
	detail ...interface{}) {
	if l.Handler != nil {
		l.Handler.OnEvent(Eventf(code, l.conn.localAddr(), raddr, format, detail...))
	}
}

func (l *listener) isFatalError(err error) (fatal bool) {
	if nerr, ok := err.(net.Error); ok {
		fatal = !nerr.Temporary()
	}
	return
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
	sconns map[ctoken]*sconn
}

func newConnMgr(cfg *ServerConfig) *connmgr {
	return &connmgr{
		ServerConfig: cfg,
		sconns:       make(map[ctoken]*sconn, sconnsInitSize),
	}
}

func (cm *connmgr) put(sc *sconn) {
	cm.removeSomeExpired()
	ct := cm.newCtoken()
	sc.ctoken = ct
	cm.sconns[ct] = sc
}

func (cm *connmgr) get(ct ctoken) (sc *sconn) {
	if sc = cm.sconns[ct]; sc == nil {
		return
	}
	if sc.expired() {
		cm.delete(ct)
	}
	return
}

func (cm *connmgr) remove(ct ctoken) (sc *sconn) {
	var ok bool
	if sc, ok = cm.sconns[ct]; ok {
		cm.delete(ct)
	}
	return
}

// removeSomeExpired checks checkExpiredCount sconns for expiration and removes
// them if expired. Yes, I know, I'm depending on Go's random map iteration
// start point, which per the language spec, I should not depend on. That said,
// this makes for a highly CPU efficient way to eventually clean up expired
// sconns, and because the Go team very intentionally made map order traversal
// random for a good reason, I don't think that's going to change any time soon.
func (cm *connmgr) removeSomeExpired() {
	i := 0
	for ct, sc := range cm.sconns {
		if sc.expired() {
			cm.delete(ct)
		}
		if i++; i >= checkExpiredCount {
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

func (cm *connmgr) delete(ct ctoken) {
	delete(cm.sconns, ct)
}
