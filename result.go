package irtt

import (
	"encoding/json"
	"math"
	"sort"
	"time"
)

// Result is returned from Run.
type Result struct {
	SystemInfo *SystemInfo `json:"system_info"`
	Config     *Config     `json:"config"`
	SendErr    error       `json:"send_err,omitempty"`
	ReceiveErr error       `json:"receive_err,omitempty"`
	*Stats     `json:"stats"`
	RoundTrips []RoundTrip `json:"round_trips"`
}

func newResult(rec *Recorder, cfg *Config, serr error, rerr error) *Result {
	stats := &Stats{Recorder: rec}
	r := &Result{Stats: stats, Config: cfg, SystemInfo: NewSystemInfo(),
		SendErr: serr, ReceiveErr: rerr}

	// create RoundTrips array
	r.RoundTrips = make([]RoundTrip, len(rec.Timestamps))
	for i := 0; i < len(r.RoundTrips); i++ {
		rt := &r.RoundTrips[i]
		rt.Seqno = Seqno(i)
		rt.Timestamps = &r.Timestamps[i]
		rt.Lost = !rt.ReplyReceived()
		rt.IPDV = InvalidDuration
		rt.SendIPDV = InvalidDuration
		rt.ReceiveIPDV = InvalidDuration
		if i > 0 {
			rtp := &r.RoundTrips[i-1]
			if rt.ReplyReceived() && rtp.ReplyReceived() {
				rt.IPDV = rt.IPDVSince(rtp.Timestamps)
				rt.SendIPDV = rt.SendIPDVSince(rtp.Timestamps)
				rt.ReceiveIPDV = rt.ReceiveIPDVSince(rtp.Timestamps)
			}
		}
	}

	// do median calculations (could figure out a rolling median one day)
	r.visitStats(&r.RTTStats, false, func(rt *RoundTrip) time.Duration {
		return rt.RTT()
	})
	r.visitStats(&r.SendDelayStats, false, func(rt *RoundTrip) time.Duration {
		return rt.SendDelay()
	})
	r.visitStats(&r.ReceiveDelayStats, false, func(rt *RoundTrip) time.Duration {
		return rt.ReceiveDelay()
	})

	// IPDV
	r.visitStats(&r.RoundTripIPDVStats, true, func(rt *RoundTrip) time.Duration {
		return AbsDuration(rt.IPDV)
	})

	// send IPDV
	r.visitStats(&r.SendIPDVStats, true, func(rt *RoundTrip) time.Duration {
		return AbsDuration(rt.SendIPDV)
	})

	// receive IPDV
	r.visitStats(&r.ReceiveIPDVStats, true, func(rt *RoundTrip) time.Duration {
		return AbsDuration(rt.ReceiveIPDV)
	})

	// calculate server processing time, if available
	for _, rt := range rec.Timestamps {
		spt := rt.ServerProcessingTime()
		if spt != InvalidDuration {
			r.ServerProcessingTimeStats.push(spt)
		}
	}

	// set packets sent and received
	r.PacketsSent = r.SendCallStats.N
	r.PacketsReceived = r.RTTStats.N + r.Duplicates

	// calculate total duration
	r.Duration = time.Since(r.Start)

	// calculate expected packets sent based on the time between the first and
	// last send
	r.ExpectedPacketsSent = pcount(r.LastSent.Sub(r.FirstSend), r.Config.Interval)

	// calculate timer stats
	r.TimerErrPercent = 100 * float64(r.TimerErrorStats.Mean()) / float64(r.Config.Interval)

	// for some reason, occasionally one more packet is sent than expected which
	// wraps around the uint, so just punt and hard prevent this for now
	if r.ExpectedPacketsSent < r.PacketsSent {
		r.TimerMisses = 0
		r.ExpectedPacketsSent = r.PacketsSent
	} else {
		r.TimerMisses = r.ExpectedPacketsSent - r.PacketsSent
	}
	r.TimerMissPercent = 100 * float64(r.TimerMisses) / float64(r.ExpectedPacketsSent)

	// calculate send rate
	r.SendRate = calculateBitrate(r.BytesSent, r.LastSent.Sub(r.FirstSend))

	// calculate receive rate (start from time of first receipt)
	r.ReceiveRate = calculateBitrate(r.BytesReceived, r.LastReceived.Sub(r.FirstReceived))

	// calculate packet loss percent
	if r.RTTStats.N > 0 {
		r.PacketLossPercent = 100 * float64(r.SendCallStats.N-r.RTTStats.N) /
			float64(r.SendCallStats.N)
	} else {
		r.PacketLossPercent = float64(100)
	}

	// calculate duplicate percent
	if r.PacketsReceived > 0 {
		r.DuplicatePercent = 100 * float64(r.Duplicates) / float64(r.PacketsReceived)
	}

	// calculate late packets percent
	if r.PacketsReceived > 0 {
		r.LatePacketsPercent = 100 * float64(r.LatePackets) / float64(r.PacketsReceived)
	}

	return r
}

