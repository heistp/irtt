package irtt

import (
	"time"
)

// Common defaults.
const (
	DefaultIPVersion  = DualStack
	DefaultPort       = "2112"
	DefaultTTL        = 0
	DefaultThreadLock = false
)

// Client defaults.
const (
	DefaultDuration                = time.Duration(1) * time.Hour
	DefaultInterval                = time.Duration(1) * time.Second
	DefaultLength                  = 0
	DefaultReceivedStats           = ReceivedStatsBoth
	DefaultStampAt                 = AtBoth
	DefaultClock                   = BothClocks
	DefaultDSCP                    = 0
	DefaultStrictParams            = false
	DefaultLocalAddress            = ":0"
	DefaultLocalPort               = "0"
	DefaultDF                      = DFDefault
	DefaultCompTimerMinErrorFactor = 0.0
	DefaultCompTimerMaxErrorFactor = 2.0
	DefaultHybridTimerSleepFactor  = 0.95
	DefaultAverageWindow           = 5
	DefaultExponentialAverageAlpha = 0.1
	DefaultEventMask               = AllEvents
)

// DefaultOpenTimeouts are the default timeouts used when the client opens a
// connection to the server.
var DefaultOpenTimeouts = Durations([]time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
})

// DefaultCompTimerAverage is the default timer error averaging algorithm for
// the CompTimer.
var DefaultCompTimerAverage = NewDefaultExponentialAverager()

// DefaultWait is the default client wait time for the final responses after all
// packets have been sent.
var DefaultWait = &WaitMaxRTT{time.Duration(4) * time.Second, 3}

// DefaultTimer is the default timer implementation, CompTimer.
var DefaultTimer = NewCompTimer(DefaultCompTimerAverage)

// DefaultFillPattern is the default fill pattern.
var DefaultFillPattern = []byte("irtt")

// DefaultServerFiller it the default filler for the server, PatternFiller.
var DefaultServerFiller = NewDefaultPatternFiller()

// Server defaults.
const (
	DefaultBindAddr    = ":2112"
	DefaultMaxDuration = time.Duration(0)
	DefaultMinInterval = time.Duration(0)
	DefaultMaxLength   = 0
	DefaultPacketBurst = 10
	DefaultAllowStamp  = DualStamps
	DefaultGoroutines  = 1
)
