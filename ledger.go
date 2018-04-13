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
    "fmt"
    "errors"
    "github.com/brejski/hid"
	"os"
	"encoding/hex"
)

var (
	errTooShort        = errors.New("too short")
	errInvalidChannel  = errors.New("invalid channel")
	errInvalidSequence = errors.New("invalid sequence")
	errInvalidTag      = errors.New("invalid tag")
)
const (
    VendorLedger = 0x2c97
    ProductNano  = 1
    Channel      = 0x8001
    PacketSize   = 64
)

type Ledger struct {
    device Device
}

func NewLedger(dev Device) *Ledger {
    return &Ledger{
        device: dev,
    }
}

func FindLedger() (*Ledger, error) {
    devs, err := hid.Devices()
    if err != nil {
        return nil, err
    }
    for _, d := range devs {
        // TODO: ProductId filter
        if d.VendorID == VendorLedger {
            ledger, err := d.Open()
            if err != nil {
                return nil, err
            }
            return NewLedger(ledger), nil
        }
    }
    return nil, errors.New("no ledger connected")
}

// A Device provides access to a HID device.
type Device interface {
    // Close closes the device and associated resources.
    Close()

    // Write writes an output report to device. The first byte must be the
    // report number to write, zero if the device does not use numbered reports.
    Write([]byte) error

    // ReadCh returns a channel that will be sent input reports from the device.
    // If the device uses numbered reports, the first byte will be the report
    // number.
    ReadCh() <-chan []byte

    // ReadError returns the read error, if any after the channel returned from
    // ReadCh has been closed.
    ReadError() error
}

func main() {
    ledger, err := FindLedger()
    if err != nil {
        fmt.Printf("Could not find ledger.")
        os.Exit(-1)
    }
    if ledger != nil {
		header := make([]byte, 4)
		header[0] = 0x80;
		header[1] = 0x01;
		header[2] = 0x01;
		header[3] = 0x01;
		fullMessage := header

		fmt.Println(fullMessage)
		adpu, _ := WrapCommandAPDU(Channel, fullMessage, PacketSize, false)

		// write all the packets
		err := ledger.device.Write(adpu[:PacketSize])
		if err != nil {
			fmt.Printf("Could not write to ledger.")
			os.Exit(-1)
		}
		for len(adpu) > PacketSize {
			adpu = adpu[PacketSize:]
			err = ledger.device.Write(adpu[:PacketSize])
			if err != nil {
				fmt.Printf("Could not write to ledger.")
				os.Exit(-1)
			}
		}

		input := ledger.device.ReadCh()
		response, err := UnwrapResponseAPDU(Channel, input, PacketSize, false)

		if len(response) < 3{
			fmt.Printf("Response length %d", len(response))
			os.Exit(-1)
		}

		swOffset := len(response) - 2
		sw := codec.Uint16(response[swOffset:])
		if sw != 0x9000 {
			fmt.Errorf("Invalid status %04x", sw)
		}
		fmt.Printf("Signature:" + hex.EncodeToString(response[:swOffset]))
	}
}
