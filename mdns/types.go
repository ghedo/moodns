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
import "net"
import "strings"
import "syscall"

type Flags uint16

const (
	FlagQR Flags = 32768
	FlagAA       = 1024
	FlagTC       = 512
	FlagRD       = 256
	FlagRA       = 128
	FlagAD       = 32
	FlagCD       = 16

	RCodeNoError  = 0
	RCodeFormErr  = 1
	RCodeServFail = 2
	RCodeNXDomain = 3
	RCodeNotImpl  = 4
	RCodeRefused  = 5
)

func (t Flags) RCode() uint8 {
	flags := t

	flags <<= 12
	flags >>= 12

	return uint8(flags)
}

func (t Flags) RCodeString() string {
	if t & RCodeFormErr != 0 {
		return "FORMERROR"
	}

	if t & RCodeServFail != 0 {
		return "SERVFAIL"
	}

	if t & RCodeNXDomain != 0 {
		return "NXDOMAIN"
	}

	if t & RCodeNotImpl != 0 {
		return "NOTIMPL"
	}

	if t & RCodeRefused != 0 {
		return "REFUSED"
	}

	return "NOERROR"
}

func (t Flags) String() string {
	var s []string

	if t & FlagQR != 0 {
		s = append(s, "qr")
	}

	if t & FlagAA != 0 {
		s = append(s, "aa")
	}

	if t & FlagTC != 0 {
		s = append(s, "rc")
	}

	if t & FlagRD != 0 {
		s = append(s, "rd")
	}

	if t & FlagRA != 0 {
		s = append(s, "ra")
	}

	if t & FlagAD != 0 {
		s = append(s, "ad")
	}

	if t & FlagCD != 0 {
		s = append(s, "cd")
	}

	return strings.Join(s, " ")
}

type Type uint16

const (
	TypeNone  Type = 0
	TypeA          = 1
	TypeCNAME      = 5
	TypePTR        = 12
	TypeHINFO      = 13
	TypeTXT        = 16
	TypeAAAA       = 28
	TypeSRV        = 33
	TypeOPT        = 41
	TypeAny        = 255
)

func (t Type) MakeRR() RData {
	switch t {
	case TypeNone:
		return nil

	case TypeA:
		return new(A)

	case TypeCNAME:
		return new(CNAME)

	case TypePTR:
		return new(PTR)

	case TypeHINFO:
		return new(HINFO)

	case TypeTXT:
		return new(TXT)

	case TypeAAAA:
		return new(AAAA)

	case TypeSRV:
		return new(SRV)

	case TypeOPT:
		return new(OPT)

	case TypeAny:
		return nil

	default:
		return nil
	}
}

func (t Type) String() string {
	switch t {
	case TypeNone:
		return "NONE"

	case TypeA:
		return "A"

	case TypeCNAME:
		return "CNAME"

	case TypePTR:
		return "PTR"

	case TypeHINFO:
		return "HINFO"

	case TypeTXT:
		return "TXT"

	case TypeAAAA:
		return "AAAA"

	case TypeSRV:
		return "SRV"

	case TypeOPT:
		return "OPT"

	case TypeAny:
		return "ANY"

	default:
		return "Unknown"
	}
}

type Class uint16

const (
	ClassInet    Class = 1
	ClassNone          = 254
	ClassAny           = 255
	ClassUnicast       = 1 << 15
)

func (c Class) String() string {
	switch c {
	case ClassInet, ClassInet | ClassUnicast:
		return "IN"

	case ClassNone:
		return "NONE"

	case ClassAny:
		return "ANY"

	default:
		return "Unknown"
	}
}

type Message struct {
	Header     Header
	Question   []*Question
	Answer     []*Record
	Authority  []*Record
	Additional []*Record
}

func (msg *Message) AppendQD(qd *Question) {
	msg.Question = append(msg.Question, qd)
	msg.Header.QDCount++
}

func (msg *Message) AppendAN(an *Record) {
	msg.Answer = append(msg.Answer, an)
	msg.Header.ANCount++
}

