//go:build !ledger_mock && !ledger_zemu
// +build !ledger_mock,!ledger_zemu

/*******************************************************************************
*   (c) Zondax AG
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

package ledger_go

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/zondax/hid"
)

const (
	VendorLedger         = 0x2c97
	UsagePageLedgerNanoS = 0xffa0
	Channel              = 0x0101
	PacketSize           = 64
)

type LedgerAdminHID struct{}

type LedgerDeviceHID struct {
	device      *hid.Device
	readCo      *sync.Once
	readChannel chan []byte
}

// list of supported product ids as well as their corresponding interfaces
// based on https://github.com/LedgerHQ/ledger-live/blob/develop/libs/ledgerjs/packages/devices/src/index.ts
var supportedLedgerProductID = map[uint8]int{
	0x40: 0, // Ledger Nano X
	0x10: 0, // Ledger Nano S
	0x50: 0, // Ledger Nano S Plus
	0x60: 0, // Ledger Stax
	0x70: 0, // Ledger Flex
}

func NewLedgerAdmin() LedgerAdmin {
	return &LedgerAdminHID{}
}

func (admin *LedgerAdminHID) ListDevices() ([]string, error) {
	devices := hid.Enumerate(0, 0)
	if len(devices) == 0 {
		log.Debug("No devices. Ledger LOCKED OR Other Program/Web Browser may have control of device.")
	}

	for _, d := range devices {
		logDeviceInfo(d)
	}

	return []string{}, nil
}

func logDeviceInfo(d hid.DeviceInfo) {
	log.Debugf("============ %s", d.Path)
	log.Debugf("VendorID      : %x", d.VendorID)
	log.Debugf("ProductID     : %x", d.ProductID)
	log.Debugf("Release       : %x", d.Release)
	log.Debugf("Serial        : %x", d.Serial)
	log.Debugf("Manufacturer  : %s", d.Manufacturer)
	log.Debugf("Product       : %s", d.Product)
	log.Debugf("UsagePage     : %x", d.UsagePage)
	log.Debugf("Usage         : %x", d.Usage)
}

func isLedgerDevice(d hid.DeviceInfo) bool {
	deviceFound := d.UsagePage == UsagePageLedgerNanoS

	// Workarounds for possible empty usage pages
	productIDMM := uint8(d.ProductID >> 8)
	if interfaceID, supported := supportedLedgerProductID[productIDMM]; deviceFound || (supported && (interfaceID == d.Interface)) {
		return true
	}

	return false
}

func (admin *LedgerAdminHID) CountDevices() int {
	devices := hid.Enumerate(0, 0)

	count := 0
	for _, d := range devices {
		if isLedgerDevice(d) {
			count++
		}
	}

	return count
}

func newDevice(dev *hid.Device) *LedgerDeviceHID {
	return &LedgerDeviceHID{
		device:      dev,
		readCo:      new(sync.Once),
		readChannel: make(chan []byte),
	}
}

func (admin *LedgerAdminHID) Connect(requiredIndex int) (LedgerDevice, error) {
	devices := hid.Enumerate(VendorLedger, 0)

	currentIndex := 0
	for _, d := range devices {
		if isLedgerDevice(d) {
			if currentIndex == requiredIndex {
				device, err := d.Open()
				if err != nil {
					return nil, err
				}
				deviceHID := newDevice(device)
				return deviceHID, nil
			}
			currentIndex++
			if currentIndex > requiredIndex {
				break
			}
		}
	}

	return nil, fmt.Errorf("LedgerHID device (idx %d) not found: device may be locked or in use by another application", requiredIndex)
}

func (ledger *LedgerDeviceHID) write(buffer []byte) (int, error) {
	totalBytes := len(buffer)
	totalWrittenBytes := 0
	for totalBytes > totalWrittenBytes {
		writtenBytes, err := ledger.device.Write(buffer)

		if err != nil {
			return totalWrittenBytes, err
		}
		buffer = buffer[writtenBytes:]
		totalWrittenBytes += writtenBytes
	}
	return totalWrittenBytes, nil
}

func (ledger *LedgerDeviceHID) Read() <-chan []byte {
	ledger.readCo.Do(ledger.initReadChannel)
	return ledger.readChannel
}

func (ledger *LedgerDeviceHID) initReadChannel() {
	ledger.readChannel = make(chan []byte, 30)
	go ledger.readThread()
}

func (ledger *LedgerDeviceHID) readThread() {
	defer close(ledger.readChannel)

	for {
		buffer := make([]byte, PacketSize)
		readBytes, err := ledger.device.Read(buffer)

		// Check for HID Read Error (May occur even during normal runtime)
		if err != nil {
			continue
		}

		// Discard all zero packets from Ledger Nano X on macOS
		allZeros := true
		for i := 0; i < len(buffer); i++ {
			if buffer[i] != 0 {
				allZeros = false
				break
			}
		}

		// Discard all zero packet
		if allZeros {
			// HID Returned Empty Packet - Retry Read
			continue
		}

		select {
		case ledger.readChannel <- buffer[:readBytes]:
			// Send data to UnwrapResponseAPDU
		default:
			// Possible source of bugs
			// Drop a buffer if ledger.readChannel is busy
		}
	}
}

func (ledger *LedgerDeviceHID) drainRead() {
	// Allow time for late packet arrivals (When main program doesn't read enough packets)
	<-time.After(50 * time.Millisecond)
	for {
		select {
		case <-ledger.readChannel:
		default:
			return
		}
	}
}

func (ledger *LedgerDeviceHID) Exchange(command []byte) ([]byte, error) {
	log.Debugf("Sending command: %X", command)
	// Purge messages that arrived after previous exchange completed
	ledger.drainRead()

	if len(command) < 5 {
		return nil, fmt.Errorf("APDU commands should not be smaller than 5")
	}

	if (byte)(len(command)-5) != command[4] {
		return nil, fmt.Errorf("APDU[data length] mismatch")
	}

	serializedCommand, err := WrapCommandAPDU(Channel, command, PacketSize)
	if err != nil {
		return nil, err
	}

	// Write all the packets
	_, err = ledger.write(serializedCommand)
	if err != nil {
		return nil, err
	}

	readChannel := ledger.Read()

	response, err := UnwrapResponseAPDU(Channel, readChannel, PacketSize)
	if err != nil {
		return nil, err
	}

	if len(response) < 2 {
		return nil, fmt.Errorf("len(response) < 2")
	}

	swOffset := len(response) - 2
	sw := codec.Uint16(response[swOffset:])

	if sw != 0x9000 {
		return response[:swOffset], errors.New(ErrorMessage(sw))
	}

	log.Debugf("Received response: %X", response)
	return response[:swOffset], nil
}

func (ledger *LedgerDeviceHID) Close() error {
	return ledger.device.Close()
}
