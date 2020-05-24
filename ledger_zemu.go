//+build ledger_zemu

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

type LedgerAdminZemu struct {
	// TODO: Add url, etc.
}

type LedgerDeviceZemu struct {
}

func NewLedgerAdmin( /*pass grpc url, maybe hardcode?, etc.*/) *LedgerAdminZemu {
	return &LedgerAdminZemu{
		// TODO: Add url, etc.
	}
}

func (admin *LedgerAdminZemu) ListDevices() ([]string, error) {
	// It does not make sense for zemu devices
	x := []string{"Zemu device"}
	return x, nil
}

func (admin *LedgerAdminZemu) CountDevices() int {
	// TODO: Always 1, maybe zero if zemu has not elf??
	return 1
}

func (admin *LedgerAdminZemu) Connect(deviceIndex int) (*LedgerDeviceZemu, error) {
	// TODO: Confirm GRPC could connect
	return &LedgerDeviceZemu{}, nil
}

func (ledger *LedgerDeviceZemu) Exchange(command []byte) ([]byte, error) {
	// Send to Zemu and return reply or error
	return []byte{}, nil
}

func (ledger *LedgerDeviceZemu) Close() error {
	// TODO: Any clean update that we may need to do here
	return nil
}
