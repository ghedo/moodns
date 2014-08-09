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

import "net"

func NewQD(name string, t Type, class Class) (*Question) {
	qd := new(Question);

	qd.Name = append(qd.Name, name...);
	qd.Name = append(qd.Name, '.');

	qd.Type  = t;
	qd.Class = class;

	return qd;
}

func NewA(addr net.IP) *A {
	a := new(A);

	a.Addr = addr.To4();

	return a;
}

func NewAAAA(addr net.IP) *AAAA {
	a := new(AAAA);

	a.Addr = addr.To16();

	return a;
}

func NewCNAME(cname string) *CNAME {
	a := new(CNAME);

	a.CNAME = []byte(cname);

	return a;
}

func (msg *Message) AppendQD(qd *Question) {
	msg.Question = append(msg.Question, qd);
	msg.Header.QDCount++;
}

func (msg *Message) AppendAN(qd *Question, rdata RData, ttl uint32) {
	if rdata == nil {
		return;
	}

	an := new(Answer);

	an.Name  = qd.Name;

	switch rdata.(type) {
		case *A:
			an.Type = TypeA;

		case *AAAA:
			an.Type = TypeAAAA;
	}

	an.Class = qd.Class;
	an.TTL   = ttl;
	an.RData = rdata;
	an.RDLen = rdata.Len();

	msg.Answer = append(msg.Answer, an);
	msg.Header.ANCount++;
}
