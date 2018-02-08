# IRTT (Isochronous Round-Trip Tester)

IRTT measures round-trip time, one-way delay and other metrics using UDP
packets sent on a fixed period, and produces both user and machine parseable
output.

IRTT has reached version 0.9, and is usable today, but needs more work until
version 1.0 can be released. I would appreciate any feedback, which you can
send under Issues. However, it could be useful to first review the
[TODO and Roadmap](#todo-and-roadmap) section of the documentation before
submitting a new bug or feature request.

## Table of Contents

1. [Motivation](#motivation)
2. [Goals](#goals)
3. [Features](#features)
4. [Limitations](#limitations)
5. [Installation](#installation)
6. [Documentation](#documentation)
7. [Frequently Asked Questions](#frequently-asked-questions)
8. [Releases](#releases)
9. [TODO and Roadmap](#todo-and-roadmap)
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
- Nanosecond time precision (where available), and robustness in the face of
	clock drift and NTP corrections through the use of both the wall and monotonic
	clocks
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

## Limitations

See the
[LIMITATIONS](http://htmlpreview.github.io/?https://github.com/peteheist/irtt/blob/master/doc/irtt.html#limitations)
section of the irtt(1) man page.

## Installation

To install IRTT manually or build from source, you must:

1. [Install Go](https://golang.org/dl/)
2. Install irtt: `go get -u github.com/peteheist/irtt/cmd/irtt`
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
then from the `github.com/peteheist/irtt` directory, do:
1. `git checkout branch`
2. `go get ./...`
3. `go install ./cmd/irtt` or `./build.sh` and move resulting `irtt` executable
   to install location

## Documentation

After installing IRTT, see the
[irtt(1)](http://htmlpreview.github.io/?https://github.com/peteheist/irtt/blob/master/doc/irtt.html),
[irtt-client(1)](http://htmlpreview.github.io/?https://github.com/peteheist/irtt/blob/master/doc/irtt-client.html)
and [irtt-server(1)](http://htmlpreview.github.io/?https://github.com/peteheist/irtt/blob/master/doc/irtt-server.html)
man pages.

To get started quickly, see the
[EXAMPLE](http://htmlpreview.github.io/?https://github.com/peteheist/irtt/blob/master/doc/irtt.html#example)
sections of each man page for common client and server usage.

## Frequently Asked Questions

1) Why not just use ping?

   Ping may be the preferred tool when measuring minimum latency, or for other
   reasons. IRTT's reported mean RTT is likely to be around 0.1-0.4 ms higher
   and a bit more variable than the results reported by ping, due to the
   overhead of entering userspace, together with Go's system call overhead and
   scheduling variability. That said, this overhead should be negligible at most
   Internet RTTs, and there are advantages that IRTT has over ping when minimum
   RTT is not what you're measuring:

	 - In addition to round-trip time, IRTT also measures OWD, IPDV and upstream
	   vs downstream packet loss.
	 - Some device vendors prioritize ICMP, so ping may not be an accurate measure
		 of user-perceived latency.
	 - IRTT can use HMACs to protect private servers from unauthorized discovery
		 and use.
	 - IRTT has a three-way handshake to prevent test traffic redirection from
		 spoofed source IPs.
	 - IRTT can fill the payload (if included) with random data.

2) Why does `irtt client` use `-l` for packet length instead of following ping
   and using `-s` for size?

   I felt it more appropriate to follow the
   [RFC 768](https://tools.ietf.org/html/rfc768) term _length_ for UDP packets,
   since IRTT uses UDP.

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

8) Why can't the client connect to the server, and instead I get
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
      IP address, and return packets are not routing properly, which can happen in
      some network configurations. Try running the server with the `--set-src-ip`
      flag, which sets the source address on all reply packets from listeners
      on unspecified IP addresses. This is not done by default in order to avoid
      the additional per-packet heap allocations required by the
      `golang.org/x/net` packages.
   6) You're using a 0.1 development version of the server with a newer client,
      in which case you'll also see `[Drop] [UnknownParam] unknown negotiation
      param (0x8 = 0)` in the server logs. Make sure both client and server are up
      to date. Going forward, the protocol is versioned (independently from IRTT in
      general), and is checked when the client connects to the server. For now, the
      protocol versions must match exactly.

9) Why don't you include median values for send call time, timer error and
   server processing time?

   Those values aren't stored for each round trip, and it's difficult to do a
	 running calculation of the median, although
	 [this method](https://rhettinger.wordpress.com/2010/02/06/lost-knowledge/) of
	 using skip lists appears to have promise. It's a possibility for the future,
	 but so far it isn't a high priority. If it is for you, file an
   [Issue](https://github.com/peteheist/irtt/issues).

10) I see you use MD5 for the HMAC. Isn't that insecure?

    MD5 should not have practical vulnerabilities when used in a message authenticate
    code. See [this page](https://en.wikipedia.org/wiki/Hash-based_message_authentication_code#Security)
    for more info.

11) Will you add unit tests?

    Maybe some. I feel that the most important thing for a project of this size
    is that the design is clear enough that bugs are next to impossible. IRTT
    is not there yet though, particularly when it comes to packet manipulation.

12) Are there any plans for translation to other languages?

    While some parts of the API were designed to keep i18n possible, there is no
    support for i18n built in to the Go standard libraries. It should be possible,
    but could be a challenge, and is not something I'm likely to undertake myself.

13) Why do I get `Error: failed to allocate results buffer for X round trips
   (runtime error: makeslice: cap out of range)`?

    Your test interval and duration probably require a results buffer that's
    larger than Go can allocate on your platform. Lower either your test
    interval or duration. See the following additional documentation for
    reference: [In-memory results storage](#in-memory-results-storage),
    `maxSliceCap` in [slice.go](https://golang.org/src/runtime/slice.go) and
    `_MaxMem` in [malloc.go](https://golang.org/src/runtime/malloc.go).

14) Why is little endian byte order used in the packet format?

    As for Google's [protobufs](https://github.com/google/protobuf), this was
    chosen because the vast majority of modern processors use little-endian byte
    order. In the future, packet manipulation may be optimized for little-endian
    architecutures by doing conversions with Go's
    [unsafe](https://golang.org/pkg/unsafe/) package, but so far this
    optimization has not been shown to be necessary.

15) Why is the virt size (vsz) memory usage so high in Linux?

    This has to do with the way Go allocates memory. See
    [this article](https://deferpanic.com/blog/understanding-golang-memory-usage/)
    for more information. File an Issue if your resident usage (rss/res) is high
    or you feel that memory consumption is somehow a problem.

## Releases

### Version 0.9

Version 0.9 is the first tagged release of IRTT. Following are the changes
from the untagged 0.1 development version:

- Command line option changes:
  - Due to adoption of the [pflag](https://github.com/ogier/pflag) package, all long
    options now start with -- and must use = with values (e.g. `--fill=rand`).
    After the subcommand, flags and arguments may now appear in any order.
  - `irtt client` changes:
    - `-rs` is renamed to `--stats`
    - `-strictparams` is removed and is now the default. `--loose` may be used
      instead to accept and use server restricted parameters, with a warning.
    - `-ts` is renamed to `--tstamp`
    - `-qq` is renamed to `-Q`
    - `-fillall` is removed and is now the default. `--fill-one` may be used as
      a small optimization, but should rarely be needed.
  - `irtt server` changes:
    - `-nodscp` is renamed to `--no-dscp`
    - `-setsrcip` is renamed to `--set-src-ip`
- The communication protocol has changed, so clients and servers must both be
  updated. The handshake now includes a protocol version, which may change
  independently of the release version. For now, the protocol version between
  client and server must match exactly or the client will refuse to connect.
- Server fills are now supported, and may be restricted on the server. See
  `--sfill` for the client and `--allow-fills` on the server. As an example, one
  can do `irtt client --fill=rand --sfill=rand -l 172 server` for random
  payloads in both directions. The server default is `--allow-fills=rand` so
  that arbitrary data cannot be relayed between two hosts. `server_fill` now
  appears under `params` in the JSON.
- Version information has been added to the JSON output.
- The default server minimum interval is now `10ms`.
- The default client duration has been changed from `1h` to `1m`.
- Some author info was changed in the commit history, so the rewritten history
  must be fetched in all forks and any changes rebased.

## TODO and Roadmap

### TODO v1.0

- Refactor packet manipulation to improve readability and prevent multiple validations
- Improve open/close process:
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
- Write a SmokePing probe

### Roadmap

_Planned for the future..._

- Add a Scheduler interface to allow non-isochronous send schedules and variable
  packet lengths
  - Find some way to determine packet interval and length distributions for
    captured traffic
  - Determine if asymmetric send schedules (between client and server) required
- Improve induced latency and jitter:
  - Use Go profiling, scheduler tracing, strace and sar
  - Do more thorough tests of `chrt -r 99`, `--thread` and `--gc`
  - Find or file issue with Go team over scheduler performance
  - Prototype doing thread scheduling or socket i/o for Linux in C
- Add different server authentication modes:
	- none (no conn token in header, for minimum packet sizes during local use)
	- token (what we have today, 64-bit token in header)
	- nacl-hmac (hmac key negotiated with public/private key encryption)
- Implement graceful server shutdown with sconn close
- Implement zero-downtime restarts

### Inbox

_Collection area for undefined or uncertain stuff..._

- Add client flag to skip sleep and catch up after timer misses
- Always return instance of irtt.Error? If so, look at exitOnError.
- Find better model for concurrency (one goroutine per sconn induces latency)
- Use error code (if available) as exit code
- Add seqno to the Max and maybe Min columns in the text output
- Prototype TCP throughput test and compare straight Go vs iperf/netperf
- Add a subcommand to the CLI to convert JSON to CSV
- Support a range of server ports to improve concurrency and maybe defeat
  latency "slotting" on multi-queue interfaces
- Prompt to write JSON file on cancellation
- Add unit tests
- Add support for load balanced conns (multiple source addresses for same conn)
- Use unsafe package to speed up packet buffer manipulation
- Add encryption
- Add estimate for HMAC calculation time and correct send timestamp by this time
- Implement web interface for client and server
- Set DSCP per-packet, at least for IPv6
- Add NAT hole punching
- Add a flag to disable per-packet results
- Use a larger, internal received window on the server to increase up/down loss accuracy
- Implement median calculation for timer error, send call time and server processing time
- Allow specifying two out of three of interval, bitrate and packet size
- Calculate per-packet arrival order during results generation using timestamps
- Add OWD compensation at results generation stage for shifting mean value to 0
  to improve readability for clocks that are badly out of sync
- Add a way to keep out "internal" info from JSON, like IP and hostname, and a
	subcommand to strip these out after the JSON is created
- Make it possible to add custom per-round-trip statistics programmatically
- Add more info on outliers and possibly a textual histogram
- Allow Client Dial to try multiple IPs when a hostname is given
- Allow Server listen to listen on multiple IPs for a hostname
- What do I do for IPDV when there are out of order packets?
- Does exposing both monotonic and wall clock values open the server to any
	timing attacks?
- Should I request a reserved IANA port?

## Thanks

Many thanks to both Toke Høiland-Jørgensen and Dave Täht from the
[Bufferbloat project](https://www.bufferbloat.net/) for their valuable
advice. Any problems in design or implementation are entirely my own.
