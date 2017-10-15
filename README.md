# IRTT (Isochronous Round-Trip Tester)

IRTT measures round-trip time and other metrics using UDP packets sent on a
fixed period, and produces both human and machine parseable output.

## Start Here!

IRTT is still under active development, and as such has not yet met all of its
[goals](#goals), and its goals may still even change. In particular:

- non-isochronous send schedules are still under consideration, which would be a
	significant design change
- it is not yet capable of distinguishing between upstream and downstream packet
	loss
- there's more work to do for public server security
- the JSON output format, packet format and API are all not finalized
- it is only available in source form
- it has only had basic testing on a couple of platforms

That said, it is working and can be used today. I would appreciate any feedback,
which you can send under Issues. However, it could be useful to first review the
[TODO and Roadmap](#todo-and-roadmap) section of the documentation before
submitting a new bug or feature request.

## Table of Contents

1. [Introduction](#introduction)
	1. [Motivation and Goals](#motivation-and-goals)
	2. [Features](#features)
	3. [Limitations](#limitations)
2. [Getting Started](#getting-started)
	1. [Installation](#installation)
	2. [Quick Start](#quick-start)
3. [Running IRTT](#running-irtt)
	1. [Client Options](#client-options)
	2. [Server Options](#server-options)
	3. [Tests and Benchmarks](#tests-and-benchmarks)
4. [Results](#results)
	1. [Test Output](#test-output)
	2. [JSON Format](#json-format)
5. [Internals](#internals)
	1. [Packet Format](#packet-format)
	2. [Security](#security)
6. [Frequently Asked Questions](#frequently-asked-questions)
7. [TODO and Roadmap](#todo-and-roadmap)
8. [Thanks](#thanks)

## Introduction

### Motivation and Goals

Latency is an under-appreciated metric in network and application performance.
There is a certain hard to quantify but visceral *"latency stress"* that comes
from waiting in expectation after a web page click, straining through a delayed
and garbled VoIP conversation, or losing at your favorite online game (unless
you'd like "lag" as an excuse). I think that relieving this stress for others
may be what drives those who work on latency related projects.

The [Bufferbloat](https://www.bufferbloat.net/projects/) and related projects
aim to reduce "chaotic and laggy network performance", which is what in my
opinion makes this project so worthwhile to anyone who uses the Internet and
values their time and sanity.

The original motivation for IRTT was to improve the latency and packet loss
measurements for the excellent [Flent](https://flent.org) tool, which was
developed by and for the Bufferbloat project. However, IRTT could be useful as a
general purpose tool as well. The goals of this project are to:

- Accurately measure latency and other relevant metrics of network behavior
- Produce statistics via both human and machine parseable output
- Provide for reasonably secure use on both public and private servers
- Support small packet sizes for [VoIP traffic](https://www.cisco.com/c/en/us/support/docs/voice/voice-quality/7934-bwidth-consume.html) simulation
- Support relevant socket options, including DSCP
- Use a single UDP port for deployment simplicity
- Provide an API for embedding and extensibility

### Features:

* Measurement of:
	- [RTT (round-trip time)](https://en.wikipedia.org/wiki/Round-trip_delay_time)
	- [OWD (one-way delay)](https://en.wikipedia.org/wiki/End-to-end_delay), given
		external clock synchronization
	- [IPDV (instantaneous packet delay variation)](https://en.wikipedia.org/wiki/Packet_delay_variation)
	- [Packet loss](https://en.wikipedia.org/wiki/Packet_loss), with upstream and downstream differentiation
	- [Out-of-order](https://en.wikipedia.org/wiki/Out-of-order_delivery) and
		duplicate packets
	- [Bitrate](https://en.wikipedia.org/wiki/Bit_rate)
	- Timer error, send call time and server processing time
* Statistics: min, max, mean, median (for most quantities) and standard deviation
* Nanosecond time precision (where available), and robustness in the face of
	clock drift and NTP corrections through the use of both the wall and monotonic
	clocks
* Binary protocol with negotiated format for test packet lengths down to 16 bytes (without timestamps)
* HMAC support for private servers, preventing unauthorized discovery and use
* Support for a wide range of Go supported [platforms](https://github.com/golang/go/wiki/MinimumRequirements)
* Timer compensation to improve sleep send schedule accuracy
* Support for IPv4 and IPv6
* Public server protections, including:
	* Three-way handshake with returned 64-bit connection token, preventing reply
		redirection to spoofed source addresses
	* Limits on maximum test duration, minimum interval and maximum packet length,
		both advertised in the negotiation and enforced with hard limits to protect
		against rogue clients
	* Packet payload filling to prevent relaying of arbitrary traffic
* Output to JSON

### Limitations

> "It is the limitations of software that give it life." *-Me, justifying my limitations*

#### Isochronous (fixed period) send schedule

Currently, IRTT only sends packets on a fixed period. I am still considering
allowing packets to be sent on varying schedules so that more types of traffic
could be simulated, but accepting this limitation offers some benefits as well:

- It's easy to implement
- It's easy to calculate how much data will be sent in a given time
- It simplifies timer error compensation

Also, isochronous packets are commonly seen in VoIP, games and streaming media,
so it already simulates an array of common types of traffic.

#### Fixed packet lengths for a given test

Packet lengths are fixed for the duration of the test. While this may not be an
accurate simulation of some types of traffic, it means that IPDV measurements
are accurate, where they wouldn't be in any other case.

#### Stateful protocol

While there are numerous benefits to stateless protocols, including simplified
server design, horizontal scalabity, and easily implemented zero-downtime
restarts, I ultimately decided that a stateless protocol brings most of its
advantages to the data center, and in this case, a stateful protocol provides
important benefits to the user, including:

- Smaller packet sizes (a design goal) as context does not need to be included in every request
- More accurate measurement of upstream vs downstream packet loss (this gets worse in a stateless protocol as RTT approaches the test duration, complicating interplanetary tests!)
- More accurate rate and test duration limiting on the server

#### In-memory results storage

Results for each round-trip are stored in memory as the test is being run. Each
result takes up to 64 bytes in memory (8 64-bit timestamps, explained later), so
this limits the effective duration of the test, especially at very small send
intervals. However, the advantages are:

- It's easier to perform statistical analysis (like calculation of the median)
	on fixed arrays than on running data values
- We don't need to either send client timestamps to the server, or maintain a
	local running window of sent packet info, because they're all in memory, no
	matter when server replies come back
- Not accessing the disk during the test to write test output prevents
	inadvertently affecting the results
- It simplifies the API

As a consequence of storing results in memory, packet sequence numbers are fixed
at 32-bits. If all 2^32 sequence numbers were used, the results would require
almost 275 Gb of RAM to record while the test is running, which is not likely
to be a number reached any time soon. That is why 64-bit sequence numbers are
unnecessary at this time.

#### Use of Go

IRTT is written in Go. While that carries with it the disadvantage of a larger
executable size than with C, for example, and somewhat slower execution speed
(although [not that much slower](https://benchmarksgame.alioth.debian.org/u64q/compare.php?lang=go&lang2=gcc), depending on the workload), Go still has benefits that are useful for this application:

- Go's target is network and server applications, with a focus on simplicity,
	reliability and efficiency, which is appropriate for this project
- Memory footprint tends to be significantly lower than with some interpreted
	languages
- It's easy to support a broad array of hardware and OS combinations

## Getting Started

### Installation

Currently, IRTT is only available in source form. To build it, you must:

- [Install Go](https://golang.org/dl/)
- Install IRTT by running `go install github.com/peteheist/irtt/cmd/irtt`

If you're not familiar with the `go` tool, the build.sh script may be used as an
example of how to cross-compile to different platforms or minimize the binary
size. For example, `build.sh min linux-amd64` would compile a minimized binary
for Linux on AMD64. See build.sh for more info and a "source-documented" list of
platforms that the script supports.

For more on cross-compilation, see [this
page](http://golang.org/doc/install/source#environment) for a full list of valid GOOS
GOARCH combinations.

### Quick Start

After installing IRTT, start a server:

```
% irtt server
IRTT server starting...
[ListenerStart] starting IPv6 listener on [::]:2112
[ListenerStart] starting IPv4 listener on 0.0.0.0:2112
```

While that's running, run a client, which will perform a default test with
duration 1 second and interval 200ms:

```
% irtt client localhost
[Connecting] connecting to localhost
[Connected] connected to 127.0.0.1:2112
[WaitForPackets] waiting 1.13ms for final packets

                        Min    Mean  Median     Max  Stddev
                        ---    ----  ------     ---  ------
                RTT  62.6µs   305µs   368µs   376µs   136µs
         send delay  32.9µs   174µs   211µs   231µs  81.7µs
      receive delay  29.7µs   131µs   155µs   170µs  57.4µs
                                                           
      IPDV (jitter)   706ns  89.6µs  26.3µs   305µs   144µs
          send IPDV  10.7µs  69.9µs  45.4µs   178µs  74.3µs
       receive IPDV    10µs  43.8µs  19.1µs   127µs  55.8µs
                                                           
     send call time  14.6µs  56.8µs          75.3µs  24.8µs
        timer error  87.5µs  1.37ms          5.04ms  2.45ms
  server proc. time  4.55µs  16.5µs          25.1µs  7.66µs

                duration: 802.1ms (wait 1.13ms)
   packets sent/received: 5/5 (0.00% loss)
     bytes sent/received: 240/240
       send/receive rate: 2.4 Kbps / 2.4 Kbps
           packet length: 48 bytes
             timer stats: 0/5 (0.00%) missed, 0.69% error
```

## Running IRTT

### Client Options

### Server Options

### Tests and Benchmarks

## Results

### Test Output

### JSON Format

*TBD ASAP*

## Internals

### Packet Format

### Security

## Frequently Asked Questions

1) Why not just use ping?

	 Ping may be the preferred tool when measuring minimum latency, or for other
	 reasons. IRTT's reported latency is likely to be somewhat higher and more
	 variable than the results reported by ping, due to task scheduling
	 variability and more to do in the stack and program. That said, there are
	 advantages that IRTT has over ping when minimum RTT is not what you're
	 measuring:

	 - Some device vendors prioritize ICMP, so ping may not be an accurate measure
		 of user-perceived latency.
	 - In addition to round-trip time, IRTT also measures OWD, IPDV and upstream
	   vs downstream packet loss.
	 - IRTT can use HMACs to protect private servers from unauthorized discovery
		 and use.
	 - IRTT has a three-way handshake to prevent test traffic redirection from
		 spoofed source IPs.
	 - IRTT can fill the payload with random data.

2) Why is the receive rate 0 when a single packet is sent?

   Receive rate is measured from the time the first packet is received to the time
the last packet is received. For a single packet, those times are the same.

3) Why does the default test with a one second duration run for around 800ms?

   The test duration is exclusive, meaning requests will not be sent exactly at
or after the test duration has elapsed. In this case, the interval is 200ms, and
the fifth and final request is sent at around 800ms from the start of the test.
The test ends when the final reply is received from the server. If all packets
have been received, there is no waiting period for final packets.

4) Why does wait fall back to fixed duration when duration is less than RTT?

   If a full RTT has not elapsed, there is no way to know how long an
	 appropriate wait time would be, so the wait falls back to a default fixed
	 time (default is 4 seconds).

5) Why don't you include median values for send call time, timer error and
server processing time?

   Those values aren't stored for each round trip, and it's difficult to do a
	 running calculation of the median, although
	 [this method](https://rhettinger.wordpress.com/2010/02/06/lost-knowledge/) of
	 using skip lists appears to have promise. I may consider it for the future,
	 but so far it isn't a high priority.

6) I see you use MD5 for the HMAC. Isn't that insecure?

   MD5 should not have practical vulnerabilities when used in a message authenticate
code. See [this page](https://en.wikipedia.org/wiki/Hash-based_message_authentication_code#Security) for more info.

7) Will you add unit tests?

   Maybe some. Although I appreciate unit tests, I probably have a dissenting
view of them. If there is a bug, it means one of two things:

	1. There is a typo or very simple logical error
	2. There is a design problem that makes it hard to determine what the code will do, making whack-a-mole like logical errors inevitable (there should be no such problems)

	 So when there is a bug, I like to not only fix it, but if I have time, fix
	 the reason it exists in the first place, so that we don't see bugs in that
	 second category. That said, there are places where IRTT falls short of this
	 (packet manipulation) and places where it could use some unit tests. I just
	 don't believe that 100% coverage is necessary.

8) Are there any plans for translation to other languages?

   While some parts of the API were designed to keep i18n possible, there is no
support for it built in to the Go standard libraries. It should be possible, but
could be a challenge, and is not something I'm likely to undertake.

## TODO and Roadmap

Definitely (in order of priority)...

- Document JSON format
- Implement server received packets feedback (to distinguish between upstream
	and downstream packet loss)
- Allow specifying two out of three of interval, bitrate and packet size to the
	client
- Refactor packet manipulation to improve maintainability and prevent multiple
	validations
- Improve robustness and security of public servers:
	- Add bitrate limiting
	- Improve server close by repeating close packets up to some limit
	- Limit open requests to prevent the equivalent of a "syn flood"
	- Add per-IP limiting
- Add different server authentication modes:
	- none (no conn token in header, for local use)
	- token (what we have today, 64-bit token in header)
	- nacl-hmac (hmac key negotiated with public/private key encryption)
- Add a subcommand to the CLI to convert JSON to CSV
- Show IPDV in continuous (-v) output
- Add a way to keep out "internal" info from JSON, like IP, hostname, and a
	subcommand to strip these details after the JSON is created
- Add more relevant statistics, including more info on outliers and a textual
	histogram(?)
- Add ability for client to request random fill from server
- Allow Client Dial to try multiple IPs when a hostname is given
- Refactor events. I was trying to make something internationalizable and
	filterable as opposed to just writing to a log, but I'm not satisfied yet.

Possibly...

- Allow non-isochronous send schedules
- Write an irtt.Timer implementation that uses Linux timerfd
- Use pflag options: https://github.com/spf13/pflag
- Implement graceful server shutdown
- Implement zero downtime restart
- Add unit tests
- Add supported for load balanced connections (packets for same connection that
	come from multiple addresses)
- Use unsafe package to increase packet buffer modification and comparison performance
- Add compression for second timestamp monotonic value as diff from first
- Add encryption
- Add estimate for HMAC calculation time and correct send timestamp by this time
- Implement web interface for client and server
- Add NAT hole punching
- Implement median for timer error, send call time and server processing time

Open questions...

- Should I request a reserved IANA port?
- Does exposing both monotonic and wall clock values open the server to any
	timing attacks?
- What do I do for IPDV and out of order packets?

## Thanks

Many thanks to both Toke Høiland-Jørgensen and Dave Täht for their valuable
advice on this project. Any problems in design or implementation are entirely
my own.
