moodns
======

![Travis CI](https://secure.travis-ci.org/ghedo/moodns.png)

**moodns** is a server implementation of [multicast DNS] [rfc]. Multicast DNS
allows programs to discover hosts running on a local network by using familiar
DNS programming interfaces without the need for a conventional DNS server.

[rfc]: http://tools.ietf.org/html/rfc6762

### FEATURES

* No configuration required.
* Support for forwarding unicast DNS queries to multicast servers which can be
  used for transparently enabling client-side multicast DNS with minimal config
  changes (see caveats below).
* No dependencies (e.g. dbus) required.

## GETTING STARTED

There's not much to it, just run:

```bash
$ moodns
```

moodns will start answering multicast DNS queries for the local host (i.e. those
asking for the `$HOST.local` domain name, unless an alternative hostname is
provided).

In order to enable multicast DNS on a client computer one can either install
[nss-mdns] [nss] (recommended), or enable the forwarding of unicast requests in
moodns.

"Multicast forwarding" allows moodns to receive DNS queries via unicast (like
any traditional DNS server) and automatically forward those queries to other
hosts via multicast DNS (essentially, moodns will receive the unicast query,
forward it to the multicast DNS address, receive the multicast response and
forward that response back to the client via unicast).

This mode can be enabled by running moodns with the `--enable-multicast-forward`
option and then configuring the client computer to use moodns as DNS server.

Start moodns like this:

```bash
$ sudo moodns --listen ':5353,:53' --enable-multicast-forward
```

Then edit the `/etc/resolv.conf` file and the add `nameserver 127.0.0.1` to the
top of the file. Note that the `--listen ':5353,:53'` option is needed because
some system DNS resolvers (e.g. glibc's) do not support setting the name server
port and always use `53`. If your system resolver supports changing the port
(e.g. Mac OS X, Solaris, OpenBSD) you don't need that option.

moodns will only answer queries for `*.local` domain names, and if it receives
a query for another domain name it will return an error so that the querier will
fallback to another DNS server without waiting for the timeout to expire
(remember to configure `/etc/resolv.conf` with multiple `nameserver` lines).

Note however that this mode is **not recommended**. It's not part of the
multicast DNS specs and is really just a hack which may break your system in
unexpected ways. Unless you know what you are doing, use [nss-mdns] [nss]
instead.

See the [man page](http://ghedo.github.io/moodns/) for more information.

[nss]: http://0pointer.de/lennart/projects/nss-mdns/

## BUILDING

moodns is distributed as source code. Install with:

```bash
$ make
```

## COPYRIGHT

Copyright (C) 2014 Alessandro Ghedini <alessandro@ghedini.me>

See COPYING for the license.
