// +build ledger_device

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

package ledger_go

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zondax/hid"
	"sync"
	"testing"
)

var mux sync.Mutex

func Test_ThereAreDevices(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	devices := hid.Enumerate(0, 0)
	assert.NotEqual(t, 0, len(devices))
}

func Test_ListDevices(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	ListDevices()
}

func Test_CountLedgerDevices(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	count := CountLedgerDevices()
	println(count)
	assert.True(t, count > 0)
}

func Test_FindLedger(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	ledger, err := FindLedger()
	defer ledger.Close()

	if err != nil {
		fmt.Println("\n*********************************")
		fmt.Println("Did you enter the password??")
		fmt.Println("*********************************")
		t.Fatalf("Error: %s", err.Error())
	}
	assert.NotNil(t, ledger)
}

func Test_GetLedger(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	ledger, err := GetLedger(1)
	defer ledger.Close()

	if err != nil {
		fmt.Println("\n*********************************")
		fmt.Println("Did you enter the password??")
		fmt.Println("*********************************")
		t.Fatalf("Error: %s", err.Error())
	}
	assert.NotNil(t, ledger)
}

func Test_BasicExchange(t *testing.T) {
	mux.Lock()
	defer mux.Unlock()

	ledger, err := FindLedger()
	defer ledger.Close()

	if err != nil {
		fmt.Println("\n*********************************")
		fmt.Println("Did you enter the password??")
		fmt.Println("*********************************")
		t.Fatalf("Error: %s", err.Error())
	}
	assert.NotNil(t, ledger)

	// Call app info (this should work in main menu and many apps)
	message := []byte{0xB0, 0x01, 0, 0, 0}

	for i := 0; i < 10; i++ {
		response, err := ledger.Exchange(message)

		if err != nil {
			fmt.Printf("iteration %d\n", i)
			t.Fatalf("Error: %s", err.Error())
		}

		assert.Equal(t, 15, len(response))
	}
}
