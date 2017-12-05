package irtt

import (
	"encoding/json"
	"net"
)

// ClientConfig defines the Client configuration.
type ClientConfig struct {
	LocalAddress  string
	RemoteAddress string
	LocalAddr     net.Addr
	RemoteAddr    net.Addr
	OpenTimeouts  Durations
	NoTest        bool
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
	ThreadLock   bool
	Supplied     *ClientConfig
}

// NewDefaultConfig returns a new Config with the default settings.
func NewDefaultConfig() *ClientConfig {
	return &ClientConfig{
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
		ThreadLock:   DefaultThreadLock,
	}
}

// validate validates the configuration
func (c *ClientConfig) validate() error {
	if c.Interval <= 0 {
		return Errorf(IntervalNonPositive, "interval (%s) must be > 0", c.Interval)
	}
	if c.Duration <= 0 {
		return Errorf(DurationNonPositive, "duration (%s) must be > 0", c.Duration)
	}
	return validateInterval(c.Interval)
}

// MarshalJSON implements the json.Marshaler interface.
func (c *ClientConfig) MarshalJSON() ([]byte, error) {
	fstr := "none"
	if c.Filler != nil {
		fstr = c.Filler.String()
	}

	j := &struct {
		LocalAddress  string `json:"local_address"`
		RemoteAddress string `json:"remote_address"`
		OpenTimeouts  string `json:"open_timeouts"`
		Params        `json:"params"`
		StrictParams  bool          `json:"strict_params"`
		IPVersion     IPVersion     `json:"ip_version"`
		DF            DF            `json:"df"`
		TTL           int           `json:"ttl"`
		Timer         string        `json:"timer"`
		Waiter        string        `json:"waiter"`
		Filler        string        `json:"filler"`
		FillAll       bool          `json:"fill_all"`
		ThreadLock    bool          `json:"thread_lock"`
		Supplied      *ClientConfig `json:"supplied,omitempty"`
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
		ThreadLock:    c.ThreadLock,
		Supplied:      c.Supplied,
	}
	return json.Marshal(j)
}
