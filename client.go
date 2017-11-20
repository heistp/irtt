package irtt

import (
	"context"
	"math/rand"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// ignore server restrictions (for testing hard limits)
const ignoreServerRestrictions = true

// settings for testing
const clientDropsPercent = 0

// Client is the Client. It must be created with NewClient. It may not be used
// concurrently.
type Client struct {
	*Config
	conn    *cconn
	rec     *Recorder
	closed  bool
	closedM sync.Mutex
}

// NewClient returns a new client.
func NewClient(cfg *Config) *Client {
	// create client
	c := *cfg
	c.Supplied = cfg
	return &Client{
		Config: &c,
	}
}

// Run runs the test and returns the Result. An error is returned if the test
// could not be started. If an error occurs during the test, the error is nil,
// partial results are returned and either or both of the SendErr or
// ReceiveErr fields of Result will be non-nil. Run may only be called once.
func (c *Client) Run(ctx context.Context) (r *Result, err error) {
	// validate config
	if err = c.validate(); err != nil {
		return
	}

	// notify about connecting
	c.eventf(Connecting, "connecting to %s", c.RemoteAddress)

	// check for closed error helper
	isClosedError := func(e error) (closedError bool) {
		if ee, ok := e.(*Error); ok {
			closedError = (ee.Code == ServerClosed)
		}
		return
	}

	// dial server
	c.conn, err = dial(ctx, c.Config)
	if err != nil && (!c.Config.NoTest || !isClosedError(err)) {
		return
	}
	defer c.close()

	// notify about connected
	c.eventf(Connected, "connected to %s", c.remoteAddr())

	// check parameter changes
	var changed bool
	if changed, err = c.checkParameters(); err != nil {
		return
	}
	if changed && c.StrictParams {
		err = Errorf(ParamsChanged, "server restricted test parameters")
		return
	}

	// return if NoTest is set
	if c.Config.NoTest {
		err = nil
		c.eventf(NoTest, "skipping test at user request")
		return
	}

	// ignore server restrictions for testing
	if ignoreServerRestrictions {
		c.Params = c.Supplied.Params
	}

	// set socket options
	if err = c.setSockOpts(); err != nil {
		return
	}

	// create recorder
	if c.rec, err = newRecorder(pcount(c.Duration, c.Interval), c.Handler); err != nil {
		return
	}

	// wait group for goroutine completion
	wg := sync.WaitGroup{}

	// collect before test
	runtime.GC()

	// disable GC
	debug.SetGCPercent(-1)

	// start receive
	var rerr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer c.close()
		rerr = c.receive()
		if rerr != nil && c.isClosed() {
			rerr = nil
		}
	}()

	// start send
	var serr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer c.close()
		serr = c.send(ctx)
		if serr == nil {
			err = c.wait(ctx)
		}
		if serr != nil && c.isClosed() {
			serr = nil
		}
	}()

	// wait for send and receive to complete
	wg.Wait()

	// re-enable GC
	debug.SetGCPercent(100)

	r = newResult(c.rec, c.Config, serr, rerr)
	return
}

func (c *Client) close() {
	c.closedM.Lock()
	defer c.closedM.Unlock()
	if !c.closed {
		c.conn.close()
		c.closed = true
	}
}

func (c *Client) isClosed() bool {
	c.closedM.Lock()
	defer c.closedM.Unlock()
	return c.closed
}

// localAddr returns the local address (non-nil after server dialed).
func (c *Client) localAddr() *net.UDPAddr {
	if c.conn == nil {
		return nil
	}
	return c.conn.localAddr()
}

// remoteAddr returns the remote address (non-nil after server dialed).
func (c *Client) remoteAddr() *net.UDPAddr {
	if c.conn == nil {
		return nil
	}
	return c.conn.remoteAddr()
}

