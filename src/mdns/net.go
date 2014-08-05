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

import "fmt"
import "net"

func NewServer(addr string, maddr string) (*net.UDPAddr, *net.UDPConn, error) {
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

	err = SetTTL(udp, 1);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set TTL: %s", err);
	}

	err = SetLoop(udp, 0);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set loop: %s", err);
	}

	err = AddMembership(udp, smaddr.IP);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not join group: %s", err);
	}

	err = SetPktInfo(udp, 1);
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set PKTINFO: %s", err);
	}

	return smaddr, udp, nil;
}

func NewClient(addr string, maddr string) (*net.UDPAddr, *net.UDPConn, error) {
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

	return smaddr, udp, nil;
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

func Read(udp *net.UDPConn) (*Message, *net.IPNet, *net.IPNet, *net.UDPAddr, error) {
	var local4 *net.IPNet;
	var local6 *net.IPNet;

	pkt := make([]byte, 65536);
	oob := make([]byte, 40);

	n, oobn, _, from, err := udp.ReadMsgUDP(pkt, oob);
	if err != nil {
		return nil, nil, nil, nil,
		  fmt.Errorf("Could not read: %s", err);
	}

	if oobn > 0 {
		pktinfo := ParseOob(oob[:oobn]);

		if pktinfo != nil {
			ifi, err := net.InterfaceByIndex(int(pktinfo.Ifindex));
			if err != nil {
				return nil, nil, nil, nil,
				  fmt.Errorf("Could not find if: %s", err);
			}

			addrs, err := ifi.Addrs();
			if err != nil {
				return nil, nil, nil, nil,
				  fmt.Errorf("Could not find addrs: %s", err);
			}

			for _, a := range addrs {
				if a.(*net.IPNet).IP.To4() != nil {
					local4 = a.(*net.IPNet);
				} else {
					local6 = a.(*net.IPNet);
				}
			}
		}
	}

	req, err := Unpack(pkt[:n]);
	if err != nil {
		return nil, nil, nil, nil,
		  fmt.Errorf("Could not unpack request: %s", err);
	}

	return req, local4, local6, from, err;
}

func Write(udp *net.UDPConn, addr *net.UDPAddr, msg *Message) (error) {
	pkt, err := Pack(msg);
	if err != nil {
		return fmt.Errorf("Could not pack response: %s", err);
	}

	_, err = udp.WriteToUDP(pkt, addr);
	if err != nil {
		return fmt.Errorf("Could not write to network: %s", err);
	}

	return nil;
}
