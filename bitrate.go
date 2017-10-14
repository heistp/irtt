package irtt

import (
	"encoding/json"
	"fmt"
	"time"
)

// Bitrate is a bit rate in bits per second.
type Bitrate uint64

func calculateBitrate(n uint64, d time.Duration) Bitrate {
	if n == 0 || d == 0 {
		return Bitrate(0)
	}
	return Bitrate(8 * float64(n) / d.Seconds())
}

// String returns a Bitrate string in appropriate units.
func (r Bitrate) String() string {
	// Yes, it's exhaustive, just for fun. A 64-int unsigned int can't hold
	// Yottabits per second as 1e21 overflows it. If this problem affects
	// you, thanks for solving climate change!
	if r < 1000 {
		return fmt.Sprintf("%d bps", r)
	} else if r < 1e6 {
		return fmt.Sprintf("%.1f Kbps", float64(r)/float64(1000))
	} else if r < 1e9 {
		return fmt.Sprintf("%.2f Mbps", float64(r)/float64(1e6))
	} else if r < 1e12 {
		return fmt.Sprintf("%.3f Gbps", float64(r)/float64(1e9))
	} else if r < 1e15 {
		return fmt.Sprintf("%.3f Pbps", float64(r)/float64(1e12))
	} else if r < 1e18 {
		return fmt.Sprintf("%.3f Ebps", float64(r)/float64(1e15))
	}
	return fmt.Sprintf("%.3f Zbps", float64(r)/float64(1e18))
}

// MarshalJSON implements the json.Marshaler interface.
func (r Bitrate) MarshalJSON() ([]byte, error) {
	type Alias DurationStats
	j := &struct {
		BPS    uint64 `json:"bps"`
		String string `json:"string"`
	}{
		BPS:    uint64(r),
		String: r.String(),
	}
	return json.Marshal(j)
}
