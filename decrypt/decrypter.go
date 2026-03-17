// Copyright 2026 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package decrypt provides the decryption function based on the payment channel driver.
package decrypt

import (
	"context"
	"net/http"

	"github.com/xgfone/go-payment-driver/driver"
)

const (
	TypeTrade  = "trade"
	TypeRefund = "refund"
	TypeIgnore = "ignore"
)

type Request struct {
	Type string

	TradeInfo  *driver.TradeInfo
	RefundInfo *driver.RefundInfo

	Requery bool
}

type Decrypter interface {
	DecryptRequest(ctx context.Context, hreq *http.Request) (dreq Request, err error)
	DecryptRequestData(ctx context.Context, req *http.Request) (data string, err error)
	DecryptResponseSend(ctx context.Context, w http.ResponseWriter, err error)
}

// IsSupported checks whether the payment channel driver supports the Decrypter interface.
func IsSupported(driver driver.Driver) bool {
	_, ok := driver.(Decrypter)
	return ok
}

// If the payment channel driver does not support the Decrypter interface, return nil.
func Get(driver driver.Driver) Decrypter {
	s, _ := driver.(Decrypter)
	return s
}
