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

import "bytes"
import "fmt"
import "log"
import "math"
import "math/rand"
import "os"
import "time"

import "github.com/docopt/docopt-go"

import "mdns"

func main() {
	log.SetFlags(0);

	usage := `Usage: moodns [options]

Options:
  -H <hostname>, --host <hostname>      Name of the local host.
  -l <addr:port>, --listen <addr:port>  Listen on this address and port [default: 0.0.0.0:5353].
  -r, --enable-multicast-forward        Enable forwarding of unicast requests to multicast.
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

	maddr, server, err := mdns.NewServer(listen, "224.0.0.251:5353");
	if err != nil {
		log.Fatal("Error starting server: ", err);
	}

	var sent_id uint16;

	for {
		req, local4, local6, client, err := mdns.Read(server);
		if err != nil {
			if args["--silent"].(bool) != true {
				log.Println("Error reading request: ", err);
				continue;
			}
		}

		if req.Header.Flags & mdns.FlagQR != 0 {
			continue;
		}

		if sent_id > 0 && req.Header.Id == sent_id {
			continue;
		}

		rsp := mdns.MakeResponse(client, req);

		for _, q := range req.Question {
			if client.Port != 5353 {
				rsp.Question = append(rsp.Question, q);
				rsp.Header.QDCount++;
			}

			if string(q.Name) != localname {
				if args["--enable-multicast-forward"].(bool) != false {
					sent_id, _ = MakeRecursive(q, rsp);
				}

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
		}

		if rsp.Header.ANCount == 0 &&
		   rsp.Header.Flags.RCode() == mdns.RCodeOK {
			continue;
		}

		if client.Port == 5353 {
			client = maddr;
		}

		err = mdns.Write(server, client, rsp);
		if err != nil {
			if args["--silent"].(bool) != true {
				log.Println("Error sending response: ", err);
				continue;
			}
		}
	}
}

func MakeRecursive(qd *mdns.Question, out *mdns.Message) (uint16, error) {
	if bytes.HasSuffix(qd.Name, []byte("local.")) != true {
		out.Header.Flags |= mdns.RCodeFmtErr;
		return 0, nil;
	}

	rand.Seed(time.Now().UTC().UnixNano());
	id := uint16(rand.Intn(math.MaxUint16));

	req := new(mdns.Message);
	req.Header.Id = id;
	req.AppendQD(qd);

	rsp, err := mdns.SendRequest(req);
	if err != nil {
		return 0, fmt.Errorf("Could not send request: %s", err);
	}

	for _, an := range rsp.Answer {
		out.Answer = append(out.Answer, an);
		out.Header.ANCount++;
	}

	return id, nil;
}
