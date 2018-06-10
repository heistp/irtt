package irtt

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// InvalidDuration indicates a duration that is not valid.
const InvalidDuration = time.Duration(math.MaxInt64)

// Durations contains a slice of time.Duration.
type Durations []time.Duration

func (ds Durations) String() string {
	dss := make([]string, len(ds))
	for i, d := range ds {
		dss[i] = d.String()
	}
	return strings.Join(dss, ",")
}

// ParseDurations returns a Durations value from a comma separated list of
// time.Duration string representations.
func ParseDurations(sdurs string) (durs Durations, err error) {
	ss := strings.Split(sdurs, ",")
	durs = make([]time.Duration, len(ss))
	for i, s := range ss {
		var err error
		durs[i], err = time.ParseDuration(s)
		if err != nil {
			return nil, err
		}
	}
	return durs, nil
}

// Time contains both wall clock (subject to system time adjustments) and
// monotonic clock (relative to a fixed start time, and not subject to system
// time adjustments) times in nanoseconds. The monotonic value should be used
// for calculating time differences, and the wall value must be used for
// comparing wall clock time. Comparisons between wall clock values are only as
// accurate as the synchronization between the clocks that produced the values.
type Time struct {
	// Wall is the wall clock value as the number of nanoseconds elapsed since
	// January 1, 1970 UTC.
	Wall int64 `json:"wall,omitempty"`

	// Monotonic is the monotonic clock value and may be relative to any start
	// point in time.
	Mono time.Duration `json:"monotonic,omitempty"`
}

// Sub returns the duration t-u. Monotonic times are used if both are non-zero.
func (t Time) Sub(u Time) time.Duration {
	if t.Mono != 0 && u.Mono != 0 {
		return t.Mono - u.Mono
	} else if t.Wall != 0 && u.Wall != 0 {
		return time.Duration(t.Wall - u.Wall)
	}
	panic("Time.Sub() clock mismatch")
}

// Add returns the time t+d.
func (t Time) Add(d time.Duration) Time {
	ok := false
	if t.Wall != 0 {
		t.Wall += int64(d)
		ok = true
	}
	if t.Mono != 0 {
		t.Mono += d
		ok = true
	}
	if !ok {
		panic("Time.Add() for zero time")
	}
	return t
}

// After returns whether the time instant t is after u.
func (t Time) After(u Time) bool {
	if t.Mono != 0 && u.Mono != 0 {
		return t.Mono > u.Mono
	} else if t.Wall != 0 && u.Wall != 0 {
		return t.Wall > u.Wall
	}
	panic("Time.After() clock mismatch")
}

// Before returns whether the time instant t is before u.
func (t Time) Before(u Time) bool {
	if t.Mono != 0 && u.Mono != 0 {
		return t.Mono < u.Mono
	} else if t.Wall != 0 && u.Wall != 0 {
		return t.Wall < u.Wall
	}
	panic("Time.Before() clock mismatch")
}

// Midpoint returns the point in time halfway between t and u.
func (t Time) Midpoint(u Time) Time {
	return t.Add(u.Sub(t) / 2)
}

// KeepClocks keeps only the specified clocks. Passing in Wall sets Mono to 0,
// Monotonic sets Wall to 0 and BothClocks does nothing.
func (t Time) KeepClocks(c Clock) Time {
	switch c {
	case Wall:
		t.Mono = 0
	case Monotonic:
		t.Wall = 0
	case BothClocks:
	default:
		panic(fmt.Sprintf("unknown clock %s", c))
	}
	return t
}

// IsWallZero returns true if Wall is zero.
func (t Time) IsWallZero() bool {
	return t.Wall == 0
}

// IsMonoZero returns true if Mono is zero.
func (t Time) IsMonoZero() bool {
	return t.Mono == 0
}

// IsZero returns true if both Wall and Mono are zero.
func (t Time) IsZero() bool {
	return t.IsWallZero() && t.IsMonoZero()
}

// Timestamp stores receive and send times. If the Timestamp was set to the
// midpoint on the server, Receive and Send will be the same.
type Timestamp struct {
	Receive Time `json:"receive"`
	Send    Time `json:"send"`
}

// IsBothMono returns true if there are both send and receive times from the
// monotonic clock.
func (t Timestamp) IsBothMono() bool {
	return !t.Receive.IsMonoZero() && !t.Send.IsMonoZero()
}

// IsBothWall returns true if there are both send and receive times from the
// wall clock.
func (t Timestamp) IsBothWall() bool {
	return !t.Receive.IsWallZero() && !t.Send.IsWallZero()
}

// BestSend returns the best send time. It prefers the actual send time, but
// returns the receive time if it's not available.
func (t Timestamp) BestSend() Time {
	if t.Send.IsZero() {
		return t.Receive
	}
	return t.Send
}

// BestReceive returns the best receive time. It prefers the actual receive
// time, but returns the receive time if it's not available.
func (t Timestamp) BestReceive() Time {
	if t.Receive.IsZero() {
		return t.Send
	}
	return t.Receive
}