// setSockOpts sets socket options
func (c *Client) setSockOpts() error {
	// set DSCP value on socket
	if c.DSCP != DefaultDSCP {
		if err := c.conn.setDSCP(c.DSCP); err != nil {
			return Errorf(DSCPError, "unable to set dscp value %d (%s)", c.DSCP, err)
		}
	}

	// set DF value on socket
	if c.DF != DefaultDF {
		if err := c.conn.setDF(c.DF); err != nil {
			return Errorf(DFError, "unable to set do not fragment bit (%s)", err)
		}
	}

	// set TTL
	if c.TTL != DefaultTTL {
		if err := c.conn.setTTL(c.TTL); err != nil {
			return Errorf(TTLError, "unable to set TTL %d (%s)", c.TTL, err)
		}
	}

	return nil
}

// checkParameters checks any changes after the server returned restricted
// parameters.
func (c *Client) checkParameters() (changed bool, err error) {
	if c.Duration < c.Supplied.Duration {
		changed = true
		c.eventf(ServerRestriction, "server restricted duration from %s to %s",
			c.Supplied.Duration, c.Duration)
	}
	if c.Duration > c.Supplied.Duration {
		changed = true
		err = Errorf(InvalidServerRestriction,
			"server tried to change duration from %s to %s",
			c.Supplied.Duration, c.Duration)
		return
	}
	if c.Interval > c.Supplied.Interval {
		changed = true
		c.eventf(ServerRestriction, "server restricted interval from %s to %s",
			c.Supplied.Interval, c.Interval)
	}
	if c.Interval < c.Supplied.Interval {
		changed = true
		err = Errorf(InvalidServerRestriction,
			"server tried to change interval from %s to %s",
			c.Supplied.Interval, c.Interval)
		return
	}
	if c.Length < c.Supplied.Length {
		changed = true
		c.eventf(ServerRestriction, "server restricted length from %d to %d",
			c.Supplied.Length, c.Length)
	}
	if c.Length > c.Supplied.Length {
		changed = true
		err = Errorf(InvalidServerRestriction,
			"server tried to change length from %d to %d",
			c.Supplied.Length, c.Length)
		return
	}
	if c.StampAt != c.Supplied.StampAt {
		changed = true
		c.eventf(ServerRestriction, "server restricted timestamps from %s to %s",
			c.Supplied.StampAt, c.StampAt)
	}
	if c.Clock != c.Supplied.Clock {
		changed = true
		c.eventf(ServerRestriction, "server restricted clocks from %s to %s",
			c.Supplied.Clock, c.Clock)
	}
	if c.DSCP != c.Supplied.DSCP {
		changed = true
		c.eventf(ServerRestriction,
			"server doesn't support DSCP, falling back to best effort")
	}
	return
}

// send sends all packets for the test to the server (called in goroutine from Run)
func (c *Client) send(ctx context.Context) error {
	if c.ThreadLock {
		runtime.LockOSThread()
	}

	// include 0 timestamp in appropriate fields
	seqno := Seqno(0)
	p := c.conn.spkt
	p.addFields(fechoRequest, true)
	p.zeroReceivedStats(c.ReceivedStats)
	p.stampZeroes(c.StampAt, c.Clock)
	p.setSeqno(seqno)
	c.Length = p.setLen(c.Length)

	// fill the first packet, if necessary
	if c.Filler != nil {
		err := p.readPayload(c.Filler)
		if err != nil {
			return err
		}
	} else {
		p.zeroPayload()
	}

	// lastly, set the HMAC
	p.updateHMAC()

	// record the start time of the test and calculate the end
	t := time.Now()
	c.rec.Start = t
	end := c.rec.Start.Add(c.Duration)

	// keep sending until the duration has passed
	for {
		// send to network and record times right before and after
		tsend := c.rec.recordPreSend()
		var err error
		var tsent time.Time
		if clientDropsPercent == 0 || rand.Float32() > clientDropsPercent {
			tsent, err = c.conn.send()
		} else {
			time.Sleep(20 * time.Microsecond)
			tsent, err = time.Now(), nil
		}

		// return on error
		if err != nil {
			c.rec.removeLastStamps()
			return err
		}

		// record send call
		c.rec.recordPostSend(tsend, tsent, uint64(p.length()))

		// prepare next packet (before sleep, so the next send time is as
		// precise as possible)
		seqno++
		p.setSeqno(seqno)
		if c.Filler != nil && c.FillAll {
			err := p.readPayload(c.Filler)
			if err != nil {
				return err
			}
		}
		p.updateHMAC()

		// set the current base interval we're at
		tnext := c.rec.Start.Add(
			c.Interval * (time.Now().Sub(c.rec.Start) / c.Interval))

		// if we're under half-way to the next interval, sleep until the next
		// interval, but if we're over half-way, sleep until the interval after
		// that
		if tsent.Sub(c.rec.Start)%c.Interval < c.Interval/2 {
			tnext = tnext.Add(c.Interval)
		} else {
			tnext = tnext.Add(2 * c.Interval)
		}

		// break if tnext if after the end of the test
		if !tnext.Before(end) {
			break
		}

		// calculate sleep duration
		tsleep := time.Now()
		dsleep := tnext.Sub(tsleep)

		// sleep
		t, err = c.Timer.Sleep(ctx, tsleep, dsleep)
		if err != nil {
			return err
		}

		// record timer error
		c.rec.recordTimerErr(t.Sub(tsleep) - dsleep)
	}

	return nil
}

