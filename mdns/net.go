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
import "syscall"
import "unsafe"

import "golang.org/x/net/ipv4"

import "github.com/ghedo/moodns/netlink"

const maddr4 = "224.0.0.251:5353"
const maddr6 = "[FF02::FB]:5353"

func NewConn(addr string) (*net.UDPAddr, *ipv4.PacketConn, error) {
	saddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, nil,
		  fmt.Errorf("Could not resolve address '%s': %s", addr, err)
	}

	smaddr, err := net.ResolveUDPAddr("udp", maddr4)
	if err != nil {
		return nil, nil,
		  fmt.Errorf("Could not resolve address '%s': %s", maddr4, err)
	}

	udp, err := net.ListenUDP("udp", saddr)
	if err != nil {
		return nil, nil,
		  fmt.Errorf("Could not listen: %s", err)
	}

	p := ipv4.NewPacketConn(udp)

	err = p.SetTTL(1)
	if err != nil {
		return nil, nil,
		  fmt.Errorf("Could not set TTL: %s", err)
	}

	err = p.SetMulticastLoopback(false)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set loop: %s", err)
	}

	err = p.SetControlMessage(ipv4.FlagInterface|ipv4.FlagDst, true)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not set ctrlmsg: %s", err)
	}

	return smaddr, p, nil
}

func NewServer(addr string) (*net.UDPAddr, *ipv4.PacketConn, error) {
	smaddr, p, err := NewConn(addr)
	if err != nil {
		return nil, nil, err
	}

	go MonitorNetwork(p, smaddr)

	return smaddr, p, nil
}

func NewClient(addr string) (*net.UDPAddr, *ipv4.PacketConn, error) {
	return NewConn(addr)
}

func Read(p *ipv4.PacketConn) (*Message, *net.IPNet, *net.IPNet, *net.UDPAddr, bool, error) {
	var local4 *net.IPNet
	var local6 *net.IPNet

	var ifi *net.Interface

	var loopback bool

	pkt := make([]byte, 9000)

	n, cm, from, err := p.ReadFrom(pkt)
	if err != nil {
		return nil, nil, nil, nil, false,
		  fmt.Errorf("Could not read: %s", err)
	}

	if cm == nil {
		ifi, err = net.InterfaceByName("lo")
		if err != nil {
			return nil, nil, nil, nil, true,
			  fmt.Errorf("Could not find if: %s", err)
		}

		loopback = true
	} else {
		ifi, err = net.InterfaceByIndex(cm.IfIndex)
		if err != nil {
			return nil, nil, nil, nil, false,
			  fmt.Errorf("Could not find if: %s", err)
		}

		loopback = false
	}

	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, nil, nil, nil, loopback,
		  fmt.Errorf("Could not find addrs: %s", err)
	}

	for _, a := range addrs {
		if a.(*net.IPNet).IP.To4() != nil {
			local4 = a.(*net.IPNet)
		} else {
			local6 = a.(*net.IPNet)
		}
	}

	req, err := Unpack(pkt[:n])
	if err != nil {
		return nil, nil, nil, nil, loopback,
		  fmt.Errorf("Could not unpack request: %s", err)
	}

	return req, local4, local6, from.(*net.UDPAddr), loopback, err
}

func Write(p *ipv4.PacketConn, addr *net.UDPAddr, msg *Message) error {
	pkt, err := Pack(msg)
	if err != nil {
		return fmt.Errorf("Could not pack response: %s", err)
	}

	_, err = p.WriteTo(pkt, nil, addr)
	if err != nil {
		return fmt.Errorf("Could not write to network: %s", err)
	}

	return nil
}

func SendRequest(req *Message) (*Message, error) {
	maddr, client, err := NewClient("0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("Could not create client: %s", err)
	}
	defer client.Close()

	seconds := 3 * time.Second
	timeout := time.Now().Add(seconds)

	err = Write(client, maddr, req)
	if err != nil {
		return nil, fmt.Errorf("Could not send request: %s", err)
	}

	client.SetReadDeadline(timeout)

	rsp, _, _, _, _, err := Read(client)
	if err != nil {
		return nil, fmt.Errorf("Could not read response: %s", err)
	}

	if rsp.Header.Id != req.Header.Id {
		return nil, fmt.Errorf("Wrong id: %d", rsp.Header.Id)
	}

	return rsp, nil
}

