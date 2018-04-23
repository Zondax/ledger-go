/*******************************************************************************
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

package ledger_goclient

import (
	"fmt"
	"errors"
	"github.com/brejski/hid"
	"math"
)

const (
	VendorLedger		= 0x2c97
	ProductNano			= 1
	Channel				= 0x0101
	PacketSize     		= 64
	CLA        			= 0x80

	GetVersionINS		= 0x00
	SignINS				= 0x01
	GetHashINS			= 0x02
	SignQuickINS  		= 0x04

	TestEchoINS         = 99
	TestGetPKINS      	= 100
	TestSignINS			= 101
	MessageChunkSize	= 250
)

type VersionInfo struct {
	AppId uint8
	Major uint8
	Minor uint8
	Patch uint8
}

type Ledger struct {
	device  Device
	Logging bool
}

func NewLedger(dev Device) *Ledger {
	return &Ledger{
		device:  dev,
		Logging: false,
	}
}

func FindLedger() (*Ledger, error) {
	devices, err := hid.Devices()
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.VendorID == VendorLedger {
			device, err := d.Open()
			if err != nil {
				return nil, err
			}
			return NewLedger(device), nil
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

func (ledger *Ledger) Exchange(command []byte) ([]byte, error) {
	if ledger.Logging {
		fmt.Printf("[%3d]=> %x\n", len(command), command)
	}

	serializedCommand, err := WrapCommandAPDU(Channel, command, PacketSize, false)

	if err != nil {
		return nil, err
	}

	// Write all the packets
	err = ledger.device.Write(serializedCommand[:PacketSize])
	if err != nil {
		return nil, err
	}
	for len(serializedCommand) > PacketSize {
		serializedCommand = serializedCommand[PacketSize:]
		err = ledger.device.Write(serializedCommand[:PacketSize])
		if err != nil {
			return nil, err
		}
	}

	input := ledger.device.ReadCh()
	response, err := UnwrapResponseAPDU(Channel, input, PacketSize, false)

	if len(response) < 2 {
		return nil, fmt.Errorf("lost connection")
	}

	swOffset := len(response) - 2
	sw := codec.Uint16(response[swOffset:])

	if sw != 0x9000 {
		switch sw {
		case 0x6400: return nil, errors.New("APDU_CODE_EXECUTION_ERROR")
		case 0x6982: return nil, errors.New("APDU_CODE_EMPTY_BUFFER")
		case 0x6983: return nil, errors.New("APDU_CODE_OUTPUT_BUFFER_TOO_SMALL")
		case 0x6986: return nil, errors.New("APDU_CODE_COMMAND_NOT_ALLOWED")
		case 0x6D00: return nil, errors.New("APDU_CODE_INS_NOT_SUPPORTED")
		case 0x6E00: return nil, errors.New("APDU_CODE_CLA_NOT_SUPPORTED")
		case 0x6F00: return nil, errors.New("APDU_CODE_UNKNOWN")
		}
		return nil, fmt.Errorf("invalid status %04x", sw)
	}

	if ledger.Logging {
		fmt.Printf("[%3d]<= %x\n", len(response[:swOffset]), response[:swOffset])
	}

	return response[:swOffset], nil
}

func (ledger *Ledger) GetVersion() (*VersionInfo, error) {
	message := make([]byte, 2)
	message[0] = CLA
	message[1] = GetVersionINS
	response, err := ledger.Exchange(message)

	if err != nil {
		return nil, err
	}

	if len(response) < 4 {
		return nil, fmt.Errorf("invalid response")
	}

	return &VersionInfo{
		AppId: response[0],
		Major: response[1],
		Minor: response[2],
		Patch: response[3],
	}, nil
}

func (ledger *Ledger) Sign(transaction []byte) ([]byte, error) {

	var packetIndex byte = 1
	var packetCount byte = byte(math.Ceil(float64(len(transaction)) / float64(MessageChunkSize)))

	var finalResponse []byte

	for packetIndex <= packetCount {
		header := make([]byte, 4)
		header[0] = CLA
		header[1] = SignINS
		header[2] = packetIndex
		header[3] = packetCount

		chunk := MessageChunkSize
		if len(transaction) < MessageChunkSize {
			chunk = len(transaction)
		}
		message := append(header, transaction[:chunk]...)
		response, err := ledger.Exchange(message)

		if err != nil {
			return nil, err
		}
		finalResponse = response
		packetIndex++
		transaction = transaction[chunk:]
	}
	return finalResponse, nil
}

func (ledger *Ledger) SignTest(transaction []byte) ([]byte, error) {

	var packetIndex byte = 1
	var packetCount byte = byte(math.Ceil(float64(len(transaction)) / float64(MessageChunkSize)))

	var finalResponse []byte

	for packetIndex <= packetCount {
		header := make([]byte, 4)
		header[0] = CLA
		header[1] = TestSignINS
		header[2] = packetIndex
		header[3] = packetCount

		chunk := MessageChunkSize
		if len(transaction) < MessageChunkSize {
			chunk = len(transaction)
		}
		message := append(header, transaction[:chunk]...)
		response, err := ledger.Exchange(message)

		if err != nil {
			return nil, err
		}
		finalResponse = response
		packetIndex++
		transaction = transaction[chunk:]
	}
	return finalResponse, nil
}

func (ledger *Ledger) Hash(transaction []byte) ([]byte, error) {

	var packetIndex byte = 1
	var packetCount byte = byte(math.Ceil(float64(len(transaction)) / float64(MessageChunkSize)))

	var finalResponse []byte
	for packetIndex <= packetCount {
		header := make([]byte, 4)
		header[0] = CLA
		header[1] = GetHashINS
		header[2] = packetIndex
		header[3] = packetCount

		chunk := MessageChunkSize
		if len(transaction) < MessageChunkSize {
			chunk = len(transaction)
		}
		message := append(header, transaction[:chunk]...)
		response, err := ledger.Exchange(message)

		if err != nil {
			return nil, err
		}
		finalResponse = response
		packetIndex++
		transaction = transaction[chunk:]
	}
	return finalResponse, nil
}

func (ledger *Ledger) Echo(transaction []byte) ([]byte, error) {

	var packetIndex byte = 1
	var packetCount byte = byte(math.Ceil(float64(len(transaction)) / float64(MessageChunkSize)))

	var finalResponse []byte
	for packetIndex <= packetCount {
		header := make([]byte, 4)
		header[0] = CLA
		header[1] = TestEchoINS
		header[2] = packetIndex
		header[3] = packetCount

		chunk := MessageChunkSize
		if len(transaction) < MessageChunkSize {
			chunk = len(transaction)
		}
		message := append(header, transaction[:chunk]...)

		response, err := ledger.Exchange(message)
		if err != nil {
			return nil, err
		}

		finalResponse = response
		packetIndex++
		transaction = transaction[chunk:]
	}
	return finalResponse, nil
}

func (ledger *Ledger) GetPKDummy() ([]byte, error) {
	message := make([]byte, 2)
	message[0] = CLA
	message[1] = TestGetPKINS
	response, err := ledger.Exchange(message)

	if err != nil {
		return nil, err
	}

	if len(response) < 4 {
		return nil, fmt.Errorf("invalid response")
	}

	return response, nil
}