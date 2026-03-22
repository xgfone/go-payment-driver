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
	"slices"
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

	ErrPaymentRefundedFully  = ErrUnallowed.WithReason("payment has been refunded fully")
	ErrTooSmallPaymentAmount = ErrUnallowed.WithReason("payment amount is too small")
)

const DefaultExpiresIn = time.Minute * 5

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

// Payment
type (
	CreatePaymentRequest struct {
		// Our unique payment id.
		PaymentId string `json:",omitzero"`

		// Payment description information
		PaymentDesc string `json:",omitzero"`

		// Payment amount, in the smallest currency unit of the payment currency, such as cents.
		PaymentAmount int64 `json:",omitzero"`

		// Payment currency, following ISO 4217 standard, all uppercase.
		PaymentCurrency string `json:",omitzero"`

		// Success callback URL passed to the payment provider when creating a payment.
		//
		// That is, after payment is successful, the payment provider should notify
		// the payment result to this callback URL as much as possible.
		//
		// If the payment provider does not support callbacks, for example,
		// the payment provider uses Webhook notifications,
		// the driver implementation should ignore it.
		CallbackUrl string `json:",omitzero"`

		// Payment validity period, after which the payment will automatically
		// become invalid or automatically close.
		//
		// Note: If the payment provider does not support automatic invalidation
		// or automatic closing, the driver implementation should ignore it.
		ExpiresIn time.Duration `json:",omitzero"`

		// Open ID, such as OpenId under WeChat Mini Program or Service Account,
		// specific usage depends on the support of the payment provider.
		OpenId string `json:",omitzero"`

		// Buyer's IP address when creating the payment.
		//
		// Note: Jialian Payment or WeChat H5 payment requires this parameter.
		ClientIp string `json:",omitzero"`

		// Extended information for specific information required
		// by specific payment providers.
		ExtInfo any `json:",omitzero"`

		// Whether to enable profit sharing when creating a payment.
		//
		// Note: For WeChat Pay, if profit sharing is needed, it requires
		// that the profit sharing flag must be set when creating the payment.
		Share bool `json:",omitzero"`
	}

	CancelPaymentRequest struct {
		// Open ID, such as OpenId under WeChat Mini Program or Service Account,
		// specific usage depends on the support of the payment provider.
		OpenId string `json:",omitzero"`

		// Our unique payment id.
		PaymentId string `json:",omitzero"`

		// Payment information on the payment provider side.
		//
		// For example: XiaoHongShu local guaranteed transactions require
		// filling in product information, so payment-related product
		// information can be put into this field.
		ChannelData string `json:",omitzero"`

		// Unique payment id corresponding to the payment provider side.
		ChannelPaymentId string `json:",omitzero"`
	}

	QueryPaymentRequest struct {
		// Open ID, such as OpenId under WeChat Mini Program or Service Account,
		// specific usage depends on the support of the payment provider.
		OpenId string `json:",omitzero"`

		// Our unique payment id.
		PaymentId string `json:",omitzero"`

		// Payment information on the payment provider side.
		//
		// For example: XiaoHongShu local guaranteed transactions require
		// filling in product information, so payment-related product
		// information can be put into this field.
		ChannelData string `json:",omitzero"`

		// Unique payment id corresponding to the payment provider side.
		ChannelPaymentId string `json:",omitzero"`
	}

	PayLinkInfo struct {
		// Link returned to the frontend for user payment.
		PayLink string `json:",omitzero"`

		// Payment information on the payment provider side.
		//
		// Note: This field can be used for subsequent payment query or cancellation.
		ChannelData string `json:",omitzero"`

		// Unique payment id corresponding to the payment provider side.
		//
		// Note: If the payment provider does not provide this information
		// when creating the payment, this field can be ignored.
		ChannelPaymentId string `json:",omitzero"`
	}

	PaymentInfo struct {
		// Our unique payment id.
		PaymentId string `json:",omitzero"`

		// Unique payment id corresponding to the payment provider side.
		ChannelPaymentId string `json:",omitzero"`

		// Original value of the payment status on the payment provider side.
		ChannelStatus string `json:",omitzero"`

		// Additional payment information on the payment provider side,
		// such as BankType.
		//
		// Note: Generally uniformly encoded in JSON.
		ChannelData string `json:",omitzero"`

		// Unique ID of the paying user on the payment provider side,
		// or other ID that can uniquely identify the paying user.
		PayerId string `json:",omitzero"`

		// Currency used by the user for payment, following ISO 4217 standard.
		PayerPaidCurrency string `json:",omitzero"`

		// Actual amount paid by the user, in the smallest currency unit, such as Cent.
		PayerPaidAmount int64 `json:",omitzero"`

		// Time when the user actually completed the payment on the payment provider side.
		PayerPaidAt time.Time `json:",omitzero"`

		// Whether the current payment has been fully refunded.
		IsRefunded bool `json:",omitzero"`

		// If payment fails, it indicates the reason for failure on the payment provider side.
		//
		// Note: If the payment provider does not support this information,
		// the driver implementation can ignore it.
		FailReason string `json:",omitzero"`

		// Current task status of the payment.
		TaskStatus TaskStatus `json:",omitzero"`
	}
)

func (r *CreatePaymentRequest) GetExpiresIn() time.Duration {
	return cmp.Or(r.ExpiresIn, DefaultExpiresIn)
}

func (r *CreatePaymentRequest) GetExipredAt() time.Time {
	return timex.Now().Add(r.GetExpiresIn())
}

