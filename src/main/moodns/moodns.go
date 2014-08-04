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

import "github.com/docopt/docopt-go"

import "mcast"
import "mdns"

func main() {
	log.SetFlags(0);

	usage := `Usage: moodns [options]

Options:
  -H <hostname>, --host <hostname>      Name of the local host.
  -l <addr:port>, --listen <addr:port>  Listen on this address and port [default: 0.0.0.0:5353].
  -s, --silent                          Print fatal errors only.
  -h, --help                            Show the program's help message and exit.`

	args, err := docopt.Parse(usage, nil, true, "", false)
	if err != nil {
		log.Fatal("Invalid arguments: ", err);
	}

	listen   := args["--listen"].(string);
	hostname := "";

	if args["--host"] == nil {
		hostname, err = os.Hostname();
		if err != nil {
			log.Fatal("Error retrieving hostname: ", err);
		}
	} else {
		hostname = args["--host"].(string);
	}

	localname := hostname + ".local.";

	maddr, server, err := mcast.NewServer(listen, "224.0.0.251:5353");
	if err != nil {
		log.Fatal("Error starting server: ", err);
	}

	pkt := make([]byte, 65536);

	for {
		n, local4, local6, client, err := mcast.Read(server, pkt);
		if err != nil {
			if args["--silent"].(bool) != true {
				log.Print("Error reading from network: ", err);
			}
		}

		req, err := mdns.Unpack(pkt[:n]);
		if err != nil {
			if args["--silent"].(bool) != true {
				log.Print("Error unpacking request: ", err);
			}
			continue;
		}

		if len(req.Question) == 0 || len(req.Answer) > 0 {
			continue;
		}

		rsp := new(mdns.Message);

		rsp.Header.Flags |= mdns.FlagQR;
		rsp.Header.Flags |= mdns.FlagAA;

		if req.Header.Flags & mdns.FlagRD != 0 {
			rsp.Header.Flags |= mdns.FlagRD;
			rsp.Header.Flags |= mdns.FlagRA
		}

		if client.Port != 5353 {
			rsp.Header.Id = req.Header.Id;
		}

		for _, q := range req.Question {
			if string(q.Name) != localname {
				continue;
			}

			switch (q.Type) {
				case mdns.TypeA:
					an := mdns.NewA(local4.IP);
					rsp.AppendAN(q, an);

				case mdns.TypeAAAA:
					an := mdns.NewAAAA(local6.IP);
					rsp.AppendAN(q, an);

				default:
					continue;
			}

			if client.Port != 5353 {
				rsp.Question = append(rsp.Question, q);
				rsp.Header.QDCount++;
			}
		}

		if len(rsp.Answer) == 0 {
			continue;
		}

		out, err := mdns.Pack(rsp);
		if err != nil {
			log.Fatal("Error packing response: ", err);
		}

		if client.Port == 5353 {
			client = maddr;
		}

		_, err = server.WriteToUDP(out, client);
		if err != nil {
			log.Fatal("Error writing to network: ", err);
		}
	}
}