// visitStats visits each RoundTrip, optionally pushes to a DurationStats, and
// at the end, sets the median value on the DurationStats.
func (r *Result) visitStats(ds *DurationStats, push bool,
	fn func(*RoundTrip) time.Duration) {
	fs := make([]float64, 0, len(r.RoundTrips))
	for i := 0; i < len(r.RoundTrips); i++ {
		d := fn(&r.RoundTrips[i])
		if d != InvalidDuration {
			if push {
				ds.push(d)
			}
			fs = append(fs, float64(d))
		}
	}
	if len(fs) > 0 {
		ds.setMedian(median(fs))
	}
}

// RoundTrip stores the Timestamps and statistics for a single round trip.
type RoundTrip struct {
	Seqno       Seqno `json:"seqno"`
	Lost        bool  `json:"lost"`
	*Timestamps `json:"timestamps"`
	IPDV        time.Duration `json:"-"`
	SendIPDV    time.Duration `json:"-"`
	ReceiveIPDV time.Duration `json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (rt *RoundTrip) MarshalJSON() ([]byte, error) {
	type Alias RoundTrip

	delay := make(map[string]interface{})
	if rt.RTT() != InvalidDuration {
		delay["rtt"] = rt.RTT()
	}
	if rt.SendDelay() != InvalidDuration {
		delay["send_delay"] = rt.SendDelay()
	}
	if rt.ReceiveDelay() != InvalidDuration {
		delay["receive_delay"] = rt.ReceiveDelay()
	}

	ipdv := make(map[string]interface{})
	if rt.IPDV != InvalidDuration {
		ipdv["round_trip"] = rt.IPDV
	}
	if rt.SendIPDV != InvalidDuration {
		ipdv["send"] = rt.SendIPDV
	}
	if rt.ReceiveIPDV != InvalidDuration {
		ipdv["receive"] = rt.ReceiveIPDV
	}

	j := &struct {
		*Alias
		Delay map[string]interface{} `json:"delay"`
		IPDV  map[string]interface{} `json:"ipdv"`
	}{
		Alias: (*Alias)(rt),
		Delay: delay,
		IPDV:  ipdv,
	}
	return json.Marshal(j)
}

// Stats are the statistics in the Result.
type Stats struct {
	*Recorder
	Duration                  time.Duration `json:"duration"`
	ExpectedPacketsSent       uint          `json:"-"`
	PacketsSent               uint          `json:"packets_sent"`
	PacketsReceived           uint          `json:"packets_received"`
	PacketLossPercent         float64       `json:"packet_loss_percent"`
	DuplicatePercent          float64       `json:"duplicate_percent"`
	LatePacketsPercent        float64       `json:"late_packets_percent"`
	SendIPDVStats             DurationStats `json:"ipdv_send"`
	ReceiveIPDVStats          DurationStats `json:"ipdv_receive"`
	RoundTripIPDVStats        DurationStats `json:"ipdv_round_trip"`
	ServerProcessingTimeStats DurationStats `json:"server_processing_time"`
	TimerErrPercent           float64       `json:"timer_err_percent"`
	TimerMisses               uint          `json:"timer_misses"`
	TimerMissPercent          float64       `json:"timer_miss_percent"`
	SendRate                  Bitrate       `json:"send_rate"`
	ReceiveRate               Bitrate       `json:"receive_rate"`
}

// median calculates the median value of the supplied float64 slice. The array
// is sorted in place, so its original order is modified.
func median(f []float64) float64 {
	sort.Float64s(f)
	l := len(f)
	if l == 0 {
		return math.NaN()
	}
	if l%2 == 0 {
		return (float64(f[l/2-1]) + float64(f[l/2])) / 2.0
	}
	return float64(f[l/2])
}
