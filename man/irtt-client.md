IRTT-CLIENT 1 "February 3, 2018" "IRTT 0.9" "Isochronous Round-Trip Time Manual"
================================================================================

NAME
----

irtt-client - Isochronous Round-Trip Time client

SYNOPSIS
--------

`foo` [`-bar`] [`-c` *config-file*] *file* ...

DESCRIPTION
-----------

`foo` frobnicates the bar library by tweaking internal symbol tables. By
default it parses all baz segments and rearranges them in reverse order by
time for the xyzzy(1) linker to find them. The symdef entry is then compressed
using the WBG (Whiz-Bang-Gizmo) algorithm. All files are processed in the
order specified.

First Header              | Second Header
------------------------- | -------------
Content Cell              | Content Cell
Content Cell that's wider | Content Cell

OPTIONS
-------

`-b`
  Do not write "busy" to stdout while processing.

`-c` *config-file*
  Use the alternate system wide *config-file* instead of */etc/foo.conf*. This
  overrides any `FOOCONF` environment variable.

`-a`
  In addition to the baz segments, also parse the blurfl headers.

`-r`
  Recursive mode. Operates as fast as lightning at the expense of a megabyte
  of virtual memory.

BUGS
----

The command name should have been chosen more carefully to reflect its
purpose.

AUTHOR
------

Pete Heist <pete@eventide.io>

SEE ALSO
--------

irtt(1), irtt-server(1)
[IRTT GitHub repository](https://github.com/peteheist/irtt/)
