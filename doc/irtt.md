% IRTT(1) v0.9 | IRTT Manual
%
% February 4, 2018

# NAME

irtt - Isochronous Round-Trip Time

# SYNOPSIS

irtt *command* [*args*]

irtt help *command*

# DESCRIPTION

IRTT measures round-trip time and other latency related metrics using UDP
packets sent on a fixed period, and produces both text and JSON output.

# COMMANDS

*client*
:   runs the client

*server*
:   runs the server

*bench*
:   runs HMAC and fill benchmarks

*clock*
:   runs wall vs monotonic clock test

*sleep*
:   runs sleep accuracy test

*version*
:   shows the version

# EXAMPLE

After installing IRTT, start a server:

```
% irtt server
IRTT server starting...
[ListenerStart] starting IPv6 listener on [::]:2112
[ListenerStart] starting IPv4 listener on 0.0.0.0:2112
```

While that's running, run a client. If no options are supplied, it will send
a request once per second, like ping. Here we simulate a one minute
G.711 VoIP conversation by using an interval of 20ms and randomly filled
payloads of 172 bytes:

```
% irtt client -i 20ms -l 172 -d 1m \
    --fill=rand --sfill=rand -q 192.168.100.10
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

In the results above, the client and server are located at two different sites,
around 50km from one another, each of which connects to the Internet via
point-to-point WiFi. The client is 3km NLOS through trees located near its
transmitter, which is likely the reason for the higher upstream packet loss, mean
send delay and IPDV.

# BUGS

- Windows is unable to set DSCP values for IPv6.
- Windows is unable to set the source IP address, so `--set-src-ip` may not be used
  on the server.
- The server doesn't run well on 32-bit Windows platforms. When connecting with
  a client, you may see `Terminated due to receive error`. To work around
  this, disable dual timestamps from the client by including `--tstamp=midpoint`.

# LIMITATIONS

> "It is the limitations of software that give it life." -Me, justifying my limitations

## Isochronous (fixed period) send schedule

Currently, IRTT only sends packets on a fixed period, foregoing the ability to
simulate arbitrary traffic. Accepting this limitation offers some benefits:

- It's easy to implement
- It's easy to calculate how many packets and how much data will be sent in a given time
- It simplifies timer error compensation

Also, isochronous packets are commonly seen in VoIP, games and some streaming media,
so it already simulates an array of common types of traffic.

## Fixed packet lengths for a given test

Packet lengths are fixed for the duration of the test. While this may not be an
accurate simulation of some types of traffic, it means that IPDV measurements
are accurate, where they wouldn't be in any other case.

## Stateful protocol

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

## In-memory results storage

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
over 300 Gb of virtual memory to record while the test is running. That is
why 64-bit sequence numbers are currently unnecessary.

## 64-bit received window

In order to determine per-packet differentiation between upstream and downstream
loss, a 64-bit "received window" may be returned with each packet that contains
the receipt status of the previous 64 packets. This can be enabled using
`--stats=window/both` with the irtt client. Its limited width and simple bitmap
format lead to some caveats:

- Per-packet differentiation is not available (for any intervening packets) if
	greater than 64 packets are lost in succession. These packets will be marked
	with the generic `Lost`.
- While any packet marked `LostDown` is guaranteed to be marked properly, there
	is no confirmation of receipt of the receive window from the client to the
	server, so packets may sometimes be erroneously marked `LostUp`, for example,
	if they arrive late to the server and slide out of the received window before
	they can be confirmed to the client, or if the received window is lost on its
	way to the client and not amended by a later packet's received window.

There are many ways that this simple approach could be improved, such as by:

- Allowing a wider window
- Encoding receipt seqnos in a more intelligent way to allow a wider seqno range
- Sending confirmation of window receipt from the client to the server and
  re-sending unreceived windows

However, the current strategy means that a good approximation of per-packet loss
results can be obtained with only 8 additional bytes in each packet. It also
requires very little computational time on the server, and almost all
computation on the client occurs during results generation, after the test is
complete. It isn't as accurate with late (out-of-order) upstream packets or with
long sequences of lost packets, but high loss or high numbers of late packets
typically indicate more severe network conditions that should be corrected first
anyway, perhaps before per-packet results matter. Note that in case of very high
packet loss, the **total** number of packets received by the server but not
returned to the client (which can be obtained using `--stats=count`) will still
be correct, which will still provide an accurate **average** loss percentage in
each direction over the course of the test.

# NOTES

Latency is an under-appreciated metric in network and application performance.
There is a certain hard to quantify but visceral "latency stress" that comes
from waiting in expectation after a web page click, straining through a delayed
and garbled VoIP conversation, or losing at your favorite online game (unless
you like "lag" as an excuse). As of this writing, many broadband connections are
well past the point of diminishing returns when it comes to throughput, yet
that's what we continue to take as the primary measure of Internet performance.
This is analogous to ordinary car buyers making top speed their first priority.

# SEE ALSO

irtt-client(1), irtt-server(1)

[IRTT GitHub repository](https://github.com/peteheist/irtt/)

# AUTHOR

Pete Heist <pete@eventide.io>

Many thanks to both Toke Høiland-Jørgensen and Dave Täht from the
[Bufferbloat project](https://www.bufferbloat.net/) for their valuable advice.
Any problems in design or implementation are entirely my own.

# HISTORY

IRTT was originally written to improve the latency and packet loss measurements
for the excellent [Flent](https://flent.org) tool. Flent was developed by and
for the [Bufferbloat](https://www.bufferbloat.net/projects/) project, which aims
to reduce "chaotic and laggy network performance," making this project valuable
for anyone who values their time and sanity while using the Internet.
