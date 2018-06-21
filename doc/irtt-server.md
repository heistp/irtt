% IRTT-SERVER(1) v0.9.0 | IRTT Manual
%
% February 11, 2018

# NAME

irtt-server - Isochronous Round-Trip Time Server

# SYNOPSIS

irtt server [*args*]

# DESCRIPTION

*irtt server* is the server for [irtt(1)](irtt.html).

# OPTIONS

-b *addresses*
:   Bind addresses (default ":2112"), comma separated list of:

    Format      | Address Type
    ----------- | ------------
    :port       | unspecified address with port, use with care
    host        | host with default port 2112, see [Host formats](#host-formats) below
    host:port   | host with specified port, see [Host formats](#host-formats) below
    %iface      | all addresses on interface iface with default port 2112
    %iface:port | all addresses on interface iface with port

    **Note:** iface strings may contain * to match multiple interfaces

-d *duration*
:   Max test duration, or 0 for no maximum (default 0s, see
    [Duration units](#duration-units) below)

-i *interval*
:   Min send interval, or 0 for no minimum (default 10ms, see
    [Duration units](#duration-units) below)

-l *length*
:   Max packet length (default 0), or 0 for no maximum. Numbers less than
    size of required headers will cause test packets to be dropped.

\--hmac=*key*
:   Add HMAC with *key* (0x for hex) to all packets, provides:

    - Dropping of all packets without a correct HMAC
    - Protection for server against unauthorized discovery and use

\--syslog=*uri*
:   Log events to syslog (default don't use syslog). URI format:
    protocol://host:port/tag. Examples:

    URI                      | Result
    ------------------------ | ------
    local                    | Log to local syslog, default tag irtt
    udp://logsrv:514/irttsrv | UDP to logsrv:514, tag irttsrv
    tcp://logsrv:8514/       | TCP to logsrv:8514, default tag irtt

\--timeout=*duration*
:   Timeout for closing connections if no requests received on a
    connection (default 1m0s, see [Duration units](#duration-units) below).
    0 means no timeout (not recommended, especially on public servers).
    Max client interval will be restricted to timeout/4.

\--pburst=*#*
:   Packet burst allowed before enforcing minimum interval (default 5)

\--fill=*fill*
:   Payload fill if not requested (default pattern:69727474). Possible values
    include:

    Value        | Fill
    ------------ | ------------
    *none*       | Echo client payload (insecure on public servers)
    *rand*       | Use random bytes from Go's math.rand
    *pattern:*XX | Use repeating pattern of hex (default 69727474)

\--allow-fills=*fills*
:   Comma separated patterns of fill requests to allow (default rand). See
    options for *--fill*. Notes:

    - Patterns may contain * for matching
    - Allowing non-random fills insecure on public servers
    - Use *\--allow-fills=""* to disallow all fill requests

\--tstamp=*modes*
:   Timestamp modes to allow (default dual). Possible values:

    Value    | Allowed Timestamps
    -------- | ------------------
    *none*   | Don't allow any timestamps
    *single* | Allow a single timestamp (send, receive or midpoint)
    *dual*   | Allow dual timestamps

\--no-dscp
:   Don't allow setting dscp (default false)

\--set-src-ip
:   Set source IP address on all outgoing packets from listeners on
    unspecified IP addresses (use for more reliable reply routing, but
    increases per-packet heap allocations)

\--gc=*mode*
:   Sets garbage collection mode (default on). Possible values:

    Value  | Meaning
    ------ | ------------------
    *on*   | Garbage collector always on
    *off*  | Garbage collector always off
    *idle* | Garbage collector enabled only when idle

\--thread
:   Lock request handling goroutines to OS threads

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

# SECURITY

Running an IRTT server that's open to the outside world requires some additional
attention. For starters, the command line flags should be used to, at a minimum:

- Restrict the duration (*-d*), interval (*-i*) and length (*-l*) of tests,
  particularly for public servers
- Set an HMAC key (*\--hmac*) for private servers to prevent unauthorized
  discovery and use

In addition, there are various systemd(1) options available for securing
services. The irtt.service file included with the distribution sets some
commonly used options, but should not be considered exhaustive.

To secure a server for public use, additional steps may be taken that are
outside of the scope of this documentation, including but not limited to:

- Installing irtt in an unprivileged container
- Setting up an iptables firewall (only UDP port 2112 must be open)
- Setting up a chroot jail

It should be noted that there are no known security vulnerabilities in the Go
language at this time, and the steps above, in particular the chroot jail, may
or may not serve to enhance security in any way. Go-based servers are generally
regarded as safe because of Go's high-level language constructs for memory
management, and at this time IRTT makes no use of Go's
[unsafe](https://golang.org/pkg/unsafe/) package.

# EXIT STATUS

*irtt server* exits with one of the following status codes:

Code | Meaning
---- | -------
0    | Success
1    | Runtime error
2    | Command line error
3    | Two interrupt signals received

# EXAMPLES

$ irtt server
:   Starts the server and listens on all addresses (unspecified address)

$ irtt server -d 30s -i 20ms -l 256 \--fill=rand \--allow-fills=""
:   Starts the server and listens on all addresses, setting the maximum
    test duration to 30 seconds, minimum interval to 20 ms, and maximum
    packet length to 256 bytes. Disallows fill requests and forces all
    return packets to be filled with random data.

$ irtt server -b 192.168.100.11:64381 \--hmac=secret
:   Starts the server and binds to IPv4 address 192.168.100.11, port 64381.
    Requires a valid HMAC on all packets with the key *secret*, otherwise
    packets are dropped.

# SEE ALSO

[irtt(1)](irtt.html), [irtt-client(1)](irtt-client.html)

[IRTT GitHub repository](https://github.com/heistp/irtt/)
