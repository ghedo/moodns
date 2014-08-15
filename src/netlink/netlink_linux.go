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

package netlink

// #include <linux/rtnetlink.h>
import "C"

import "fmt"
import "syscall"
import "unsafe"

type NetlinkListener struct {
	fd int;
	sa *syscall.SockaddrNetlink;
}

func ListenNetlink() (*NetlinkListener, error) {
	groups := C.RTMGRP_LINK        |
	          C.RTMGRP_IPV4_IFADDR |
	          C.RTMGRP_IPV6_IFADDR;

	s, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, syscall.NETLINK_ROUTE);
	if err != nil {
		return nil, fmt.Errorf("socket: %s", err);
	}

	saddr := new(syscall.SockaddrNetlink);
	saddr.Family = syscall.AF_NETLINK;
	saddr.Pid    = uint32(0);
	saddr.Groups = uint32(groups);

	err = syscall.Bind(s, saddr);
	if err != nil {
		return nil, fmt.Errorf("bind: %s", err);
	}

	l := new(NetlinkListener);
	l.fd = s;
	l.sa = saddr;

	return l, nil;
}

func (l *NetlinkListener) ReadMsgs() ([]syscall.NetlinkMessage, error) {
	defer func() {
		recover();
	}();

	pkt := make([]byte, 2048);

	n, err := syscall.Read(l.fd, pkt);
	if err != nil {
		return nil, fmt.Errorf("read: %s", err);
	}

	msgs, err := syscall.ParseNetlinkMessage(pkt[:n]);
	if err != nil {
		return nil, fmt.Errorf("parse: %s", err);
	}

	return msgs, nil;
}

func (l *NetlinkListener) SendRouteRequest(proto, family int) error {
	wb := newNetlinkRouteRequest(proto, 1, family);

	err := syscall.Sendto(l.fd, wb, 0, l.sa);
	if err != nil {
		return err;
	}

	return nil;
}

func toWireFormat(rr *syscall.NetlinkRouteRequest) []byte {
	b := make([]byte, rr.Header.Len);

	*(*uint32)(unsafe.Pointer(&b[0:4][0])) = rr.Header.Len;
	*(*uint16)(unsafe.Pointer(&b[4:6][0])) = rr.Header.Type;
	*(*uint16)(unsafe.Pointer(&b[6:8][0])) = rr.Header.Flags;
	*(*uint32)(unsafe.Pointer(&b[8:12][0])) = rr.Header.Seq;
	*(*uint32)(unsafe.Pointer(&b[12:16][0])) = rr.Header.Pid;

	b[16] = byte(rr.Data.Family);
	return b;
}

func newNetlinkRouteRequest(proto, seq, family int) []byte {
	rr := &syscall.NetlinkRouteRequest{};
	rr.Header.Len = uint32(syscall.NLMSG_HDRLEN + syscall.SizeofRtGenmsg);
	rr.Header.Type = uint16(proto);
	rr.Header.Flags = syscall.NLM_F_DUMP | syscall.NLM_F_REQUEST;
	rr.Header.Seq = uint32(seq);
	rr.Data.Family = uint8(family);

	return toWireFormat(rr);
}

func IsNewAddr(msg *syscall.NetlinkMessage) bool {
	if msg.Header.Type == C.RTM_NEWADDR {
		return true;
	}

	return false;
}

func IsDelAddr(msg *syscall.NetlinkMessage) bool {
	if msg.Header.Type == C.RTM_DELADDR {
		return true;
	}

	return false;
}

func IsRelevant(msg *syscall.IfAddrmsg) bool {
	if msg.Scope == C.RT_SCOPE_UNIVERSE ||
	   msg.Scope == C.RT_SCOPE_SITE {
		return true;
	}

	return false;
}
