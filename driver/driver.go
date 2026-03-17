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

// Package driver provides the interface and implementations of the payment channel driver.
package driver

import (
	"context"
	"net/http"
	"time"

	"github.com/xgfone/go-toolkit/codeint"
)

var (
	ErrPaid                = codeint.ErrPaid
	ErrUnallowed           = codeint.ErrUnallowed
	ErrBadRequest          = codeint.ErrBadRequest
	ErrUnsupported         = codeint.ErrUnsupported
	ErrBalanceInsufficient = codeint.ErrInsufficientBalance
	ErrTooSmallTradeAmount = ErrUnallowed.WithReason("trade amount is too small")
)

const (
	TaskStatusClosed     TaskStatus = "Closed"
	TaskStatusUnknown    TaskStatus = "Unknown"
	TaskStatusSuccess    TaskStatus = "Success"
	TaskStatusFailure    TaskStatus = "Failure"
	TaskStatusProcessing TaskStatus = "Processing"
)

const (
	LinkTypeCodeUrl LinkType = "code_url"
)

type (
	LinkType   string
	TaskStatus string
)

func (lt LinkType) LinkInfo(paylink string) LinkInfo {
	return LinkInfo{LinkType: lt, PayLink: paylink}
}

// Trade
type (
	CreateTradeRequest struct {
		// Required.
		TradeNo       string
		TradeDesc     string
		TradeAmount   int64  // the smallest currency unit, such as Cent
		TradeCurrency string `json:",omitzero"`
		CallbackUrl   string `json:",omitzero"`

		Timeout time.Duration `json:",omitzero"`

		// Optional.
		OpenId   string `json:",omitzero"`
		ClientIp string `json:",omitzero"`
		ExtInfo  any    `json:",omitzero"`

		Share bool `json:",omitzero"`
	}

	CancelTradeRequest struct {
		OpenId         string `json:",omitzero"`
		TradeNo        string `json:",omitzero"`
		ChannelTradeNo string `json:",omitzero"`
	}

	QueryTradeRequest struct {
		OpenId         string `json:",omitzero"`
		TradeNo        string `json:",omitzero"`
		ChannelTradeNo string `json:",omitzero"`
	}

	LinkInfo struct {
		PayLink  string   `json:",omitzero"`
		LinkType LinkType `json:",omitzero"`

		ChannelTradeNo string `json:",omitzero"`
	}

	TradeInfo struct {
		TradeNo       string `json:",omitzero"`
		TradeAmount   int64  `json:",omitzero"` // the smallest currency unit, such as Cent
		TradeCurrency string `json:",omitzero"`

		ChannelTradeNo string `json:",omitzero"`
		ChannelStatus  string `json:",omitzero"`
		ChannelData    string `json:",omitzero"` // such as BankType

		PayerId      string    `json:",omitzero"`
		PaidCurrency string    `json:",omitzero"`
		PaidAmount   int64     `json:",omitzero"` // the smallest currency unit, such as Cent
		PaidAt       time.Time `json:",omitzero"`

		IsRefunded bool       `json:",omitzero"`
		FailReason string     `json:",omitzero"`
		TaskStatus TaskStatus `json:",omitzero"`
	}
)

// Refund
type (
	RefundTradeRequest struct {
		OpenId string `json:",omitzero"`

		TradeNo     string `json:",omitzero"`
		TradeAmount int64  `json:",omitzero"` // the smallest currency unit, such as Cent

		RefundNo       string `json:",omitzero"`
		RefundReason   string `json:",omitzero"`
		RefundAmount   int64  `json:",omitzero"` // the smallest currency unit, such as Cent
		RefundCurrency string `json:",omitzero"`
		FundAccount    string `json:",omitzero"`

		CallbackUrl string `json:",omitzero"`

		ChannelData    string `json:",omitzero"`
		ChannelTradeNo string `json:",omitzero"`
	}

	QueryRefundRequest struct {
		OpenId string `json:",omitzero"`

		TradeNo  string
		RefundNo string

		ChannelTradeNo string `json:",omitzero"`
	}

	RefundInfo struct {
		TradeNo     string `json:",omitzero"`
		TradeAmount int64  `json:",omitzero"` // the smallest currency unit, such as Cent

		RefundNo     string `json:",omitzero"`
		RefundReason string `json:",omitzero"`
		RefundAmount int64  `json:",omitzero"` // the smallest currency unit, such as Cent

		ChannelTradeNo  string `json:",omitzero"`
		ChannelRefundNo string `json:",omitzero"`
		ChannelStatus   string `json:",omitzero"`
		ChannelData     string `json:",omitzero"`

		RefundedAt time.Time  `json:",omitzero"`
		FailReason string     `json:",omitzero"`
		TaskStatus TaskStatus `json:",omitzero"`
	}
)

func (r RefundTradeRequest) QueryRefundRequest() QueryRefundRequest {
	return QueryRefundRequest{
		OpenId: r.OpenId,

		TradeNo:  r.TradeNo,
		RefundNo: r.RefundNo,

		ChannelTradeNo: r.ChannelTradeNo,
	}
}

type Metadata struct {
	// The unique type of the payment channel driver, such as weixin_h5, alipay_jsapi, etc.
	Type string

	// The link type of the payment channel driver, such as weixin_h5, alipay_jsapi, etc.
	//
	// If empty, use Type instead.
	LinkType LinkType

	// The provider of the payment service, such as weixin, alipay, stripe, etc.
	Provider string

	// The scene of the payment service, such as h5, app, jsapi, native, qrcode, etc.
	PayScene string

	// The channels of the payment service, for example,
	// ["weixin"] for "weixin", ["alipay"] for "alipay",
	// ["weixin", "alipay"] for some aggregation providers,
	// etc.
	Channels []string
}

type Driver interface {
	Metadata() Metadata

	CreateTrade(ctx context.Context, req CreateTradeRequest) (info LinkInfo, err error)
	QueryTrade(ctx context.Context, req QueryTradeRequest) (info TradeInfo, ok bool, err error)

	// If has paid, return ErrPaid
	// If the trade has been canceled, return nil.
	CancelTrade(ctx context.Context, req CancelTradeRequest) (err error)

	// If the trade has been fully refunded, return nil.
	// If the balance is insufficient, return ErrBalanceInsufficient.
	// If it's not allowed to refund the trade, return ErrUnallowed.
	RefundTrade(ctx context.Context, req RefundTradeRequest) (info RefundInfo, err error)
	QueryRefund(ctx context.Context, req QueryRefundRequest) (info RefundInfo, ok bool, err error)

	ParseTradeCallbackRequest(ctx context.Context, r *http.Request) (info TradeInfo, err error)
	SendTradeCallbackResponse(ctx context.Context, w http.ResponseWriter, err error)

	ParseRefundCallbackRequest(ctx context.Context, r *http.Request) (info RefundInfo, err error)
	SendRefundCallbackResponse(ctx context.Context, w http.ResponseWriter, err error)
}
