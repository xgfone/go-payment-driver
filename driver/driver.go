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
	"cmp"
	"context"
	"net/http"
	"time"

	"github.com/xgfone/go-toolkit/codeint"
	"github.com/xgfone/go-toolkit/timex"
)

var (
	ErrPaid                = codeint.ErrPaid
	ErrUnallowed           = codeint.ErrUnallowed
	ErrBadRequest          = codeint.ErrBadRequest
	ErrUnsupported         = codeint.ErrUnsupported
	ErrBalanceInsufficient = codeint.ErrInsufficientBalance
	ErrTooSmallTradeAmount = ErrUnallowed.WithReason("trade amount is too small")
	ErrTradeRefundedFully  = ErrUnallowed.WithReason("trade has been refunded fully")
)

const DefaultTimeout = time.Minute * 5

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

// Trade
type (
	CreateTradeRequest struct {
		// Required.
		TradeNo       string `json:",omitzero"`
		TradeDesc     string `json:",omitzero"`
		TradeAmount   int64  `json:",omitzero"` // the smallest currency unit, such as Cent
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
		OpenId  string `json:",omitzero"`
		TradeNo string `json:",omitzero"`

		ChannelData    string `json:",omitzero"`
		ChannelTradeNo string `json:",omitzero"`
	}

	QueryTradeRequest struct {
		OpenId  string `json:",omitzero"`
		TradeNo string `json:",omitzero"`

		ChannelData    string `json:",omitzero"`
		ChannelTradeNo string `json:",omitzero"`
	}

	LinkInfo struct {
		PayLink string `json:",omitzero"`

		ChannelData    string `json:",omitzero"`
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

func (r *CreateTradeRequest) GetTimeout() time.Duration {
	return cmp.Or(r.Timeout, DefaultTimeout)
}

func (r *CreateTradeRequest) ExipredAt() time.Time {
	return timex.Now().Add(r.GetTimeout())
}

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

		TradeNo  string `json:",omitzero"`
		RefundNo string `json:",omitzero"`

		ChannelData    string `json:",omitzero"`
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

func NewMetadata(provider, payscene string) Metadata {
	return Metadata{Provider: provider, PayScene: payscene}
}

func (md Metadata) WithLinkType(linktype LinkType) Metadata {
	md.LinkType = linktype
	return md
}

func (md Metadata) WithChannels(channels []string) Metadata {
	md.Channels = channels
	return md
}

func (md Metadata) WithCurrencies(currencies []string) Metadata {
	md.Currencies = currencies
	return md
}

type Metadata struct {
	// The unique type of the payment channel driver, such as weixin_h5, alipay_jsapi, etc.
	//
	// If empty, maybe use "${Provider}_${PayScene}" instead.
	Type string

	// The link type of the payment channel driver, such as weixin_h5, alipay_jsapi, etc.
	//
	// If empty, maybe use Type instead.
	LinkType LinkType

	// The provider of the payment service, such as weixin, alipay, stripe, etc.
	//
	// Required.
	Provider string

	// The scene of the payment service, such as h5, app, jsapi, native, qrcode, etc.
	//
	// Required.
	PayScene string

	// The channels supported by the payment service, for example,
	// ["weixin"] for "weixin", ["alipay"] for "alipay",
	// ["weixin", "alipay"] for some aggregation providers,
	// etc.
	Channels []string

	// The ISO 4127 currency list supported by the payment channel driver,
	// for example, ["USD"], ["CNY"], ["USD", "CNY"].
	Currencies []string

	// Whether to need to use ChannelData in XxxRequest after creating trade.
	NeedChannelData bool
}

type Driver interface {
	Metadata() Metadata

	CreateTrade(ctx context.Context, req CreateTradeRequest) (info LinkInfo, err error)
	QueryTrade(ctx context.Context, req QueryTradeRequest) (info TradeInfo, ok bool, err error)

	// If has paid, return ErrPaid
	// If the trade has been canceled, return nil.
	CancelTrade(ctx context.Context, req CancelTradeRequest) (err error)

	// If the trade has been fully refunded, return ErrTradeRefundedFully.
	// If the balance is insufficient, return ErrBalanceInsufficient.
	// If it's not allowed to refund the trade, return ErrUnallowed.
	RefundTrade(ctx context.Context, req RefundTradeRequest) (info RefundInfo, err error)
	QueryRefund(ctx context.Context, req QueryRefundRequest) (info RefundInfo, ok bool, err error)

	ParseTradeCallbackRequest(ctx context.Context, r *http.Request) (info TradeInfo, err error)
	SendTradeCallbackResponse(ctx context.Context, w http.ResponseWriter, err error)

	ParseRefundCallbackRequest(ctx context.Context, r *http.Request) (info RefundInfo, err error)
	SendRefundCallbackResponse(ctx context.Context, w http.ResponseWriter, err error)
}
