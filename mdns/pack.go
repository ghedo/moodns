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
import "reflect"

func Pack(msg *Message) ([]byte, error) {
	b := bytes.NewBuffer([]byte{})

	err := PackStruct(b, &msg.Header)
	if err != nil {
		return nil, fmt.Errorf("Could not pack header: %s", err)
	}

	for i := uint16(0); i < msg.Header.QDCount; i++ {
		err := PackStruct(b, msg.Question[i])
		if err != nil {
			return nil, fmt.Errorf("Could not pack qd: %s", err)
		}
	}

	for i := uint16(0); i < msg.Header.ANCount; i++ {
		err := PackStruct(b, msg.Answer[i])
		if err != nil {
			return nil, fmt.Errorf("Could not pack an: %s", err)
		}
	}

	for i := uint16(0); i < msg.Header.NSCount; i++ {
		err := PackStruct(b, msg.Authority[i])
		if err != nil {
			return nil, fmt.Errorf("Could not pack ns: %s", err)
		}
	}

	for i := uint16(0); i < msg.Header.ARCount; i++ {
		err := PackStruct(b, msg.Additional[i])
		if err != nil {
			return nil, fmt.Errorf("Could not pack ar: %s", err)
		}
	}

	return b.Bytes(), nil
}

func PackStruct(r io.Writer, data interface{}) error {
	value := reflect.ValueOf(data).Elem()

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		kind  := field.Kind()
		tag   := value.Type().Field(i).Tag

		switch {
		case tag == `mdns:"name"`:
			name := field.Bytes()

			err := PackName(r, name)
			if err != nil {
				return fmt.Errorf("name: %s", err)
			}

		case tag == `mdns:"a"` || tag == `mdns:"aaaa"`:
			for i := 0; i < field.Len(); i++ {
				b := byte(field.Index(i).Uint())

				err := binary.Write(
					r, binary.BigEndian, &b,
				)

				if err != nil {
					return fmt.Errorf("write: %s", err)
				}
			}

		case kind == reflect.Uint16:
			v := uint16(field.Uint())

			err := binary.Write(r, binary.BigEndian, &v)
			if err != nil {
				return fmt.Errorf("write: %s", err)
			}

		case kind == reflect.Uint32:
			v := uint32(field.Uint())

			err := binary.Write(r, binary.BigEndian, &v)
			if err != nil {
				return fmt.Errorf("write: %s", err)
			}

		case kind == reflect.String:
			str := field.String()

			err := PackString(r, str)
			if err != nil {
				return fmt.Errorf("string: %s", err)
			}

		case kind == reflect.Interface || kind == reflect.Struct:
			err := PackStruct(r, field.Interface())
			if err != nil {
				return fmt.Errorf("struct: %s", err)
			}
		}
	}

	return nil
}

func PackName(w io.Writer, name []byte) error {
	for _, label := range bytes.Split(name, []byte{'.'}) {
		l := uint8(len(label))

		err := binary.Write(w, binary.BigEndian, &l)
		if err != nil {
			return fmt.Errorf("write: %s", err)
		}

		err = binary.Write(w, binary.BigEndian, label)
		if err != nil {
			return fmt.Errorf("write: %s", err)
		}
	}

	return nil
}

func PackString(w io.Writer, str string) error {
	l := uint8(len(str))

	err := binary.Write(w, binary.BigEndian, &l)
	if err != nil {
		return fmt.Errorf("write: %s", err)
	}

	err = binary.Write(w, binary.BigEndian, []byte(str))
	if err != nil {
		return fmt.Errorf("write: %s", err)
	}

	return nil
}
