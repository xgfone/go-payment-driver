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

// Package share provides the share function based on the payment channel driver.
package share

import (
	"context"
	"time"

	"github.com/xgfone/go-payment-driver/driver"
)

var ErrReturnAccountIsShareAccount = driver.ErrUnallowed.WithReason("return account is equal to share account")

const (
	StatusProcessing Status = "Processing"
	StatusFinished   Status = "Finished"
)

const (
	ResultProcess Result = "Process"
	ResultSuccess Result = "Success"
	ResultFailure Result = "Failure"
)

type (
	Status string
	Result string
)

// Share
type (
	QueryShareRequest struct {
		ShareNo        string `json:",omitzero"`
		TradeNo        string `json:",omitzero"`
		ChannelTradeNo string `json:",omitzero"`
	}

	ApplyShareRequest struct {
		ShareNo         string
		TradeNo         string
		ChannelTradeNo  string
		ShareReceivers  []ShareReceiver `json:",omitzero"`
		UnfreezeUnsplit bool            `json:",omitzero"`
	}

	ShareReceiver struct {
		ShareAmount int64       `json:",omitzero"` // the smallest currency unit, such as Cent
		ShareDesc   string      `json:",omitzero"` // max length: 32
		Receiver    AccountInfo `json:",omitzero"`
	}

	AccountInfo struct {
		AccountType AccountType `json:",omitzero"`
		Account     string      `json:",omitzero"`
	}

	ShareInfo struct {
		ShareNo        string `json:",omitzero"`
		ChannelShareNo string `json:",omitzero"`
		ChannelTradeNo string `json:",omitzero"`

		ShareRecords []ShareRecord `json:",omitzero"`
		ShareStatus  Status        `json:",omitzero"`
	}

	ShareRecord struct {
		ShareReceiver
		Sender AccountInfo `json:",omitzero"`

		IsInitiator     bool   `json:",omitzero"`
		ChannelDetailId string `json:",omitzero"`

		CreatedAt  time.Time `json:",omitzero"`
		FinishedAt time.Time `json:",omitzero"`

		FailReason  string `json:",omitzero"`
		ShareResult Result `json:",omitzero"`
	}
)

// Return
type (
	ReturnShareRequest struct {
		TradeNo  string `json:",omitzero"`
		ShareNo  string `json:",omitzero"`
		ReturnNo string `json:",omitzero"`

		ChannelShareNo string `json:",omitzero"`
		ChannelTradeNo string `json:",omitzero"`

		ReturnAccount string `json:",omitzero"`
		ReturnAmount  int64  // the smallest currency unit, such as Cent
		ReturnDesc    string
	}

	QueryReturnRequest struct {
		TradeNo  string `json:",omitzero"`
		ShareNo  string `json:",omitzero"`
		ReturnNo string `json:",omitzero"`
	}

	ReturnInfo struct {
		TradeNo  string `json:",omitzero"`
		ShareNo  string `json:",omitzero"`
		ReturnNo string `json:",omitzero"`

		ChannelShareNo  string `json:",omitzero"`
		ChannelReturnNo string `json:",omitzero"`

		ReturnAmount  int64  `json:",omitzero"` // the smallest currency unit, such as Cent
		ReturnAccount string `json:",omitzero"`
		ReturnDesc    string `json:",omitzero"`

		FailReason   string `json:",omitzero"`
		ReturnResult Result `json:",omitzero"`

		CreatedAt  time.Time `json:",omitzero"`
		FinishedAt time.Time `json:",omitzero"`
	}
)

type Sharer interface {
	// If the balance is insufficient, return driver.ErrBalanceInsufficient.
	ApplyShare(ctx context.Context, req ApplyShareRequest) (info ShareInfo, err error)
	QueryShare(ctx context.Context, req QueryShareRequest) (info ShareInfo, ok bool, err error)

	// If the return account is equal to the share account, return share.ErrReturnAccountIsShareAccount.
	ReturnShare(ctx context.Context, req ReturnShareRequest) (info ReturnInfo, err error)
	QueryReturn(ctx context.Context, req QueryReturnRequest) (info ReturnInfo, ok bool, err error)

	AddShareReceiver(ctx context.Context, r Receiver) (err error)
	DeleteShareReceiver(ctx context.Context, r Receiver) (err error)
}

// IsSupported checks whether the payment channel driver supports the Sharer interface.
func IsSupported(driver driver.Driver) bool {
	_, ok := driver.(Sharer)
	return ok
}

// If the payment channel driver does not support the Sharer interface, return nil.
func Get(driver driver.Driver) Sharer {
	sharer, _ := driver.(Sharer)
	return sharer
}
