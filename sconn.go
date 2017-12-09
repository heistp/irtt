package irtt

import (
	"math/rand"
	"net"
	"time"
)

// settings for testing
const serverDupsPercent = 0
const serverDropsPercent = 0

// time after which sconns expire and may be removed
const expirationTime = 1 * time.Minute

// max duration grace period
const maxDurationGrace = 2 * time.Second

// sconn stores the state for a client's connection to the server
type sconn struct {
	*listener
	ctoken         ctoken
	raddr          *net.UDPAddr
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

func newSconn(l *listener, raddr *net.UDPAddr) *sconn {
	return &sconn{
		listener:     l,
		raddr:        raddr,
		created:      time.Now(),
		lastSeqno:    InvalidSeqno,
		packetBucket: float64(l.PacketBurst),
	}
}

func accept(l *listener, p *packet) (sc *sconn, err error) {
	// create sconn
	sc = newSconn(l, p.raddr)

	// parse, restrict and set params
	var params *Params
	params, err = parseParams(p.payload())
	if err != nil {
		return
	}
	sc.restrictParams(params)
	sc.params = params

	// determine state of connection
	if params.ProtoVersion != ProtoVersion {
		l.eventf(ProtocolVersionMismatch, p.raddr,
			"close connection, client version %d != server version %d",
			params.ProtoVersion, ProtoVersion)
		p.setFlagBits(flClose)
	} else if p.flags()&flClose != 0 {
		l.eventf(OpenClose, p.raddr, "open-close connection")
	} else {
		l.cmgr.put(sc)
		l.eventf(NewConn, p.raddr, "new connection, token=%016x", sc.ctoken)
	}

	// prepare and send open reply
	if sc.SetSrcIP {
		p.srcIP = p.dstIP
	}
	p.setConnToken(sc.ctoken)
	p.setReply(true)
	p.setPayload(params.bytes())
	err = l.conn.send(p)
	return
}

func (sc *sconn) serve(p *packet) (closed bool, err error) {
	if !udpAddrsEqual(p.raddr, sc.raddr) {
		err = Errorf(AddressMismatch, "address mismatch (expected %s for %016x)",
			sc.raddr, p.ctoken())
		return
	}
	if p.flags()&flClose != 0 {
		closed = true
		err = sc.serveClose(p)
		return
	}
	closed, err = sc.serveEcho(p)
	return
}

func (sc *sconn) serveClose(p *packet) (err error) {
	if err = p.addFields(fcloseRequest, false); err != nil {
		return
	}
	sc.eventf(CloseConn, p.raddr, "close connection, token=%016x", sc.ctoken)
	if scr := sc.cmgr.remove(sc.ctoken); scr == nil {
		sc.eventf(RemoveNoConn, p.raddr,
			"sconn not in connmgr, token=%016x", sc.ctoken)
	}
	return
}

func (sc *sconn) serveEcho(p *packet) (closed bool, err error) {
	// handle echo request
	if err = p.addFields(fechoRequest, false); err != nil {
		return
	}

	// check that request isn't too large
	if sc.MaxLength > 0 && p.length() > sc.MaxLength {
		err = Errorf(LargeRequest, "request too large (%d > %d)",
			p.length(), sc.MaxLength)
		return
	}

	// update first used
	now := time.Now()
	if sc.firstUsed.IsZero() {
		sc.firstUsed = now
	}

	// enforce minimum interval
	if sc.MinInterval > 0 {
		if !sc.lastUsed.IsZero() {
			earned := float64(now.Sub(sc.lastUsed)) / float64(sc.MinInterval)
			sc.packetBucket += earned
			if sc.packetBucket > float64(sc.PacketBurst) {
				sc.packetBucket = float64(sc.PacketBurst)
			}
		}
		if sc.packetBucket < 1 {
			sc.lastUsed = now
			err = Errorf(ShortInterval, "drop due to short packet interval")
			return
		}
		sc.packetBucket--
	}

	// set reply flag
	p.setReply(true)

	// update last used
	sc.lastUsed = now

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

	// check if max test duration exceeded (but still return packet)
	if sc.MaxDuration > 0 && time.Since(sc.firstUsed) >
		sc.MaxDuration+maxDurationGrace {
		sc.eventf(ExceededDuration, p.raddr,
			"closing connection due to duration limit exceeded")
		sc.cmgr.remove(sc.ctoken)
		p.setFlagBits(flClose)
		closed = true
	}

	// set packet dscp value
	if sc.AllowDSCP && sc.conn.dscpSupport {
		p.dscp = sc.params.DSCP
	}

	// set source IP, if necessary
	if sc.SetSrcIP {
		p.srcIP = p.dstIP
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

	// fill payload
	if sc.Filler != nil {
		if err = p.readPayload(sc.Filler); err != nil {
			return
		}
	}

	// simulate dropped packets, if necessary
	if serverDropsPercent > 0 && rand.Float32() < serverDropsPercent {
		return
	}

	// simulate duplicates, if necessary
	if serverDupsPercent > 0 {
		for rand.Float32() < serverDupsPercent {
			if err = sc.conn.send(p); err != nil {
				return
			}
		}
	}

	// send reply
	err = sc.conn.send(p)
	return
}

func (sc *sconn) expired() bool {
	return !sc.lastUsed.IsZero() && time.Since(sc.lastUsed) > expirationTime
}

func (sc *sconn) restrictParams(p *Params) {
	if p.ProtoVersion != ProtoVersion {
		p.ProtoVersion = ProtoVersion
	}
	if sc.MaxDuration > 0 && p.Duration > sc.MaxDuration {
		p.Duration = sc.MaxDuration
	}
	if sc.MinInterval > 0 && p.Interval < sc.MinInterval {
		p.Interval = sc.MinInterval
	}
	if sc.MaxLength > 0 && p.Length > sc.MaxLength {
		p.Length = sc.MaxLength
	}
	p.StampAt = sc.AllowStamp.Restrict(p.StampAt)
	if !sc.AllowDSCP || !sc.conn.dscpSupport {
		p.DSCP = 0
	}
	return
}
