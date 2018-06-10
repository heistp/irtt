package irtt

import (
	"fmt"
	"time"
)

// TimeSource provides wall and monotonic clock values.
type TimeSource interface {
	// Now returns the current time, with wall or monotonic clock values set
	// according to the specified Clock.
	Now(c Clock) Time

	String() string
}

// GoTimeSource uses Go's default time functions.
type GoTimeSource struct {
	monotonicStart time.Time
}

// Now returns a Time containing the current time.
func (g *GoTimeSource) Now(clock Clock) Time {
	now := time.Now()
	switch clock {
	case Wall:
		return Time{now.UnixNano(), time.Duration(0)}
	case Monotonic:
		return Time{0, now.Sub(g.monotonicStart)}
	case BothClocks:
		return Time{now.UnixNano(), now.Sub(g.monotonicStart)}
	default:
		panic(fmt.Sprintf("unknown clock %s", clock))
	}
}

func (g *GoTimeSource) String() string {
	return "go"
}

// NewGoTimeSource returns a new Go TimeSource.
func NewGoTimeSource() *GoTimeSource {
	return &GoTimeSource{time.Now()}
}

// TimeSourceFactories are the registered TimeSource factories.
var TimeSourceFactories = make([]TimeSourceFactory, 0)

// TimeSourceFactory can create a TimeSource from a string.
type TimeSourceFactory struct {
	FactoryFunc func(string) (TimeSource, error)
	Usage       string
}

// RegisterTimeSource registers a new TimeSource.
func RegisterTimeSource(fn func(string) (TimeSource, error), usage string) {
	TimeSourceFactories = append(TimeSourceFactories, TimeSourceFactory{fn, usage})
}

// NewTimeSource returns a TimeSource from a string.
func NewTimeSource(s string) (TimeSource, error) {
	for _, fac := range TimeSourceFactories {
		t, err := fac.FactoryFunc(s)
		if err != nil {
			return nil, err
		}
		if t != nil {
			return t, nil
		}
	}
	return nil, Errorf(NoSuchTimeSource, "no such TimeSource %s", s)
}

func init() {
	RegisterTimeSource(
		func(s string) (t TimeSource, err error) {
			if s == "go" {
				t = NewGoTimeSource()
			}
			return
		},
		"go: Go's standard time.Time functions",
	)
}
