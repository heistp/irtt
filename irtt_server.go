package irtt

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func serverUsage() {
	setBufio()
	printf("Options:")
	printf("--------")
	printf("")
	printf("-b addresses   bind addresses (default \"%s\"), comma separated list of:", strings.Join(DefaultBindAddrs, ","))
	printf("               :port (unspecified address with port, use with care)")
	printf("               host (IPv4 addr or hostname with default port %s)", DefaultPort)
	printf("               host:port (IPv4 addr or hostname with port)")
	printf("               [ipv6-host%%zone] (IPv6 addr or hostname with default port %s)", DefaultPort)
	printf("               [ipv6-host%%zone]:port (IPv6 addr or hostname with port)")
	printf("               %%iface (all addresses on interface iface with default port %s)", DefaultPort)
	printf("               %%iface:port (all addresses on interface iface with port)")
	printf("               note: iface strings may contain * to match multiple interfaces")
	printf("-d duration    max test duration, or 0 for no maximum")
	printf("               (default %s, see Duration units below)", DefaultMaxDuration)
	printf("-i interval    min send interval, or 0 for no minimum")
	printf("               (default %s, see Duration units below)", DefaultMinInterval)
	printf("-l length      max packet length (default %d), or 0 for no maximum", DefaultMaxLength)
	printf("               numbers too small will cause test packets to be dropped")
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
	printf("-nodscp        don't allow setting dscp (default %t)", !DefaultAllowDSCP)
	printf("-setsrcip      set source IP address on all outgoing packets from listeners")
	printf("               on unspecified IP addresses (use for more reliable reply")
	printf("               routing, but increases per-packet heap allocations)")
	printf("-gc mode       sets garbage collection mode (default %s)", DefaultGCMode)
	printf("               on: garbage collector always on")
	printf("               off: garbage collector always off")
	printf("               idle: garbage collector enabled only when idle")
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
	var baddrsStr = fs.String("b", strings.Join(DefaultBindAddrs, ","), "bind addresses")
	var maxDuration = fs.Duration("d", DefaultMaxDuration, "max duration")
	var minInterval = fs.Duration("i", DefaultMinInterval, "min interval")
	var maxLength = fs.Int("l", DefaultMaxLength, "max length")
	var allowTimestampStr = fs.String("ts", DefaultAllowStamp.String(), "allow timestamp")
	var hmacStr = fs.String("hmac", defaultHMACKey, "HMAC key")
	var packetBurst = fs.Int("pburst", DefaultPacketBurst, "packet burst")
	var fillStr = fs.String("fill", DefaultServerFiller.String(), "filler")
	var ipv4 = fs.Bool("4", false, "IPv4 only")
	var ipv6 = fs.Bool("6", false, "IPv6 only")
	var ttl = fs.Int("ttl", DefaultTTL, "IP time to live")
	var noDSCP = fs.Bool("nodscp", !DefaultAllowDSCP, "no DSCP")
	var setSrcIP = fs.Bool("setsrcip", DefaultSetSrcIP, "set source IP")
	var gcModeStr = fs.String("gc", DefaultGCMode.String(), "gc mode")
	var lockOSThread = fs.Bool("thread", DefaultThreadLock, "thread")
	fs.Parse(args)

	// start profiling, if enabled in build
	if profileEnabled {
		defer startProfile("./server.pprof").Stop()
	}

	// determine IP version
	ipVer := IPVersionFromBooleans(*ipv4, *ipv6, DualStack)

	// parse allow stamp string
	allowStamp, err := ParseAllowStamp(*allowTimestampStr)
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

	// parse GC mode
	gcMode, err := ParseGCMode(*gcModeStr)
	exitOnError(err, exitCodeBadCommandLine)

	// create server config
	cfg := NewServerConfig()
	cfg.Addrs = strings.Split(*baddrsStr, ",")
	cfg.MaxDuration = *maxDuration
	cfg.MinInterval = *minInterval
	cfg.AllowStamp = allowStamp
	cfg.HMACKey = hmacKey
	cfg.PacketBurst = *packetBurst
	cfg.MaxLength = *maxLength
	cfg.Filler = filler
	cfg.AllowDSCP = !*noDSCP
	cfg.TTL = *ttl
	cfg.Handler = &serverHandler{}
	cfg.IPVersion = ipVer
	cfg.SetSrcIP = *setSrcIP
	cfg.GCMode = gcMode
	cfg.ThreadLock = *lockOSThread

	// create server
	s := NewServer(cfg)

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

	if err := s.ListenAndServe(); err != nil {
		printf("Error: %s", err)
	}
}

type serverHandler struct {
}

func (s *serverHandler) OnEvent(ev *Event) {
	println(ev.String())
}
