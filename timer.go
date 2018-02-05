package irtt

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Timer is implemented to wait for the next send.
type Timer interface {
	// Sleep waits for at least duration d and returns the current time. The
	// current time is passed as t as a convenience for timers performing error
	// compensation. Timers should obey the Context and use a select that
	// includes ctx.Done() so that the sleep can be terminated early. In that
	// case, ctx.Err() should be returned.
	Sleep(ctx context.Context, t time.Time, d time.Duration) (time.Time, error)

	String() string
}

// SimpleTimer uses Go's default time functions. It must be created using
// NewSimpleTimer.
type SimpleTimer struct {
	timer *time.Timer
}

// NewSimpleTimer returns a new SimpleTimer.
func NewSimpleTimer() *SimpleTimer {
	t := time.NewTimer(0)
	<-t.C
	return &SimpleTimer{t}
}

// Sleep selects on both a time.Timer channel and the done channel.
func (st *SimpleTimer) Sleep(ctx context.Context, t time.Time, d time.Duration) (time.Time, error) {
	st.timer.Reset(d)
	select {
	case t := <-st.timer.C:
		return t, nil
	case <-ctx.Done():
		// stop and drain timer for cleanliness
		if !st.timer.Stop() {
			<-st.timer.C
		}
		return time.Now(), ctx.Err()
	}
}

func (st *SimpleTimer) String() string {
	return "simple"
}

// CompTimer uses Go's default time functions and performs compensation by
// continually measuring the timer error and applying a correction factor to try
// to improve precision. It must be created using NewCompTimer. MinErrorFactor
// and MaxErrorFactor may be adjusted to reject correction factor outliers,
// which may be seen before enough data is collected. They default to 0 and 2,
// respectively.
type CompTimer struct {
	MinErrorFactor float64 `json:"min_error_factor"`
	MaxErrorFactor float64 `json:"max_error_factor"`
	avg            Averager
	stimer         *SimpleTimer
}

// NewCompTimer returns a new CompTimer with the specified Average.
// MinErrorFactor and MaxErrorFactor may be changed before use.
func NewCompTimer(a Averager) *CompTimer {
	return &CompTimer{DefaultCompTimerMinErrorFactor,
		DefaultCompTimerMaxErrorFactor, a, NewSimpleTimer()}
}

// NewDefaultCompTimer returns a new CompTimer with the default Average.
// MinErrorFactor and MaxErrorFactor may be changed before use.
func NewDefaultCompTimer() *CompTimer {
	return NewCompTimer(DefaultCompTimerAverage)
}

// Sleep selects on both a time.Timer channel and the done channel.
func (ct *CompTimer) Sleep(ctx context.Context, t time.Time, d time.Duration) (time.Time, error) {
	comp := ct.avg.Average()

	// do compensation
	if comp != 0 {
		d = time.Duration(float64(d) / comp)
	}

	// sleep and calculate error
	t2, err := ct.stimer.Sleep(ctx, t, d)
	erf := float64(t2.Sub(t)) / float64(d)

	// reject outliers
	if erf >= ct.MinErrorFactor && erf <= ct.MaxErrorFactor {
		ct.avg.Push(erf)
	}

	return t2, err
}

func (ct *CompTimer) String() string {
	return "comp"
}

// BusyTimer uses a busy wait loop to wait for the next send. It wastes CPU
// and should only be used for extremely tight timing requirements.
type BusyTimer struct {
}

// Sleep waits with a busy loop and checks the done channel every iteration.
func (bt *BusyTimer) Sleep(ctx context.Context, t time.Time, d time.Duration) (time.Time, error) {
	e := t.Add(d)
	for t.Before(e) {
		select {
		case <-ctx.Done():
			return t, ctx.Err()
		default:
			t = time.Now()
		}
	}
	return t, nil
}

func (bt *BusyTimer) String() string {
	return "busy"
}

