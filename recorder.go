package irtt

import (
	"encoding/json"
	"math"
	"sync"
	"time"
)

// maxStampsInitCap is the maximum initial capacity of the Timestamps slice.
// The size of Stamps is 64 bytes, so this should be up to 4 megs in RAM.
// This allows around 65 seconds of recording at interval 1ms.
//const maxStampsInitCap = 64 * 1024
const maxStampsInitCap = 0

// Recorder is used to record data during the test. It is available to the
// Handler during the test for display of basic statistics, and may be used
// later to create a Result for further statistical analysis and storage.
// Recorder is accessed concurrently while the test is running, so its RLock and
// RUnlock methods must be used during read access to prevent race conditions. It
// is not possible to lock Recorder externally for write, since all recording
// should be done internally.
type Recorder struct {
	Start             time.Time     `json:"start_time"`
	FirstSend         time.Time     `json:"-"`
	LastSent          time.Time     `json:"-"`
	FirstReceived     time.Time     `json:"-"`
	LastReceived      time.Time     `json:"-"`
	SendCallStats     DurationStats `json:"send_call"`
	TimerErrorStats   DurationStats `json:"timer_error"`
	RTTStats          DurationStats `json:"rtt"`
	SendDelayStats    DurationStats `json:"send_delay"`
	ReceiveDelayStats DurationStats `json:"receive_delay"`
	BytesSent         uint64        `json:"bytes_sent"`
	BytesReceived     uint64        `json:"bytes_received"`
	Duplicates        uint          `json:"duplicates"`
	LatePackets       uint          `json:"late_packets"`
	Wait              time.Duration `json:"wait"`
	Timestamps        []Timestamps  `json:"-"`
	lastSeqno         Seqno
	mtx               sync.RWMutex
}

// RLock locks the Recorder for reading.
func (r *Recorder) RLock() {
	r.mtx.RLock()
}

// RUnlock unlocks the Recorder for reading.
func (r *Recorder) RUnlock() {
	r.mtx.RUnlock()
}

func newRecorder(dur time.Duration, interval time.Duration) *Recorder {
	pcap := pcount(dur, interval)
	if maxStampsInitCap > 0 && pcap > maxStampsInitCap {
		pcap = maxStampsInitCap
	}
	return &Recorder{
		Timestamps: make([]Timestamps, 0, pcap),
	}
}

func (r *Recorder) recordPreSend() time.Time {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	// add round trip before timestamp, so any re-allocation happens before the
	// time is set
	r.Timestamps = append(r.Timestamps, Timestamps{})
	tsend := time.Now()
	r.Timestamps[len(r.Timestamps)-1].Client.Send.set(tsend)
	return tsend
}

func (r *Recorder) removeLastStamps() {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.Timestamps = r.Timestamps[:len(r.Timestamps)-1]
}

func (r *Recorder) recordPostSend(tsend time.Time, tsent time.Time, n uint64) Timestamps {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	// add send call duration
	r.SendCallStats.push(tsent.Sub(tsend))

	// update bytes sent
	r.BytesSent += n

	// update send and sent times
	if r.FirstSend.IsZero() {
		r.FirstSend = tsend
	}
	r.LastSent = tsent
	return r.Timestamps[len(r.Timestamps)-1]
}

func (r *Recorder) recordTimerErr(terr time.Duration) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.TimerErrorStats.push(AbsDuration(terr))
}

func (r *Recorder) recordReceive(seqno Seqno, trecv time.Time, sts *Timestamp,
	n uint64) (Timestamps, bool, bool) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	// check for invalid sequence number
	if int(seqno) >= len(r.Timestamps) {
		return Timestamps{}, false, false
	}

	tt := &r.Timestamps[seqno]

	// check for duplicate (don't update stats for duplicates)
	if !tt.Client.Receive.IsZero() {
		r.Duplicates++
		return *tt, true, true
	}

	// check for late packet
	if seqno < r.lastSeqno {
		r.LatePackets++
	}
	r.lastSeqno = seqno

	// update client received times
	tt.Client.Receive.set(trecv)

	// update RTT and RTT stats
	tt.Server = *sts
	r.RTTStats.push(tt.RTT())

	// update one-way delay stats
	if !tt.Server.BestReceive().IsWallZero() {
		r.SendDelayStats.push(tt.SendDelay())
	}
	if !tt.Server.BestSend().IsWallZero() {
		r.ReceiveDelayStats.push(tt.ReceiveDelay())
	}

	// set received times
	if r.FirstReceived.IsZero() {
		r.FirstReceived = trecv
	}
	r.LastReceived = trecv

	// update bytes received
	r.BytesReceived += n

	return *tt, false, true
}

