package irtt

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	flag "github.com/ogier/pflag"
)

func clientUsage() {
	setBufio()
	printf("Usage: client [flags] host|host:port (see Host formats below)")
	printf("")
	printf("Flags:")
	printf("------")
	printf("")
	printf("-d duration    total time to send (default %s, see Duration units below)", DefaultDuration)
	printf("-i interval    send interval (default %s, see Duration units below)", DefaultInterval)
	printf("-l length      length of packet (including irtt headers, default %d)", DefaultLength)
	printf("               increased as necessary for irtt headers, common values:")
	printf("               1472 (max unfragmented size of IPv4 datagram for 1500 byte MTU)")
	printf("               1452 (max unfragmented size of IPv6 datagram for 1500 byte MTU)")
	printf("-o file        write JSON output to file (use '-' for stdout)")
	printf("               if file has no extension, .json.gz is added, output is gzipped")
	printf("               if extension is .json.gz, output is gzipped")
	printf("               if extension is .gz, it's changed to .json.gz, output is gzipped")
	printf("               if extension is .json, output is not gzipped")
	printf("               output to stdout is not gzipped, pipe to gzip if needed")
	printf("-q             quiet, suppress per-packet output")
	printf("-Q             really quiet, suppress all output except errors to stderr")
	printf("-n             no test, connect to the server and validate test parameters")
	printf("               but don't run the test")
	printf("--stats=stats  server stats on received packets (default %s)", DefaultReceivedStats.String())
	printf("               none: no server stats on received packets")
	printf("               count: total count of received packets")
	printf("               window: receipt status of last 64 packets with each reply")
	printf("               both: both count and window")
	printf("--tstamp=mode  server timestamp mode (default %s)", DefaultStampAt.String())
	printf("               none: request no timestamps")
	printf("               send: request timestamp at server send")
	printf("               receive: request timestamp at server receive")
	printf("               both: request both send and receive timestamps")
	printf("               midpoint: request midpoint timestamp (send/receive avg)")
	printf("--clock=clock  clock/s used for server timestamps (default %s)", DefaultClock)
	printf("               wall: wall clock only")
	printf("               monotonic: monotonic clock only")
	printf("               both: both clocks")
	printf("--dscp=dscp    dscp value (default %s, 0x prefix for hex), common values:", strconv.Itoa(DefaultDSCP))
	printf("               0 (Best effort)")
	printf("               8 (Bulk)")
	printf("               40 (CS5)")
	printf("               46 (Expedited forwarding)")
	printf("--df=string    setting for do not fragment (DF) bit in all packets:")
	printf("               default: OS default")
	printf("               false: DF bit not set")
	printf("               true: DF bit set")
	printf("--wait=wait    wait time at end of test for unreceived replies (default %s)", DefaultWait.String())
	printf("               - Valid formats -")
	for _, wfac := range WaiterFactories {
		printf("               %s", wfac.Usage)
	}
	printf("               - Examples -")
	printf("               3x4s: 3 times max RTT, or 4 seconds if no response")
	printf("               1500ms: fixed 1500 milliseconds")
	printf("--timer=timer  timer for waiting to send packets (default %s)", DefaultTimer.String())
	for _, tfac := range TimerFactories {
		printf("               %s", tfac.Usage)
	}
	printf("--tcomp=alg    comp timer averaging algorithm (default %s)", DefaultCompTimerAverage.String())
	for _, afac := range AveragerFactories {
		printf("               %s", afac.Usage)
	}
	printf("--fill=fill    fill payload with given data (default none)")
	printf("               none: leave payload as all zeroes")
	for _, ffac := range FillerFactories {
		printf("               %s", ffac.Usage)
	}
	printf("--fill-one     fill only once and repeat for all packets")
	printf("--sfill=fill   request server fill (default not specified)")
	printf("               see options for --fill")
	printf("               server must support and allow this fill with --allow-fills")
	printf("--local=addr   local address (default from OS), valid formats:")
	printf("               :port (all IPv4/IPv6 addresses with port)")
	printf("               host (host with dynamic port, see Host formats below)")
	printf("               host:port (host with specified port, see Host formats below)")
	printf("--hmac=key     add HMAC with key (0x for hex) to all packets, provides:")
	printf("               dropping of all packets without a correct HMAC")
	printf("               protection for server against unauthorized discovery and use")
	printf("-4             IPv4 only")
	printf("-6             IPv6 only")
	printf("--timeouts=drs timeouts used when connecting to server (default %s)", DefaultOpenTimeouts.String())
	printf("               comma separated list of durations (see Duration units below)")
	printf("               total wait time will be up to the sum of these Durations")
	printf("               max packets sent is up to the number of Durations")
	printf("               minimum timeout duration is %s", minOpenTimeout)
	printf("--ttl=ttl      time to live (default %d, meaning use OS default)", DefaultTTL)
	printf("--loose        accept and use any server restricted test parameters instead")
	printf("               of exiting with nonzero status")
	printf("--thread       lock sending and receiving goroutines to OS threads")
	printf("-h             show help")
	printf("-v             show version")
	printf("")
	hostUsage()
	printf("")
	durationUsage()
}

