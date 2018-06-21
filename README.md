# IRTT (Isochronous Round-Trip Tester)

IRTT measures round-trip time, one-way delay and other metrics using UDP
packets sent on a fixed period, and produces both user and machine parseable
output.

IRTT has reached version 0.9.0, and is usable today, but needs more work until
version 1.0.0 can be released. I would appreciate any feedback, which you can
send under Issues. However, it could be useful to first review the
[Roadmap](#roadmap) section of the documentation before submitting a new bug or
feature request.

## Table of Contents

1. [Motivation](#motivation)
2. [Goals](#goals)
3. [Features](#features)
4. [Limitations](#limitations)
5. [Installation](#installation)
6. [Documentation](#documentation)
7. [Frequently Asked Questions](#frequently-asked-questions)
8. [Roadmap](#roadmap)
9. [Changes](#changes)
10. [Thanks](#thanks)

## Motivation

Latency is an under-appreciated metric in network and application performance.
As of this writing, many broadband connections are well past the point of
diminishing returns when it comes to throughput, yet that’s what we continue to
take as the primary measure of Internet performance. This is analogous to
ordinary car buyers making top speed their first priority.

There is a certain hard to quantify but visceral “latency stress” that comes
from waiting in expectation after a web page click, straining through a delayed
and garbled VoIP conversation, or losing at your favorite online game (unless
you like “lag” as an excuse). Those who work on reducing latency and improving
network performance characteristics beyond just throughput may be driven by the
idea of helping relieve this stress for others.

IRTT was originally written to improve the latency and packet loss measurements
for the excellent [Flent](https://flent.org) tool, but should be useful as a
standalone tool as well. Flent was developed by and for the
[Bufferbloat](https://www.bufferbloat.net/projects/) project, which aims to
reduce "chaotic and laggy network performance," making this project valuable to
anyone who values their time and sanity while using the Internet.

## Goals

The goals of this project are to:

- Accurately measure latency and other relevant metrics of network behavior
- Produce statistics via both human and machine parseable output
- Provide for reasonably secure use on both public and private servers
- Support small enough packet sizes for [VoIP](https://www.cisco.com/c/en/us/support/docs/voice/voice-quality/7934-bwidth-consume.html) simulation
- Support relevant socket options, including DSCP
- Use a single UDP port for deployment simplicity
- Provide an API for embedding and extensibility

## Features:

- Measurement of:
	- [RTT (round-trip time)](https://en.wikipedia.org/wiki/Round-trip_delay_time)
	- [OWD (one-way delay)](https://en.wikipedia.org/wiki/End-to-end_delay), given
		external clock synchronization
	- [IPDV (instantaneous packet delay variation)](https://en.wikipedia.org/wiki/Packet_delay_variation), usually referred to as jitter
	- [Packet loss](https://en.wikipedia.org/wiki/Packet_loss), with upstream and downstream differentiation
	- [Out-of-order](https://en.wikipedia.org/wiki/Out-of-order_delivery)
		(measured using late packets metric) and [duplicate](https://wiki.wireshark.org/DuplicatePackets) packets
	- [Bitrate](https://en.wikipedia.org/wiki/Bit_rate)
	- Timer error, send call time and server processing time
- Statistics: min, max, mean, median (for most quantities) and standard deviation
- One nanosecond time precision on Linux and OS/X, and 100ns on Windows
- Robustness in the face of clock drift and NTP corrections through the use of
  both wall and monotonic clocks
- Binary protocol with negotiated format for test packet lengths down to 16 bytes (without timestamps)
- HMAC support for private servers, preventing unauthorized discovery and use
- Support for a wide range of Go supported [platforms](https://github.com/golang/go/wiki/MinimumRequirements)
- Timer compensation to improve sleep send schedule accuracy
- Support for IPv4 and IPv6
- Public server protections, including:
	- Three-way handshake with returned 64-bit connection token, preventing reply
		redirection to spoofed source addresses
	- Limits on maximum test duration, minimum interval and maximum packet length,
		both advertised in the negotiation and enforced with hard limits to protect
		against rogue clients
	- Packet payload filling to prevent relaying of arbitrary traffic
- Output to JSON
- An available [SmokePing](https://oss.oetiker.ch/smokeping/) probe
  ([code](https://github.com/heistp/SmokePing/blob/master/lib/Smokeping/probes/IRTT.pm),
  [pull request](https://github.com/oetiker/SmokePing/pull/110))

## Limitations

See the
[LIMITATIONS](http://htmlpreview.github.io/?https://github.com/heistp/irtt/blob/master/doc/irtt.html#limitations)
section of the irtt(1) man page.

## Installation

To install IRTT manually or build from source, you must:

1. [Install Go](https://golang.org/dl/)
2. Install irtt: `go get -u github.com/heistp/irtt/cmd/irtt`
3. For convenience, copy the `irtt` executable, which should be in
   `$HOME/go/bin`, or `$GOPATH/bin` if you have `$GOPATH` defined, to somewhere
   on your `PATH`.

If you want to build the source for development, you must also:

1. Install the `pandoc` utility for generating man pages and HTML documentation
   from their markdown source files. This can be done with `apt-get install
   pandoc` on Debian flavors of Linux or `brew install pandoc` on OS/X. See the
   [Pandoc](http://pandoc.org/) site for more information.
2. Install the `stringer` utility by doing
   `go get -u golang.org/x/tools/cmd/stringer`.
   This is only necessary if you need to re-generate the `*_string.go` files that
   are generated by this tool, otherwise the checked in versions may also be
   used.
3. Use `build.sh` to build during development, which helps with development
   related tasks, such as generating source files and docs, and cross-compiling
   for testing. For example, `build.sh min linux-amd64` would compile a
   minimized binary for Linux on AMD64. See `build.sh` for more info and a
   "source-documented" list of platforms that the script supports. See [this
   page](http://golang.org/doc/install/source#environment) for a full list of
   valid GOOS GOARCH combinations. `build.sh install` runs Go's install command,
   which puts the resulting executable in `$GOPATH/bin`.

If you want to build from a branch, you should first follow the steps above,
then from the `github.com/heistp/irtt` directory, do:
1. `git checkout branch`
2. `go get ./...`
3. `go install ./cmd/irtt` or `./build.sh` and move resulting `irtt` executable
   to install location

## Documentation

After installing IRTT, see the man pages and their corresponding EXAMPLES
sections to get started quickly:
- [irtt(1)](http://htmlpreview.github.io/?https://github.com/heistp/irtt/blob/master/doc/irtt.html) | [EXAMPLES](http://htmlpreview.github.io/?https://github.com/heistp/irtt/blob/master/doc/irtt.html#examples)
- [irtt-client(1)](http://htmlpreview.github.io/?https://github.com/heistp/irtt/blob/master/doc/irtt-client.html) | [EXAMPLES](http://htmlpreview.github.io/?https://github.com/heistp/irtt/blob/master/doc/irtt-client.html#examples)
- [irtt-server(1)](http://htmlpreview.github.io/?https://github.com/heistp/irtt/blob/master/doc/irtt-server.html) | [EXAMPLES](http://htmlpreview.github.io/?https://github.com/heistp/irtt/blob/master/doc/irtt-server.html#examples)

## Frequently Asked Questions

1) Why not just use ping?

   Ping may be the preferred tool when measuring minimum latency, or for other
   reasons. IRTT's reported mean RTT is likely to be a bit higher (on the order
   of a couple hundred microseconds) and a bit more variable than the results
   reported by ping, due to the overhead of entering userspace, together with
   Go's system call overhead and scheduling variability. That said, this
   overhead should be negligible at most Internet RTTs, and there are advantages
   that IRTT has over ping when minimum RTT is not what you're measuring:

	 - In addition to round-trip time, IRTT also measures OWD, IPDV and upstream
	   vs downstream packet loss.
	 - Some device vendors prioritize ICMP, so ping may not be an accurate measure
		 of user-perceived latency.
	 - IRTT can use HMACs to protect private servers from unauthorized discovery
		 and use.
	 - IRTT has a three-way handshake to prevent test traffic redirection from
		 spoofed source IPs.
	 - IRTT can fill the payload (if included) with random or arbitrary data.
   - On Windows, ping has a precision of 0.5ms, while IRTT uses high resolution
     timer functions for a precision of 100ns (high resolution wall clock only
     available on Windows 8 or Windows 2012 Server and later).

   Also note the following behavioral differences between ping and IRTT:

   - IRTT makes a stateful connection to the server, whereas ping is stateless.
   - By default, ping waits for a reply before sending its next request, while
     IRTT keeps sending requests on the specified interval regardless of whether
     or not replies are received. The effect of this, for example, is that a
     fixed-length pause in server packet processing (with packets buffered
     during the pause) will look like a single high RTT in ping, and multiple
     high then descending RTTs in irtt for the duration of the maximum RTT.

2) Why can't the client connect to the server, and instead I get
   `Error: no reply from server`?

   There are a number of possible reasons for this:

   1) You've specified an incorrect hostname or IP address for the server.
   2) There is a firewall blocking packets from the client to the server.
      Traffic must be allowed on the chosen UDP port (default 2112).
   3) There is high packet loss. By default, up to four packets are sent when
      the client tries to connect to the server, using timeouts of 1, 2, 4 and 8
      seconds. If all of these are lost, the client won't connect to the server.
      In environments with known high packet loss, the `--timeouts` flag may
      be used to send more packets with the chosen timeouts before abandoning
      the connection.
   4) The server has an HMAC key set with `--hmac` and the client either has
      not specified a key or it's incorrect. Make sure the client has the
      correct HMAC key, also specified with the `--hmac` flag.
   5) You're trying to connect to a listener that's listening on an unspecified
      IP address, but reply packets are coming back on a different route from the
      requests, or not coming back at all. This can happen for example in
      network environments with [asymmetric routing and a firewall or NAT]
      (https://www.cisco.com/web/services/news/ts_newsletter/tech/chalktalk/archives/200903.html).
      The best solution may be to change the network configuration to avoid this
      problem, but when this is not possible, try running the server with the
      `--set-src-ip` flag, which explicitly sets the source address on all reply
      packets from listeners on unspecified IP addresses to the destination
      address that the request was received on. This is not done by default in
      order to avoid the additional per-packet heap allocations required by the
      `golang.org/x/net` packages.

3) Why is the send (or receive) delay negative or much larger than I expect?

	 The client and server clocks must be synchronized for one-way delay values to
	 be meaningful (although, the relative change of send and receive delay may be
   useful to look at even without clock synchronization). Well-configured NTP
   hosts may be able to synchronize to within a few milliseconds.
	 [PTP](https://en.wikipedia.org/wiki/Precision_Time_Protocol)
	 ([Linux](http://linuxptp.sourceforge.net) implementation here) is capable of
   much higher precision. For example, using two
   [PCEngines APU2](http://pcengines.ch/apu2.htm) boards (which support PTP
   hardware timestamps) connected directly by Ethernet, the clocks
   may be synchronized within a few microseconds.
   
	 Note that client and server synchronization is not needed for either RTT or
	 IPDV (even send and receive IPDV) values to be correct. RTT is measured with
	 client times only, and since IPDV is measuring differences between successive
	 packets, it's not affected by time synchronization.

4) Why is the receive rate 0 when a single packet is sent?

   Receive rate is measured from the time the first packet is received to the time
   the last packet is received. For a single packet, those times are the same.

5) Why does a test with a one second duration and 200ms interval run for around
   800ms and not one second?

   The test duration is exclusive, meaning requests will not be sent exactly at
   or after the test duration has elapsed. In this case, the interval is 200ms, and
   the fifth and final request is sent at around 800ms from the start of the test.
   The test ends when all replies have been received from the server, so it may
	 end shortly after 800ms. If there are any outstanding packets, the wait time
	 is observed, which by default is a multiple of the maximum RTT.

6) Why is IPDV not reported when only one packet is received?

   [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation) is the
   difference in delay between successfully returned replies, so at least two
   reply packets are required to make this calculation.

7) Why does wait fall back to fixed duration when duration is less than RTT?

   If a full RTT has not elapsed, there is no way to know how long an
	 appropriate wait time would be, so the wait falls back to a default fixed
	 time (default is 4 seconds, same as ping).

8) Why can't the client connect to the server, and I either see `[Drop]
   [UnknownParam] unknown negotiation param (0x8 = 0)` on the server, or a strange
   message on the client like `[InvalidServerRestriction] server tried to reduce
   interval to < 1s, from 1s to 92ns`?

   You're using a 0.1 development version of the server with a newer client.
   Make sure both client and server are up to date. Going forward, the protocol
   is versioned (independently from IRTT in general), and is checked when the
   client connects to the server. For now, the protocol versions must match
   exactly.

9) Why don't you include median values for send call time, timer error and
   server processing time?

   Those values aren't stored for each round trip, and it's difficult to do a
   running calculation of the median, although
   [this method](https://rhettinger.wordpress.com/2010/02/06/lost-knowledge/) of
   using skip lists appears to have promise. It's a possibility for the future,
   but so far it isn't a high priority. If it is for you, file an
   [Issue](https://github.com/heistp/irtt/issues).

10) I see you use MD5 for the HMAC. Isn't that insecure?

    MD5 should not have practical vulnerabilities when used in a message authenticate
    code. See
    [this page](https://en.wikipedia.org/wiki/Hash-based_message_authentication_code#Security)
    for more info.

11) Are there any plans for translation to other languages?

    While some parts of the API were designed to keep i18n possible, there is no
    support for i18n built in to the Go standard libraries. It should be possible,
    but could be a challenge, and is not something I'm likely to undertake myself.

12) Why do I get `Error: failed to allocate results buffer for X round trips
    (runtime error: makeslice: cap out of range)`?

    Your test interval and duration probably require a results buffer that's
    larger than Go can allocate on your platform. Lower either your test
    interval or duration. See the following additional documentation for
    reference: [In-memory results storage](#in-memory-results-storage),
    `maxSliceCap` in [slice.go](https://golang.org/src/runtime/slice.go) and
    `_MaxMem` in [malloc.go](https://golang.org/src/runtime/malloc.go).

13) Why is little endian byte order used in the packet format?

    As for Google's [protobufs](https://github.com/google/protobuf), this was
    chosen because the vast majority of modern processors use little-endian byte
    order. In the future, packet manipulation may be optimized for little-endian
    architecutures by doing conversions with Go's
    [unsafe](https://golang.org/pkg/unsafe/) package, but so far this
    optimization has not been shown to be necessary.

14) Why does `irtt client` use `-l` for packet length instead of following ping
    and using `-s` for size?

    I felt it more appropriate to follow the
    [RFC 768](https://tools.ietf.org/html/rfc768) term _length_ for UDP packets,
    since IRTT uses UDP.

15) Why is the virt size (vsz) memory usage for the server so high in Linux?

    This has to do with the way Go allocates memory, but should not cause a
    problem. See [this
    article](https://deferpanic.com/blog/understanding-golang-memory-usage/) for
    more information. File an Issue if your resident usage (rss/res) is high or
    you feel that memory consumption is somehow a problem.

## Changes

See [CHANGES.md](CHANGES.md).

## Roadmap

### v1.0.0

_Planned for v1.0.0..._

- Document irtt.openrc and public server
- Solidify TimeSource, Time and new Windows timer support:
  - Add --timesrc to client and server
  - Fall back to Go functions as necessary for older Windows versions
  - Make sure all calls to TimeSource.Now pass in only needed clocks
  - Find a better way to log warnings than fmt.Fprintf(os.Stderr) in timesrc_win.go
  - Rename Time.Mono to Monotonic, or others from Monotonic to Mono for
    consistency
  - Document 100ns resolution for Windows
- Improve diagnostic commands:
  - Change bench command to output in columns
  - Rename sleep command to timer and add --timesrc, --sleep, --timer and --tcomp
  - Rename timer command to resolution and add --timesrc
  - Rename clock command to drift and add --timesrc
- Improve client output flexibility:
  - Allow specifying a format string for text output with optional units for times
  - Add format abbreviations for CSV, space delimited, etc.
  - Add a subcommand to the CLI to convert JSON to CSV
  - Add a way to disable per-packet results in JSON
  - Add a way to keep out "internal" info from JSON, like IP and hostname, and a
	  subcommand to strip these out after the JSON is created
  - Add more info on outliers and possibly a textual histogram
- Change some defaults:
  - Use receive timestamp, as dual timestamps rarely improve accuracy
  - Use wall clock timestamps, as send and receive IPDV are still close
- Add DSCP text values and return an error when ECN bits are passed to --dscp
- Refactor packet manipulation to improve readability, prevent multiple validations
  and support unit tests
- Improve open/close process:
  - Do Happy Eyeballs (RFC 8305) to better handle multiple address families and
    addresses
  - Make timeout support automatic exponential backoff, like 4x15s
  - Repeat close packets until acknowledgement, like open
  - Include final stats in the close acknowledgement from the server
- Improve robustness and security of public servers:
	- Add bitrate limiting
	- Limit open requests rate and coordinate with sconn cleanup
  - Add separate, shorter timeout for open
  - Specify close timeout as param from client, which may be restricted
  - Make connref mechanism robust to listener failure
	- Add per-IP limiting
  - Add a more secure way than cmdline flag to specify --hmac
- Add [ping-pair](https://www.microsoft.com/en-us/research/wp-content/uploads/2017/09/PingPair-CoNEXT2017.pdf) functionality
- Stabilize API:
  - Always return instance of irtt.Error? If so, look at exitOnError.
  - Use error code (if available) as exit code
- Improve induced latency and jitter:
  - Use Go profiling, scheduler tracing, strace and sar
  - Do more thorough tests of `chrt -r 99`, `--thread` and `--gc`
  - Find or file issue with Go team over scheduler performance, if needed
  - Prototype doing thread scheduling or socket i/o for Linux in C
- Show actual size of header in text and json, and add calculation to doc
- Measure and document local differences between ping and irtt response times
- Create a backports version for Debian stable

### Inbox

_Collection area for the future..._

- Add different server authentication modes:
	- none (no conn token in header, for minimum packet sizes during local use)
	- token (what we have today, 64-bit token in header)
	- nacl-hmac (hmac key negotiated with public/private key encryption)
- Implement graceful server shutdown with sconn close
- Implement zero-downtime restarts
- Add a Scheduler interface to allow non-isochronous send schedules and variable
  packet lengths
  - Find some way to determine packet interval and length distributions for
    captured traffic
  - Determine if asymmetric send schedules (between client and server) required
- Add an overhead test mode to compare ping vs irtt
- Add client flag to skip sleep and catch up after timer misses
- Add seqno to the Max and maybe Min columns in the text output
- Prototype TCP throughput test and compare straight Go vs iperf/netperf
- Support a range of server ports to improve concurrency and maybe defeat
  latency "slotting" on multi-queue interfaces
- Add more unit tests
- Add support for load balanced conns (multiple source addresses for same conn)
- Use unsafe package to speed up packet buffer manipulation
- Add encryption
- Add estimate for HMAC calculation time and correct send timestamp by this time
- Implement web interface for client and server
- Set DSCP per-packet, at least for IPv6
- Add NAT hole punching
- Use a larger, internal received window on the server to increase up/down loss accuracy
- Allow specifying two out of three of interval, bitrate and packet size
- Calculate per-packet arrival order during results generation using timestamps
- Make it possible to add custom per-round-trip statistics programmatically
- Allow Server to listen on multiple IPs for a hostname
- Prompt to write JSON file on cancellation
- Open questions:
  - What do I do for IPDV when there are out of order packets?
  - Does exposing both monotonic and wall clock values, as well as dual
    timestamps, open the server to any timing attacks?
  - Is there any way to make the server concurrent without inducing latency?
  - Should I request a reserved IANA port?

## Thanks

Many thanks to both Toke Høiland-Jørgensen and Dave Täht from the
[Bufferbloat project](https://www.bufferbloat.net/) for their valuable
advice. Any problems in design or implementation are entirely my own.