// receive receives packets from the server (called in goroutine from Run)
func (c *Client) receive() error {
	if c.ThreadLock {
		runtime.LockOSThread()
	}

	p := c.conn.rpkt

	for {
		// read a packet
		trecv, err := c.conn.receive()
		if err != nil {
			return err
		}

		// drop packets with open flag set
		if p.flags()&flOpen != 0 {
			c.eventf(DropUnexpectedOpenFlag,
				"receiver dropped packet with unexpected open flag set")
			continue
		}

		// add expected echo reply fields
		p.addFields(fechoReply, false)

		// return an error if reply packet was too small
		if p.length() < c.Length {
			return Errorf(ShortReply,
				"sent %d byte request but received %d byte reply",
				c.conn.spkt.length(), p.length())
		}

		// add expected received stats fields
		p.addReceivedStatsFields(c.ReceivedStats)

		// add expected timestamp fields
		p.addTimestampFields(c.StampAt, c.Clock)

		// get timestamps and return an error if the timestamp setting is
		// different (server doesn't support timestamps)
		at := p.stampAt()
		if at != c.StampAt {
			return Errorf(StampAtMismatch, "server stamped at %s, but %s was requested",
				at, c.StampAt)
		}
		if at != AtNone {
			cl := p.clock()
			if cl != c.Clock {
				return Errorf(ClockMismatch, "server clock %s, but %s was requested", cl, c.Clock)
			}
		}
		sts := p.timestamp()

		// record receive if all went well (may fail if seqno not found)
		ok := c.rec.recordReceive(p, trecv, &sts)
		if !ok {
			return Errorf(UnexpectedSequenceNumber, "unexpected reply sequence number %d", p.seqno())
		}
	}
}

// wait waits for final packets
func (c *Client) wait(ctx context.Context) (err error) {
	// return if all packets have been received
	c.rec.RLock()
	if c.rec.RTTStats.N >= c.rec.SendCallStats.N {
		c.rec.RUnlock()
		return
	}
	c.rec.RUnlock()

	// wait
	dwait := c.Waiter.Wait(c.rec)
	if dwait > 0 {
		c.rec.Wait = dwait
		c.eventf(WaitForPackets, "waiting %s for final packets", rdur(dwait))
		select {
		case <-time.After(dwait):
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	return
}

func (c *Client) eventf(code EventCode, format string, args ...interface{}) {
	if c.Handler != nil && c.EventMask&code != 0 {
		c.Handler.OnEvent(Eventf(code, c.localAddr(), c.remoteAddr(), format, args...))
	}
}

// ClientHandler is called with client events, as well as separately when
// packets are sent and received. See the documentation for Recorder for
// information on locking for concurrent access.
type ClientHandler interface {
	Handler

	RecorderHandler
}