func hostUsage() {
	printf("Host formats:")
	printf("-------------")
	printf("")
	printf("Hosts may be either hostnames (for IPv4 or IPv6) or IP addresses. IPv6")
	printf("addresses must be surrounded by brackets and may include a zone after the %%")
	printf("character. Examples:")
	printf("")
	printf("IPv4 IP: 192.168.1.10")
	printf("IPv6 IP: [fe80::426c:8fff:fe13:9feb%%en0]")
	printf("IPv4/6 hostname: localhost")
	printf("")
	printf("Note: IPv6 addresses must be quoted in most shells.")
}

func durationUsage() {
	printf("Duration units:")
	printf("---------------")
	printf("")
	printf("Durations are a sequence of decimal numbers, each with optional fraction, and")
	printf("unit suffix, such as: \"300ms\", \"1m30s\" or \"2.5m\". Sanity not enforced.")
	printf("")
	printf("h              hours")
	printf("m              minutes")
	printf("s              seconds")
	printf("ms             milliseconds")
	printf("ns             nanoseconds")
}

// runClientCLI runs the client command line interface.
func runClientCLI(args []string) {
	// client flags
	fs := flag.NewFlagSet("client", flag.ContinueOnError)
	fs.Usage = func() {
		usageAndExit(clientUsage, exitCodeBadCommandLine)
	}
	var durationStr = fs.StringP("d", "d", DefaultDuration.String(), "total time to send")
	var intervalStr = fs.StringP("i", "i", DefaultInterval.String(), "send interval")
	var length = fs.IntP("l", "l", DefaultLength, "packet length")
	var noTest = fs.BoolP("n", "n", false, "no test")
	var rsStr = fs.String("stats", DefaultReceivedStats.String(), "received stats")
	var tsatStr = fs.String("tstamp", DefaultStampAt.String(), "stamp at")
	var clockStr = fs.String("clock", DefaultClock.String(), "clock")
	var outputStr = fs.StringP("o", "o", "", "output file")
	var quiet = fs.BoolP("q", "q", defaultQuiet, "quiet")
	var reallyQuiet = fs.BoolP("Q", "Q", defaultReallyQuiet, "really quiet")
	var dscpStr = fs.String("dscp", strconv.Itoa(DefaultDSCP), "dscp value")
	var dfStr = fs.String("df", DefaultDF.String(), "do not fragment")
	var waitStr = fs.String("wait", DefaultWait.String(), "wait")
	var timerStr = fs.String("timer", DefaultTimer.String(), "timer")
	var tcompStr = fs.String("tcomp", DefaultCompTimerAverage.String(),
		"timer compensation algorithm")
	var fillStr = fs.String("fill", "none", "fill")
	var fillOne = fs.Bool("fill-one", false, "fill one")
	var sfillStr = fs.String("sfill", "", "sfill")
	var laddrStr = fs.String("local", DefaultLocalAddress, "local address")
	var hmacStr = fs.String("hmac", defaultHMACKey, "HMAC key")
	var ipv4 = fs.BoolP("4", "4", false, "IPv4 only")
	var ipv6 = fs.BoolP("6", "6", false, "IPv6 only")
	var timeoutsStr = fs.String("timeouts", DefaultOpenTimeouts.String(), "open timeouts")
	var ttl = fs.Int("ttl", DefaultTTL, "IP time to live")
	var loose = fs.Bool("loose", DefaultLoose, "loose")
	var threadLock = fs.Bool("thread", DefaultThreadLock, "thread")
	var version = fs.BoolP("version", "v", false, "version")
	err := fs.Parse(args)

	// start profiling, if enabled in build
	if profileEnabled {
		defer startProfile("./client.pprof").Stop()
	}

	// version
	if *version {
		runVersion(args)
		os.Exit(0)
	}

	// parse duration
	duration, err := time.ParseDuration(*durationStr)
	if err != nil {
		exitOnError(fmt.Errorf("%s (use s for seconds)", err),
			exitCodeBadCommandLine)
	}

	// parse interval
	interval, err := time.ParseDuration(*intervalStr)
	if err != nil {
		exitOnError(fmt.Errorf("%s (use s for seconds)", err),
			exitCodeBadCommandLine)
	}

	// determine IP version
	ipVer := IPVersionFromBooleans(*ipv4, *ipv6, DualStack)

	// parse DSCP
	dscp, err := strconv.ParseInt(*dscpStr, 0, 32)
	exitOnError(err, exitCodeBadCommandLine)

	// parse DF
	df, err := ParseDF(*dfStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse wait
	waiter, err := NewWaiter(*waitStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse received stats
	rs, err := ParseReceivedStats(*rsStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse timestamp string
	at, err := ParseStampAt(*tsatStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse clock
	clock, err := ParseClock(*clockStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse timer compensation
	timerComp, err := NewAverager(*tcompStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse timer
	timer, err := NewTimer(*timerStr, timerComp)
	exitOnError(err, exitCodeBadCommandLine)

	// parse fill
	filler, err := NewFiller(*fillStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse open timeouts
	timeouts, err := ParseDurations(*timeoutsStr)
	if err != nil {
		exitOnError(fmt.Errorf("%s (use s for seconds)", err),
			exitCodeBadCommandLine)
	}

	// parse HMAC key
	var hmacKey []byte
	if *hmacStr != "" {
		hmacKey, err = decodeHexOrNot(*hmacStr)
		exitOnError(err, exitCodeBadCommandLine)
	}

	// check for remote address argument
	if len(fs.Args()) != 1 {
		usageAndExit(clientUsage, exitCodeBadCommandLine)
	}
	raddrStr := fs.Args()[0]

	// send regular output to stderr if json going to stdout
	if *outputStr == "-" {
		printTo = os.Stderr
	}

	// create context
	ctx, cancel := context.WithCancel(context.Background())

	// install signal handler to cancel context, which stops the test
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-sigs
		if !*reallyQuiet {
			printf("%s", sig)
		}
		cancel()

		sig = <-sigs
		if !*reallyQuiet {
			printf("second interrupt, exiting")
		}
		os.Exit(exitCodeDoubleSignal)
	}()

	// create config
	cfg := NewClientConfig()
	cfg.LocalAddress = *laddrStr
	cfg.RemoteAddress = raddrStr
	cfg.OpenTimeouts = timeouts
	cfg.NoTest = *noTest
	cfg.Duration = duration
	cfg.Interval = interval
	cfg.Length = *length
	cfg.ReceivedStats = rs
	cfg.StampAt = at
	cfg.Clock = clock
	cfg.DSCP = int(dscp)
	cfg.ServerFill = *sfillStr
	cfg.Loose = *loose
	cfg.IPVersion = ipVer
	cfg.DF = df
	cfg.TTL = int(*ttl)
	cfg.Timer = timer
	cfg.Waiter = waiter
	cfg.Filler = filler
	cfg.FillOne = *fillOne
	cfg.HMACKey = hmacKey
	cfg.Handler = &clientHandler{*quiet, *reallyQuiet}
	cfg.ThreadLock = *threadLock

	// run test
	c := NewClient(cfg)
	r, err := c.Run(ctx)
	if err != nil {
		exitOnError(err, exitCodeRuntimeError)
	}

	// exit if NoTest set
	if cfg.NoTest {
		return
	}

	// print results
	if !*reallyQuiet {
		printResult(r)
	}

	// write results to JSON
	if *outputStr != "" {
		if err := writeResultJSON(r, *outputStr, ctx.Err() != nil); err != nil {
			exitOnError(err, exitCodeRuntimeError)
		}
	}
}

func printResult(r *Result) {
	// set some stat variables for later brevity
	rtts := r.RTTStats
	rttvs := r.RoundTripIPDVStats
	sds := r.SendDelayStats
	svs := r.SendIPDVStats
	rds := r.ReceiveDelayStats
	rvs := r.ReceiveIPDVStats
	ss := r.SendCallStats
	tes := r.TimerErrorStats
	sps := r.ServerProcessingTimeStats

	if r.SendErr != nil {
		if r.SendErr != context.Canceled {
			printf("\nTerminated due to send error: %s", r.SendErr)
		}
	}
	if r.ReceiveErr != nil {
		printf("\nTerminated due to receive error: %s", r.ReceiveErr)
	}
	printf("")

	printStats := func(title string, s DurationStats) {
		if s.N > 0 {
			var med string
			if m, ok := s.Median(); ok {
				med = rdur(m).String()
			}
			printf("%s\t%s\t%s\t%s\t%s\t%s\t", title, rdur(s.Min), rdur(s.Mean()),
				med, rdur(s.Max), rdur(s.Stddev()))
		}
	}

	setTabWriter(tabwriter.AlignRight)

	printf("\tMin\tMean\tMedian\tMax\tStddev\t")
	printf("\t---\t----\t------\t---\t------\t")
	printStats("RTT", rtts)
	printStats("send delay", sds)
	printStats("receive delay", rds)
	printf("\t\t\t\t\t\t")
	printStats("IPDV (jitter)", rttvs)
	printStats("send IPDV", svs)
	printStats("receive IPDV", rvs)
	printf("\t\t\t\t\t\t")
	printStats("send call time", ss)
	printStats("timer error", tes)
	printStats("server proc. time", sps)
	printf("")
	printf("                duration: %s (wait %s)", rdur(r.Duration), rdur(r.Wait))
	printf("   packets sent/received: %d/%d (%.2f%% loss)", r.PacketsSent,
		r.PacketsReceived, r.PacketLossPercent)
	if r.PacketsReceived > 0 && r.ServerPacketsReceived > 0 {
		printf(" server packets received: %d/%d (%.2f%%/%.2f%% loss up/down)",
			r.ServerPacketsReceived, r.PacketsSent, r.UpstreamLossPercent,
			r.DownstreamLossPercent)
	}
	if r.Duplicates > 0 {
		printf("          *** DUPLICATES: %d (%.2f%%)", r.Duplicates,
			r.DuplicatePercent)
	}
	if r.LatePackets > 0 {
		printf("late (out-of-order) pkts: %d (%.2f%%)", r.LatePackets,
			r.LatePacketsPercent)
	}
	printf("     bytes sent/received: %d/%d", r.BytesSent, r.BytesReceived)
	printf("       send/receive rate: %s / %s", r.SendRate, r.ReceiveRate)
	printf("           packet length: %d bytes", r.Config.Length)
	printf("             timer stats: %d/%d (%.2f%%) missed, %.2f%% error",
		r.TimerMisses, r.ExpectedPacketsSent, r.TimerMissPercent,
		r.TimerErrPercent)

	flush()
}

func writeResultJSON(r *Result, output string, cancelled bool) error {
	var jout io.Writer

	var gz bool
	if output == "-" {
		if cancelled {
			return nil
		}
		jout = os.Stdout
	} else {
		gz = true
		if strings.HasSuffix(output, ".json") {
			gz = false
		} else if !strings.HasSuffix(output, ".json.gz") {
			if strings.HasSuffix(output, ".gz") {
				output = output[:len(output)-3] + ".json.gz"
			} else {
				output = output + ".json.gz"
			}
		}
		of, err := os.Create(output)
		if err != nil {
			exitOnError(err, exitCodeRuntimeError)
		}
		defer of.Close()
		jout = of
	}
	if gz {
		gzw := gzip.NewWriter(jout)
		defer func() {
			gzw.Flush()
			gzw.Close()
		}()
		jout = gzw
	}
	e := json.NewEncoder(jout)
	e.SetIndent("", "    ")
	return e.Encode(r)
}

type clientHandler struct {
	quiet       bool
	reallyQuiet bool
}

func (c *clientHandler) OnSent(seqno Seqno, rtd *RoundTripData) {
}

func (c *clientHandler) OnReceived(seqno Seqno, rtd *RoundTripData,
	prtd *RoundTripData, late bool, dup bool) {
	if !c.reallyQuiet {
		if dup {
			printf("DUP! seq=%d", seqno)
			return
		}

		if !c.quiet {
			ipdv := "n/a"
			if prtd != nil {
				dv := rtd.IPDVSince(prtd)
				if dv != InvalidDuration {
					ipdv = rdur(AbsDuration(dv)).String()
				}
			}
			rd := ""
			if rtd.ReceiveDelay() != InvalidDuration {
				rd = fmt.Sprintf(" rd=%s", rdur(rtd.ReceiveDelay()))
			}
			sd := ""
			if rtd.SendDelay() != InvalidDuration {
				sd = fmt.Sprintf(" sd=%s", rdur(rtd.SendDelay()))
			}
			sl := ""
			if late {
				sl = " (LATE)"
			}
			printf("seq=%d rtt=%s%s%s ipdv=%s%s", seqno, rdur(rtd.RTT()),
				rd, sd, ipdv, sl)
		}
	}
}

func (c *clientHandler) OnEvent(ev *Event) {
	if !c.reallyQuiet {
		printf("%s", ev)
	}
}
