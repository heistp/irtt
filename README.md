# IRTT (Isochronous Round-Trip Tester)

IRTT measures round-trip time and other metrics using UDP packets sent on a
fixed period, and produces both human and machine parseable output.

## Start Here!

IRTT is still under development, and as such has not yet met all of its
[goals](#goals). In particular:

- it can't distinguish between upstream and downstream packet loss
- there's more work to do for public server security
- the JSON output format, packet format and API are all not finalized
- it is only available in source form
- it has only had very basic testing on a couple of platforms

Also, I'm still considering removing the isochronous send schedule limitation,
although there are implications to doing that that I'm still sorting through.

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
you like "lag" as an excuse). I think that relieving this stress for others
may be what drives those who work on reducing latency.

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
	- [IPDV (instantaneous packet delay variation)](https://en.wikipedia.org/wiki/Packet_delay_variation), usually referred to as jitter
	- [Packet loss](https://en.wikipedia.org/wiki/Packet_loss), with upstream and downstream differentiation
	- [Out-of-order](https://en.wikipedia.org/wiki/Out-of-order_delivery)
		(measured by late packets metric) and duplicate packets
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

There are numerous benefits to stateless protocols, particularly for developers
and data centers, including simplified server design, horizontal scalabity, and
easily implemented zero-downtime restarts. However, in this case, a stateful
protocol provides important benefits to the user, including:

- Smaller packet sizes (a design goal) as context does not need to be included
  in every request
- More accurate measurement of upstream vs downstream packet loss (this gets
  worse in a stateless protocol as RTT approaches the test duration,
  complicating interplanetary tests!)
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

While that's running, run a client. If no options are supplied, it will send
a request once per second, like ping, but here we use an interval of 10ms
and a test duration of 1m, with a payload of 160 bytes, to roughly simulate
a G.711 VoIP conversation:

```
% irtt client -i 20ms -l 160 -d 1m -q 192.168.100.10
[Connecting] connecting to 192.168.100.10
[Connected] connected to 192.168.100.10:2112

                        Min     Mean   Median      Max  Stddev
                        ---     ----   ------      ---  ------
                RTT  12.2ms  24.87ms  23.21ms  115.9ms   9.1ms
         send delay  5.89ms  16.46ms  15.08ms  97.29ms  8.04ms
      receive delay  5.64ms   8.41ms   7.55ms  39.26ms  2.95ms
                                                              
      IPDV (jitter)  11.6µs   6.69ms   5.03ms  83.23ms  6.57ms
          send IPDV  3.74µs   6.24ms   4.61ms  84.81ms  6.22ms
       receive IPDV  1.05µs   1.96ms   1.06ms  31.99ms  2.88ms
                                                              
     send call time  56.2µs   79.4µs           11.88ms   227µs
        timer error     5ns   64.3µs           11.68ms   579µs
  server proc. time  18.9µs     21µs             262µs  9.62µs

                duration: 1m0s (wait 347.8ms)
   packets sent/received: 2986/2966 (0.67% loss)
     bytes sent/received: 477760/474560
       send/receive rate: 63.7 Kbps / 63.3 Kbps
           packet length: 160 bytes
             timer stats: 14/3000 (0.47%) missed, 0.32% error
```

## Running IRTT

### Client Options

### Server Options

### Tests and Benchmarks

## Results

### Test Output

### JSON Format

IRTT's JSON output format consists of four top-level objects. These are documented
through the examples below. All attributes are present unless otherwise noted in
_italics._

1. [system_info](#system_info)
2. [config](#config)
3. [stats](#stats)
4. [round_trips](#round_trips)

#### system_info

a few basic pieces of system information

```
"system_info": {
    "os": "darwin",
    "cpus": 8,
    "go_version": "go1.9.1",
    "hostname": "tron.local"
},
```

- `os` the Operating System from Go's `runtime.GOOS`
- `cpus` the number of CPUs reported by Go's `runtime.NumCPU()`, which reflects
  the number of logical rather than physical CPUs. In the example below, the
	number 8 is reported for a Core i7 (quad core) with hyperthreading (2 threads
	per core).
- `go_version` the version of Go the executable was built with
- `hostname` the local hostname

#### config

the configuration used for the test

```
"config": {
    "local_address": "127.0.0.1:51203",
    "remote_address": "127.0.0.1:2112",
    "params": {
        "duration": 600000000,
        "interval": 200000000,
        "length": 48,
        "stamp_at": "both",
        "clock": "both",
        "dscp": 0
    },
    "ip_version": "IPv4",
    "df": 0,
    "ttl": 0,
    "timer": "comp",
    "waiter": "3x4s",
    "filler": "none",
    "fill_all": false,
    "lock_os_thread": false,
    "supplied": {
        "local_address": ":0",
        "remote_address": "localhost",
        "params": {
            "duration": 600000000,
            "interval": 200000000,
            "length": 0,
            "stamp_at": "both",
            "clock": "both",
            "dscp": 0
        },
        "ip_version": "IPv4+6",
        "df": 0,
        "ttl": 0,
        "timer": "comp",
        "waiter": "3x4s",
        "filler": "none",
        "fill_all": false,
        "lock_os_thread": false
    }
},
```

- `local_address` the local address (IP:port) for the client
- `remote_address` the remote address (IP:port) for the server
- `params` are the parameters that were negotiated with the server, including:
  - `duration` duration of the test, in nanoseconds
  - `interval` send interval, in nanoseconds
  - `length` packet length
  - `stamp_at` timestamp selection parameter (none, send, receive, both or
		midpoint, -ts flag for irtt client)
  - `clock` clock selection parameter (wall or monotonic, -clock flag for irtt client)
  - `dscp` the [DSCP](https://en.wikipedia.org/wiki/Differentiated_services)
		value
- `ip_version` the IP version used (IPv4 or IPv6)
- `df` the do-not-fragment setting (0 == OS default, 1 == false, 2 == true)
- `ttl` the IP [time-to-live](https://en.wikipedia.org/wiki/Time_to_live) value
- `timer` the timer used: simple, comp, hybrid or busy (irtt client -timer parameter)
- `waiter` the waiter used: fixed duration, multiple of RTT or multiple of max RTT
  (irtt client -wait parameter)
- `filler` the packet filler used: none, rand or pattern (irtt client -fill
	parameter)
- `fill_all` whether to fill all packets (irtt client -fillall parameter)
- `lock_os_thread` whether to lock packet handling goroutines to OS threads
- `supplied` a nested `config` object with the configuration as
  originally supplied to the API or `irtt` command. The supplied configuration can
	differ from the final configuration in the following ways:
	- `local_address` and `remote_address` may have hostnames or named ports before
	  being resolved to an IP and numbered port
	- `ip_version` may be IPv4+6 before it is determined after address resolution
	- `params` may be different before the server applies restrictions based on
		its configuration

#### stats

statistics for the results

```
"stats": {
    "start_time": "2017-10-16T21:05:23.502719056+02:00",
    "send_call": {
        "total": 79547,
        "n": 3,
        "min": 17790,
        "max": 33926,
        "mean": 26515,
        "stddev": 8148,
        "variance": 66390200
    },
    "timer_error": {
        "total": 227261,
        "n": 2,
        "min": 59003,
        "max": 168258,
        "mean": 113630,
        "stddev": 77254,
        "variance": 5968327512
    },
    "rtt": {
        "total": 233915,
        "n": 2,
        "min": 99455,
        "max": 134460,
        "mean": 116957,
        "median": 116957,
        "stddev": 24752,
        "variance": 612675012
    },
    "send_delay": {
        "total": 143470,
        "n": 2,
        "min": 54187,
        "max": 89283,
        "mean": 71735,
        "median": 71735,
        "stddev": 24816,
        "variance": 615864608
    },
    "receive_delay": {
        "total": 90445,
        "n": 2,
        "min": 45177,
        "max": 45268,
        "mean": 45222,
        "median": 45222,
        "stddev": 64,
        "variance": 4140
    },
    "bytes_sent": 144,
    "bytes_received": 96,
    "duplicates": 0,
    "late_packets": 0,
    "wait": 403380,
    "duration": 400964028,
    "packets_sent": 3,
    "packets_received": 2,
    "packet_loss_percent": 33.333333333333336,
    "duplicate_percent": 0,
    "late_packets_percent": 0,
    "ipdv_send": {
        "total": 35096,
        "n": 1,
        "min": 35096,
        "max": 35096,
        "mean": 35096,
        "median": 35096,
        "stddev": 0,
        "variance": 0
    },
    "ipdv_receive": {
        "total": 91,
        "n": 1,
        "min": 91,
        "max": 91,
        "mean": 91,
        "median": 91,
        "stddev": 0,
        "variance": 0
    },
    "ipdv_round_trip": {
        "total": 35005,
        "n": 1,
        "min": 35005,
        "max": 35005,
        "mean": 35005,
        "median": 35005,
        "stddev": 0,
        "variance": 0
    },
    "server_processing_time": {
        "total": 20931,
        "n": 2,
        "min": 9979,
        "max": 10952,
        "mean": 10465,
        "stddev": 688,
        "variance": 473364
    },
    "timer_err_percent": 0.056815,
    "timer_misses": 0,
    "timer_miss_percent": 0,
    "send_rate": {
        "bps": 2878,
        "string": "2.9 Kbps"
    },
    "receive_rate": {
        "bps": 3839,
        "string": "3.8 Kbps"
    }
},
```

**Note:** In the `stats` object, a _duration stats_ class of object repeats and
will not be repeated in the individual descriptions. It contains statistics about
nanosecond duration values and has the following attributes:
- `total` the total of the duration values
- `n` the number of duration values
- `min` the minimum duration value
- `max` the maximum duration value
- `mean` the mean duration value
- `stddev` the standard deviation
- `variance` the variance

The regular attributes in `stats` are as follows:

- `start_time` the start time of the test, in TZ format
- `send_call` a duration stats object for the call time when sending packets
- `timer_error` a duration stats object for the observed sleep time error
- `rtt` a duration stats object for the round-trip time
- `send_delay` a duration stats object for the one-way send delay
   _(only available if server timestamps are enabled)_
- `receive_delay` a duration stats object for the one-way receive delay
   _(only available if server timestamps are enabled)_
- `bytes_sent` the number of UDP payload bytes sent during the test
- `bytes_received` the number of UDP payload bytes received during the test
- `duplicates` the number of packets received with the same sequence number
- `late_packets` the number of packets received with a sequence number lower
	than the previously received sequence number (one simple metric for
	out-of-order packets)
- `wait` the actual time spent waiting for final packets, in nanoseconds
- `duration` the actual duration of the test, in nanoseconds, from the time just
	before the first packet was sent to the time after the last packet was
	received and results are starting to be calculated
- `packets_sent` the number of packets sent
- `packets_received` the number of packets received
- `packet_loss_percent` 100 * `packets_received` / `packets_sent`
- `duplicate_percent` 100 * `duplicates` / `packets_received`
- `late_packets_percent` 100 * `late_packets` / `packets_received`
- `ipdv_send` a duration stats object for the send
   [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation)
   _(only available if server timestamps are enabled)_
- `ipdv_receive` a duration stats object for the receive
   [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation)
   _(only available if server timestamps are enabled)_
- `ipdv_round_trip` a duration stats object for the round-trip
   [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation)
   (available regardless of whether server timestamps are enabled or not)
- `server_processing_time` a duration stats object for the time the server took
   after it received the packet to when it sent the response _(only available
   when both send and receive timestamps are enabled)_
- `timer_err_percent` the mean of the absolute values of the timer error, as a
	percentage of the interval
- `timer_misses` the number of times the timer missed the interval (was at least
	50% over the scheduled time)
- `timer_miss_percent` 100 * `timer_misses` / expected packets sent
- `send_rate` the send bitrate (bits-per-second and corresponding string),
	calculated using the number of UDP payload bytes sent between the time right
	before the first send call and the time right after the last send call
- `receive_rate` the receive bitrate (bits-per-second and corresponding string),
	calculated using the number of UDP payload bytes received between the time right
	after the first receive call and the time right after the last receive call

#### round_trips

each round-trip is a single request to / reply from the server

```
"round_trips": [
    {
        "seqno": 0,
        "lost": false,
        "timestamps": {
            "client": {
                "receive": {
                    "wall": 1508180723502871779,
                    "monotonic": 2921143
                },
                "send": {
                    "wall": 1508180723502727340,
                    "monotonic": 2776704
                }
            },
            "server": {
                "receive": {
                    "wall": 1508180723502816623,
                    "monotonic": 32644353327
                },
                "send": {
                    "wall": 1508180723502826602,
                    "monotonic": 32644363306
                }
            }
        },
        "delay": {
            "receive": 45177,
            "rtt": 134460,
            "send": 89283
        },
        "ipdv": {}
    },
    {
        "seqno": 1,
        "lost": false,
        "timestamps": {
            "client": {
                "receive": {
                    "wall": 1508180723702917735,
                    "monotonic": 202967099
                },
                "send": {
                    "wall": 1508180723702807328,
                    "monotonic": 202856692
                }
            },
            "server": {
                "receive": {
                    "wall": 1508180723702861515,
                    "monotonic": 32844398219
                },
                "send": {
                    "wall": 1508180723702872467,
                    "monotonic": 32844409171
                }
            }
        },
        "delay": {
            "receive": 45268,
            "rtt": 99455,
            "send": 54187
        },
        "ipdv": {
            "receive": 91,
            "rtt": -35005,
            "send": -35096
        }
    },
    {
        "seqno": 2,
        "lost": true,
        "timestamps": {
            "client": {
                "receive": {},
                "send": {
                    "wall": 1508180723902925971,
                    "monotonic": 402975335
                }
            },
            "server": {
                "receive": {},
                "send": {}
            }
        },
        "delay": {},
        "ipdv": {}
    }
]
```

**Note:** `wall` values are from Go's `time.Time.UnixNano()`, the number of nanoseconds
elapsed since January 1, 1970 UTC

**Note:** `monotonic` values are the number of nanoseconds since the start of the test for
the client, and since start of the process for the server

- `seqno` the sequence number
- `lost` whether the packet was lost or not
- `timestamps` the client and server timestamps
  - `client` the client send and receive wall and monotonic timestamps
    _(`receive` values not present if `lost` is true)_
  - `server` the server send and receive wall and monotonic timestamps _(both
		`send` and `receive` values not present if `lost` is true), and
		additionally:_
    - `send` values are not present if the StampAt (irtt client -ts parameter) does not
      include send timestamps
    - `receive` values are not present if the StampAt (irtt client -ts parameter) does not
      include receive timestamps
    - `wall` values are not present if the Clock (irtt client -clock parameter) does
      not include wall values or server timestamps are not enabled
    - `monotonic` values are not present if the Clock (irtt client -clock parameter)
      does not include monotonic values or server timestamps are not enabled
- `delay` an object containing the delay values
  - `receive` the one-way receive delay, in nanoseconds _(present only if
    server timestamps are enabled and at least one wall clock value is
		available)_
  - `rtt` the round-trip time, in nanoseconds, always present
  - `send` the one-way send delay, in nanoseconds _(present only if server
    timestamps are enabled and at least one wall clock value is available)_
- `ipdv` an object containing the
  [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation) values
  _(attributes present only for `seqno` > 0, and if `lost` is `false` for both
  the current and previous `round_trip`)_
	- `receive` the difference in receive delay relative to the previous packet
    _(present only if at least one server timestamp is available)_
	- `rtt` the difference in round-trip time relative to the previous packet
    _(always present for `seqno` > 0)_
	- `send` the difference in send delay relative to the previous packet
    _(present only if at least one server timestamp is available)_

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

2) Why is the send (or receive) delay negative?

	 The client and server clocks must be synchronized for one-way delay values to
	 be meaningful.  Well-configured NTP hosts may be able to synchronize within a
	 few milliseconds.
	 [PTP](https://en.wikipedia.org/wiki/Precision_Time_Protocol)
	 ([Linux](http://linuxptp.sourceforge.net) implementation here) should be
	 capable of higher precision, but I've not tried it myself as I don't have any
	 supported hardware.

	 Note that client and server synchronization is not needed for either RTT or
	 IPDV (even send and receive IPDV) values to be correct. RTT is measured with
	 client times only, and since IPDV is measuring differences between successive
	 packets, it's not affected by time synchronization.

3) Why is the receive rate 0 when a single packet is sent?

   Receive rate is measured from the time the first packet is received to the time
   the last packet is received. For a single packet, those times are the same.

4) Why does a test with a one second duration and 200ms interval run for around
   800ms and not one second?

   The test duration is exclusive, meaning requests will not be sent exactly at
   or after the test duration has elapsed. In this case, the interval is 200ms, and
   the fifth and final request is sent at around 800ms from the start of the test.
   The test ends when all replies have been received from the server, so it may
	 end shortly after 800ms. If there are any outstanding packets, the wait time
	 is observed, which by default is a multiple of the maximum RTT.

5) Why does wait fall back to fixed duration when duration is less than RTT?

   If a full RTT has not elapsed, there is no way to know how long an
	 appropriate wait time would be, so the wait falls back to a default fixed
	 time (default is 4 seconds, same as ping).

6) Why can't the client connect to the server, and instead I get
   `Error: no reply from server`?

   There are a number of possible reasons for this:

   1) You've specified an incorrect hostname or IP address for the server.
   2) There is a firewall blocking packets from the client to the server.
      Traffic must be allowed on the chosen UDP port (default 2112).
   3) There is high packet loss. Up to four packets are sent when the client
      tries to connect to the server. If all of these are lost, the client
      won't connect to the server.
   4) The server has an HMAC key set with `-hmac` and the client either has
      not specified a key or it's incorrect. Make sure the client has the
      correct HMAC key, specified also with the `-hmac` parameter.
   5) The server has multiple IP addresses and you've specified a hostname or
      IP to the client that is not the same IP that the server uses to reply.
      This can be verified using `tcpdump -i eth0 udp port 2112`, replacing eth0
      with the actual interface and 2112 with the actual port used. If you see
      that the server is replying, but its source IP is different than the IP
      you specified to the client, there are two possible solutions:
      1) Ideally, the server should be started with explicit bind addresses
         using the `-b` parameter, so that replies always come back from the
         IP address that requests were received on.
      2) If you do not have access to the server, you can work around it by
         using the tcpdump command above and finding out what IP the server is
         replying with. Specify that IP address to the client. Notify the server
         admin to configure the server correctly with explicit bind addresses.

