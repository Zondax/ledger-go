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

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
)

type LedgerAdminZemu struct {
	grpcURL  	string
	grpcPort 	string
}

type LedgerDeviceZemu struct {
	connection 	*grpc.ClientConn
	client		ZemuCommandClient
}

func NewLedgerAdmin() *LedgerAdminZemu {
	return &LedgerAdminZemu{
		grpcURL: "localhost", //TODO get this from flag value
		grpcPort: "3002", //TODO get this from flag value
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
	serverAddr := admin.grpcURL +  ":" + admin.grpcPort
	//TODO: check Dial flags
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())

	if err != nil {
		err = fmt.Errorf("could not connect to rpc server at %q : %q", serverAddr, err)
		return &LedgerDeviceZemu{}, err
	}

	client := NewZemuCommandClient(conn)

	return &LedgerDeviceZemu{connection: conn, client: client}, nil
}

func (ledger *LedgerDeviceZemu) Exchange(command []byte) ([]byte, error) {
	// Send to Zemu and return reply or error
	r, err := ledger.client.Exchange(context.Background(), &ExchangeRequest{Command: command})

	if err != nil {
		err = fmt.Errorf("could not call rpc service: %q", err)
		return []byte{}, err
	}

	return r.Reply, nil
}

func (ledger *LedgerDeviceZemu) Close() error {
	err := ledger.connection.Close()

	if err != nil {
		err = fmt.Errorf("could not close connection to rpc server")
		return err
	}

	return nil
}
