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
import "encoding/binary"
import "fmt"
import "io"
import "net"
import "reflect"

func Unpack(pkt []byte) (*Message, error) {
	msg := new(Message)

	r := bytes.NewReader(pkt)

	err := UnpackStruct(r, &msg.Header)
	if err != nil {
		return nil, fmt.Errorf("Could not unpack header: %s", err)
	}

	for i := uint16(0); i < msg.Header.QDCount; i++ {
		qd := new(Question)

		err := UnpackStruct(r, qd)
		if err != nil {
			return nil, fmt.Errorf("Could not unpack qd: %s", err)
		}

		msg.Question = append(msg.Question, qd)
	}

	for i := uint16(0); i < msg.Header.ANCount; i++ {
		an := new(Record)

		err := UnpackStruct(r, an)
		if err != nil {
			return nil, fmt.Errorf("Could not unpack an: %s", err)
		}

		msg.Answer = append(msg.Answer, an)
	}

	for i := uint16(0); i < msg.Header.NSCount; i++ {
		an := new(Record)

		err := UnpackStruct(r, an)
		if err != nil {
			return nil, fmt.Errorf("Could not unpack an: %s", err)
		}

		msg.Authority = append(msg.Authority, an)
	}

	for i := uint16(0); i < msg.Header.ARCount; i++ {
		ar := new(Record)

		err := UnpackStruct(r, ar)
		if err != nil {
			return nil, fmt.Errorf("Could not unpack ar: %s", err)
		}

		msg.Additional = append(msg.Additional, ar)
	}

	if r.Len() != 0 {
		l    := r.Len()
		rest := make([]byte, l)

		io.ReadFull(r, rest)

		return nil, fmt.Errorf("trailing data: %d %s %v", l, msg, rest)
	}

	return msg, nil
}

func UnpackStruct(r io.Reader, data interface{}) error {
	value := reflect.ValueOf(data).Elem()

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		kind  := field.Kind()
		tag   := value.Type().Field(i).Tag

		switch {
		case tag == `mdns:"name"`:
			name, err := UnpackName(r)
			if err != nil {
				return fmt.Errorf("name: %s", err)
			}

			field.SetBytes(name)

		case tag == `mdns:"a"`:
			var addr [4]byte

			err := binary.Read(r, binary.BigEndian, &addr)
			if err != nil {
				return fmt.Errorf("read: %s", err)
			}

			ip := net.IP(addr[:]).To4()

			field.Set(reflect.ValueOf(ip))

		case tag == `mdns:"aaaa"`:
			var addr [16]byte

			err := binary.Read(r, binary.BigEndian, &addr)
			if err != nil {
				return fmt.Errorf("read: %s", err)
			}

			ip := net.IP(addr[:]).To16()

			field.Set(reflect.ValueOf(ip))

		case tag == `mdns:"opt"`:
			optlen := value.FieldByName("OptLen").Uint()
			opt := make([]byte, optlen)

			err := binary.Read(r, binary.BigEndian, &opt)
			if err != nil {
				return fmt.Errorf("read: %s", err)
			}

			field.Set(reflect.ValueOf(opt))

		case tag == `mdns:"rdata"`:
			rdlen := value.FieldByName("RDLen").Uint()

			if rdlen == 0 {
				continue
			}

			rdtype := value.FieldByName("Type").Interface().(Type)
			rdata  := rdtype.MakeRR()
			if rdata == nil {
				return fmt.Errorf("%d not implemented",
					rdtype)
			}

			err := UnpackStruct(r, rdata)
			if err != nil {
				return fmt.Errorf("struct: %s", err)
			}

			field.Set(reflect.ValueOf(rdata))

		case kind == reflect.Uint16:
			var v uint16

			err := binary.Read(r, binary.BigEndian, &v)
			if err != nil {
				return fmt.Errorf("read: %s", err)
			}

			field.SetUint(uint64(v))

		case kind == reflect.Uint32:
			var v uint32

			err := binary.Read(r, binary.BigEndian, &v)
			if err != nil {
				return fmt.Errorf("read: %s", err)
			}

			field.SetUint(uint64(v))

		case kind == reflect.String:
			s, err := UnpackString(r)
			if err != nil {
				return fmt.Errorf("string: %s", err)
			}

			field.SetString(s)

		case kind == reflect.Interface || kind == reflect.Struct:
			err := UnpackStruct(r, field.Interface())
			if err != nil {
				return fmt.Errorf("struct: %s", err)
			}
		}
	}

	return nil
}

func UnpackName(r io.Reader) ([]byte, error) {
	var name []byte

	for {
		var v uint8

		err := binary.Read(r, binary.BigEndian, &v)
		if err != nil {
			return nil, fmt.Errorf("read: %s", err)
		}

		if v&0xC0 == 0xC0 {
			var p uint8

			err := binary.Read(r, binary.BigEndian, &p)
			if err != nil {
				return nil, fmt.Errorf("read: %s", err)
			}

			off := (int(v)^0xC0)<<8 | int(p)

			return UnpackNameAt(r.(*bytes.Reader), int64(off))
		}

		if v == 0 {
			break
		}

		label := make([]byte, v)

		err = binary.Read(r, binary.BigEndian, &label)
		if err != nil {
			return nil, fmt.Errorf("read: %s", err)
		}

		name = append(name, label...)
		name = append(name, '.')
	}

	return name, nil
}

func UnpackNameAt(r io.ReaderAt, off int64) ([]byte, error) {
	var name []byte

	for {
		v := make([]byte, 1)

		n, err := r.ReadAt(v, off)
		if err != nil {
			return nil, fmt.Errorf("read at: %s", err)
		}

		off += int64(n)

		if v[0]&0xC0 == 0xC0 {
			p := make([]byte, 1)

			_, err := r.ReadAt(p, off)
			if err != nil {
				return nil, fmt.Errorf("read at: %s", err)
			}

			off := (int(v[0])^0xC0)<<8 | int(p[0])

			return UnpackNameAt(r, int64(off))
		}

		if v[0] == 0 {
			break
		}

		label := make([]byte, v[0])

		n, err = r.ReadAt(label, off)
		if err != nil {
			return nil, fmt.Errorf("read at: %s", err)
		}

		off += int64(n)

		name = append(name, label...)
		name = append(name, '.')
	}

	return name, nil
}

func UnpackString(r io.Reader) (string, error) {
	var v uint8

	err := binary.Read(r, binary.BigEndian, &v)
	if err != nil {
		return "", fmt.Errorf("read: %s", err)
	}

	if v == 0 {
		return "", nil
	}

	s := make([]byte, v)

	err = binary.Read(r, binary.BigEndian, &s)
	if err != nil {
		return "", fmt.Errorf("read: %s", err)
	}

	return string(s), nil
}