func SendRecursiveRequest(msg *Message, q *Question) uint16 {
	if bytes.HasSuffix(q.Name, []byte("local.")) != true {
		msg.Header.Flags |= RCodeServFail
		return 0
	}

	rand.Seed(time.Now().UTC().UnixNano())
	id := uint16(rand.Intn(math.MaxUint16))

	req := new(Message)

	req.Header.Id = id
	req.AppendQD(q)

	rsp, err := SendRequest(req)
	if err != nil {
		return 0
	}

	for _, an := range rsp.Answer {
		msg.Answer = append(msg.Answer, an)
		msg.Header.ANCount++
	}

	return id
}

func Serve(p *ipv4.PacketConn, maddr *net.UDPAddr, localname string, silent, forward bool) {
	var sent_id uint16

	for {
		req, local4, local6, client, loopback, err := Read(p)
		if err != nil {
			if silent != true {
				log.Println("Error reading request: ", err)
				continue
			}
		}

		if req.Header.Flags&FlagQR != 0 {
			continue
		}

		if sent_id > 0 && req.Header.Id == sent_id {
			continue
		}

		rsp := new(Message)

		rsp.Header.Flags |= FlagQR
		rsp.Header.Flags |= FlagAA

		if req.Header.Flags&FlagRD != 0 {
			rsp.Header.Flags |= FlagRD
			rsp.Header.Flags |= FlagRA
		}

		if client.Port != 5353 {
			rsp.Header.Id = req.Header.Id
		}

		for _, q := range req.Question {
			switch q.Class {
			case ClassInet:
			case ClassInet | ClassUnicast:
			case ClassAny:

			default:
				continue /* unsupport class */
			}

			if client.Port != 5353 {
				rsp.Question = append(rsp.Question, q)
				rsp.Header.QDCount++
			}

			if string(q.Name) != localname {
				if loopback && forward != false {
					sent_id = SendRecursiveRequest(rsp, q)
				}

				continue
			}

			var rdata []RData

			switch q.Type {
			case TypeA:
				rdata = append(rdata, NewA(local4.IP))

			case TypeAAAA:
				rdata = append(rdata, NewAAAA(local6.IP))

			case TypeHINFO:
				rdata = append(rdata, NewHINFO())

			case TypeAny:
				rdata = append(rdata, NewA(local4.IP))
				rdata = append(rdata, NewAAAA(local6.IP))
				rdata = append(rdata, NewHINFO())

			default:
				continue
			}

			for _, rd := range rdata {
				an := NewAN(q.Name, q.Class, 120, rd)
				rsp.AppendAN(an)
			}
		}

		if rsp.Header.ANCount       == 0 &&
		   rsp.Header.Flags.RCode() == RCodeNoError {
			continue /* no answers and no error, skip */
		}

		if client.Port == 5353 {
			client = maddr
		}

		err = Write(p, client, rsp)
		if err != nil {
			if silent != true {
				log.Println("Error sending response: ", err)
				continue
			}
		}
	}
}

func MonitorNetwork(p *ipv4.PacketConn, group net.Addr) error {
	l, _ := netlink.ListenNetlink()

	l.SendRouteRequest(syscall.RTM_GETADDR, syscall.AF_UNSPEC)

	for {
		msgs, err := l.ReadMsgs()
		if err != nil {
			return fmt.Errorf("Could not read netlink: %s", err)
		}

		for _, m := range msgs {
			if netlink.IsNewAddr(&m) {
				err := JoinGroup(p, &m, group)
				if err != nil {
					return err
				}
			}

			if netlink.IsDelAddr(&m) {
				err := LeaveGroup(p, &m, group)
				if err != nil {
					return err
				}
			}
		}
	}
}

func JoinGroup(p *ipv4.PacketConn, msg *syscall.NetlinkMessage, group net.Addr) error {
	ifaddrmsg := (*syscall.IfAddrmsg)(unsafe.Pointer(&msg.Data[0]))

	if netlink.IsRelevant(ifaddrmsg) != true {
		return nil
	}

	ifi, err := net.InterfaceByIndex(int(ifaddrmsg.Index))
	if err != nil {
		return fmt.Errorf("Could not get interface: %s", err)
	}

	err = p.JoinGroup(ifi, group)
	if err != nil {
		return fmt.Errorf("Could not join group: %s", err)
	}

	return nil
}

func LeaveGroup(p *ipv4.PacketConn, msg *syscall.NetlinkMessage, group net.Addr) error {
	ifaddrmsg := (*syscall.IfAddrmsg)(unsafe.Pointer(&msg.Data[0]))

	ifi, err := net.InterfaceByIndex(int(ifaddrmsg.Index))
	if err != nil {
		return fmt.Errorf("Could not get interface: %s", err)
	}

	err = p.LeaveGroup(ifi, group)
	if err != nil {
		return fmt.Errorf("Could not leave group: %s", err)
	}

	return nil
}
