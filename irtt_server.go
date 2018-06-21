package irtt

import (
	"log/syslog"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	flag "github.com/ogier/pflag"
)

const defaultSyslogTag = "irtt"

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
	printf("--syslog=uri   log events to syslog (default don't use syslog)")
	printf("               URI format: protocol://host:port/tag, examples:")
	printf("               local: log to local syslog, default tag irtt")
	printf("               udp://logsrv:514/irttsrv: UDP to logsrv:514, tag irttsrv")
	printf("               tcp://logsrv:8514/: TCP to logsrv:8514, default tag irtt")
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
	printf("--set-src-ip   set source IP address on all outgoing packets from listeners")
	printf("               on unspecified IP addresses (use for more reliable reply")
	printf("               routing, but increases per-packet heap allocations)")
	printf("--gc=mode      sets garbage collection mode (default %s)", DefaultGCMode)
	printf("               on: garbage collector always on")
	printf("               off: garbage collector always off")
	printf("               idle: garbage collector enabled only when idle")
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
	var baddrsStr = fs.StringP("b", "b", strings.Join(DefaultBindAddrs, ","), "bind addresses")
	var maxDuration = fs.DurationP("d", "d", DefaultMaxDuration, "max duration")
	var minInterval = fs.DurationP("i", "i", DefaultMinInterval, "min interval")
	var maxLength = fs.IntP("l", "l", DefaultMaxLength, "max length")
	var allowTimestampStr = fs.String("tstamp", DefaultAllowStamp.String(), "allow timestamp")
	var hmacStr = fs.String("hmac", defaultHMACKey, "HMAC key")
	var syslogStr = fs.String("syslog", "", "syslog uri")
	var timeout = fs.Duration("timeout", DefaultServerTimeout, "timeout")
	var packetBurst = fs.Int("pburst", DefaultPacketBurst, "packet burst")
	var fillStr = fs.String("fill", DefaultServerFiller.String(), "fill")
	var allowFillsStr = fs.String("allow-fills", strings.Join(DefaultAllowFills, ","), "sfill")
	var ipv4 = fs.BoolP("4", "4", false, "IPv4 only")
	var ipv6 = fs.BoolP("6", "6", false, "IPv6 only")
	var ttl = fs.Int("ttl", DefaultTTL, "IP time to live")
	var noDSCP = fs.Bool("no-dscp", !DefaultAllowDSCP, "no DSCP")
	var setSrcIP = fs.Bool("set-src-ip", DefaultSetSrcIP, "set source IP")
	var gcModeStr = fs.String("gc", DefaultGCMode.String(), "gc mode")
	var lockOSThread = fs.Bool("thread", DefaultThreadLock, "thread")
	var version = fs.BoolP("version", "v", false, "version")
	fs.Parse(args)

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

	// parse syslog URI
	var syslogWriter *syslog.Writer
	if *syslogStr != "" {
		var suri *url.URL
		suri, err = parseSyslogURI(*syslogStr)
		exitOnError(err, exitCodeBadCommandLine)

		prio := syslog.LOG_DAEMON | syslog.LOG_INFO
		if suri.Scheme == "local" {
			syslogWriter, err = syslog.New(prio, suri.Path)
		} else {
			syslogWriter, err = syslog.Dial(suri.Scheme, suri.Host, prio, suri.Path)
		}
		exitOnError(err, exitCodeRuntimeError)
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
	cfg.Timeout = *timeout
	cfg.PacketBurst = *packetBurst
	cfg.MaxLength = *maxLength
	cfg.Filler = filler
	cfg.AllowFills = strings.Split(*allowFillsStr, ",")
	cfg.AllowDSCP = !*noDSCP
	cfg.TTL = *ttl
	cfg.Handler = &serverHandler{syslogWriter}
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

	err = s.ListenAndServe()
	exitOnError(err, exitCodeRuntimeError)
}

type serverHandler struct {
	syslogWriter *syslog.Writer
}

func (s *serverHandler) OnEvent(ev *Event) {
	println(ev.String())

	if s.syslogWriter != nil {
		if ev.IsError() {
			s.syslogWriter.Err(ev.String())
		} else {
			s.syslogWriter.Info(ev.String())
		}
	}
}

func parseSyslogURI(suri string) (u *url.URL, err error) {
	if u, err = url.Parse(suri); err != nil {
		return
	}
	if u.Path[0] == '/' {
		u.Path = u.Path[1:]
	}
	if u.Path == "" {
		u.Path = defaultSyslogTag
	}
	return
}
