//go:build ledger_mock
// +build ledger_mock

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
	"encoding/hex"
	"fmt"
)

type LedgerAdminMock struct{}

type LedgerDeviceMock struct {
	commands map[string][]byte
}

func NewLedgerAdmin() *LedgerAdminMock {
	return &LedgerAdminMock{}
}

func (admin *LedgerAdminMock) ListDevices() ([]string, error) {
	x := []string{"Mock device"}
	return x, nil
}

func (admin *LedgerAdminMock) CountDevices() int {
	return 1
}

func (admin *LedgerAdminMock) Connect(deviceIndex int) (*LedgerDeviceMock, error) {
	return &LedgerDeviceMock{}, nil
}

func NewLedgerDeviceMock() *LedgerDeviceMock {
	return &LedgerDeviceMock{
		commands: map[string][]byte{
			"E001000000": []byte{0x31, 0x10, 0x00, 0x04, 0x08, 0x53, 0x70, 0x65, 0x63, 0x75, 0x6c, 0x6f, 0x73, 0x00, 0x0b, 0x53, 0x70, 0x65, 0x63, 0x75, 0x6c, 0x6f, 0x73, 0x4d, 0x43, 0x55},
		},
	}
}

func (ledger *LedgerDeviceMock) Exchange(command []byte) ([]byte, error) {
	hexCommand := hex.EncodeToString(command)
	if reply, ok := ledger.commands[hexCommand]; ok {
		return reply, nil
	}
	return nil, fmt.Errorf("unknown command: %s", hexCommand)
}

func (ledger *LedgerDeviceMock) Close() error {
	// Nothing to do here
	return nil
}