func (m *Message) String() string {
	b := new(bytes.Buffer)

	fmt.Fprintf(b, ";;")
	fmt.Fprintf(b, " opcode: %d,", 255)
	fmt.Fprintf(b, " status: %s,", m.Header.Flags.RCodeString())
	fmt.Fprintf(b, " id: %d", m.Header.Id)
	fmt.Fprintf(b, "\n")

	fmt.Fprintf(b, ";;")
	fmt.Fprintf(b, " flags: %s;", m.Header.Flags)
	fmt.Fprintf(b, " QUERY: %d,", m.Header.QDCount)
	fmt.Fprintf(b, " ANSWER: %d,", m.Header.ANCount)
	fmt.Fprintf(b, " AUTHORITY: %d,", m.Header.NSCount)
	fmt.Fprintf(b, " ADDITIONAL: %d", m.Header.NSCount)
	fmt.Fprintf(b, "\n\n")

	if m.Header.QDCount > 0 {
		fmt.Fprintf(b, ";; QUESTION SECTION:\n")
	}

	for _, qd := range m.Question {
		fmt.Fprintf(b, ";%s\t\t\t%s\t%s\n",
		            string(qd.Name), qd.Class, qd.Type)
	}

	if m.Header.QDCount > 0 {
		fmt.Fprintln(b, "")
	}

	if m.Header.ANCount > 0 {
		fmt.Fprintf(b, ";; ANSWER SECTION:\n")
	}

	for _, an := range m.Answer {
		fmt.Fprintf(b, ";%s\t\t%d\t%s\t%s\t%s\n",
		            string(an.Name), an.TTL, an.Class,
			    an.Type, an.RData)
	}

	if m.Header.ANCount > 0 {
		fmt.Fprintln(b, "")
	}

	return b.String()
}

type Header struct {
	Id    uint16
	Flags Flags

	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type Question struct {
	Name  []byte `mdns:"name"`
	Type  Type
	Class Class
}

func NewQD(name []byte, t Type, class Class) *Question {
	return &Question{
		Name:  name,
		Type:  t,
		Class: class,
	}
}

type Record struct {
	Name  []byte `mdns:"name"`
	Type  Type
	Class Class
	TTL   uint32
	RDLen uint16
	RData RData `mdns:"rdata"`
}

func NewAN(name []byte, class Class, ttl uint32, rd RData) *Record {
	if rd == nil {
		return nil
	}

	an := &Record{
		Name:  name,
		Class: class,
		TTL:   ttl,
		RData: rd,
		RDLen: rd.Len(),
	}

	switch rd.(type) {
	case *A:
		an.Type = TypeA

	case *AAAA:
		an.Type = TypeAAAA

	case *HINFO:
		an.Type = TypeHINFO
	}

	return an
}

type RData interface {
	Len() uint16
	String() string
}

type A struct {
	Addr net.IP `mdns:"a"`
}

func NewA(addr net.IP) *A {
	return &A{ Addr: addr.To4() }
}

func (rr *A) Len() uint16 {
	return uint16(4)
}

func (rr *A) String() string {
	return rr.Addr.String()
}

type CNAME struct {
	CNAME []byte `mdns:"name"`
}

func NewCNAME(cname string) *CNAME {
	return &CNAME{ CNAME: []byte(cname) }
}

func (rr *CNAME) Len() uint16 {
	return uint16(len(rr.CNAME) + 1)
}

func (rr *CNAME) String() string {
	return string(rr.CNAME)
}

type PTR struct {
	PTRNAME []byte `mdns:"name"`
}

func (rr *PTR) Len() uint16 {
	return uint16(len(rr.PTRNAME) + 1)
}

func (rr *PTR) String() string {
	return string(rr.PTRNAME)
}

type HINFO struct {
	CPU string
	OS  string
}

func NewHINFO() *HINFO {
	var uname syscall.Utsname

	err := syscall.Uname(&uname)
	if err != nil {
		return nil
	}

	var cpu, os []byte

	for _, c := range uname.Machine {
		if c == 0 {
			break
		}

		cpu = append(cpu, byte(c))
	}

	for _, c := range uname.Sysname {
		if c == 0 {
			break
		}

		os = append(os, byte(c))
	}

	return &HINFO{
		CPU: strings.ToUpper(string(cpu)),
		OS:  strings.ToUpper(string(os)),
	}
}

func (rr *HINFO) Len() uint16 {
	return uint16(len(rr.CPU) + 1 + len(rr.OS) + 1)
}

func (rr *HINFO) String() string {
	return "\"" + rr.CPU + "\" " + rr.OS + "\""
}

type TXT struct {
	TXT string
}

func (rr *TXT) Len() uint16 {
	return uint16(len(rr.TXT) + 1)
}

func (rr *TXT) String() string {
	return rr.TXT
}

type AAAA struct {
	Addr net.IP `mdns:"aaaa"`
}

func NewAAAA(addr net.IP) *AAAA {
	a := new(AAAA)

	a.Addr = addr.To16()

	return a
}

func (rr *AAAA) Len() uint16 {
	return uint16(16)
}

func (rr *AAAA) String() string {
	return rr.Addr.String()
}

type SRV struct {
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   []byte `mdns:"name"`
}

func (rr *SRV) Len() uint16 {
	return uint16(2 + 2 + 2 + len(rr.Target))
}

func (rr *SRV) String() string {
	return fmt.Sprintf("%d %d %d %s",
	                   rr.Priority, rr.Weight, rr.Port, rr.Target)
}

type OPT struct {
	Code   uint16
	OptLen uint16
	OPT    []byte `mdns:"opt"`
}

func (rr *OPT) Len() uint16 {
	return uint16(len(rr.OPT))
}

func (rr *OPT) String() string {
	return ""
}
