package irtt

import (
	"context"
	"log"
	"os"
	"strings"

	flag "github.com/ogier/pflag"
)

func serverUsage() {
	setBufio()
	printf("Options:")
	printf("--------")
	printf("")
	printf("-b addresses   bind addresses (default \"%s\"), comma separated list of:", strings.Join(DefaultBindAddrs, ","))
	printf("               :port (unspecified address with port, use with care)")
	printf("               host (host with default port %s, see Host formats below)", DefaultPort)
	printf("               host:port (host with specified port, see Host formats below)")
	printf("               %%iface (all addresses on interface iface with default port %s)", DefaultPort)
	printf("               %%iface:port (all addresses on interface iface with port)")
	printf("               note: iface strings may contain * to match multiple interfaces")
	printf("-d duration    max test duration, or 0 for no maximum")
	printf("               (default %s, see Duration units below)", DefaultMaxDuration)
	printf("-i interval    min send interval, or 0 for no minimum")
	printf("               (default %s, see Duration units below)", DefaultMinInterval)
	printf("-l length      max packet length (default %d), or 0 for no maximum", DefaultMaxLength)
	printf("               numbers too small will cause test packets to be dropped")
	printf("--hmac=key     add HMAC with key (0x for hex) to all packets, provides:")
	printf("               dropping of all packets without a correct HMAC")
	printf("               protection for server against unauthorized discovery and use")
	if syslogSupport {
		printf("--syslog=uri   log events to syslog (default don't use syslog)")
		printf("               URI format: scheme://host:port/tag, examples:")
		printf("               local: log to local syslog, default tag irtt")
		printf("               local:/irttsrv: log to local syslog, tag irttsrv")
		printf("               udp://logsrv:514/irttsrv: UDP to logsrv:514, tag irttsrv")
		printf("               tcp://logsrv:8514/: TCP to logsrv:8514, default tag irtt")
	}
	printf("--timeout=dur  timeout for closing connections if no requests received")
	printf("               0 means no timeout (not recommended on public servers)")
	printf("               max client interval will be restricted to timeout/%d", maxIntervalTimeoutFactor)
	printf("               (default %s, see Duration units below)", DefaultServerTimeout)
	printf("--pburst=#     packet burst allowed before enforcing minimum interval")
	printf("               (default %d)", DefaultPacketBurst)
	printf("--fill=fill    payload fill if not requested (default %s)", DefaultServerFiller.String())
	printf("               none: echo client payload (insecure on public servers)")
	for _, ffac := range FillerFactories {
		printf("               %s", ffac.Usage)
	}
	printf("--allow-fills= comma separated patterns of fill requests to allow (default %s)", strings.Join(DefaultAllowFills, ","))
	printf("  fills        see options for --fill")
	printf("               allowing non-random fills insecure on public servers")
	printf("               use --allow-fills=\"\" to disallow all fill requests")
	printf("               note: patterns may contain * for matching")
	printf("--tstamp=modes timestamp modes to allow (default %s)", DefaultAllowStamp)
	printf("               none: don't allow timestamps")
	printf("               single: allow a single timestamp (send, receive or midpoint)")
	printf("               dual: allow dual timestamps")
	printf("--no-dscp      don't allow setting dscp (default %t)", !DefaultAllowDSCP)
	printf("-4             IPv4 only")
	printf("-6             IPv6 only")
	printf("--set-src-ip   set source IP address on all outgoing packets from listeners")
	printf("               on unspecified IP addresses (use for more reliable reply")
	printf("               routing, but increases per-packet heap allocations)")
	printf("--ecn          Ship ECN bits to be logged by the client.  Forces --set-src-ip, disables UDP replies")
	printf("--thread       lock request handling goroutines to OS threads")
	printf("-h             show help")
	printf("-v             show version")
	printf("")
	hostUsage()
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
	var (
		baddrsStr         = fs.StringP("b", "b", strings.Join(DefaultBindAddrs, ","), "bind addresses")
		maxDuration       = fs.DurationP("d", "d", DefaultMaxDuration, "max duration")
		minInterval       = fs.DurationP("i", "i", DefaultMinInterval, "min interval")
		maxLength         = fs.IntP("l", "l", DefaultMaxLength, "max length")
		allowTimestampStr = fs.String("tstamp", DefaultAllowStamp.String(), "allow timestamp")
		hmacStr           = fs.String("hmac", defaultHMACKey, "HMAC key")
		timeout           = fs.Duration("timeout", DefaultServerTimeout, "timeout")
		packetBurst       = fs.Int("pburst", DefaultPacketBurst, "packet burst")
		fillStr           = fs.String("fill", DefaultServerFiller.String(), "fill")
		allowFillsStr     = fs.String("allow-fills", strings.Join(DefaultAllowFills, ","), "sfill")
		ipv4              = fs.BoolP("4", "4", false, "IPv4 only")
		ipv6              = fs.BoolP("6", "6", false, "IPv6 only")
		ttl               = fs.Int("ttl", DefaultTTL, "IP time to live")
		noDSCP            = fs.Bool("no-dscp", !DefaultAllowDSCP, "no DSCP")
		setSrcIP          = fs.Bool("set-src-ip", DefaultSetSrcIP, "set source IP")
		ecn               = fs.Bool("ecn", DefaultSetECN, "enable ECN capture - disables UDP replies from server")
		lockOSThread      = fs.Bool("thread", DefaultThreadLock, "thread")
		version           = fs.BoolP("version", "v", false, "version")
	)
	var syslogStr *string
	if syslogSupport {
		syslogStr = fs.String("syslog", "", "syslog uri")
	}

	err := fs.Parse(args)
	if err != nil {
		log.Fatal("runServerCLI fs.Parse(args) err:", err)
	}

	// start profiling, if enabled in build
	if profileEnabled {
		defer startProfile("./server.pprof").Stop()
	}

	// version
	if *version {
		runVersion(args)
		os.Exit(0)
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

	// create event handler with console handler as default
	handler := &MultiHandler{[]Handler{&consoleHandler{}}}

	// add syslog event handler
	if syslogStr != nil && *syslogStr != "" {
		sh, err := newSyslogHandler(*syslogStr)
		exitOnError(err, exitCodeRuntimeError)
		handler.AddHandler(sh)
	}

	// create server config
	cfg := NewServerConfig()
	cfg.Addrs = strings.Split(*baddrsStr, ",")
	cfg.MaxDuration = *maxDuration
	cfg.MinInterval = *minInterval
	cfg.AllowStamp = allowStamp
	cfg.HMACKey = hmacKey
	cfg.Timeout = *timeout
	cfg.PacketBurst = *packetBurst
	cfg.MaxLength = *maxLength
	cfg.Filler = filler
	cfg.AllowFills = strings.Split(*allowFillsStr, ",")
	cfg.AllowDSCP = !*noDSCP
	cfg.TTL = *ttl
	cfg.Handler = handler
	cfg.IPVersion = ipVer
	cfg.SetSrcIP = *setSrcIP || *ecn
	cfg.ThreadLock = *lockOSThread

	// create server
	s := NewServer(cfg)

	// create context
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	go initSignalHandler(cancel, false)

	err = s.ListenAndServe()
	exitOnError(err, exitCodeRuntimeError)
}
