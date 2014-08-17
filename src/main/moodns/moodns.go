/*
 * Minimal multicast DNS server.
 *
 * Copyright (c) 2014, Alessandro Ghedini
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *     * Redistributions of source code must retain the above copyright
 *       notice, this list of conditions and the following disclaimer.
 *
 *     * Redistributions in binary form must reproduce the above copyright
 *       notice, this list of conditions and the following disclaimer in the
 *       documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS
 * IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
 * THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
 * PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR
 * CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
 * EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
 * PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
 * PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
 * LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
 * NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package main

import "log"
import "os"
import "strings"

import "github.com/docopt/docopt-go"

import "mdns"

func main() {
	log.SetFlags(0);

	usage := `Usage: moodns [options]

Options:
  -H <hostname>, --host <hostname>      Name of the local host.
  -l <addr:port>, --listen <addr:port>  Listen on this local address and port [default: 0.0.0.0:5353].
  -r, --enable-multicast-forward        Enable forwarding of unicast requests to multicast.
  -s, --silent                          Print fatal errors only.
  -h, --help                            Show the program's help message and exit.`

	args, err := docopt.Parse(usage, nil, true, "", false)
	if err != nil {
		log.Fatalf("Invalid arguments: %s", err);
	}

	listen := args["--listen"].(string);

	hostname, _ := os.Hostname();
	if args["--host"] != nil {
		hostname = args["--host"].(string);
	}

	localname := hostname + ".local.";
	silent    := args["--silent"].(bool);
	forward   := args["--enable-multicast-forward"].(bool);

	for _, addr := range strings.Split(listen, ",") {
		maddr, server, err := mdns.NewServer(addr);
		if err != nil {
			log.Fatalf("Error starting server: %s", err);
		}

		go mdns.Serve(server, maddr, localname, silent, forward);
	}

	select {}
}
