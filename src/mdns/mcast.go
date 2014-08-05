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
import "syscall"
import "unsafe"

func SetTTL(udp *net.UDPConn, value int) error {
	return SetsockoptInt(udp, syscall.IP_MULTICAST_TTL, value);
}

func SetLoop(udp *net.UDPConn, value int) error {
	return SetsockoptInt(udp, syscall.IP_MULTICAST_LOOP, value);
}

func AddMembership(udp *net.UDPConn, addr net.IP) error {
	var mreq syscall.IPMreq;

	for i := 0; i < net.IPv4len; i++ {
		mreq.Multiaddr[i] = addr.To4()[i];
	}

	file, err := udp.File();
	if err != nil {
		return fmt.Errorf("Could not get socket file: %s", err);
	}

	fd := file.Fd();

	err = syscall.SetsockoptIPMreq(
		int(fd), syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP, &mreq,
	);
	if err != nil {
		return fmt.Errorf("Could not set socket opt: %s", err);
	}

	return nil;
}

func SetPktInfo(udp *net.UDPConn, value int) error {
	return SetsockoptInt(udp, syscall.IP_PKTINFO, value);
}

func SetsockoptInt(udp *net.UDPConn, opt int, value int) error {
	file, err := udp.File();
	if err != nil {
		return fmt.Errorf("Could not get socket file: %s", err);
	}

	fd := file.Fd();

	err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, opt, value);
	if err != nil {
		return fmt.Errorf("Could not set socket opt: %s", err);
	}

	return nil;
}

func ParseOob(oob []byte) (*syscall.Inet4Pktinfo) {
	cmsgs, err := syscall.ParseSocketControlMessage(oob);
	if err != nil {
		fmt.Println("error parsing");
	}

	for _, m := range cmsgs {
		if m.Header.Level != 0 {
			continue
		}

		switch (m.Header.Type) {
			case syscall.IP_PKTINFO:
				return ParsePktInfo(m.Data);
		}
	}

	return nil;
}

func ParsePktInfo(b []byte) (*syscall.Inet4Pktinfo) {
	return (*syscall.Inet4Pktinfo)(unsafe.Pointer(&b[0]));
}