// Timestamps contains Timestamps for both the client and server.
type Timestamps struct {
	Client Timestamp `json:"client"`
	Server Timestamp `json:"server"`
}

// ReplyReceived returns true if a reply was received from the server.
func (ts *Timestamps) ReplyReceived() bool {
	return !ts.Client.Receive.IsZero()
}

// RTT returns the round-trip time. The monotonic clock values are used
// for accuracy, and the server processing time is subtracted out if
// both send and receive timestamps are enabled.
func (ts *Timestamps) RTT() (rtt time.Duration) {
	if !ts.ReplyReceived() {
		return InvalidDuration
	}
	rtt = ts.Client.Receive.Mono - ts.Client.Send.Mono
	spt := ts.ServerProcessingTime()
	if spt != InvalidDuration {
		rtt -= ts.ServerProcessingTime()
	}
	return
}

// IPDVSince returns the instantaneous packet delay variation since the
// specified Stamps.
func (ts *Timestamps) IPDVSince(pts *Timestamps) time.Duration {
	if !ts.ReplyReceived() || !pts.ReplyReceived() {
		return InvalidDuration
	}
	return ts.RTT() - pts.RTT()
}

// SendIPDVSince returns the send instantaneous packet delay variation since the
// specified Stamps.
func (ts *Timestamps) SendIPDVSince(pts *Timestamps) (d time.Duration) {
	d = InvalidDuration
	if ts.IsTimestamped() && pts.IsTimestamped() {
		if ts.IsMonoTimestamped() && pts.IsMonoTimestamped() {
			d = ts.SendMonoDiff() - pts.SendMonoDiff()
		} else if ts.IsWallTimestamped() && pts.IsWallTimestamped() {
			d = ts.SendWallDiff() - pts.SendWallDiff()
		}
	}
	return
}

// ReceiveIPDVSince returns the receive instantaneous packet delay variation
// since the specified Stamps.
func (ts *Timestamps) ReceiveIPDVSince(pts *Timestamps) (d time.Duration) {
	d = InvalidDuration
	if ts.IsTimestamped() && pts.IsTimestamped() {
		if ts.IsMonoTimestamped() && pts.IsMonoTimestamped() {
			d = ts.ReceiveMonoDiff() - pts.ReceiveMonoDiff()
		} else if ts.IsWallTimestamped() && pts.IsWallTimestamped() {
			d = ts.ReceiveWallDiff() - pts.ReceiveWallDiff()
		}
	}
	return
}

// SendDelay returns the estimated one-way send delay, valid only if wall clock timestamps
// are available and the server's system time has been externally synchronized.
func (ts *Timestamps) SendDelay() time.Duration {
	if !ts.IsWallTimestamped() {
		return InvalidDuration
	}
	return time.Duration(ts.Server.BestReceive().Wall - ts.Client.Send.Wall)
}

// ReceiveDelay returns the estimated one-way receive delay, valid only if wall
// clock timestamps are available and the server's system time has been
// externally synchronized.
func (ts *Timestamps) ReceiveDelay() time.Duration {
	if !ts.IsWallTimestamped() {
		return InvalidDuration
	}
	return time.Duration(ts.Client.Receive.Wall - ts.Server.BestSend().Wall)
}

// SendMonoDiff returns the difference in send values from the monotonic clock.
// This is useful for measuring send IPDV (jitter), but not for absolute send delay.
func (ts *Timestamps) SendMonoDiff() time.Duration {
	return ts.Server.BestReceive().Mono - ts.Client.Send.Mono
}

// ReceiveMonoDiff returns the difference in receive values from the monotonic
// clock. This is useful for measuring receive IPDV (jitter), but not for
// absolute receive delay.
func (ts *Timestamps) ReceiveMonoDiff() time.Duration {
	return ts.Client.Receive.Mono - ts.Server.BestSend().Mono
}

// SendWallDiff returns the difference in send values from the wall
// clock. This is useful for measuring receive IPDV (jitter), but not for
// absolute send delay. Because the wall clock is used, it is subject to wall
// clock variability.
func (ts *Timestamps) SendWallDiff() time.Duration {
	return time.Duration(ts.Server.BestReceive().Wall - ts.Client.Send.Wall)
}

// ReceiveWallDiff returns the difference in receive values from the wall
// clock. This is useful for measuring receive IPDV (jitter), but not for
// absolute receive delay. Because the wall clock is used, it is subject to wall
// clock variability.
func (ts *Timestamps) ReceiveWallDiff() time.Duration {
	return time.Duration(ts.Client.Receive.Wall - ts.Server.BestSend().Wall)
}

// IsTimestamped returns true if the server returned any timestamp.
func (ts *Timestamps) IsTimestamped() bool {
	return ts.IsReceiveTimestamped() || ts.IsSendTimestamped()
}

