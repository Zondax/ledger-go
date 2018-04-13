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

package ledger_goclient

import (
	"github.com/pkg/errors"
	"encoding/binary"
)

var codec = binary.BigEndian

func Packetize(
	channel uint16,
	command []byte,
	packetSize int,
	sequenceIdx uint16,
	ble bool)	(result []byte, offset int, err error) {

	if packetSize < 3 {
		return nil, 0, errors.New("Packet size must be at least 3")
	}

	var headerSize uint16

	result = make([]byte, packetSize)
	var buffer = result

	// Insert channel (2 bytes)
	if !ble {
		codec.PutUint16(buffer, channel)
		headerSize += 2
		buffer = buffer[2:]
	}

	// Insert tag (1 byte)
	buffer[0] = 0x05
	headerSize += 1

	var commandLength uint16
	commandLength = uint16(len(command))

	// Insert sequenceIdx (2 bytes)
	codec.PutUint16(buffer[1:], sequenceIdx)
	headerSize += 2

	// Only insert total size of the command in the first package
	if sequenceIdx == 0 {
		// Insert sequenceIdx (2 bytes)
		codec.PutUint16(buffer[3:], commandLength)
		headerSize += 2
	}

	buffer = buffer[5:]
	offset = copy(buffer, command)
	return result, offset, nil
}

// WrapCommandAPDU turns the command into a sequence of 64 byte packets
func WrapCommandAPDU(
	channel uint16,
	command []byte,
	packetSize int,
	ble bool) (result []byte, err error) {

	var offset int
	var totalResult []byte
	var sequenceIdx uint16
	for len(command) > 0 {
		result, offset, err = Packetize(channel, command, packetSize, sequenceIdx, ble)
		if err != nil {
			return nil, err
		}
		command = command[offset:]
		totalResult = append(totalResult, result...)
		sequenceIdx++
	}
	return totalResult, nil
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