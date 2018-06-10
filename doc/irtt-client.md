% IRTT-CLIENT(1) v0.9.0 | IRTT Manual
%
% February 11, 2018

# NAME

irtt-client - Isochronous Round-Trip Time Client

# SYNOPSIS

irtt client [*args*]

# DESCRIPTION

*irtt client* is the client for [irtt(1)](irtt.html).

# OPTIONS

-d *duration*
:   Total time to send (default 1m0s, see [Duration units](#duration-units)
    below)

-i *interval*
:   Send interval (default 1s, see [Duration units](#duration-units) below)

-l *length*
:   Length of packet (default 0, increased as necessary for required headers),
    common values:

    - 1472 (max unfragmented size of IPv4 datagram for 1500 byte MTU)
    - 1452 (max unfragmented size of IPv6 datagram for 1500 byte MTU)

-o *file*
:   Write JSON output to file (use '-' for stdout).  The extension used for
    *file* controls the gzip behavior as follows (output to stdout is not
    gzipped):

    Extension | Behavior
    --------- | --------
    none      | extension .json.gz is added, output is gzipped
    .json.gz  | output is gzipped
    .gz       | output is gzipped, extension changed to .json.gz
    .json     | output is not gzipped

-q
:   Quiet, suppress per-packet output

-Q
:   Really quiet, suppress all output except errors to stderr

-n
:   No test, connect to the server and validate test parameters but don't run
    the test

\--stats=*stats*
:   Server stats on received packets (default *both*). Possible values:

    Value    | Meaning
    -------- | -------
    *none*   | no server stats on received packets
    *count*  | total count of received packets  
    *window* | receipt status of last 64 packets with each reply  
    *both*   | both count and window

\--tstamp=*mode*
:   Server timestamp mode (default *both*). Possible values:

    Value      | Meaning
    ---------- | -------
    *none*     | request no timestamps
    *send*     | request timestamp at server send  
    *receive*  | request timestamp at server receive  
    *both*     | request both send and receive timestamps
    *midpoint* | request midpoint timestamp (send/receive avg)

\--clock=*clock*
:   Clock/s used for server timestamps (default *both*). Possible values:

    Value       | Meaning
    ----------- | -------
    *wall*      | wall clock only
    *monotonic* | monotonic clock only  
    *both*      | both clocks  

\--dscp=*dscp*
:   DSCP (ToS) value (default 0, 0x prefix for hex). Common values:

    Value | Meaning
    ----- | -------
    0     | Best effort
    8     | CS1- Bulk
    40    | CS5- Video
    46    | EF- Expedited forwarding

    [DSCP & ToS](https://www.tucny.com/Home/dscp-tos)

\--df=*DF*
:   Setting for do not fragment (DF) bit in all packets. Possible values:

    Value     | Meaning
    --------- | -------
    *default* | OS default
    *false*   | DF bit not set
    *true*    | DF bit set

\--wait=*wait*
:   Wait time at end of test for unreceived replies (default 3x4s).
    Possible values:

    Format       | Meaning
    ------------ | -------
    #*x*duration | # times max RTT, or duration if no response
    #*r*duration | # times RTT, or duration if no response  
    duration     | fixed duration (see [Duration units](#duration-units) below)

    Examples:

    Example | Meaning
    ------- | -------
    3x4s    | 3 times max RTT, or 4 seconds if no response
    1500ms  | fixed 1500 milliseconds  

\--timer=*timer*
:   Timer for waiting to send packets (default comp). Possible values:

    Value      | Meaning
    ---------- | -------
    *simple*   | Go's standard time.Timer
    *comp*     | Simple timer with error compensation (see -tcomp)
    *hybrid:*# | Hybrid comp/busy timer with sleep factor (default 0.95)
    *busy*     | busy wait loop (high precision and CPU, blasphemy)

\--tcomp=*alg*
:   Comp timer averaging algorithm (default exp:0.10). Possible values:

    Value   | Meaning
    ------- | -------
    *avg*   | Cumulative average error
    *win:*# | Moving average error with window # (default 5)
    *exp:*# | Exponential average with alpha # (default 0.10)

\--fill=*fill*
:   Fill payload with given data (default none). Possible values:

    Value        | Meaning
    ------------ | -------
    *none*       | Leave payload as all zeroes
    *rand*       | Use random bytes from Go's math.rand
    *pattern:*XX | Use repeating pattern of hex (default 69727474)

\--fill-one
:   Fill only once and repeat for all packets

\--sfill=fill
:   Request server fill (default not specified). See values for --fill.
    Server must support and allow this fill with --allow-fills.

\--local=addr
:   Local address (default from OS). Possible values:

    Value       | Meaning
    ----------- | -------
    *:port*     | Unspecified address (all IPv4/IPv6 addresses) with port
    *host*      | Host with dynamic port, see [Host formats](#host-formats) below
    *host:port* | Host with specified port, see [Host formats](#host-formats) below

\--hmac=key
:   Add HMAC with key (0x for hex) to all packets, provides:

    - Dropping of all packets without a correct HMAC
    - Protection for server against unauthorized discovery and use

-4
:   IPv4 only

-6
:   IPv6 only

\--timeouts=*durations*
:   Timeouts used when connecting to server (default 1s,2s,4s,8s).
    Comma separated list of durations (see [Duration units](#duration-units) below).
    Total wait time will be up to the sum of these Durations.
    Max packets sent is up to the number of Durations.
    Minimum timeout duration is 200ms.

\--ttl=*ttl*
:   Time to live (default 0, meaning use OS default)

\--loose
:   Accept and use any server restricted test parameters instead of
    exiting with nonzero status.

\--thread
:   Lock sending and receiving goroutines to OS threads

-h
:   Show help

-v
:   Show version

## Host formats

Hosts may be either hostnames (for IPv4 or IPv6) or IP addresses. IPv6
addresses must be surrounded by brackets and may include a zone after the %
character. Examples:

Type            | Example
--------------- | -------
IPv4 IP         | 192.168.1.10
IPv6 IP         | [2001:db8:8f::2/32]
IPv4/6 hostname | localhost

**Note:** IPv6 addresses must be quoted in most shells.

## Duration units

Durations are a sequence of decimal numbers, each with optional fraction, and
unit suffix, such as: "300ms", "1m30s" or "2.5m". Sanity not enforced.

Suffix | Unit
------ | ----
h      | hours
m      | minutes
s      | seconds
ms     | milliseconds
ns     | nanoseconds

# OUTPUT

IRTT's JSON output format consists of five top-level objects:

1. [version](#version)
2. [system_info](#system_info)
3. [config](#config)
4. [stats](#stats)
5. [round_trips](#round_trips)

These are documented through the examples below. All attributes are present
unless otherwise **noted**.

## version

version information

```
"version": {
    "irtt": "0.9.0",
    "protocol": 1,
    "json_format": 1
},
```

- *irtt* the IRTT version number
- *protocol* the protocol version number (increments mean incompatible changes)
- *json_format* the JSON format number (increments mean incompatible changes)

## system_info

a few basic pieces of system information

```
"system_info": {
    "os": "darwin",
    "cpus": 8,
    "go_version": "go1.9.2",
    "hostname": "tron.local"
},
```

- *os* the Operating System from Go's *runtime.GOOS*
- *cpus* the number of CPUs reported by Go's *runtime.NumCPU()*, which reflects
  the number of logical rather than physical CPUs. In the example below, the
	number 8 is reported for a Core i7 (quad core) with hyperthreading (2 threads
	per core).
- *go_version* the version of Go the executable was built with
- *hostname* the local hostname

## config

the configuration used for the test

```
"config": {
    "local_address": "127.0.0.1:51203",
    "remote_address": "127.0.0.1:2112",
    "open_timeouts": "1s,2s,4s,8s",
    "params": {
        "proto_version": 1,
        "duration": 600000000,
        "interval": 200000000,
        "length": 48,
        "received_stats": "both",
        "stamp_at": "both",
        "clock": "both",
        "dscp": 0,
        "server_fill": ""
    },
    "loose": false,
    "ip_version": "IPv4",
    "df": 0,
    "ttl": 0,
    "timer": "comp",
    "waiter": "3x4s",
    "filler": "none",
    "fill_one": false,
    "thread_lock": false,
    "supplied": {
        "local_address": ":0",
        "remote_address": "localhost",
        "open_timeouts": "1s,2s,4s,8s",
        "params": {
            "proto_version": 1,
            "duration": 600000000,
            "interval": 200000000,
            "length": 0,
            "received_stats": "both",
            "stamp_at": "both",
            "clock": "both",
            "dscp": 0,
            "server_fill": ""
        },
        "loose": false,
        "ip_version": "IPv4+6",
        "df": 0,
        "ttl": 0,
        "timer": "comp",
        "waiter": "3x4s",
        "filler": "none",
        "fill_one": false,
        "thread_lock": false
    }
},
```

- *local_address* the local address (IP:port) for the client
- *remote_address* the remote address (IP:port) for the server
- *open_timeouts* a list of timeout durations used after an open packet is sent
- *params* are the parameters that were negotiated with the server, including:
  - *proto_version* protocol version number
  - *duration* duration of the test, in nanoseconds
  - *interval* send interval, in nanoseconds
  - *length* packet length
  - *received_stats* statistics for packets received by server (none, count,
    window or both, *\--stats* flag for irtt client)
  - *stamp_at* timestamp selection parameter (none, send, receive, both or
		midpoint, *\--tstamp* flag for irtt client)
  - *clock* clock selection parameter (wall or monotonic, *\--clock* flag for irtt client)
  - *dscp* the [DSCP](https://en.wikipedia.org/wiki/Differentiated_services)
		value
  - *server_fill* the requested server fill (*\--sfill* flag for irtt client)
- *loose* if true, client accepts and uses restricted server parameters, with a
  warning
- *ip_version* the IP version used (IPv4 or IPv6)
- *df* the do-not-fragment setting (0 == OS default, 1 == false, 2 == true)
- *ttl* the IP [time-to-live](https://en.wikipedia.org/wiki/Time_to_live) value
- *timer* the timer used: simple, comp, hybrid or busy (irtt client \--timer flag)
- *time_source* the time source used: go or windows
- *waiter* the waiter used: fixed duration, multiple of RTT or multiple of max RTT
  (irtt client *\--wait* flag)
- *filler* the packet filler used: none, rand or pattern (irtt client *\--fill*
	flag)
- *fill_one* whether to fill only once and repeat for all packets
  (irtt client *\--fill-one* flag)
- *thread_lock* whether to lock packet handling goroutines to OS threads
- *supplied* a nested *config* object with the configuration as
  originally supplied to the API or *irtt* command. The supplied configuration can
	differ from the final configuration in the following ways:
	- *local_address* and *remote_address* may have hostnames or named ports before
	  being resolved to an IP and numbered port
	- *ip_version* may be IPv4+6 before it is determined after address resolution
	- *params* may be different before the server applies restrictions based on
		its configuration

## stats

statistics for the results

```
"stats": {
    "start_time": {
        "wall": 1528621979787034330,
        "monotonic": 5136247
    },
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

**Note:** In the *stats* object, a _duration stats_ class of object repeats and
will not be repeated in the individual descriptions. It contains statistics about
nanosecond duration values and has the following attributes:

- *total* the total of the duration values
- *n* the number of duration values
- *min* the minimum duration value
- *max* the maximum duration value
- *mean* the mean duration value
- *stddev* the standard deviation
- *variance* the variance

The regular attributes in *stats* are as follows:

- *start_time* the start time of the test (see *round_trips* Notes for
  descriptions of *wall* and *monotonic* values)
- *send_call* a duration stats object for the call time when sending packets
- *timer_error* a duration stats object for the observed sleep time error
- *rtt* a duration stats object for the round-trip time
- *send_delay* a duration stats object for the one-way send delay
  **(only available if server timestamps are enabled)**
- *receive_delay* a duration stats object for the one-way receive delay
  **(only available if server timestamps are enabled)**
- *server_packets_received* the number of packets received by the server,
  including duplicates (always present, but only valid if the *ReceivedStats*
  parameter includes *ReceivedStatsCount*, or the *\--stats* flag to the irtt
  client is *count* or *both*)
- *bytes_sent* the number of UDP payload bytes sent during the test
- *bytes_received* the number of UDP payload bytes received during the test
- *duplicates* the number of packets received with the same sequence number
- *late_packets* the number of packets received with a sequence number lower
	than the previously received sequence number (one simple metric for
	out-of-order packets)
- *wait* the actual time spent waiting for final packets, in nanoseconds
- *duration* the actual duration of the test, in nanoseconds, from the time just
	before the first packet was sent to the time after the last packet was
	received and results are starting to be calculated
- *packets_sent* the number of packets sent to the server
- *packets_received* the number of packets received from the server
- *packet_loss_percent* 100 * (*packets_sent* - *packets_received*) / *packets_sent*
- *upstream_loss_percent* 100 * (*packets_sent* - *server_packets_received* /
  *packets_sent*) (always present, but only valid if *server_packets_received*
  is valid)
- *downstream_loss_percent* 100 * (*server_packets_received* - *packets_received* /
  *server_packets_received*) (always present, but only valid if
  *server_packets_received* is valid)
- *duplicate_percent* 100 * *duplicates* / *packets_received*
- *late_packets_percent* 100 * *late_packets* / *packets_received*
- *ipdv_send* a duration stats object for the send
   [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation)
   **(only available if server timestamps are enabled)**
- *ipdv_receive* a duration stats object for the receive
   [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation)
   **(only available if server timestamps are enabled)**
- *ipdv_round_trip* a duration stats object for the round-trip
   [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation)
   **(available regardless of whether server timestamps are enabled or not)**
- *server_processing_time* a duration stats object for the time the server took
   after it received the packet to when it sent the response **(only available
   when both send and receive timestamps are enabled)**
- *timer_err_percent* the mean of the absolute values of the timer error, as a
	percentage of the interval
- *timer_misses* the number of times the timer missed the interval (was at least
	50% over the scheduled time)
- *timer_miss_percent* 100 * *timer_misses* / expected packets sent
- *send_rate* the send bitrate (bits-per-second and corresponding string),
	calculated using the number of UDP payload bytes sent between the time right
	before the first send call and the time right after the last send call
- *receive_rate* the receive bitrate (bits-per-second and corresponding string),
	calculated using the number of UDP payload bytes received between the time right
	after the first receive call and the time right after the last receive call

## round_trips

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

**Note:** *wall* values are from Go's *time.Time.UnixNano()*, the number of nanoseconds
elapsed since January 1, 1970 UTC

**Note:** *monotonic* values are the number of nanoseconds since some arbitrary
point in time, so can only be relied on to measure duration

- *seqno* the sequence number
- *lost* the lost status of the packet, which can be one of *false*, *true*,
  *true_down* or *true_up*. The *true_down* and *true_up* values are only
  possible if the *ReceivedStats* parameter includes *ReceivedStatsWindow*
  (irtt client *\--stats* flag). Even then, if it could not be determined whether
  the packet was lost upstream or downstream, the value *true* is used.
- *timestamps* the client and server timestamps
  - *client* the client send and receive wall and monotonic timestamps
    **(*receive* values only present if *lost* is false)**
  - *server* the server send and receive wall and monotonic timestamps **(both
		*send* and *receive* values not present if *lost* is true)**, and
		additionally:
    - *send* values are not present if the StampAt (irtt client *\--tstamp* flag)
      does not include send timestamps
    - *receive* values are not present if the StampAt (irtt client *\--tstamp*
      flag) does not include receive timestamps
    - *wall* values are not present if the Clock (irtt client *\--clock* flag) does
      not include wall values or server timestamps are not enabled
    - *monotonic* values are not present if the Clock (irtt client *\--clock*
      flag) does not include monotonic values or server timestamps are not enabled
- *delay* an object containing the delay values
  - *receive* the one-way receive delay, in nanoseconds **(present only if
    server timestamps are enabled and at least one wall clock value is
		available)**
  - *rtt* the round-trip time, in nanoseconds, always present
  - *send* the one-way send delay, in nanoseconds **(present only if server
    timestamps are enabled and at least one wall clock value is available)**
- *ipdv* an object containing the
  [IPDV](https://en.wikipedia.org/wiki/Packet_delay_variation) values
  **(attributes present only for *seqno* > 0, and if *lost* is *false* for both
  the current and previous *round_trip*)**
	- *receive* the difference in receive delay relative to the previous packet
    **(present only if at least one server timestamp is available)**
	- *rtt* the difference in round-trip time relative to the previous packet
    (always present for *seqno* > 0)
	- *send* the difference in send delay relative to the previous packet
    **(present only if at least one server timestamp is available)**

# EXIT STATUS

*irtt client* exits with one of the following status codes:

Code | Meaning
---- | -------
0    | Success
1    | Runtime error
2    | Command line error
3    | Two interrupt signals received

# WARNINGS

It is possible with the irtt client to dramatically harm network performance
by using intervals that are too low, particularly in combination with large
packet lengths. Careful consideration should be given before using
sub-millisecond intervals, not only because of the impact on the network, but
also because:

- Timer accuracy at sub-millisecond intervals may begin to suffer without
  the use of a custom kernel or the busy timer (which pins the CPU)
- Memory consumption for results storage and system CPU time both rise rapidly
- The granularity of the results reported may very well not be required

# EXAMPLES

$ irtt client localhost
:   Sends requests once per second for one minute to localhost.

$ irtt client -i 200ms -d 10s -o - localhost
:   Sends requests every 0.2 sec for 10 seconds to localhost. Writes JSON
    output to stdout.

$ irtt client -i 20ms -d 1m -l 172 \--fill=rand \--sfill=rand 192.168.100.10
:   Sends requests every 20ms for one minute to 192.168.100.10. Fills both the
    client and server payload with random data. This simulates a G.711 VoIP
    conversation, one of the most commonly used codecs for VoIP as of this
    writing.

$ irtt client -i 0.1s -d 5s -6 \--dscp=46 irtt.example.org
:   Sends requests with IPv6 every 100ms for 5 seconds to irtt.example.org.
    Sets the DSCP value (ToS field) of requests and responses to 46
    (Expedited Forwarding).

$ irtt client \--hmac=secret -d 10s "[2001:db8:8f::2/32]:64381"
:   Sends requests to the specified IPv6 IP on port 64381 every second for
    10 seconds. Adds an HMAC to each packet with the key *secret*.

# SEE ALSO

[irtt(1)](irtt.html), [irtt-server(1)](irtt-server.html)

[IRTT GitHub repository](https://github.com/heistp/irtt/)
