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

const mockDeviceName = "Mock device"

type LedgerAdminMock struct{}

type LedgerDeviceMock struct {
	commands map[string]string
}

func NewLedgerAdmin() LedgerAdmin {
	return &LedgerAdminMock{}
}

func (admin *LedgerAdminMock) ListDevices() ([]string, error) {
	return []string{mockDeviceName}, nil
}

func (admin *LedgerAdminMock) CountDevices() int {
	return 1
}

func (admin *LedgerAdminMock) Connect(deviceIndex int) (LedgerDevice, error) {
	return NewLedgerDeviceMock(), nil
}

func NewLedgerDeviceMock() *LedgerDeviceMock {
	return &LedgerDeviceMock{
		commands: make(map[string]string),
	}
}

func (ledger *LedgerDeviceMock) Exchange(command []byte) ([]byte, error) {
	hexCommand := hex.EncodeToString(command)
	if reply, ok := ledger.commands[hexCommand]; ok {
		return hex.DecodeString(reply)
	}
	return nil, fmt.Errorf("unknown command: %s", hexCommand)
}

func (ledger *LedgerDeviceMock) SetCommandReplies(commands map[string]string) {
	ledger.commands = commands
}

func (ledger *LedgerDeviceMock) ClearCommands() {
	ledger.commands = make(map[string]string)
}

func (ledger *LedgerDeviceMock) Close() error {
	// Nothing to do here
	return nil
}
