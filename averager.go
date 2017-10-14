package irtt

import (
	"fmt"
	"strconv"
	"strings"
)

// Averager is implemented to return an average of a series of given values.
type Averager interface {
	// Push adds a value to be averaged.
	Push(val float64)

	// Average returns the average.
	Average() float64

	String() string
}

// CumulativeAverager implements the cumulative moving average (takes into account
// all values equally).
type CumulativeAverager struct {
	sum float64
	n   float64
}

// Push adds a value.
func (ca *CumulativeAverager) Push(val float64) {
	ca.sum += val
	ca.n++
}

// Average gets the cumulative average.
func (ca *CumulativeAverager) Average() float64 {
	if ca.n == 0 {
		return 0
	}
	return ca.sum / ca.n
}

func (ca *CumulativeAverager) String() string {
	return "avg"
}

// ExponentialAverager implements the exponential moving average. More recent
// values are given higher consideration. Alpha must be between 0 and 1, where a
// higher Alpha discounts older values faster. An Alpha of 0.1 - 0.2 may give
// good results for timer compensation, but experimentation is required as
// results are dependent on hardware and test config.
type ExponentialAverager struct {
	Alpha float64
	avg   float64
	prev  float64
}

// Push adds a value.
func (ea *ExponentialAverager) Push(val float64) {
	if ea.avg == 0 {
		ea.prev = val
		ea.avg = val
		return
	}

	ea.prev = ea.avg
	ea.avg = ea.Alpha*val + (1-ea.Alpha)*ea.prev
}

// Average gets the exponential average.
func (ea *ExponentialAverager) Average() float64 {
	return ea.avg
}

func (ea *ExponentialAverager) String() string {
	return fmt.Sprintf("exp:%.2f", ea.Alpha)
}

// NewExponentialAverager returns a new ExponentialAverage with the specified
// Alpha.
func NewExponentialAverager(alpha float64) *ExponentialAverager {
	return &ExponentialAverager{Alpha: alpha}
}

// NewDefaultExponentialAverager returns a new ExponentialAverage with the
// default Alpha. This may be changed before used.
func NewDefaultExponentialAverager() *ExponentialAverager {
	return NewExponentialAverager(DefaultExponentialAverageAlpha)
}

// WindowAverager implements the moving average with a specified window.
type WindowAverager struct {
	Window int
	values []float64
	pos    int
	filled bool
}

// Push adds a value.
func (wa *WindowAverager) Push(val float64) {
	wa.values[wa.pos] = val
	wa.pos++
	if wa.pos == wa.Window {
		wa.pos = 0
		wa.filled = true
	}
}

// Average gets the moving average.
func (wa *WindowAverager) Average() float64 {
	var sum = float64(0)
	var c = wa.Window - 1

	// ignore unavailable values
	if !wa.filled {
		c = wa.pos - 1
		if c < 0 {
			return 0
		}
	}

	// sum values
	var ic = 0
	for i := 0; i <= c; i++ {
		sum += wa.values[i]
		ic++
	}

	// calculate average and return
	avg := sum / float64(ic)
	return avg
}

func (wa *WindowAverager) String() string {
	return fmt.Sprintf("win:%d", wa.Window)
}

// NewWindowAverage returns a new WindowAverage with the specified window.
func NewWindowAverage(window int) *WindowAverager {
	return &WindowAverager{
		Window: window,
		values: make([]float64, window),
		pos:    0,
		filled: false,
	}
}

// NewDefaultWindowAverager returns a new WindowAverage with the default window.
func NewDefaultWindowAverager() *WindowAverager {
	return NewWindowAverage(DefaultAverageWindow)
}

// AveragerFactories are the registered Averager factories.
var AveragerFactories = make([]AveragerFactory, 0)

// AveragerFactory is the definition for an Averager.
type AveragerFactory struct {
	FactoryFunc func(string) (Averager, error)
	Usage       string
}

// RegisterAverager registers a new Averager.
func RegisterAverager(fn func(string) (Averager, error), usage string) {
	AveragerFactories = append(AveragerFactories, AveragerFactory{fn, usage})
}

// NewAverager returns an Averager from a string.
func NewAverager(s string) (Averager, error) {
	for _, fac := range AveragerFactories {
		a, err := fac.FactoryFunc(s)
		if err != nil {
			return nil, err
		}
		if a != nil {
			return a, nil
		}
	}
	return nil, Errorf(NoSuchAverager, "no such Averager %s", s)
}

func init() {
	RegisterAverager(
		func(s string) (a Averager, err error) {
			if s == "avg" {
				a = &CumulativeAverager{}
			}
			return
		},
		"avg: cumulative average error",
	)

	RegisterAverager(
		func(s string) (Averager, error) {
			args := strings.Split(s, ":")
			if args[0] != "win" {
				return nil, nil
			}
			if len(args) == 1 {
				return NewDefaultWindowAverager(), nil
			}
			w, err := strconv.Atoi(args[1])
			if err != nil || w < 1 {
				return nil, Errorf(InvalidWinAvgWindow, "invalid window %s to window average", args[1])
			}
			return NewWindowAverage(w), nil
		},
		fmt.Sprintf("win:#: moving average error with window # (default %d)",
			DefaultAverageWindow),
	)

	RegisterAverager(
		func(s string) (Averager, error) {
			args := strings.Split(s, ":")
			if args[0] != "exp" {
				return nil, nil
			}
			if len(args) == 1 {
				return NewDefaultExponentialAverager(), nil
			}
			a, err := strconv.ParseFloat(args[1], 64)
			if err != nil || a < 0 || a > 1 {
				return nil, Errorf(InvalidExpAvgAlpha, "invalid alpha %s to exponential average", args[1])
			}
			return NewExponentialAverager(a), nil
		},
		fmt.Sprintf("exp:#: exponential average with alpha # (default %.2f)",
			DefaultExponentialAverageAlpha),
	)
}
