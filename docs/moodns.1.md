moodns(1) -- minimal multicast DNS server
=========================================

## SYNOPSIS

`moodns [OPTIONS]`

## DESCRIPTION

**moodns** is a server implementation of multicast DNS. Multicast DNS allows
programs to discover hosts running on a local network by using familiar DNS
programming interfaces without the need for a conventional DNS server.

## OPTIONS ##
`-H, --host`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
Name of the local host. If no name is provided, moodns will retrieve the local
computer hostname.

`-l, --listen`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
Listen on this address:port [default: 0.0.0.0:5353].

`-s, --silent`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
Print fatal errors only.

`-h, --help`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
Show the program's help message and exit.

## AUTHOR ##

Alessandro Ghedini <alessandro@ghedini.me>

## COPYRIGHT ##

Copyright (C) 2014 Alessandro Ghedini <alessandro@ghedini.me>

This program is released under the 2 clause BSD license.
