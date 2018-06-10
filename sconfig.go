package irtt

import "time"

// ServerConfig defines the Server configuration.
type ServerConfig struct {
	Addrs       []string
	HMACKey     []byte
	MaxDuration time.Duration
	MinInterval time.Duration
	MaxLength   int
	Timeout     time.Duration
	PacketBurst int
	Filler      Filler
	AllowFills  []string
	AllowStamp  AllowStamp
	AllowDSCP   bool
	TTL         int
	IPVersion   IPVersion
	Handler     Handler
	SetSrcIP    bool
	TimeSource  TimeSource
	GCMode      GCMode
	ThreadLock  bool
}

// NewServerConfig returns a new ServerConfig with the default settings.
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		Addrs:       DefaultBindAddrs,
		MaxDuration: DefaultMaxDuration,
		MinInterval: DefaultMinInterval,
		MaxLength:   DefaultMaxLength,
		Timeout:     DefaultServerTimeout,
		PacketBurst: DefaultPacketBurst,
		Filler:      DefaultServerFiller,
		AllowFills:  DefaultAllowFills,
		AllowStamp:  DefaultAllowStamp,
		AllowDSCP:   DefaultAllowDSCP,
		TTL:         DefaultTTL,
		IPVersion:   DefaultIPVersion,
		SetSrcIP:    DefaultSetSrcIP,
		TimeSource:  DefaultTimeSource,
		GCMode:      DefaultGCMode,
		ThreadLock:  DefaultThreadLock,
	}
}