// Refund
type (
	CreateRefundRequest struct {
		// Open ID, such as OpenId under WeChat Mini Program or Service Account,
		// specific usage depends on the support of the payment provider.
		OpenId string `json:",omitzero"`

		// Our unique payment id.
		PaymentId string `json:",omitzero"`

		// Payment amount, in the smallest currency unit of the payment currency, such as cents.
		PaymentAmount int64 `json:",omitzero"`

		// Currency corresponding to the payment amount, following ISO 4217 standard.
		PaymentCurrency string `json:",omitzero"`

		// Our unique payment refund id.
		RefundId string `json:",omitzero"`

		// Refund amount, in the smallest currency unit of the payment currency, such as cents.
		RefundAmount int64 `json:",omitzero"`

		// Refund reason.
		RefundReason string `json:",omitzero"`

		// Refund success callback URL passed to the payment provider during refund.
		//
		// That is, after refund is successful, the payment provider should notify
		// the refund result to this callback URL as much as possible.
		//
		// If the payment provider does not support callbacks,
		// such as when the payment provider uses Webhook notifications,
		// the driver implementation should ignore it.
		CallbackUrl string `json:",omitzero"`

		// Additional payment information on the payment provider side.
		//
		// Note: For Xiaohongshu local guaranteed transactions, when refunding,
		// it is necessary to provide the product information in the original payment,
		// and the ChannelData field in the payment can be passed as this field.
		ChannelData string `json:",omitzero"`

		// Unique payment id corresponding to the payment provider side.
		ChannelPaymentId string `json:",omitzero"`
	}

	QueryRefundRequest struct {
		// Open ID, such as OpenId under WeChat Mini Program or Service Account,
		// specific usage depends on the support of the payment provider.
		OpenId string `json:",omitzero"`

		// Our unique payment id.
		PaymentId string `json:",omitzero"`

		// Our unique payment refund id.
		RefundId string `json:",omitzero"`

		// Additional payment information on the payment provider side.
		//
		// Note: For Xiaohongshu local guaranteed transactions, when refunding,
		// it is necessary to provide the product information in the original payment,
		// and the ChannelData field in the payment can be passed as this field.
		ChannelData string `json:",omitzero"`

		// Unique payment id corresponding to the payment provider side.
		ChannelPaymentId string `json:",omitzero"`
	}

	RefundInfo struct {
		// Our unique payment id.
		PaymentId string `json:",omitzero"`

		// Our unique payment refund ID.
		RefundId string `json:",omitzero"`

		// Refund reason on the payment provider side.
		//
		// Note: If the payment provider does not support this parameter,
		// the driver implementation can ignore it.
		RefundReason string `json:",omitzero"`

		// Unique payment id corresponding to the payment provider side.
		//
		// Deprecated. No longer recommended for use.
		// ChannelPaymentId string `json:",omitzero"`

		// Unique refund ID corresponding to the payment provider side.
		ChannelRefundId string `json:",omitzero"`

		// Original value of the refund status on the payment provider side.
		ChannelStatus string `json:",omitzero"`

		// Additional refund information on the payment provider side.
		//
		// Note: Generally uniformly encoded in JSON.
		ChannelData string `json:",omitzero"`

		// Time when the payment provider side completed the refund.
		RefundedAt time.Time `json:",omitzero"`

		// If refund fails, it indicates the reason for failure on the payment provider side.
		//
		// Note: If the payment provider does not support this information,
		// the driver implementation can ignore it.
		FailReason string `json:",omitzero"`

		// Current task status of the refund operation.
		TaskStatus TaskStatus `json:",omitzero"`
	}
)

func (r CreateRefundRequest) QueryRefundRequest() QueryRefundRequest {
	return QueryRefundRequest{
		OpenId: r.OpenId,

		RefundId:  r.RefundId,
		PaymentId: r.PaymentId,

		ChannelData:      r.ChannelData,
		ChannelPaymentId: r.ChannelPaymentId,
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
}

// CurrencyIsSupported reports whether the currency is supported by the payment channel driver.
func (md *Metadata) CurrencyIsSupported(currencyCode string) bool {
	return slices.Contains(md.Currencies, currencyCode)
}

// ChannelIsSupported reports whether the channel is supported by the payment channel driver.
func (md *Metadata) ChannelIsSupported(channel string) bool {
	return slices.Contains(md.Channels, channel)
}

type Driver interface {
	Metadata() Metadata

	CreatePayment(ctx context.Context, req CreatePaymentRequest) (info PayLinkInfo, err error)
	QueryPayment(ctx context.Context, req QueryPaymentRequest) (info PaymentInfo, ok bool, err error)

	// If has paid, return ErrPaid
	// If the payment has been canceled, return nil.
	CancelPayment(ctx context.Context, req CancelPaymentRequest) (err error)

	// If the payment has been fully refunded, return ErrPaymentRefundedFully.
	// If the balance is insufficient, return ErrBalanceInsufficient.
	// If it's not allowed to refund the payment, return ErrUnallowed.
	RefundPayment(ctx context.Context, req CreateRefundRequest) (info RefundInfo, err error)
	QueryRefund(ctx context.Context, req QueryRefundRequest) (info RefundInfo, ok bool, err error)

	ParsePaymentCallbackRequest(ctx context.Context, req *http.Request) (info PaymentInfo, err error)
	ParseRefundCallbackRequest(ctx context.Context, req *http.Request) (info RefundInfo, err error)

	SendPaymentCallbackResponse(ctx context.Context, rw http.ResponseWriter, err error)
	SendRefundCallbackResponse(ctx context.Context, rw http.ResponseWriter, err error)
}
