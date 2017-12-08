package irtt

import "time"

// ServerConfig defines the Server configuration.
type ServerConfig struct {
	Addrs       []string
	HMACKey     []byte
	MaxDuration time.Duration
	MinInterval time.Duration
	MaxLength   int
	PacketBurst int
	Filler      Filler
	AllowStamp  AllowStamp
	AllowDSCP   bool
	TTL         int
	IPVersion   IPVersion
	Handler     Handler
	SetSrcIP    bool
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
		PacketBurst: DefaultPacketBurst,
		Filler:      DefaultServerFiller,
		AllowStamp:  DefaultAllowStamp,
		AllowDSCP:   DefaultAllowDSCP,
		TTL:         DefaultTTL,
		IPVersion:   DefaultIPVersion,
		SetSrcIP:    DefaultSetSrcIP,
		GCMode:      DefaultGCMode,
		ThreadLock:  DefaultThreadLock,
	}
}
