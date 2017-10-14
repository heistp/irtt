# IRTT (Isochronous Round-Trip Time)

IRTT measures round-trip time and other metrics using UDP packets sent on a
fixed period, and produces both human and machine parseable output.

## Why?

Latency is an under-appreciated metric in network and application
performance. I think this may be what subconciously drives those who work on
latency related projects. There is a certain hard to quantify but visceral
*"latency stress"* that comes from waiting in expectation after a web page
click, or tap on your smartphone.

The [Bufferbloat](https://www.bufferbloat.net/projects/) and related projects
aim to reduce this latency stress, which is what in my opinion makes them so
worthwhile to anyone who uses the Internet and values their time and sanity.

The original motivation for IRTT was to improve the latency and packet loss
measurements in the excellent [Flent](https://flent.org) tool, which was
developed by and for the Bufferbloat project. However, IRTT could be useful as a
general purpose tool as well.

## Goals

- Accurately measure:
	- [RTT (round-trip time)](https://en.wikipedia.org/wiki/Round-trip_delay_time)
	- [OWD (one-way delay)](https://en.wikipedia.org/wiki/End-to-end_delay)
	- [IPDV (instantaneous packet delay variation)](https://en.wikipedia.org/wiki/Packet_delay_variation)
	- [Packet loss](https://en.wikipedia.org/wiki/Packet_loss), with upstream and downstream differentiation
	- [Out-of-order](https://en.wikipedia.org/wiki/Out-of-order_delivery) and
		duplicate packets
	- [Bitrate](https://en.wikipedia.org/wiki/Bit_rate)
- Produce relevant statistics via both human and machine parseable output
- Provide for reasonably secure use on public and private servers
- Support DSCP
- Support small packet sizes for [VoIP traffic](https://www.cisco.com/c/en/us/support/docs/voice/voice-quality/7934-bwidth-consume.html) simulation
- Use a single UDP port for deployment simplicity
- Provide an API for embedding and extensibility

## Limitations

> "It is the limitations of software that give it life." *-Me, justifying my limitations*

### Isochronous (fixed period) send schedule

Currently, IRTT only sends packets on a fixed period. I am still considering
allowing packets to be sent on varying schedules so that more types of traffic
could be simulated, but accepting this limitation offers some benefits as well:

- It's easy to implement
- It's easy to calculate how much data will be sent in a given time
- It allows for effective timer error compensation

Also, isochronous packets are commonly seen in VoIP, games and streaming media,
so we can already simulate an array of different types of traffic.

### Fixed packet lengths for a given test

Packet lengths are fixed for the duration of the test. While this may not be an
accurate simulation of some types of traffic, it means that IPDV measurements
are accurate, where they wouldn't be in any other case.

### Stateful protocol

While there are numerous benefits to stateless protocols, including simplified
server design, horizontal scalabity, and easily implemented zero-downtime
restarts, I ultimately decided that a stateless protocol brings most of its
advantages to the data center, while in this case, a stateful protocol provides
important benefits to the user, including:

- Smaller packet sizes (a design goal) as context does not need to be included in every request
- More accurate measurement of upstream vs downstream packet loss (this gets worse in a stateless protocol as RTT approaches the test duration, complicating interplanetary tests!)
- More accurate rate and test duration limiting on the server

### In-memory results storage

Results for each round-trip are stored in memory as the test is being run. Each
result takes up to 64 bytes in memory (8 64-bit timestamps, explained later), so
this limits the effective duration of the test, especially at very small send
intervals. However, the advantages are:

- Statistical analysis (like calculation of the median) is more easily performed on fixed arrays in memory than on running data values
- Not accessing the disk during the test prevents inadvertently affecting the
	results
- It simplifies the API

As a consequence of storing results in memory, packet sequence numbers are fixed
at 32-bits. If all 2^32 sequence numbers were used, the results would require
almost 275 Gb of RAM, which is not likely to be a limit reached any time soon.

### Use of Go

IRTT is written in Go. While that carries with it the disadvantage of a larger
executable size than with C, for example, Go still has benefits that are
useful for this application:

- It offers high execution speed by compiling to native executables
- It's easy to support a broad array of hardware and OS combinations
- Its simple syntax typically makes implementation easier, more robust and more
	readable than with C, for example

*TBD...*

## Features:

* Use of both wall and monotonic clocks
* Timers and timer compensation
* IPv4 and IPv6 support
* Flexible format negotiation and fixed format test packets
* HMAC support
* Conn token
* Public server protection
* Packet filling
* Output to JSON

## Security:

* Off-path and on-path attacks
* Public servers:
	* Conn token
	* Server restrictions
	* Server fill
	* NaCl-HMAC
* Private servers:
	* HMAC

## Specs

* Packet format (and endianness)
* Parameters:
	* StampAt
	* Clock
	* DSCP
* Statistics:
	* Duplicate handling
	* Bit rates
* Socket options and limitations
* Size of executable, stripping binary, executable compression