// HybridTimer uses Go's default time functions and performs compensation to try
// to improve precision. To further improve precision, it sleeps to within some
// factor of the target value, then uses a busy wait loop for the remainder.
// The CPU will be in a busy wait for 1 - sleep factor for each sleep performed,
// so ideally the sleep factor should be increased to some threshold before
// precision starts to be lost, or some balance between the desired precision
// and CPU load is struck. The sleep factor typically can be increased for
// longer intervals and must be decreased for shorter intervals to keep high
// precision. In one example, a sleep factor of 0.95 could be used for 15ns
// precision at an interval of 200ms, but a sleep factor of 0.80 was required
// for 100ns precision at an interval of 1ms. These requirements will likely
// vary for different hardware and OS combinations.
type HybridTimer struct {
	ctimer *CompTimer
	slfct  float64
}

// NewHybridTimer returns a new HybridTimer using the given Average algorithm
// and sleep factor (0 - 1.0) before the busy wait.
func NewHybridTimer(a Averager, sleepFactor float64) *HybridTimer {
	return &HybridTimer{NewCompTimer(a), sleepFactor}
}

// NewDefaultHybridTimer returns a new HybridTimer using the default Average
// and sleep factor.
func NewDefaultHybridTimer() *HybridTimer {
	return NewHybridTimer(DefaultCompTimerAverage, DefaultHybridTimerSleepFactor)
}

// SleepFactor returns the sleep factor.
func (ht *HybridTimer) SleepFactor() float64 {
	return ht.slfct
}

// Sleep selects on both a time.Timer channel and the done channel.
func (ht *HybridTimer) Sleep(ctx context.Context, t time.Time, d time.Duration) (time.Time, error) {
	e := t.Add(d)
	d = time.Duration(float64(d) * ht.slfct)
	t2, err := ht.ctimer.Sleep(ctx, t, d)
	if err != nil {
		return t2, err
	}
	for t2.Before(e) {
		select {
		case <-ctx.Done():
			return t2, ctx.Err()
		default:
			t2 = time.Now()
		}
	}
	return t2, nil
}

func (ht *HybridTimer) String() string {
	return fmt.Sprintf("hybrid:%f", ht.slfct)
}

// TimerFactories are the registered Timer factories.
var TimerFactories = make([]TimerFactory, 0)

// TimerFactory can create a Timer from a string.
type TimerFactory struct {
	FactoryFunc func(string, Averager) (Timer, error)
	Usage       string
}

// RegisterTimer registers a new Timer.
func RegisterTimer(fn func(string, Averager) (Timer, error), usage string) {
	TimerFactories = append(TimerFactories, TimerFactory{fn, usage})
}

// NewTimer returns a Timer from a string.
func NewTimer(s string, a Averager) (Timer, error) {
	for _, fac := range TimerFactories {
		t, err := fac.FactoryFunc(s, a)
		if err != nil {
			return nil, err
		}
		if t != nil {
			return t, nil
		}
	}
	return nil, Errorf(NoSuchTimer, "no such Timer %s", s)
}

func init() {
	RegisterTimer(
		func(s string, a Averager) (t Timer, err error) {
			if s == "simple" {
				t = NewSimpleTimer()
			}
			return
		},
		"simple: Go's standard time.Timer",
	)

	RegisterTimer(
		func(s string, a Averager) (t Timer, err error) {
			if s == "comp" {
				t = NewCompTimer(a)
			}
			return
		},
		"comp: simple timer with error compensation (see --tcomp)",
	)

	RegisterTimer(
		func(s string, a Averager) (t Timer, err error) {
			args := strings.Split(s, ":")
			if args[0] != "hybrid" {
				return nil, nil
			}
			if len(args) == 1 {
				return NewHybridTimer(a, DefaultHybridTimerSleepFactor), nil
			}
			sfct, err := strconv.ParseFloat(args[1], 64)
			if err != nil || sfct <= 0 || sfct >= 1 {
				return nil, Errorf(InvalidSleepFactor,
					"invalid sleep factor %s to hybrid timer", args[1])
			}
			return NewHybridTimer(a, sfct), nil
		},
		fmt.Sprintf("hybrid:#: hybrid comp/busy timer w/ sleep factor (dfl %.2f)",
			DefaultHybridTimerSleepFactor),
	)

	RegisterTimer(
		func(s string, a Averager) (t Timer, err error) {
			if s == "busy" {
				t = &BusyTimer{}
			}
			return
		},
		"busy: busy wait loop (high precision and CPU, blasphemy)",
	)
}
