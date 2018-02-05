% IRTT-CLIENT(1) v0.9 | IRTT Manual
%
% February 4, 2018

# NAME

irtt-client - Isochronous Round-Trip Time Client

# SYNOPSIS

irtt client [*args*]

# DESCRIPTION

"irtt client" is the client for irtt(1).

# OPTIONS

-d *duration*
:   Total time to send (default 1m0s, see Duration units below)

-i *interval*
:   Send interval (default 1s, see Duration units below)

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
    duration     | fixed duration (see Duration units below)

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
    *host*      | Host with dynamic port, see Host formats below
    *host:port* | Host with specified port, see Host formats below

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
    Comma separated list of durations (see Duration units below).
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
IPv6 IP         | [fe80::426c:8fff:fe13:9feb%en0]
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

# RETURN VALUE

# EXAMPLES

# SEE ALSO

irtt(1), irtt-server(1)

[IRTT GitHub repository](https://github.com/peteheist/irtt/)
