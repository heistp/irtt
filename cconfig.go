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
	LogCSV        bool
	NoTest        bool
	Params
	Loose      bool
	IPVersion  IPVersion
	DF         DF
	TTL        int
	Timer      Timer
	TimeSource TimeSource
	Waiter     Waiter
	Filler     Filler
	FillOne    bool
	HMACKey    []byte
	Handler    ClientHandler
	ThreadLock bool
	Supplied   *ClientConfig
}

// NewClientConfig returns a new ClientConfig with the default settings.
func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		LocalAddress: DefaultLocalAddress,
		OpenTimeouts: DefaultOpenTimeouts,
		Params: Params{
			ProtocolVersion: ProtocolVersion,
			Duration:        DefaultDuration,
			Interval:        DefaultInterval,
			Length:          DefaultLength,
			StampAt:         DefaultStampAt,
			Clock:           DefaultClock,
			DSCP:            DefaultDSCP,
		},
		Loose:      DefaultLoose,
		IPVersion:  DefaultIPVersion,
		DF:         DefaultDF,
		TTL:        DefaultTTL,
		Timer:      DefaultTimer,
		TimeSource: DefaultTimeSource,
		Waiter:     DefaultWait,
		ThreadLock: DefaultThreadLock,
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
	if len(c.ServerFill) > maxServerFillLen {
		return Errorf(ServerFillTooLong,
			"server fill string (%s) must be less than %d characters",
			c.ServerFill, maxServerFillLen)
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
		Loose         bool          `json:"loose"`
		IPVersion     IPVersion     `json:"ip_version"`
		DF            DF            `json:"df"`
		TTL           int           `json:"ttl"`
		Timer         string        `json:"timer"`
		TimeSource    string        `json:"time_source"`
		Waiter        string        `json:"waiter"`
		Filler        string        `json:"filler"`
		FillOne       bool          `json:"fill_one"`
		ServerFill    string        `json:"server_fill"`
		ThreadLock    bool          `json:"thread_lock"`
		Supplied      *ClientConfig `json:"supplied,omitempty"`
	}{
		LocalAddress:  c.LocalAddress,
		RemoteAddress: c.RemoteAddress,
		OpenTimeouts:  c.OpenTimeouts.String(),
		Params:        c.Params,
		Loose:         c.Loose,
		IPVersion:     c.IPVersion,
		DF:            c.DF,
		TTL:           c.TTL,
		Timer:         c.Timer.String(),
		TimeSource:    c.TimeSource.String(),
		Waiter:        c.Waiter.String(),
		Filler:        fstr,
		FillOne:       c.FillOne,
		ServerFill:    c.ServerFill,
		ThreadLock:    c.ThreadLock,
		Supplied:      c.Supplied,
	}
	return json.Marshal(j)
}
