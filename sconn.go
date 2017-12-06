package irtt

import (
	"net"
	"time"
)

// time after which sconns expire and may be removed
const expirationTime = 1 * time.Minute

// sconn stores the state for a client's connection to the server
type sconn struct {
	*listener
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

func accept(l *listener, p *packet) (err error) {
	// create sconn
	sc := &sconn{
		listener:     l,
		raddr:        *p.raddr,
		created:      time.Now(),
		lastSeqno:    InvalidSeqno,
		packetBucket: float64(l.PacketBurst),
	}

	// parse, restrict and set params
	var params *Params
	params, err = parseParams(p.payload())
	if err != nil {
		return
	}
	l.restrictParams(p, params)
	sc.params = params

	// put in connmgr if close flag not set (assigns ctoken)
	if p.flags()&flClose == 0 {
		l.cmgr.put(sc)
		l.eventf(NewConn, p.raddr, "new connection, token=%016x",
			sc.ctoken)
	} else {
		l.eventf(OpenClose, p.raddr, "open-close connection")
	}

	// prepare and send packet
	p.setConnToken(sc.ctoken)
	p.setReply(true)
	p.setPayload(params.bytes())
	err = l.conn.send(p)
	return
}

func (sc *sconn) expired() bool {
	return !sc.lastUsed.IsZero() && time.Since(sc.lastUsed) > expirationTime
}
