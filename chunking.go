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
	"math"
)

const (
	// DefaultChunkSize is the standard chunk size used across all Ledger apps
	// This replaces userMessageChunkSize from individual apps
	DefaultChunkSize = 48

	// Chunk payload descriptors
	ChunkInit = 0
	ChunkAdd  = 1
	ChunkLast = 2
)

// PrepareChunks splits the transaction data into chunks for sending to the Ledger device
// This matches the exact implementation from ledger-filecoin-go and ledger-avalanche-go
func PrepareChunks(bip44PathBytes []byte, transaction []byte) [][]byte {
	var packetIndex = 0
	// first chunk + number of chunk needed for transaction
	var packetCount = 1 + int(math.Ceil(float64(len(transaction))/float64(DefaultChunkSize)))

	chunks := make([][]byte, packetCount)

	// First chunk is path
	chunks[0] = bip44PathBytes
	packetIndex++

	for packetIndex < packetCount {
		var start = (packetIndex - 1) * DefaultChunkSize
		var end = packetIndex * DefaultChunkSize

		if end >= len(transaction) {
			chunks[packetIndex] = transaction[start:]
		} else {
			chunks[packetIndex] = transaction[start:end]
		}
		packetIndex++
	}

	return chunks
}

// ErrorHandler is a function type for custom error handling in ProcessChunks
type ErrorHandler func(error, []byte, byte) error

// ProcessChunks sends chunks to the Ledger device and collects the response
// This supports both Avalanche and Filecoin error handling patterns via the optional errorHandler
func ProcessChunks(device LedgerDevice, chunks [][]byte, cla, instruction, p2 byte, errorHandler ErrorHandler) ([]byte, error) {
	var finalResponse []byte

	for chunkIndex, chunk := range chunks {
		payloadLen := byte(len(chunk))
		payloadDesc := ChunkAdd

		if chunkIndex == 0 {
			payloadDesc = ChunkInit
		} else if chunkIndex == len(chunks)-1 {
			payloadDesc = ChunkLast
		}

		header := []byte{cla, instruction, byte(payloadDesc), p2, payloadLen}
		message := append(header, chunk...)

		response, err := device.Exchange(message)
		if err != nil {
			// Use custom error handler if provided
			if errorHandler != nil {
				return nil, errorHandler(err, response, instruction)
			}
			return nil, err
		}

		finalResponse = response
	}

	return finalResponse, nil
}

// ProcessChunksSimple sends chunks to the Ledger device with basic error handling
// This is a convenience function for apps that don't need custom error handling
func ProcessChunksSimple(device LedgerDevice, chunks [][]byte, cla, instruction, p2 byte) ([]byte, error) {
	return ProcessChunks(device, chunks, cla, instruction, p2, nil)
}

// BuildChunkedAPDU builds an APDU command for chunked data transmission
// cla is the APDU class byte
// ins is the APDU instruction byte
// p1 is the APDU P1 parameter (typically the chunk descriptor)
// p2 is the APDU P2 parameter
// data is the chunk data to send
func BuildChunkedAPDU(cla, ins, p1, p2 byte, data []byte) []byte {
	// APDU format: [CLA INS P1 P2 LC DATA]
	dataLen := len(data)
	command := make([]byte, 5+dataLen)

	command[0] = cla
	command[1] = ins
	command[2] = p1
	command[3] = p2
	command[4] = byte(dataLen)

	if dataLen > 0 {
		copy(command[5:], data)
	}

	return command
}
