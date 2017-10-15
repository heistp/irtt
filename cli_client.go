package irtt

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
)

func clientUsage() {
	setBufio()
	printf("Usage: client [options] host|host:port")
	printf("")
	printf("Options:")
	printf("--------")
	printf("")
	printf("-d duration    total time to send (default %s, see Duration units below)", DefaultDuration)
	printf("-i interval    send interval (default %s, see Duration units below)", DefaultInterval)
	printf("-l length      length of packet (including irtt headers, default %d)", DefaultLength)
	printf("               increased as necessary for irtt headers, common values:")
	printf("               1472 (max unfragmented size of IPv4 datagram for 1500 byte MTU)")
	printf("               1452 (max unfragmented size of IPv6 datagram for 1500 byte MTU)")
	printf("-ts mode       timestamp mode (timestamps can estimate one-way delay and IPDV")
	printf("               if clocks are synchronized externally, default %s):", DefaultStampAt.String())
	printf("               none: request no timestamps")
	printf("               send: request timestamp from server send")
	printf("               receive: request timestamp from server receive")
	printf("               both: request both send and receive timestamps")
	printf("               midpoint: request midpoint timestamp (send/receive avg)")
	printf("-clock clock   clock/s used for server timestamps (default %s)", DefaultClock)
	printf("               wall: wall clock only")
	printf("               mono: monotonic clock only")
	printf("               both: both clocks")
	printf("-o file        write JSON output to file (or 'stdout' for stdout)")
	printf("               extension .json or .json.gz added as appropriate")
	printf("-nogzip        do not gzip JSON output")
	printf("-q             quiet, suppress all output")
	printf("-v             verbose, show received packets")
	printf("-dscp dscp     dscp value (default %s, 0x prefix for hex), common values:", strconv.Itoa(DefaultDSCP))
	printf("               0 (Best effort)")
	printf("               8 (Bulk)")
	printf("               40 (CS5)")
	printf("               46 (Expedited forwarding)")
	printf("-df string     setting for do not fragment (DF) bit in all packets:")
	printf("               default: OS default")
	printf("               false: DF bit not set")
	printf("               true: DF bit set")
	printf("-wait wait     wait time at end of test for unreceived replies (default %s)", DefaultWait.String())
	printf("               - Valid formats -")
	for _, wfac := range WaiterFactories {
		printf("               %s", wfac.Usage)
	}
	printf("               - Examples -")
	printf("               3x4s: 3 times max RTT, or 4 seconds if no response")
	printf("               1500ms: fixed 1500 milliseconds")
	printf("-timer timer   timer for waiting to send packets (default %s)", DefaultTimer.String())
	for _, tfac := range TimerFactories {
		printf("               %s", tfac.Usage)
	}
	printf("-tcomp alg     comp timer averaging algorithm (default %s)", DefaultCompTimerAverage.String())
	for _, afac := range AveragerFactories {
		printf("               %s", afac.Usage)
	}
	printf("-fill fill     fill payload with given data (default none)")
	printf("               none: leave payload as all zeroes")
	for _, ffac := range FillerFactories {
		printf("               %s", ffac.Usage)
	}
	printf("-fillall       fill all packets instead of repeating the first")
	printf("               (makes rand unique per packet and pattern continue)")
	printf("-local addr    local address (default from OS), valid formats:")
	printf("               :port (all IPv4/IPv6 addresses with port)")
	printf("               host (IPv4 addr or hostname with dynamic port)")
	printf("               host:port (IPv4 addr or hostname with port)")
	printf("               [ipv6-host%%zone] (IPv6 addr or hostname with dynamic port)")
	printf("               [ipv6-host%%zone]:port (IPv6 addr or hostname with port)")
	printf("-hmac key      add HMAC with key (0x for hex) to all packets, provides:")
	printf("               dropping of all packets without a correct HMAC")
	printf("               protection for server against unauthorized discovery and use")
	printf("-4             IPv4 only")
	printf("-6             IPv6 only")
	printf("-ttl ttl       time to live (default %d, meaning use OS default)", DefaultTTL)
	printf("-thread        lock sending and receiving goroutines to OS threads (may")
	printf("               reduce mean latency, but may also add outliers)")
	printf("")
	durationUsage()
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
	fs := flag.NewFlagSet("client", 0)
	fs.Usage = func() {
		usageAndExit(clientUsage, exitCodeBadCommandLine)
	}
	var duration = fs.Duration("d", DefaultDuration, "length of time to run test")
	var interval = fs.Duration("i", DefaultInterval, "send interval")
	var length = fs.Int("l", DefaultLength, "packet length")
	var tsatStr = fs.String("ts", DefaultStampAt.String(), "stamp at")
	var clockStr = fs.String("clock", DefaultClock.String(), "clock")
	var outputStr = fs.String("o", "", "output file")
	var noGzip = fs.Bool("nogzip", false, "no GZIP")
	var quiet = fs.Bool("q", defaultQuiet, "quiet")
	var verbose = fs.Bool("v", defaultVerbose, "verbose")
	var dscpStr = fs.String("dscp", strconv.Itoa(DefaultDSCP), "dscp value")
	var dfStr = fs.String("df", DefaultDF.String(), "do not fragment")
	var waitStr = fs.String("wait", DefaultWait.String(), "wait")
	var timerStr = fs.String("timer", DefaultTimer.String(), "timer")
	var tcompStr = fs.String("tcomp", DefaultCompTimerAverage.String(),
		"timer compensation algorithm")
	var fillStr = fs.String("fill", "none", "fill")
	var fillAll = fs.Bool("fillall", false, "fill all")
	var laddrStr = fs.String("local", DefaultLocalAddress, "local address")
	var hmacStr = fs.String("hmac", defaultHMACKey, "HMAC key")
	var ipv4 = fs.Bool("4", false, "IPv4 only")
	var ipv6 = fs.Bool("6", false, "IPv6 only")
	var ttl = fs.Int("ttl", DefaultTTL, "IP time to live")
	var lockOSThread = fs.Bool("thread", DefaultLockOSThread, "thread")
	fs.Parse(args)

	// start profiling, if enabled in build
	if profileEnabled {
		defer startProfile("./client.pprof").Stop()
	}

	// determine IP version
	ipVer := IPVersionFromBooleans(*ipv4, *ipv6, DualStack)

	// parse DSCP
	dscp, err := strconv.ParseInt(*dscpStr, 0, 32)
	exitOnError(err, exitCodeBadCommandLine)

	// parse DF
	df, err := DFFromString(*dfStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse wait
	waiter, err := NewWaiter(*waitStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse timestamp string
	at, err := StampAtFromString(*tsatStr)
	exitOnError(err, exitCodeBadCommandLine)

	// parse clock
	clock, err := ClockFromString(*clockStr)
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

	// parse HMAC key
	var hmacKey []byte
	if *hmacStr != "" {
		hmacKey, err = decodeHexOrNot(*hmacStr)
		exitOnError(err, exitCodeBadCommandLine)
	}

	// check for remote address argument
	if len(fs.Args()) < 1 {
		usageAndExit(clientUsage, exitCodeBadCommandLine)
	}
	raddrStr := fs.Args()[0]

	// send regular output to stderr if json going to stdout
	if *outputStr == "stdout" {
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
		printf("%s", sig)
		cancel()

		sig = <-sigs
		printf("second interrupt, exiting")
		os.Exit(exitCodeDoubleSignal)
	}()

	// create config
	cfg := NewDefaultConfig()
	cfg.LocalAddress = *laddrStr
	cfg.RemoteAddress = raddrStr
	cfg.Duration = *duration
	cfg.Interval = *interval
	cfg.Length = *length
	cfg.StampAt = at
	cfg.Clock = clock
	cfg.IPVersion = ipVer
	cfg.DSCP = int(dscp)
	cfg.DF = df
	cfg.TTL = int(*ttl)
	cfg.Timer = timer
	cfg.Filler = filler
	cfg.FillAll = *fillAll
	cfg.Waiter = waiter
	cfg.HMACKey = hmacKey
	cfg.Handler = &clientHandler{*quiet, *verbose}
	cfg.EventMask = AllEvents
	cfg.LockOSThread = *lockOSThread

	// run test
	c := NewClient(cfg)
	r, err := c.Run(ctx)
	if err != nil {
		exitOnError(err, exitCodeRuntimeError)
	}

	// print results
	if !*quiet {
		printResult(r)
	}

	// write results to JSON
	if *outputStr != "" {
		if err := writeResultJSON(r, *outputStr, *noGzip); err != nil {
			exitOnError(err, exitCodeRuntimeError)
		}
	}
}

func printResult(r *Result) {
	// set some stat variables just for later brevity
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
		if r.SendErr == context.Canceled {
			printf("\nEarly termination due to cancellation")
		} else {
			printf("\nEarly termination due to send error: %s", r.SendErr)
		}
	}
	if r.ReceiveErr != nil {
		printf("\nEarly termination due to receive error: %s", r.ReceiveErr)
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

func writeResultJSON(r *Result, output string, noGzip bool) error {
	var jout io.Writer

	addExtension := func(path string, ext string) string {
		if strings.HasSuffix(path, ext) {
			return path
		}
		return path + ext
	}

	if output == "stdout" {
		jout = os.Stdout
	} else {
		if !noGzip {
			output = addExtension(output, ".json.gz")
		} else {
			output = addExtension(output, ".json")
		}
		of, err := os.Create(output)
		if err != nil {
			exitOnError(err, exitCodeRuntimeError)
		}
		defer of.Close()
		jout = of
	}
	if !noGzip {
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
	quiet   bool
	verbose bool
}

func (c *clientHandler) OnSent(seqno Seqno, rt Timestamps, length int,
	rec *Recorder) {
}

func (c *clientHandler) OnReceived(seqno Seqno, rt Timestamps, length int,
	dup bool, rec *Recorder) {
	if !c.quiet {
		if dup {
			printf("DUP! len=%d seq=%d", length, seqno)
			return
		}

		if c.verbose {
			rec.RLock()
			defer rec.RUnlock()
			rd := ""
			if rt.ReceiveDelay() != 0 {
				rd = fmt.Sprintf(" rd=%s", rdur(rt.ReceiveDelay()))
			}
			sd := ""
			if rt.SendDelay() != 0 {
				sd = fmt.Sprintf(" sd=%s", rdur(rt.SendDelay()))
			}
			printf("seq=%d len=%d rtt=%s%s%s", seqno, length, rdur(rt.RTT()), rd, sd)
		}
	}
}

func (c *clientHandler) OnEvent(ev *Event) {
	printf("%s", ev)
}

func rdur(dur time.Duration) time.Duration {
	d := dur
	if d < 0 {
		d = -d
	}
	if d < 1000 {
		return dur
	} else if d < 10000 {
		return dur.Round(10 * time.Nanosecond)
	} else if d < 100000 {
		return dur.Round(100 * time.Nanosecond)
	} else if d < 1000000 {
		return dur.Round(1 * time.Microsecond)
	} else if d < 100000000 {
		return dur.Round(10 * time.Microsecond)
	} else if d < 1000000000 {
		return dur.Round(100 * time.Microsecond)
	} else if d < 10000000000 {
		return dur.Round(10 * time.Millisecond)
	} else if d < 60000000000 {
		return dur.Round(100 * time.Millisecond)
	}
	return dur.Round(time.Second)
}
