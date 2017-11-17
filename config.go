package irtt

import (
	"encoding/json"
	"net"
)

// Config defines the test configuration.
type Config struct {
	LocalAddress  string
	RemoteAddress string
	LocalAddr     net.Addr
	RemoteAddr    net.Addr
	OpenTimeouts  Durations
	Params
	StrictParams bool
	IPVersion    IPVersion
	DF           DF
	TTL          int
	Timer        Timer
	Waiter       Waiter
	Filler       Filler
	FillAll      bool
	HMACKey      []byte
	Handler      ClientHandler
	EventMask    EventCode
	LockOSThread bool
	Supplied     *Config
}

// NewDefaultConfig returns a new Config with the default settings.
func NewDefaultConfig() *Config {
	return &Config{
		LocalAddress: DefaultLocalAddress,
		OpenTimeouts: DefaultOpenTimeouts,
		Params: Params{
			Duration: DefaultDuration,
			Interval: DefaultInterval,
			Length:   DefaultLength,
			StampAt:  DefaultStampAt,
			Clock:    DefaultClock,
			DSCP:     DefaultDSCP,
		},
		StrictParams: DefaultStrictParams,
		IPVersion:    DefaultIPVersion,
		DF:           DefaultDF,
		TTL:          DefaultTTL,
		Timer:        DefaultTimer,
		Waiter:       DefaultWait,
		EventMask:    DefaultEventMask,
		LockOSThread: DefaultLockOSThread,
	}
}

// validate validates the configuration
func (c *Config) validate() error {
	if c.Interval <= 0 {
		return Errorf(IntervalNonPositive, "interval (%s) must be > 0", c.Interval)
	}
	if c.Duration <= 0 {
		return Errorf(DurationNonPositive, "duration (%s) must be > 0", c.Duration)
	}
	return validateInterval(c.Interval)
}

// MarshalJSON implements the json.Marshaler interface.
func (c *Config) MarshalJSON() ([]byte, error) {
	fstr := "none"
	if c.Filler != nil {
		fstr = c.Filler.String()
	}

	j := &struct {
		LocalAddress  string `json:"local_address"`
		RemoteAddress string `json:"remote_address"`
		OpenTimeouts  string `json:"open_timeouts"`
		Params        `json:"params"`
		StrictParams  bool      `json:"strict_params"`
		IPVersion     IPVersion `json:"ip_version"`
		DF            DF        `json:"df"`
		TTL           int       `json:"ttl"`
		Timer         string    `json:"timer"`
		Waiter        string    `json:"waiter"`
		Filler        string    `json:"filler"`
		FillAll       bool      `json:"fill_all"`
		LockOSThread  bool      `json:"lock_os_thread"`
		Supplied      *Config   `json:"supplied,omitempty"`
	}{
		LocalAddress:  c.LocalAddress,
		RemoteAddress: c.RemoteAddress,
		OpenTimeouts:  c.OpenTimeouts.String(),
		Params:        c.Params,
		StrictParams:  c.StrictParams,
		IPVersion:     c.IPVersion,
		DF:            c.DF,
		TTL:           c.TTL,
		Timer:         c.Timer.String(),
		Waiter:        c.Waiter.String(),
		Filler:        fstr,
		FillAll:       c.FillAll,
		LockOSThread:  c.LockOSThread,
		Supplied:      c.Supplied,
	}
	return json.Marshal(j)
}
