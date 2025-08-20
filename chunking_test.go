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
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestPrepareChunks(t *testing.T) {
	tests := []struct {
		name           string
		bip44PathBytes []byte
		transaction    []byte
		expected       int
	}{
		{
			name:           "Large transaction",
			bip44PathBytes: []byte{0x01, 0x02, 0x03},
			transaction:    make([]byte, 500),
			expected:       12, // 1 path chunk + 11 data chunks (500/48 = 10.4, so 11 chunks)
		},
		{
			name:           "Small transaction",
			bip44PathBytes: []byte{0x01, 0x02, 0x03},
			transaction:    make([]byte, 100),
			expected:       4, // 1 path chunk + 3 data chunks (100/48 = 2.08, so 3 chunks)
		},
		{
			name:           "Exact chunk size transaction",
			bip44PathBytes: []byte{0x01, 0x02, 0x03},
			transaction:    make([]byte, 48),
			expected:       2, // 1 path chunk + 1 data chunk
		},
		{
			name:           "One byte over chunk size",
			bip44PathBytes: []byte{0x01, 0x02, 0x03},
			transaction:    make([]byte, 49),
			expected:       3, // 1 path chunk + 2 data chunks
		},
		{
			name:           "Empty transaction",
			bip44PathBytes: []byte{0x01, 0x02, 0x03},
			transaction:    []byte{},
			expected:       1, // 1 path chunk only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, err := PrepareChunks(tt.bip44PathBytes, tt.transaction)
			if err != nil {
				t.Fatalf("PrepareChunks failed: %v", err)
			}
			
			if len(chunks) != tt.expected {
				t.Errorf("Expected %d chunks, got %d", tt.expected, len(chunks))
			}

			// Verify first chunk is the BIP44 path
			if !bytes.Equal(chunks[0], tt.bip44PathBytes) {
				t.Error("First chunk doesn't match BIP44 path")
			}

			// Verify transaction data is properly chunked
			if len(tt.transaction) > 0 {
				reconstructed := []byte{}
				for i := 1; i < len(chunks); i++ {
					reconstructed = append(reconstructed, chunks[i]...)
				}
				
				if !bytes.Equal(reconstructed, tt.transaction) {
					t.Error("Transaction data not properly chunked")
				}
			}
		})
	}
}

func TestBuildChunkedAPDU(t *testing.T) {
	tests := []struct {
		name     string
		cla      byte
		ins      byte
		p1       byte
		p2       byte
		data     []byte
		expected []byte
	}{
		{
			name:     "Basic APDU",
			cla:      0x80,
			ins:      0x02,
			p1:       ChunkInit,
			p2:       0x00,
			data:     []byte{0x01, 0x02, 0x03},
			expected: []byte{0x80, 0x02, 0x00, 0x00, 0x03, 0x01, 0x02, 0x03},
		},
		{
			name:     "Empty data",
			cla:      0x80,
			ins:      0x02,
			p1:       ChunkLast,
			p2:       0x00,
			data:     []byte{},
			expected: []byte{0x80, 0x02, 0x02, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildChunkedAPDU(tt.cla, tt.ins, tt.p1, tt.p2, tt.data)
			
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Expected %X, got %X", tt.expected, result)
			}
		})
	}
}

// MockLedgerDevice for testing ProcessChunks
type MockLedgerDevice struct {
	responses [][]byte
	callCount int
	commands  [][]byte
}

func (m *MockLedgerDevice) Exchange(command []byte) ([]byte, error) {
	m.commands = append(m.commands, command)
	if m.callCount < len(m.responses) {
		response := m.responses[m.callCount]
		m.callCount++
		return response, nil
	}
	return []byte{0x90, 0x00}, nil
}

func (m *MockLedgerDevice) Close() error {
	return nil
}

func TestProcessChunks(t *testing.T) {
	mockDevice := &MockLedgerDevice{
		responses: [][]byte{
			{0x01, 0x02},
			{0x03, 0x04},
			{0x05, 0x06},
		},
	}

	chunks := [][]byte{
		{0xAA, 0xBB},
		{0xCC, 0xDD},
		{0xEE, 0xFF},
	}

	response, err := ProcessChunksSimple(mockDevice, chunks, 0x80, 0x02, 0x00)
	if err != nil {
		t.Fatalf("ProcessChunks failed: %v", err)
	}

	// Check that we got the last response
	if !bytes.Equal(response, []byte{0x05, 0x06}) {
		t.Errorf("Expected last response, got %X", response)
	}

	// Check number of calls
	if mockDevice.callCount != 3 {
		t.Errorf("Expected 3 Exchange calls, got %d", mockDevice.callCount)
	}

	// Check P1 values for chunk descriptors
	expectedP1 := []byte{ChunkInit, ChunkAdd, ChunkLast}
	for i, cmd := range mockDevice.commands {
		if cmd[2] != expectedP1[i] {
			t.Errorf("Chunk %d: expected P1=%X, got %X", i, expectedP1[i], cmd[2])
		}
	}
}

// MockErrorDevice for testing error handling
type MockErrorDevice struct {
	errorToReturn error
	responseBytes []byte
}

func (m *MockErrorDevice) Exchange(command []byte) ([]byte, error) {
	return m.responseBytes, m.errorToReturn
}

func (m *MockErrorDevice) Close() error {
	return nil
}

func TestProcessChunksWithErrorHandler(t *testing.T) {
	mockDevice := &MockErrorDevice{
		errorToReturn: errors.New("[APDU_CODE_BAD_KEY_HANDLE] The parameters in the data field are incorrect"),
		responseBytes: []byte{0xAB, 0xCD},
	}

	chunks := [][]byte{
		{0xAA, 0xBB},
	}

	// Test with custom error handler (similar to Avalanche)
	_, err := ProcessChunks(mockDevice, chunks, 0x80, 0x02, 0x00, func(err error, response []byte, instruction byte) error {
		if instruction == 0x02 && err.Error() == "[APDU_CODE_BAD_KEY_HANDLE] The parameters in the data field are incorrect" {
			return fmt.Errorf("%w extra_info=(%s)", err, string(response))
		}
		return err
	})

	if err == nil {
		t.Fatal("Expected error but got none")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("extra_info=")) {
		t.Errorf("Expected custom error handling with extra_info, got: %v", err)
	}
}