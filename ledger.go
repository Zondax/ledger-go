/*******************************************************************************
*   (c) 2016 Ledger
*   (c) 2018 ZondaX GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
********************************************************************************/

// A simple command line tool that outputs json messages representing transactions
// Usage: samples [0-3] [binary|text]
// Note: Use build_samples.sh script to update correctly update dependencies

package main

import (
    "fmt"
    "errors"
    "github.com/brejski/hid"
	"os"
	"encoding/binary"
	"encoding/hex"
	"github.com/zondax/ledger/samples"
)

var codec = binary.BigEndian
var (
	errTooShort        = errors.New("too short")
	errInvalidChannel  = errors.New("invalid channel")
	errInvalidSequence = errors.New("invalid sequence")
	errInvalidTag      = errors.New("invalid tag")
)
const (
    VendorLedger = 0x2c97
    ProductNano  = 1
    Channel      = 0x8001
    PacketSize   = 64
)

type Ledger struct {
    device Device
}

func NewLedger(dev Device) *Ledger {
    return &Ledger{
        device: dev,
    }
}

func FindLedger() (*Ledger, error) {
    devs, err := hid.Devices()
    if err != nil {
        return nil, err
    }
    for _, d := range devs {
        // TODO: ProductId filter
        if d.VendorID == VendorLedger {
            ledger, err := d.Open()
            if err != nil {
                return nil, err
            }
            return NewLedger(ledger), nil
        }
    }
    return nil, errors.New("no ledger connected")
}

// A Device provides access to a HID device.
type Device interface {
    // Close closes the device and associated resources.
    Close()

    // Write writes an output report to device. The first byte must be the
    // report number to write, zero if the device does not use numbered reports.
    Write([]byte) error

    // ReadCh returns a channel that will be sent input reports from the device.
    // If the device uses numbered reports, the first byte will be the report
    // number.
    ReadCh() <-chan []byte

    // ReadError returns the read error, if any after the channel returned from
    // ReadCh has been closed.
    ReadError() error
}

// WrapCommandAPDU turns the command into a sequence of 64 byte packets
func WrapCommandAPDU(channel uint16, command []byte, packetSize int, ble bool) []byte {
	if packetSize < 3 {
		panic("packet size must be at least 3")
	}

	var sequenceIdx uint16
	var offset, extraHeaderSize, blockSize int
	var result = make([]byte, 64)
	var buf = result

	if !ble {
		codec.PutUint16(buf, channel)
		extraHeaderSize = 2
		buf = buf[2:]
	}

	buf[0] = 0x05
	codec.PutUint16(buf[1:], sequenceIdx)
	codec.PutUint16(buf[3:], uint16(len(command)))
	sequenceIdx++
	buf = buf[5:]

	blockSize = packetSize - 5 - extraHeaderSize
	copy(buf, command)
	offset += blockSize

	for offset < len(command) {
		// TODO: optimize this
		end := len(result)
		result = append(result, make([]byte, 64)...)
		buf = result[end:]
		if !ble {
			codec.PutUint16(buf, channel)
			buf = buf[2:]
		}
		buf[0] = 0x05
		codec.PutUint16(buf[1:], sequenceIdx)
		sequenceIdx++
		buf = buf[3:]

		blockSize = packetSize - 3 - extraHeaderSize
		copy(buf, command[offset:])
		offset += blockSize
	}

	return result
}

func validatePrefix(buf []byte, channel, sequenceIdx uint16, ble bool) ([]byte, error) {
	if !ble {
		if codec.Uint16(buf) != channel {
			return nil, errInvalidChannel
		}
		buf = buf[2:]
	}

	if buf[0] != 0x05 {
		return nil, errInvalidTag
	}
	if codec.Uint16(buf[1:]) != sequenceIdx {
		return nil, errInvalidSequence
	}
	return buf[3:], nil
}

// UnwrapResponseAPDU parses a response of 64 byte packets into the real data
func UnwrapResponseAPDU(channel uint16, dev <-chan []byte, packetSize int, ble bool) ([]byte, error) {
	var err error
	var sequenceIdx uint16
	var extraHeaderSize int
	if !ble {
		extraHeaderSize = 2
	}
	buf := <-dev
	if len(buf) < 5+extraHeaderSize+5 {
		return nil, errTooShort
	}

	buf, err = validatePrefix(buf, channel, sequenceIdx, ble)
	if err != nil {
		return nil, err
	}

	responseLength := int(codec.Uint16(buf))
	buf = buf[2:]
	result := make([]byte, responseLength)
	out := result

	blockSize := packetSize - 5 - extraHeaderSize
	if blockSize > len(buf) {
		blockSize = len(buf)
	}
	copy(out, buf[:blockSize])

	// if there is anything left to read...
	for len(out) > blockSize {
		out = out[blockSize:]
		buf = <-dev

		sequenceIdx++
		buf, err = validatePrefix(buf, channel, sequenceIdx, ble)
		if err != nil {
			return nil, err
		}

		blockSize = packetSize - 3 - extraHeaderSize
		if blockSize > len(buf) {
			blockSize = len(buf)
		}
		copy(out, buf[:blockSize])
	}
	return result, nil
}


func main() {
    messages := samples.GetMessages()

    ledger, err := FindLedger()
    if err != nil {
        fmt.Printf("Could not find ledger.")
        os.Exit(-1)
    }
    if ledger != nil {
		header := make([]byte, 4)
		header[0] = 0x80;
		header[1] = 0x01;
		header[2] = 0x01;
		header[3] = 0x01;
		fullMessage := append(header, messages[0].GetSignBytes()...)

		fmt.Println(fullMessage)
		adpu := WrapCommandAPDU(Channel, fullMessage, PacketSize, false)

		// write all the packets
		err := ledger.device.Write(adpu[:PacketSize])
		if err != nil {
			fmt.Printf("Could not write to ledger.")
			os.Exit(-1)
		}
		for len(adpu) > PacketSize {
			adpu = adpu[PacketSize:]
			err = ledger.device.Write(adpu[:PacketSize])
			if err != nil {
				fmt.Printf("Could not write to ledger.")
				os.Exit(-1)
			}
		}

		input := ledger.device.ReadCh()
		response, err := UnwrapResponseAPDU(Channel, input, PacketSize, false)

		swOffset := len(response) - 2
		sw := codec.Uint16(response[swOffset:])
		if sw != 0x9000 {
			fmt.Errorf("Invalid status %04x", sw)
		}
		fmt.Printf("Signature:" + hex.EncodeToString(response[:swOffset]))
	}
}

