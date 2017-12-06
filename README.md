# IRTT (Isochronous Round-Trip Tester)

IRTT measures round-trip time and other metrics using UDP packets sent on a
fixed period, and produces both human and machine parseable output.

## Start Here!

IRTT is still under development, and as such has not yet met all of its
[goals](#goals). In particular:

- there's more work to do for public server security
- the protocol and API are not finalized
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
    4. [Known Issues](#known-issues)
2. [Getting Started](#getting-started)
    1. [Installation](#installation)
    2. [Quick Start](#quick-start)
3. [Running IRTT](#running-irtt)
    1. [Client Options](#client-options)
    2. [Server Options](#server-options)
    3. [Running Server at Startup](#running-server-at-startup)
    4. [Tests and Benchmarks](#tests-and-benchmarks)
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
you like "lag" as an excuse). Those who work on reducing latency and improving
network performance may be driven by the idea of helping relieve this stress
for others.

The [Bufferbloat](https://www.bufferbloat.net/projects/) and related projects
aim to reduce "chaotic and laggy network performance", which is what in my
opinion makes this project so worthwhile to anyone who uses the Internet and
values their time and sanity.

The original motivation for IRTT was to improve the latency and packet loss
measurements for the excellent [Flent](https://flent.org) tool, which was
developed by and for the Bufferbloat project. However, IRTT may be useful as a
general purpose tool as well. The goals of this project are to:

- Accurately measure latency and other relevant metrics of network behavior
- Produce statistics via both human and machine parseable output
- Provide for reasonably secure use on both public and private servers
- Support small packet sizes for [VoIP traffic](https://www.cisco.com/c/en/us/support/docs/voice/voice-quality/7934-bwidth-consume.html) simulation
- Support relevant socket options, including DSCP
- Use a single UDP port for deployment simplicity
- Provide an API for embedding and extensibility

### Features:

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

### Limitations

> "It is the limitations of software that give it life." *-Me, justifying my limitations*

#### Isochronous (fixed period) send schedule

Currently, IRTT only sends packets on a fixed period. I am still considering
allowing packets to be sent on varying schedules so that more types of traffic
could be simulated, but accepting this limitation offers some benefits as well:

- It's easy to implement
- It's easy to calculate how much data will be sent in a given time
- It simplifies timer error compensation

Also, isochronous packets are commonly seen in VoIP, games and some streaming media,
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
result takes 72 bytes in memory (8 64-bit timestamps and a 64-bit server
received packet window), so this limits the effective duration of the test,
especially at very small send intervals. However, the advantages are:

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
over 300 Gb of virtual memory to record while the test is running, which is not
likely to be a practical number for most hardware at this time. That is why
64-bit sequence numbers are currently unnecessary.

#### 64-bit received window

In order to determine per-packet differentiation between upstream and downstream
loss, a 64-bit "received window" may be returned with each packet that contains
the receipt status of the previous 64 packets. This can be enabled with either
`-rs window/both` with the irtt client or `ReceivedStatsWindow/ReceivedStatsBoth`
with the API. Its limited width and simple bitmap format lead to some
caveats:

- Per-packet differentiation is not available (for any intervening packets) if
  greater than 64 packets are lost in succession. These packets will be marked
  with the generic `Lost` (`lost` in the JSON).
- While any packet marked `LostDown` (`lost_down`) is guaranteed to be marked
  properly, there is no confirmation of receipt of the receive window from the
  client to the server, so packets may sometimes be erroneously marked `LostUp`
  (`lost_up`), for example, if they arrive late to the server and slide out of
  the received window before they can be confirmed to the client, or if the
  received window is lost on its way to the client and not amended by a later
  packet's received window.

There are many ways that this simple approach could be improved, such as by:

- Allowing a wider window
- Encoding receipt seqnos in a more intelligent way to allow a wider seqno range
- Sending confirmation of window receipt from the client to the server and
  re-sending unreceived windows

However, the current strategy means that a good approximation of per-packet loss
results can be obtained with only 8 additional bytes in each packet. It also
requires very little computational time on the server, and almost all
computation on the client occurs during results generation, after the test is
over. It isn't as accurate with late (out-of-order) upstream packets or with
long sequences of lost packets, but high loss or high numbers of late packets
typically indicate more severe network conditions that should be corrected first
anyway, perhaps before per-packet results matter. Note that in case of very high
packet loss, the **total** number of packets received by the server but not
returned to the client (which can be obtained using `-rs count/both` or
`ReceivedStatsCount/ReceivedStatsBoth`) will still be correct, which will still
provide an accurate _average_ loss percentage in each direction over the course
of the test.

If this limitation adversely affects your results, file an
[Issue](https://github.com/peteheist/irtt/issues) so it can be discussed further.

#### Use of Go

IRTT is written in Go. That carries with it a few disadvantages:

- Non-negligible scheduling delays and system call overhead
- Larger executable size than with C
- Somewhat slower execution speed than C (although [not that much slower](https://benchmarksgame.alioth.debian.org/u64q/compare.php?lang=go&lang2=gcc))

But Go also has benefits that are useful for this application:

- Go's target is network and server applications, with a focus on simplicity,
	reliability and efficiency, which is appropriate for this project
- Memory footprint tends to be significantly lower than with some interpreted
	languages
- It's easy to support a broad array of hardware and OS combinations

### Known Issues

- Windows is unable to set DSCP values.
- The server doesn't run well on 32-bit Windows platforms. When connecting with
  a client, you may see `Terminated due to receive error...`. To work around
  this, disable dual timestamps from the client by including `-ts midpoint`.
  This appears to be a bug in either Go's 32-bit compiler or runtime for
  Windows.

## Getting Started

### Installation

Currently, IRTT is only available in source form. To build it, you must:

- [Install Go](https://golang.org/dl/) (Note: If you're installing Go for the
  first time, you should add `$HOME/go/bin` to your `PATH` so that the `irtt`
  and other Go executables are found. It isn't strictly necessary, but it makes
  things easier. Personally, I like to set `GOPATH` to $HOME, rather than
  leaving the default of `$HOME/go`, so that compiled binaries go in
  `$HOME/bin`, which I have added to my `PATH` anyway. There is no need to
  set `$GOROOT`.)
- Install irtt by running `go get -u github.com/peteheist/irtt/cmd/irtt`. This
  should place the `irtt` executable in `$HOME/go/bin`, or `$GOPATH/bin` if you have
  `$GOPATH` defined.
- If you want to build the source for development, the `stringer` utility should
  be installed by doing `go get -u golang.org/x/tools/cmd/stringer`.  This is
  only necessary though if you need to re-generate the `*_string.go` files that
  are generated by this tool, otherwise the checked in versions may also be
  used and the source should still build.

The `build.sh` script may be used to build irtt during development, or to aid
in minimizing and cross-compiling binaries. For example, `build.sh min
linux-amd64` would compile a minimized binary for Linux on AMD64. See `build.sh`
for more info and a "source-documented" list of platforms that the script
supports. `build.sh install` runs Go's install command, which puts the resulting
executable in `$GOPATH/bin`.

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
a request once per second, like ping, but here we simulate a one minute
G.711 VoIP conversation by using an interval of 20ms and randomly filled
payload of 172 bytes (160 bytes voice data plus 12 byte RTP header):

```
% irtt client -i 20ms -l 172 -d 1m -fill rand -fillall -q 192.168.100.10
[Connecting] connecting to 192.168.100.10
[Connected] connected to 192.168.100.10:2112

                         Min     Mean   Median      Max  Stddev
                         ---     ----   ------      ---  ------
                RTT  11.93ms  20.88ms   19.2ms  80.49ms  7.02ms
         send delay   4.99ms  12.21ms  10.83ms  50.45ms  5.73ms
      receive delay   6.38ms   8.66ms   7.86ms  69.11ms  2.89ms
                                                               
      IPDV (jitter)    782ns   4.53ms   3.39ms  64.66ms   4.2ms
          send IPDV    256ns   3.99ms   2.98ms  35.28ms  3.69ms
       receive IPDV    896ns   1.78ms    966µs  62.28ms  2.86ms
                                                               
     send call time   56.5µs   82.8µs           18.99ms   348µs
        timer error       0s   21.7µs           19.05ms   356µs
  server proc. time   23.9µs   26.9µs             141µs  11.2µs

                duration: 1m0s (wait 241.5ms)
   packets sent/received: 2996/2979 (0.57% loss)
 server packets received: 2980/2996 (0.53%/0.03% loss up/down)
     bytes sent/received: 515312/512388
       send/receive rate: 68.7 Kbps / 68.4 Kbps
           packet length: 172 bytes
             timer stats: 4/3000 (0.13%) missed, 0.11% error
```

In the results above, the client is a
[Raspberry Pi 2 Model B](https://www.raspberrypi.org/products/raspberry-pi-2-model-b/)
and the server is a
[Raspberry Pi 3 Model B](https://www.raspberrypi.org/products/raspberry-pi-3-model-b/).
They are located at two different sites, around 50km from one another, each of which
connects to the Internet via point-to-point WiFi. The client is 3km
[NLOS](https://en.wikipedia.org/wiki/Non-line-of-sight_propagation) through trees
located near the client's transmitter, which is likely the reason for the higher
upstream packet loss, mean send delay and IPDV. That said, these conditions would
likely provide for a decent VoIP conversation.

## Running IRTT

### Client Options

### Server Options

### Running Server at Startup

There are many ways to run a service at startup that depend on the OS used and
other specific requirements. For Linux, the recommended way to start Go servers
in general is to use `systemd`. It can provide a lot of flexibility for logging,
service management or other custom configuration. Tutorials may be found on the
Internet that describe how to do use systemd in greater detail, but a very
simple method is as follows:

1) Install the `irtt` executable into `/usr/bin`

2) Create a file `/etc/systemd/system/irtt.service` with the following contents
   (note that some commented out parts may be used to bind to a specific
   interface, and that you may also find the files `irtt.service` and
   `irtt@.service` in the source tree):

```
[Unit]
Description=irtt server
After=network.target
#BindsTo=sys-subsystem-net-devices-%i.device
#After=sys-subsystem-net-devices-%i.device

[Service]
ExecStart=/usr/bin/irtt server # -b %%%i
User=nobody
Restart=on-failure

# Sandboxing
# Some of these are not present in old versions of systemd.
# Comment out as appropriate.
PrivateTmp=yes
PrivateDevices=yes
ProtectControlGroups=yes
ProtectKernelTunables=yes
ProtectSystem=strict
ProtectHome=yes
NoNewPrivileges=yes

[Install]
WantedBy=multi-user.target
```

3) Reload systemd with `sudo systemctl daemon-reload`

4) Start the service with `sudo systemctl start irtt.service`

5) Check the service status with `sudo systemctl status irtt.service`

6) Set the service to start at boot with `sudo systemctl enable irtt.service`

7) View and follow the irtt server logs with `sudo journalctl -f -u irtt.service`

See the irtt server's command line usage with `irtt server -h` to explore
additional parameters you may want when starting your server, such as an hmac
key or restrictions on test parameters. To secure a server for public use, you
may want to take additional steps outside of the scope of this tutorial for
securing Linux services in general, including but not limited to:

- Setting up an iptables firewall (only UDP port 2112 must be open)
- Setting up a chroot jail

It should be noted that there are no known security vulnerabilities in the Go
language at this time, and the steps above, in particular the chroot jail, may
or may not serve to enhance security in any way. Go-based servers are generally
regarded as safe because of Go's high-level language constructs for memory
management, and at this time IRTT makes no use of Go's
[unsafe](https://golang.org/pkg/unsafe/) package.

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
    "go_version": "go1.9.2",
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
    "open_timeouts": "1s,2s,4s,8s",
    "params": {
        "duration": 600000000,
        "interval": 200000000,
        "length": 48,
        "received_stats": "both",
        "stamp_at": "both",
        "clock": "both",
        "dscp": 0
    },
    "strict_params": false,
    "ip_version": "IPv4",
    "df": 0,
    "ttl": 0,
    "timer": "comp",
    "waiter": "3x4s",
    "filler": "none",
    "fill_all": false,
    "thread_lock": false,
    "supplied": {
        "local_address": ":0",
        "remote_address": "localhost",
        "open_timeouts": "1s,2s,4s,8s",
        "params": {
            "duration": 600000000,
            "interval": 200000000,
            "length": 0,
            "received_stats": "both",
            "stamp_at": "both",
            "clock": "both",
            "dscp": 0
        },
        "strict_params": false,
        "ip_version": "IPv4+6",
        "df": 0,
        "ttl": 0,
        "timer": "comp",
        "waiter": "3x4s",
        "filler": "none",
        "fill_all": false,
        "thread_lock": false
    }
},
```

- `local_address` the local address (IP:port) for the client
- `remote_address` the remote address (IP:port) for the server
- `open_timeouts` a list of timeout durations used after an open packet is sent
- `params` are the parameters that were negotiated with the server, including:
  - `duration` duration of the test, in nanoseconds
  - `interval` send interval, in nanoseconds
  - `length` packet length
  - `received_stats` statistics for packets received by server (none, count,
    window or both, -rs flag for irtt client)
  - `stamp_at` timestamp selection parameter (none, send, receive, both or
		midpoint, -ts flag for irtt client)
  - `clock` clock selection parameter (wall or monotonic, -clock flag for irtt client)
  - `dscp` the [DSCP](https://en.wikipedia.org/wiki/Differentiated_services)
		value
- `strict_params` if true, test is aborted if server restricts parameters
- `ip_version` the IP version used (IPv4 or IPv6)
- `df` the do-not-fragment setting (0 == OS default, 1 == false, 2 == true)
- `ttl` the IP [time-to-live](https://en.wikipedia.org/wiki/Time_to_live) value
- `timer` the timer used: simple, comp, hybrid or busy (irtt client -timer parameter)
- `waiter` the waiter used: fixed duration, multiple of RTT or multiple of max RTT
  (irtt client -wait parameter)
- `filler` the packet filler used: none, rand or pattern (irtt client -fill
	parameter)
- `fill_all` whether to fill all packets (irtt client -fillall parameter)
- `thread_lock` whether to lock packet handling goroutines to OS threads
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
    "server_packets_received": 2,
    "bytes_sent": 144,
    "bytes_received": 96,
    "duplicates": 0,
    "late_packets": 0,
    "wait": 403380,
    "duration": 400964028,
    "packets_sent": 3,
    "packets_received": 2,
    "packet_loss_percent": 33.333333333333336,
    "upstream_loss_percent": 33.333333333333336,
    "downstream_loss_percent": 0,
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
- `server_packets_received` the number of packets received by the server,
  including duplicates (always present, but only valid if the `ReceivedStats`
  parameter includes `ReceivedStatsCount`, or the -rs parameter to the irtt
  client is `count` or `both`)
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
- `packets_sent` the number of packets sent to the server
- `packets_received` the number of packets received from the server
- `packet_loss_percent` 100 * (`packets_sent` - `packets_received`) / `packets_sent`
- `upstream_loss_percent` 100 * (`packets_sent` - `server_packets_received` /
  `packets_sent`) (always present, but only valid if `server_packets_received`
  is valid)
- `downstream_loss_percent` 100 * (`server_packets_received` - `packets_received` /
  `server_packets_received`) (always present, but only valid if
  `server_packets_received` is valid)
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
- `lost` the lost status of the packet, which can be one of `false`, `true`,
  `true_down` or `true_up`. The `true_down` and `true_up` values are only
  available if the `ReceivedStats` parameter includes `ReceivedStatsWindow`
  (irtt client -rs parameter). Even then, if it could not be determined whether
  the packet was lost upstream or downstream, the value `true` is used.
- `timestamps` the client and server timestamps
  - `client` the client send and receive wall and monotonic timestamps
    _(`receive` values not present if `lost` is not false)_
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
   variable than the results reported by ping, due to Go's scheduling
   variability and system call overhead. That said, there are advantages that
   IRTT has over ping when minimum RTT is not what you're measuring:

	 - Some device vendors prioritize ICMP, so ping may not be an accurate measure
		 of user-perceived latency.
	 - In addition to round-trip time, IRTT also measures OWD, IPDV and upstream
	   vs downstream packet loss.
	 - IRTT can use HMACs to protect private servers from unauthorized discovery
		 and use.
	 - IRTT has a three-way handshake to prevent test traffic redirection from
		 spoofed source IPs.
	 - IRTT can fill the payload (if included) with random data.

2) Why is the send (or receive) delay negative or much larger than I expect?

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
   3) There is high packet loss. By default, up to four packets are sent when
      the client tries to connect to the server, using timeouts of 1, 2, 4 and 8
      seconds. If all of these are lost, the client won't connect to the server.
      In environments with known high packet loss, the `-timeouts` parameter may
      be used to send more packets with the chosen timeouts before abandoning
      the connection.
   4) The server has an HMAC key set with `-hmac` and the client either has
      not specified a key or it's incorrect. Make sure the client has the
      correct HMAC key, also specified with the `-hmac` parameter.
   5) You're trying to connect to a listener that's listening on an unspecified
      IP address, and return packets are not routing properly, which can happen in
      some network configurations. Try running the server with the `-setsrcip`
      parameter, which sets the source address on all reply packets from listeners
      on unspecified IP addresses. This is not done by default in order to avoid
      the additional per-packet heap allocations required by the
      `golang.org/x/net` packages.

7) Why don't you include median values for send call time, timer error and
   server processing time?

   Those values aren't stored for each round trip, and it's difficult to do a
	 running calculation of the median, although
	 [this method](https://rhettinger.wordpress.com/2010/02/06/lost-knowledge/) of
	 using skip lists appears to have promise. It's a possibility for the future,
	 but so far it isn't a high priority. If it is for you, file an
   [Issue](https://github.com/peteheist/irtt/issues).

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
    support for i18n built in to the Go standard libraries. It should be possible,
    but could be a challenge, and is not something I'm likely to undertake myself.

11) Why do I get `Error: failed to allocate results buffer for X round trips
   (runtime error: makeslice: cap out of range)`?

    Your test interval and duration probably require a results buffer that's
    larger than Go can allocate on your platform. Lower either your test
    interval or duration. See the following additional documentation for
    reference: [In-memory results storage](#in-memory-results-storage),
    `maxSliceCap` in [slice.go](https://golang.org/src/runtime/slice.go) and
    `_MaxMem` in [malloc.go](https://golang.org/src/runtime/malloc.go).

12) Why when I start a server on Windows do I see these warnings:
    - `[NoDSCPSupport] no DSCP support available (operation not supported)`
    - `[NoReceiveDstAddrSupport] no support for determining packet destination
      address (not supported by windows)`
    - `[MultipleAddresses] warning: multiple IP addresses, all bind addresses
      should be explicitly specified with -b or clients may not be able to
      connect`

    These are due to limitations in Go's support for the Windows platform. The
    consequences of these limitations are:

    - Packet DSCP (TOS) values can not be set.
    - The server cannot determine the original destination address of incoming
      packets, so it can't use the same address when sending replies. This means
      that on machines with multiple network adapters, and when using
      unspecified bind IP addresses, packets may not always return to clients
      properly. Rather than using an unspecified IP address though, you may also
      listen on all addresses on all adapters with `-b "%*"`. This however has
      the limitation that it will not listen on new adapters and addresses
      dynamically like using an unspecified IP does, so on Windows, you'll have
      to accept either one limitation or the other.

13) Why is little endian byte order used in the packet format?

    As for Google's [protobufs](https://github.com/google/protobuf), this was
    chosen because the vast majority of modern processors use little-endian byte
    order. In the future, packet manipulation may be optimized for little-endian
    architecutures by doing conversions with Go's
    [unsafe](https://golang.org/pkg/unsafe/) package, but so far this
    optimization has not been shown to be necessary.

## TODO and Roadmap

### TODO v0.9

_Concrete tasks that just need doing..._

- Move server communication and update logic into sconn
  - Get rid of remaining specific drop events, use generic Drop + error
  - Improve connRef design
- Add `-concurrent` flag to server for one goroutine per client conn
- Check or replace session cleanup mechanism
- Add a session timeout and max interval so client doesn't send to a closed conn
- Check that listeners exit only due to permanent errors, and exit code is set
- Add ability for client to request random fill from server
- Add protocol version number along with client check
- Refactor packet manipulation to improve readability and prevent multiple validations
- Improve client connection closure by:
  - Repeating close packets up to four times until acknowledgement, like open
  - Including received packet stats in the acknowledgement from the server
- Use pflag options or something GNU compatible: https://github.com/spf13/pflag
- Run heap profiler on client

### TODO v1.0

- Improve robustness and security of public servers:
	- Add bitrate limiting
	- Limit open requests to prevent the equivalent of a "syn flood"
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
  - Do more thorough tests of `chrt -r 99`, `-thread` and `-gc`
  - Find or file issue with Go team over scheduler performance
  - Prototype doing all thread scheduling for Linux in C
- Add different server authentication modes:
	- none (no conn token in header, for minimum packet sizes during local use)
	- token (what we have today, 64-bit token in header)
	- nacl-hmac (hmac key negotiated with public/private key encryption)
- Implement graceful server shutdown with sconn close
- Implement zero-downtime restarts

### Inbox

_Collection area for undefined or uncertain stuff..._

- Map error codes to exit codes
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
advice on this project. Any problems in design or implementation are entirely
my own.