7) Why don't you include median values for send call time, timer error and
   server processing time?

   Those values aren't stored for each round trip, and it's difficult to do a
	 running calculation of the median, although
	 [this method](https://rhettinger.wordpress.com/2010/02/06/lost-knowledge/) of
	 using skip lists appears to have promise. I may consider it for the future,
	 but so far it isn't a high priority.

8) I see you use MD5 for the HMAC. Isn't that insecure?

   MD5 should not have practical vulnerabilities when used in a message authenticate
   code. See [this page](https://en.wikipedia.org/wiki/Hash-based_message_authentication_code#Security)
	 for more info.

9) Will you add unit tests?

   Maybe some. I feel that the most important thing for a project of this size
	 is that the design is clear enough that bugs are next to impossible. IRTT
	 is not there yet though, particularly when it comes to packet manipulation.

10) Are there any plans for translation to other languages?

   While some parts of the API were designed to keep i18n possible, there is no
   support for it built in to the Go standard libraries. It should be possible,
	 but could be a challenge, and is not something I'm likely to undertake myself.

## TODO and Roadmap

Definitely (in order of priority)...

- Fix that minifying removes version number
- Implement server received packets feedback (to distinguish between upstream
	and downstream packet loss)
- Calculate arrival order for round trips during results generation using
  timestamps
