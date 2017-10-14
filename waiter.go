package irtt

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Waiter is implemented to return a wait time for final replies. See the
// documentation for Recorder for information on locking for concurrent access.
type Waiter interface {
	// Wait returns the wait duration.
	Wait(r *Recorder) time.Duration

	String() string
}

// WaitDuration waits for a specific period of time.
type WaitDuration struct {
	D time.Duration `json:"d"`
}

// Wait returns the wait duration.
func (w *WaitDuration) Wait(r *Recorder) time.Duration {
	return w.D
}

func (w *WaitDuration) String() string {
	return w.D.String()
}

// WaitMaxRTT waits for a factor of the maximum RTT
type WaitMaxRTT struct {
	D      time.Duration `json:"d"`
	Factor int           `json:"factor"`
}

// Wait returns the wait duration.
func (w *WaitMaxRTT) Wait(r *Recorder) time.Duration {
	r.RLock()
	defer r.RUnlock()
	if r.RTTStats.N == 0 {
		return w.D
	}
	return time.Duration(w.Factor) * r.RTTStats.Max
}

func (w *WaitMaxRTT) String() string {
	return fmt.Sprintf("%dx%s", w.Factor, w.D)
}

// WaitMeanRTT waits for a factor of the mean RTT.
type WaitMeanRTT struct {
	D      time.Duration `json:"d"`
	Factor int           `json:"factor"`
}

// Wait returns the wait duration.
func (w *WaitMeanRTT) Wait(r *Recorder) time.Duration {
	r.RLock()
	defer r.RUnlock()
	if r.RTTStats.N == 0 {
		return w.D
	}
	return time.Duration(w.Factor) * r.RTTStats.Mean()
}

func (w *WaitMeanRTT) String() string {
	return fmt.Sprintf("%dr%s", w.Factor, w.D)
}

// WaiterFactories are the registered Waiter factories.
var WaiterFactories = make([]WaiterFactory, 0)

// WaiterFactory can create a Waiter from a string.
type WaiterFactory struct {
	FactoryFunc func(string) (Waiter, error)
	Usage       string
}

// RegisterWaiter registers a new Waiter.
func RegisterWaiter(fn func(string) (Waiter, error), usage string) {
	WaiterFactories = append(WaiterFactories, WaiterFactory{fn, usage})
}

// NewWaiter returns a Waiter from a string.
func NewWaiter(s string) (Waiter, error) {
	for _, fac := range WaiterFactories {
		t, err := fac.FactoryFunc(s)
		if err != nil {
			return nil, err
		}
		if t != nil {
			return t, nil
		}
	}
	return nil, Errorf(NoSuchWaiter, "no such Waiter %s", s)
}

func init() {
	RegisterWaiter(
		func(s string) (t Waiter, err error) {
			i := strings.Index(s, "x")
			if i != -1 {
				f, d, err := parseWait(s[:i], s[i+1:])
				if err != nil {
					return nil, Errorf(InvalidWaitString, "invalid wait %s (%s)", s, err)
				}
				return &WaitMaxRTT{D: d, Factor: f}, nil
			}
			return nil, nil
		},
		"#xduration: # times max RTT, or duration if no response",
	)

	RegisterWaiter(
		func(s string) (t Waiter, err error) {
			i := strings.Index(s, "r")
			if i != -1 {
				f, d, err := parseWait(s[:i], s[i+1:])
				if err != nil {
					return nil, Errorf(InvalidWaitString, "invalid wait %s (%s)", s, err)
				}
				return &WaitMeanRTT{D: d, Factor: f}, nil
			}
			return nil, nil
		},
		"#rduration: # times RTT, or duration if no response",
	)

	RegisterWaiter(
		func(s string) (Waiter, error) {
			if d, err := time.ParseDuration(s); err == nil {
				return &WaitDuration{D: d}, nil
			}
			return nil, nil
		},
		"duration: fixed duration (see Duration units below)",
	)
}

func parseWait(fstr string, dstr string) (factor int, dur time.Duration, err error) {
	factor, err = strconv.Atoi(fstr)
	if err != nil {
		err = Errorf(InvalidWaitFactor, "unparseable factor %s", fstr)
		return
	}
	dur, err = time.ParseDuration(dstr)
	if err != nil {
		err = Errorf(InvalidWaitDuration, "not a duration %s", dstr)
		return
	}
	return
}
