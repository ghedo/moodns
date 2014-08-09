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

package mdns

import "bytes"
import "fmt"
import "log"
import "math"
import "math/rand"
import "net"
import "time"

import "code.google.com/p/go.net/ipv4"

func NewServer(addr string, maddr string) (*net.UDPAddr, *ipv4.PacketConn, error) {
	saddr, err := net.ResolveUDPAddr("udp", addr);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not resolve address '%s': %s", addr, err);
	}

	smaddr, err := net.ResolveUDPAddr("udp", maddr);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not resolve address '%s': %s", maddr, err);
	}

	udp, err := net.ListenUDP("udp", saddr);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not listen: %s", err);
	}

	p := ipv4.NewPacketConn(udp);
	if err := p.JoinGroup(nil, smaddr); err != nil {
		return nil, nil, fmt.Errorf("Could not join group: %s", err);
	}

	err = p.SetTTL(1);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set TTL: %s", err);
	}

	err = p.SetMulticastLoopback(false);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set loop: %s", err);
	}

	err = p.SetControlMessage(ipv4.FlagInterface | ipv4.FlagDst, true);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set ctrlmsg: %s", err);
	}

	return smaddr, p, nil;
}

func NewClient(addr string, maddr string) (*net.UDPAddr, *ipv4.PacketConn, error) {
	saddr, err := net.ResolveUDPAddr("udp", addr);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not resolve address '%s': %s", addr, err);
	}

	smaddr, err := net.ResolveUDPAddr("udp", maddr);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not resolve address '%s': %s", maddr, err);
	}

	udp, err := net.ListenUDP("udp", saddr);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not listen: %s", err);
	}

	p := ipv4.NewPacketConn(udp);

	err = p.SetMulticastLoopback(false);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set loop: %s", err);
	}

	err = p.SetControlMessage(ipv4.FlagInterface | ipv4.FlagDst, true);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set ctrlmsg: %s", err);
	}

	return smaddr, p, nil;
}

func MakeResponse(client *net.UDPAddr, req *Message) (*Message) {
	rsp := new(Message);

	rsp.Header.Flags |= FlagQR;
	rsp.Header.Flags |= FlagAA;

	if req.Header.Flags & FlagRD != 0 {
		rsp.Header.Flags |= FlagRD;
		rsp.Header.Flags |= FlagRA;
	}

	if client.Port != 5353 {
		rsp.Header.Id = req.Header.Id;
	}

	return rsp;
}

func Read(p *ipv4.PacketConn) (*Message, net.IP, *net.IPNet, *net.IPNet, *net.UDPAddr, error) {
	var local4 *net.IPNet;
	var local6 *net.IPNet;

	pkt := make([]byte, 65536);

	n, cm, from, err := p.ReadFrom(pkt);
	if err != nil {
		return nil, nil, nil, nil, nil,
		  fmt.Errorf("Could not read: %s", err);
	}

	ifi, err := net.InterfaceByIndex(cm.IfIndex);
	if err != nil {
		return nil, nil, nil, nil, nil,
		    fmt.Errorf("Could not find if: %s", err);
	}

	addrs, err := ifi.Addrs();
	if err != nil {
		return nil, nil, nil, nil, nil,
		  fmt.Errorf("Could not find addrs: %s", err);
	}

	for _, a := range addrs {
		if a.(*net.IPNet).IP.To4() != nil {
			local4 = a.(*net.IPNet);
		} else {
			local6 = a.(*net.IPNet);
		}
	}

	req, err := Unpack(pkt[:n]);
	if err != nil {
		return nil, nil, nil, nil, nil,
		  fmt.Errorf("Could not unpack request: %s", err);
	}

	return req, cm.Dst, local4, local6, from.(*net.UDPAddr), err;
}

func Write(p *ipv4.PacketConn, addr *net.UDPAddr, msg *Message) (error) {
	pkt, err := Pack(msg);
	if err != nil {
		return fmt.Errorf("Could not pack response: %s", err);
	}

	_, err = p.WriteTo(pkt, nil, addr);
	if err != nil {
		return fmt.Errorf("Could not write to network: %s", err);
	}

	return nil;
}

func SendRequest(req *Message) (*Message, error) {
	maddr, client, err := NewClient("0.0.0.0:0", "224.0.0.251:5353");
	if err != nil {
		return nil, fmt.Errorf("Could not create client: %s", err);
	}
	defer client.Close();

	seconds := 3 * time.Second;
	timeout := time.Now().Add(seconds);

	err = Write(client, maddr, req);
	if err != nil {
		return nil, fmt.Errorf("Could not send request: %s", err);
	}

	client.SetReadDeadline(timeout);

	rsp, _, _, _, _, err := Read(client);
	if err != nil {
		return nil, fmt.Errorf("Could not read response: %s", err);
	}

	if rsp.Header.Id != req.Header.Id {
		return nil, fmt.Errorf("Wrong id: %d", rsp.Header.Id);
	}

	return rsp, nil;
}

func Serve(p *ipv4.PacketConn, maddr *net.UDPAddr, localname string, silent, forward bool) {
	var sent_id uint16;

	for {
		req, dest, local4, local6, client, err := Read(p);
		if err != nil {
			if silent != true {
				log.Println("Error reading request: ", err);
				continue;
			}
		}

		if req.Header.Flags & FlagQR != 0 {
			continue;
		}

		if sent_id > 0 && req.Header.Id == sent_id {
			continue;
		}

		rsp := MakeResponse(client, req);

		for _, q := range req.Question {
			if client.Port != 5353 {
				rsp.Question = append(rsp.Question, q);
				rsp.Header.QDCount++;
			}

			if string(q.Name) != localname {
				if dest.IsLoopback() && forward != false {
					sent_id, _ = MakeRecursive(q, rsp);
				}

				continue;
			}

			switch (q.Type) {
				case TypeA:
					an := NewA(local4.IP);
					rsp.AppendAN(q, an, 120);

				case TypeAAAA:
					an := NewAAAA(local6.IP);
					rsp.AppendAN(q, an, 120);

				default:
					continue;
			}
		}

		if rsp.Header.ANCount == 0 &&
		   rsp.Header.Flags.RCode() == RCodeOK {
			continue;
		}

		if client.Port == 5353 {
			client = maddr;
		}

		err = Write(p, client, rsp);
		if err != nil {
			if silent != true {
				log.Println("Error sending response: ", err);
				continue;
			}
		}
	}
}

func MakeRecursive(qd *Question, out *Message) (uint16, error) {
	if bytes.HasSuffix(qd.Name, []byte("local.")) != true {
		out.Header.Flags |= RCodeFmtErr;
		return 0, nil;
	}

	rand.Seed(time.Now().UTC().UnixNano());
	id := uint16(rand.Intn(math.MaxUint16));

	req := new(Message);
	req.Header.Id = id;
	req.AppendQD(qd);

	rsp, err := SendRequest(req);
	if err != nil {
		return 0, fmt.Errorf("Could not send request: %s", err);
	}

	for _, an := range rsp.Answer {
		out.Answer = append(out.Answer, an);
		out.Header.ANCount++;
	}

	return id, nil;
}