- Write a SmokePing probe
- Refactor packet manipulation to improve maintainability and prevent multiple
	validations
- Add a flag to disable per-packet results
- Make sure no garbage created during data collection
- Allow specifying two out of three of interval, bitrate and packet size to the
	client
- Improve robustness and security of public servers:
	- Add bitrate limiting
	- Improve server close by repeating close packets up to some limit
	- Limit open requests to prevent the equivalent of a "syn flood"
	- Add per-IP limiting
- Add different server authentication modes:
	- none (no conn token in header, for local use)
	- token (what we have today, 64-bit token in header)
	- nacl-hmac (hmac key negotiated with public/private key encryption)
- Write an irtt.Timer implementation that uses Linux timerfd
- Add a subcommand to the CLI to convert JSON to CSV
- Show IPDV in continuous (-v) output
- Add a way to keep out "internal" info from JSON, like IP, hostname, and a
	subcommand to strip these details after the JSON is created
- Make it possible to add custom per-round-trip statistics programmatically
- Add more relevant statistics, including more info on outliers and a textual
	histogram(?)
- Add ability for client to request random fill from server
- Allow Client Dial to try multiple IPs when a hostname is given
- Refactor events. I was trying to make something internationalizable and
	filterable as opposed to just writing to a log, but I'm not satisfied yet.

Possibly...

- Prompt to write JSON file on cancellation
- Don't output JSON on interrupt (maybe just to stdout)
- Allow non-isochronous send schedules
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

Many thanks to both Toke Høiland-Jørgensen and Dave Täht from the
[Bufferbloat project](https://www.bufferbloat.net/) for their valuable
advice on this project. Any problems in design or implementation are entirely
my own.