// IsMonoTimestamped returns true if the server returned any timestamp with a
// valid monotonic clock value.
func (ts *Timestamps) IsMonoTimestamped() bool {
	return !ts.Server.Receive.IsMonoZero() || !ts.Server.Send.IsMonoZero()
}

// IsWallTimestamped returns true if the server returned any timestamp with a
// valid wall clock value.
func (ts *Timestamps) IsWallTimestamped() bool {
	return !ts.Server.Receive.IsWallZero() || !ts.Server.Send.IsWallZero()
}

// IsReceiveTimestamped returns true if the server returned a receive timestamp.
func (ts *Timestamps) IsReceiveTimestamped() bool {
	return !ts.Server.Receive.IsZero()
}

// IsSendTimestamped returns true if the server returned a send timestamp.
func (ts *Timestamps) IsSendTimestamped() bool {
	return !ts.Server.Send.IsZero()
}

// IsBothTimestamped returns true if the server returned both a send and receive
// timestamp.
func (ts *Timestamps) IsBothTimestamped() bool {
	return ts.IsReceiveTimestamped() && ts.IsSendTimestamped()
}

// ServerProcessingTime returns the amount of time between when the server
// received a request and when it sent its reply.
func (ts *Timestamps) ServerProcessingTime() (d time.Duration) {
	d = InvalidDuration
	if ts.Server.IsMidpoint() {
		return
	}
	if ts.Server.IsBothMono() {
		d = time.Duration(ts.Server.Send.Mono - ts.Server.Receive.Mono)
	} else if ts.Server.IsBothWall() {
		d = time.Duration(ts.Server.Send.Wall - ts.Server.Receive.Wall)
	}
	return
}

// DurationStats keeps basic time.Duration statistics. Welford's method is used
// to keep a running mean and standard deviation. In testing, this seemed to be
// worth the extra muls and divs necessary to maintain these stats. Worst case,
// there was a 2% reduction in the send rate on a Raspberry Pi 2 when sending
// the smallest packets possible at the smallest interval possible. This is not
// a typical test, however, and the argument is, it's worth paying this price to
// add standard deviation and variance for timer error and send call time, and
// running standard deviation for all packet times.
type DurationStats struct {
	Total    time.Duration `json:"total"`
	N        uint          `json:"n"`
	Min      time.Duration `json:"min"`
	Max      time.Duration `json:"max"`
	m        float64
	s        float64
	mean     float64
	median   float64
	medianOk bool
}

func (s *DurationStats) push(d time.Duration) {
	if s.N == 0 {
		s.Min = d
		s.Max = d
		s.Total = d
	} else {
		if d < s.Min {
			s.Min = d
		}
		if d > s.Max {
			s.Max = d
		}
		s.Total += d
	}
	s.N++
	om := s.mean
	fd := float64(d)
	s.mean += (fd - om) / float64(s.N)
	s.s += (fd - om) * (fd - s.mean)
}

// IsZero returns true if DurationStats has no recorded values.
func (s *DurationStats) IsZero() bool {
	return s.N == 0
}

// Mean returns the arithmetical mean.
func (s *DurationStats) Mean() time.Duration {
	return time.Duration(s.mean)
}

// Variance returns the variance.
func (s *DurationStats) Variance() float64 {
	if s.N > 1 {
		return s.s / float64(s.N-1)
	}
	return 0.0
}

// Stddev returns the standard deviation.
func (s *DurationStats) Stddev() time.Duration {
	return time.Duration(math.Sqrt(s.Variance()))
}

// Median returns the median (externally calculated).
func (s *DurationStats) Median() (dur time.Duration, ok bool) {
	ok = s.medianOk
	dur = time.Duration(s.median)
	return
}

func (s *DurationStats) setMedian(m float64) {
	s.median = m
	s.medianOk = true
}

// MarshalJSON implements the json.Marshaler interface.
func (s *DurationStats) MarshalJSON() ([]byte, error) {
	type Alias DurationStats
	j := &struct {
		*Alias
		Mean     time.Duration `json:"mean"`
		Median   time.Duration `json:"median,omitempty"`
		Stddev   time.Duration `json:"stddev"`
		Variance time.Duration `json:"variance"`
	}{
		Alias:    (*Alias)(s),
		Mean:     s.Mean(),
		Stddev:   s.Stddev(),
		Variance: time.Duration(s.Variance()),
	}
	if m, ok := s.Median(); ok {
		j.Median = m
	}
	return json.Marshal(j)
}

// AbsDuration returns the absolute value of a duration.
func AbsDuration(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return time.Duration(-d)
}

// pcount returns the number of packets that should be sent for a given
// duration and interval.
func pcount(d time.Duration, i time.Duration) uint {
	return 1 + uint(d/i)
}
