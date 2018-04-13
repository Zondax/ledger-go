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
	"testing"
	"github.com/stretchr/testify/assert"
	"unsafe"
	"bytes"
)

func Test_Packetizer_EmptyCommand(t *testing.T) {
	var command= make([]byte, 1)

	_, _, err := Packetize(0x0101, command, 64, 0, false)
	assert.Nil(t, err, "Commands smaller than 3 bytes should return error")
}

func Test_Packetizer_PacketSize(t *testing.T) {

	var packetSize int = 64
	type header struct {
		channel     uint16
		tag         uint8
		sequenceIdx uint16
		commandLen  uint16
	}

	h := header{channel: 0x0101, tag: 0x05, sequenceIdx:0, commandLen: 32}

	var command= make([]byte, h.commandLen)

	result, _, _ := Packetize(
		h.channel,
		command,
		packetSize,
		h.sequenceIdx,
		false)

	assert.Equal(t, len(result), packetSize, "Packet size is wrong")
}

func Test_Packetizer_Header(t *testing.T) {

	var packetSize int = 64
	type header struct {
		channel     uint16
		tag         uint8
		sequenceIdx uint16
		commandLen  uint16
	}

	h := header{channel: 0x0101, tag: 0x05, sequenceIdx:0, commandLen: 32}

	var command= make([]byte, h.commandLen)

	result, _, _ := Packetize(
		h.channel,
		command,
		packetSize,
		h.sequenceIdx,
		false)

	assert.Equal(t, codec.Uint16(result), h.channel, "Channel not properly serialized")
	assert.Equal(t, result[2], h.tag, "Tag not properly serialized")
	assert.Equal(t, codec.Uint16(result[3:]), h.sequenceIdx, "SequenceIdx not properly serialized")
	assert.Equal(t, codec.Uint16(result[5:]), h.commandLen, "Command len not properly serialized")
}

func Test_Packetizer_Offset(t *testing.T) {

	var packetSize int = 64
	type header struct {
		channel     uint16
		tag         uint8
		sequenceIdx uint16
		commandLen  uint16
	}

	h := header{channel: 0x0101, tag: 0x05, sequenceIdx:0, commandLen: 100}

	var command= make([]byte, h.commandLen)

	_, offset, _ := Packetize(
		h.channel,
		command,
		packetSize,
		h.sequenceIdx,
		false)

	assert.Equal(t, packetSize - int(unsafe.Sizeof(h))+1, offset, "Wrong offset returned. Offset must point to the next comamnd byte that needs to be packet-ized.")
}

func Test_ApduWrapper_NumberOfPackets(t *testing.T) {

	var packetSize int = 64
	type firstHeader struct {
		channel     uint16
		sequenceIdx uint16
		commandLen  uint16
		tag         uint8
	}
	type secondHeader struct {
		channel     uint16
		sequenceIdx uint16
		tag         uint8
	}

	h1 := firstHeader{channel: 0x0101, tag: 0x05, sequenceIdx:0, commandLen: 100}

	var command= make([]byte, h1.commandLen)

	result, _ := WrapCommandAPDU(
		h1.channel,
		command,
		packetSize,
		false)

	assert.Equal(t, packetSize*2, len(result), "Result buffer size is not correct")
}

func Test_ApduWrapper_CheckHeaders(t *testing.T) {

	var packetSize int = 64
	type firstHeader struct {
		channel     uint16
		sequenceIdx uint16
		commandLen  uint16
		tag         uint8
	}
	type secondHeader struct {
		channel     uint16
		sequenceIdx uint16
		tag         uint8
	}

	h1 := firstHeader{channel: 0x0101, tag: 0x05, sequenceIdx:0, commandLen: 100}

	var command= make([]byte, h1.commandLen)

	result, _ := WrapCommandAPDU(
		h1.channel,
		command,
		packetSize,
		false)

	assert.Equal(t, h1.channel, codec.Uint16(result), "Channel not properly serialized")
	assert.Equal(t, h1.tag, result[2], "Tag not properly serialized")
	assert.Equal(t, 0, int(codec.Uint16(result[3:])), "SequenceIdx not properly serialized")
	assert.Equal(t, int(h1.commandLen), int(codec.Uint16(result[5:])), "Command len not properly serialized")

	var offsetOfSecondPacket = packetSize
	assert.Equal(t, h1.channel, codec.Uint16(result[offsetOfSecondPacket:]), "Channel not properly serialized")
	assert.Equal(t, h1.tag, result[offsetOfSecondPacket+2], "Tag not properly serialized")
	assert.Equal(t, 1, int(codec.Uint16(result[offsetOfSecondPacket+3:])), "SequenceIdx not properly serialized")
}

func Test_ApduWrapper_CheckData(t *testing.T) {

	var packetSize int = 64
	type firstHeader struct {
		channel     uint16
		sequenceIdx uint16
		commandLen  uint16
		tag         uint8
	}
	type secondHeader struct {
		channel     uint16
		sequenceIdx uint16
		tag         uint8
	}

	h1 := firstHeader{channel: 0x0101, tag: 0x05, sequenceIdx:0, commandLen: 200}

	var command= make([]byte, h1.commandLen)

	for i := range command {
		command[i] = byte(i % 256)
	}

	result, _ := WrapCommandAPDU(
		h1.channel,
		command,
		packetSize,
		false)

	// Check data in the first packet
	assert.True(t, bytes.Compare(command[0:64-7], result[7:64]) == 0)

	result = result[64:]
	command = command[64-7:]
	// Check data in the second packet
	assert.True(t, bytes.Compare(command[0:64-5], result[5:64]) == 0)

	result = result[64:]
	command = command[64-5:]
	// Check data in the third packet
	assert.True(t, bytes.Compare(command[0:64-5], result[5:64]) == 0)

	result = result[64:]
	command = command[64-5:]

	// Check data in the last packet
	assert.True(t, bytes.Compare(command[0:len(command)], result[5:5+len(command)]) == 0)

	// The remaining bytes in the result should be zeros
	result = result[5+len(command):]
	assert.True(t, bytes.Compare(result, make([]byte, len(result))) == 0)
}