// StampAt selects the time/s when timestamps are made on the server.
type StampAt int

// StampAt constants.
const (
	AtNone     StampAt = 0x00
	AtSend     StampAt = 0x01
	AtReceive  StampAt = 0x02
	AtBoth     StampAt = AtSend | AtReceive
	AtMidpoint StampAt = 0x04
)

var sas = [...]string{"none", "send", "receive", "both", "midpoint"}

func (sa StampAt) String() string {
	if int(sa) < 0 || int(sa) >= len(sas) {
		return fmt.Sprintf("StampAt:%d", sa)
	}
	return sas[sa]
}

// StampAtFromInt returns a StampAt value from its int constant.
func StampAtFromInt(v int) (StampAt, error) {
	if v < int(AtNone) || v > int(AtMidpoint) {
		return AtNone, Errorf(InvalidStampAtInt, "invalid StampAt int: %d", v)
	}
	return StampAt(v), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (sa StampAt) MarshalJSON() ([]byte, error) {
	return json.Marshal(sa.String())
}

// ParseStampAt returns a StampAt value from its string.
func ParseStampAt(s string) (StampAt, error) {
	for i, v := range sas {
		if v == s {
			return StampAt(i), nil
		}
	}
	return AtNone, Errorf(InvalidStampAtString, "invalid StampAt string: %s", s)
}

// Clock selects the clock/s to use for timestamps.
type Clock int

// Clock constants.
const (
	Wall       Clock = 0x01
	Monotonic  Clock = 0x02
	BothClocks Clock = Wall | Monotonic
)

var tcs = [...]string{"wall", "monotonic", "both"}

func (tc Clock) String() string {
	if int(tc) < 1 || int(tc) > len(tcs) {
		return fmt.Sprintf("Clock:%d", tc)
	}
	return tcs[tc-1]
}

// MarshalJSON implements the json.Marshaler interface.
func (tc Clock) MarshalJSON() ([]byte, error) {
	return json.Marshal(tc.String())
}

// ClockFromInt returns a Clock value from its int constant.
func ClockFromInt(v int) (Clock, error) {
	if v < int(Wall) || v > int(BothClocks) {
		return Clock(0), Errorf(InvalidClockInt, "invalid Clock int: %d", v)
	}
	return Clock(v), nil
}

// ParseClock returns a Clock from a string.
func ParseClock(s string) (Clock, error) {
	for i, v := range tcs {
		if s == v {
			return Clock(i + 1), nil
		}
	}
	return Clock(0), Errorf(InvalidClockString, "invalid Clock string: %s", s)
}

// clockFromBools returns a Clock for wall and monotonic booleans. Either w or m
// must be true.
func clockFromBools(w bool, m bool) Clock {
	if w {
		if m {
			return BothClocks
		}
		return Wall
	}
	if m {
		return Monotonic
	}
	panic(fmt.Sprintf("invalid clock booleans %t, %t", w, m))
}

// AllowStamp selects the timestamps that are allowed by the server.
type AllowStamp int

// AllowStamp constants.
const (
	NoStamp AllowStamp = iota
	SingleStamp
	DualStamps
)

var als = [...]string{"none", "single", "dual"}

// Restrict returns the StampAt allowed for a given StampAt requested.
func (a AllowStamp) Restrict(at StampAt) StampAt {
	if at == AtNone {
		return AtNone
	}
	switch a {
	case NoStamp:
		return AtNone
	case SingleStamp:
		switch at {
		case AtBoth:
			return AtMidpoint
		default:
			return at
		}
	case DualStamps:
		return at
	default:
		panic(fmt.Sprintf("unknown AllowStamp %d", a))
	}
}

func (a AllowStamp) String() string {
	if int(a) < 0 || int(a) >= len(als) {
		return fmt.Sprintf("AllowStamp:%d", a)
	}
	return als[a]
}

// ParseAllowStamp returns an AllowStamp from a string.
func ParseAllowStamp(s string) (AllowStamp, error) {
	for i, v := range als {
		if s == v {
			return AllowStamp(i), nil
		}
	}
	return NoStamp, Errorf(InvalidAllowStampString, "invalid AllowStamp string: %s", s)
}

// rdur rounds a Duration for improved readability.
func rdur(dur time.Duration) time.Duration {
	d := dur
	if d < 0 {
		d = -d
	}
	if d < 1000 {
		return dur
	} else if d < 10000 {
		return dur.Round(10 * time.Nanosecond)
	} else if d < 100000 {
		return dur.Round(100 * time.Nanosecond)
	} else if d < 1000000 {
		return dur.Round(1 * time.Microsecond)
	} else if d < 100000000 {
		return dur.Round(10 * time.Microsecond)
	} else if d < 1000000000 {
		return dur.Round(100 * time.Microsecond)
	} else if d < 10000000000 {
		return dur.Round(10 * time.Millisecond)
	} else if d < 60000000000 {
		return dur.Round(100 * time.Millisecond)
	}
	return dur.Round(time.Second)
}
