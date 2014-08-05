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

import "github.com/docopt/docopt-go"

import "mdns"

func main() {
	log.SetFlags(0);

	usage := `Usage: moodns-resolve [options] <name>

Options:
  -6, --ipv6  Request IPv6 address too [default: false].
  -h, --help  Show the program's help message and exit.`

	args, err := docopt.Parse(usage, nil, true, "", false)
	if err != nil {
		log.Fatal("Invalid arguments: ", err);
	}

	name := args["<name>"].(string);

	maddr, client, err := mdns.NewServer("0.0.0.0:0", "224.0.0.251:5353");
	if err != nil {
		log.Fatal("Error creating client: ", err);
	}

	req := new(mdns.Message);

	req.AppendQD(mdns.NewQD(name, mdns.TypeA, mdns.ClassInet));

	if args["--ipv6"].(bool) {
		req.AppendQD(mdns.NewQD(name, mdns.TypeAAAA, mdns.ClassInet));
	}

	err = mdns.Write(client, maddr, req);
	if err != nil {
		log.Fatal("Error sending request: ", err);
	}

	rsp, _, _, _, err := mdns.Read(client);
	if err != nil {
		log.Fatal("Error reading request: ", err);
	}

	log.Println(rsp);
}
