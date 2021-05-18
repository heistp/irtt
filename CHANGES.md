# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

## 0.9.1 - 2021-05-18

### Added

- Improve Windows time support by using `QueryPerformanceCounter` and
  `GetSystemTimePreciseAsFileTime`. Time precision on Windows 10 can now
  reach 100ns.
- Add syslog support for the server (`--syslog` flag).
- Add a [SmokePing](https://oss.oetiker.ch/smokeping/) probe, available from
  SmokePing 2.7.2. It's possible to copy the
  [code](https://github.com/oetiker/SmokePing/blob/master/lib/Smokeping/probes/IRTT.pm)
  into older versions.
- Update OM2P build to use the new
  [MIPS softfloat support](https://github.com/golang/go/issues/18162) in Go 1.10.

### Changed

- Re-write git history with new email address.
- Re-licensed to GPLv2.

### Removed

- Stop disabling GC on client and remove `--gc` flag from server as disabling GC
  no longer offers a demonstrable performance benefit. See
  [The Journey of Go's Garbage Collector](https://blog.golang.org/ismmkeynote).

### Fixed

- Fix potential client race at startup.
- Fix issue on platforms with timer resolution not accurate enough
  to measure server processing time (closes #13).
- Change handled interrupt signals to only `os.Interrupt` and `syscall.SIGTERM`
  to fix Plan 9 build.
- Various documentation fixes.

## 0.9.0 - 2018-02-11

### Added

- Server fills are now supported, and may be restricted on the server. See
  `--sfill` for the client and `--allow-fills` on the server. As an example, one
  can do `irtt client --fill=rand --sfill=rand -l 172 server` for random
  payloads in both directions. The server default is `--allow-fills=rand` so
  that arbitrary data cannot be relayed between two hosts. `server_fill` now
  appears under `params` in the JSON.
- Version information has been added to the JSON output.

### Changed

- Due to adoption of the [pflag](https://github.com/ogier/pflag) package, all long
  options now start with -- and must use = with values (e.g. `--fill=rand`).
  After the subcommand, flags and arguments may now appear in any order.
- `irtt client` syntax changes:
  - `-rs` is renamed to `--stats`
  - `-strictparams` is removed and is now the default. `--loose` may be used
    instead to accept and use server restricted parameters, with a warning.
  - `-ts` is renamed to `--tstamp`
  - `-qq` is renamed to `-Q`
  - `-fillall` is removed and is now the default. `--fill-one` may be used as
    a small optimization, but should rarely be needed.
- `irtt server` syntax changes:
  - `-nodscp` is renamed to `--no-dscp`
  - `-setsrcip` is renamed to `--set-src-ip`
- The communication protocol has changed, so clients and servers must both be
  updated.
- The handshake now includes a protocol version, which may change independently
  of the release version, and must match exactly between the client and server
  or the client will refuse to connect.
- The default server minimum interval is now `10ms`.
- The default client duration has been changed from `1h` to `1m`.
- Some author info was changed in the commit history, so the rewritten history
  must be fetched in all forks and any changes rebased.

## 0.1.0 - 2017-10-15

### Added

- Initial, untagged development release.
