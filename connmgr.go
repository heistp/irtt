package irtt

import (
	"crypto/rand"
	"encoding/binary"
	"net"
	"time"
)

// time after which conns expire and may be removed
const expirationTime = 1 * time.Minute

// number of conns to check to remove on each add (two seems to be the least
// aggresive number where the map size still levels off over time)
const checkExpiredCount = 5

// allocate space for this number of concurrent conns, initially
const connmgrInitSize = 128

type sconn struct {
	ctoken         ctoken
	raddr          net.UDPAddr
	params         *Params
	created        time.Time
	firstUsed      time.Time
	lastUsed       time.Time
	packetBucket   float64
	lastSeqno      Seqno
	receivedCount  ReceivedCount
	receivedWindow ReceivedWindow
	rwinValid      bool
	bytes          uint64
}

func (sc *sconn) expired() bool {
	return !sc.lastUsed.IsZero() && time.Since(sc.lastUsed) > expirationTime
}

type connmgr struct {
	conns       map[ctoken]*sconn
	packetBurst float64
	minInterval time.Duration
}

func newConnMgr(packetBurst int, minInterval time.Duration) *connmgr {
	return &connmgr{
		conns:       make(map[ctoken]*sconn, connmgrInitSize),
		packetBurst: float64(packetBurst),
		minInterval: minInterval,
	}
}

func (cm *connmgr) newConn(raddr *net.UDPAddr, p *Params, temporary bool) *sconn {
	cm.removeSomeExpired()
	ct := cm.newCtoken()
	sc := &sconn{
		ctoken:       ct,
		raddr:        *raddr,
		params:       p,
		created:      time.Now(),
		lastSeqno:    InvalidSeqno,
		packetBucket: float64(cm.packetBurst),
	}
	if !temporary {
		cm.conns[ct] = sc
	}
	return sc
}

func (cm *connmgr) conn(p *packet, raddr *net.UDPAddr) (sconn *sconn,
	exists bool, addrOk bool, intervalOk bool) {
	ct := p.ctoken()
	sc := cm.conns[ct]
	if sc == nil {
		return
	}
	exists = true
	if sc.expired() {
		delete(cm.conns, ct)
		return
	}
	if !udpAddrsEqual(raddr, &sc.raddr) {
		return
	}
	addrOk = true
	now := time.Now()
	if sc.firstUsed.IsZero() {
		sc.firstUsed = now
	}
	if cm.minInterval > 0 {
		if !sc.lastUsed.IsZero() {
			earned := float64(now.Sub(sc.lastUsed)) / float64(cm.minInterval)
			sc.packetBucket += earned
			if sc.packetBucket > float64(cm.packetBurst) {
				sc.packetBucket = float64(cm.packetBurst)
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
	if sc, ok = cm.conns[ct]; ok {
		delete(cm.conns, ct)
	}
	return
}

// removeSomeExpired checks checkExpiredCount conns for expiration and removes
// them if expired. Yes, I know, I'm depending on Go's random map iteration,
// which per the language spec, I should not depend on. That said, this makes
// for a highly CPU efficient way to eventually clean up expired conns, and
// because the Go team very intentionally made map order traversal random for a
// good reason, I don't think that's going to change any time soon.
func (cm *connmgr) removeSomeExpired() {
	for i := 0; i < checkExpiredCount; i++ {
		for ct, sc := range cm.conns {
			if sc.expired() {
				delete(cm.conns, ct)
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
		if _, ok := cm.conns[ct]; !ok {
			break
		}
	}
	return ct
}
