moodns
======

![Travis CI](https://secure.travis-ci.org/ghedo/moodns.png)

**moodns** is a server implementation of [multicast DNS] [rfc]. Multicast DNS
allows programs to discover hosts running on a local network by using familiar
DNS programming interfaces without the need for a conventional DNS server.

[rfc]: http://tools.ietf.org/html/rfc6762

## GETTING STARTED

There's not much to it, just run:

```bash
$ moodns
```

moodns will start answering multicast DNS queries for the local host (i.e. those
asking for the `$HOST.local` domain name, unless an alternative hostname is
provided).

See the [man page](http://ghedo.github.io/moodns/) for more information.

## BUILDING

moodns is distributed as source code. Install with:

```bash
$ make
```

## COPYRIGHT

Copyright (C) 2014 Alessandro Ghedini <alessandro@ghedini.me>

See COPYING for the license.
