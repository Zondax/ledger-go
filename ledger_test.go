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
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mux sync.Mutex

func TestLedger(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"CountLedgerDevices", Test_CountLedgerDevices},
		{"ListDevices", TestListDevices},
		{"GetLedger", Test_GetLedger},
		{"BasicExchange", Test_BasicExchange},
		{"Connect", TestConnect},
		{"GetVersion", TestGetVersion},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func Test_CountLedgerDevices(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	ledgerAdmin := NewLedgerAdmin()
	count := ledgerAdmin.CountDevices()
	require.True(t, count > 0)
}

func TestListDevices(t *testing.T) {
	ledgerAdmin := NewLedgerAdmin()

	devices, err := ledgerAdmin.ListDevices()
	if err != nil {
		t.Fatalf("Error listing devices: %v", err)
	}
	assert.NotNil(t, devices, "Devices should not be nil")
}

func Test_GetLedger(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	ledgerAdmin := NewLedgerAdmin()
	count := ledgerAdmin.CountDevices()
	require.True(t, count > 0)

	ledger, err := ledgerAdmin.Connect(0)
	if err != nil {
		t.Fatalf("Error connecting to ledger: %v", err)
	}
	defer ledger.Close()

	assert.NoError(t, err)
	assert.NotNil(t, ledger)
}

func Test_BasicExchange(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	ledgerAdmin := NewLedgerAdmin()
	count := ledgerAdmin.CountDevices()
	require.True(t, count > 0)

	ledger, err := ledgerAdmin.Connect(0)
	if err != nil {
		t.Fatalf("Error connecting to ledger: %v", err)
	}
	defer ledger.Close()

	// Call device info (this should work in main menu and many apps)
	message := []byte{0xE0, 0x01, 0, 0, 0}

	for i := 0; i < 10; i++ {
		response, err := ledger.Exchange(message)

		if err != nil {
			fmt.Printf("iteration %d\n", i)
			t.Fatalf("Error: %s", err.Error())
		}

		require.True(t, len(response) > 0)
	}
}

func TestConnect(t *testing.T) {
	ledgerAdmin := NewLedgerAdmin()

	ledger, err := ledgerAdmin.Connect(0)
	if err != nil {
		t.Fatalf("Error connecting to ledger: %v", err)
	}
	defer ledger.Close()

	assert.NotNil(t, ledger, "Ledger should not be nil")
}

func TestGetVersion(t *testing.T) {
	ledgerAdmin := NewLedgerAdmin()

	ledger, err := ledgerAdmin.Connect(0)
	if err != nil {
		t.Fatalf("Error connecting to ledger: %v", err)
	}
	defer ledger.Close()

	// Call device info (this should work in main menu and many apps)
	message := []byte{0xE0, 0x01, 0, 0, 0}

	response, err := ledger.Exchange(message)
	assert.NoError(t, err)
	assert.NotEmpty(t, response, "Response should not be empty")

}
