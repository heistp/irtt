package irtt

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

func serverUsage() {
	setBufio()
	printf("Options:")
	printf("--------")
	printf("")
	printf("-b addresses   bind addresses (default %s), comma separated list of:", DefaultBindAddr)
	printf("               :port (all IPv4/IPv6 addresses with port)")
	printf("               host (IPv4 addr or hostname with default port %s)", DefaultPort)
	printf("               host:port (IPv4 addr or hostname with port)")
	printf("               [ipv6-host%%zone] (IPv6 addr or hostname with default port %s)", DefaultPort)
	printf("               [ipv6-host%%zone]:port (IPv6 addr or hostname with port)")
	printf("-d duration    max test duration, or 0 for no maximum")
	printf("               (default %s, see Duration units below)", DefaultMaxDuration)
	printf("-i interval    min send interval, or 0 for no minimum")
	printf("               (default %s, see Duration units below)", DefaultMinInterval)
	printf("-l length      max packet length (default %d)", DefaultMaxLength)
	printf("               0 means calculate from max MTU of listen interfaces")
	printf("-hmac key      add HMAC with key (0x for hex) to all packets, provides:")
	printf("               dropping of all packets without a correct HMAC")
	printf("               protection for server against unauthorized discovery and use")
	printf("-pburst #      packet burst allowed before enforcing minimum interval")
	printf("               (default %d)", DefaultPacketBurst)
	printf("-fill fill     fill payload with given data (default %s)", DefaultServerFiller.String())
	printf("               none: leave payload as all zeroes")
	for _, ffac := range FillerFactories {
		printf("               %s", ffac.Usage)
	}
	printf("-ts tsmode     timestamp modes to allow (default %s)", DefaultAllowStamp)
	printf("               none: don't allow timestamps")
	printf("               single: allow a single timestamp (send, receive or midpoint)")
	printf("               dual: allow dual timestamps")
	printf("-goroutines #  number of goroutines to serve requests with (default %d)", DefaultGoroutines)
	printf("               0 means use the number of CPUs reported by Go (%d)", runtime.NumCPU())
	printf("               increasing this adds both concurrency and lock contention")
	printf("-thread        lock request handling goroutines to OS threads (may reduce")
	printf("               mean latency, but may also add outliers)")
	printf("")
	durationUsage()
}

// runServerCLI runs the server command line interface.
func runServerCLI(args []string) {
	// server flags
	fs := flag.NewFlagSet("server", 0)
	fs.Usage = func() {
		usageAndExit(serverUsage, exitCodeBadCommandLine)
	}
	var baddrsStr = fs.String("b", DefaultBindAddr, "bind addresses")
	var maxDuration = fs.Duration("d", DefaultMaxDuration, "max duration")
	var minInterval = fs.Duration("i", DefaultMinInterval, "min interval")
	var maxLength = fs.Int("l", DefaultMaxLength, "max length")
	var allowTimestampStr = fs.String("ts", DefaultAllowStamp.String(), "allow timestamp")
	var goroutines = fs.Int("goroutines", DefaultGoroutines, "goroutines")
	var hmacStr = fs.String("hmac", defaultHMACKey, "HMAC key")
	var packetBurst = fs.Int("pburst", DefaultPacketBurst, "packet burst")
	var fillStr = fs.String("fill", DefaultServerFiller.String(), "filler")
	var ipv4 = fs.Bool("4", false, "IPv4 only")
	var ipv6 = fs.Bool("6", false, "IPv6 only")
	var ttl = fs.Int("ttl", DefaultTTL, "IP time to live")
	var lockOSThread = fs.Bool("thread", DefaultLockOSThread, "thread")
	fs.Parse(args)

	// start profiling, if enabled in build
	if profileEnabled {
		defer startProfile("./server.pprof").Stop()
	}

	// determine IP version
	ipVer := IPVersionFromBooleans(*ipv4, *ipv6, DualStack)

	// parse allow stamp string
	allowStamp, err := AllowStampFromString(*allowTimestampStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse fill
	filler, err := NewFiller(*fillStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse HMAC key
	var hmacKey []byte
	if *hmacStr != "" {
		hmacKey, err = decodeHexOrNot(*hmacStr)
		exitOnError(err, exitCodeBadCommandLine)
	}

	// create server
	s := NewServer()
	s.Addrs = strings.Split(*baddrsStr, ",")
	s.MaxDuration = *maxDuration
	s.MinInterval = *minInterval
	s.AllowStamp = allowStamp
	s.HMACKey = hmacKey
	s.PacketBurst = *packetBurst
	s.MaxLength = *maxLength
	s.Filler = filler
	s.TTL = *ttl
	s.Goroutines = *goroutines
	s.Handler = &serverHandler{}
	s.IPVersion = ipVer
	s.LockOSThread = *lockOSThread

	// install signal handler to stop server
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-sigs
		printf("%s", sig)
		s.Shutdown()

		sig = <-sigs
		os.Exit(exitCodeDoubleSignal)
	}()

	printf("IRTT server starting...")
	if err := s.ListenAndServe(); err != nil {
		printf("Error: %s", err)
	}
}

type serverHandler struct {
}

func (s *serverHandler) OnEvent(ev *Event) {
	printf(ev.String())
}
