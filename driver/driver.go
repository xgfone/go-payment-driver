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
	"github.com/xgfone/go-toolkit/jsonx"
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
	Buyer struct {
		// Buyer's Open ID, such as OpenId under WeChat Mini Program or Service Account,
		// specific usage depends on the support of the payment provider.
		OpenId string `json:",omitzero"`

		// Buyer's IP address when creating the payment.
		//
		// Note: Jialian Payment or WeChat H5 payment requires this parameter.
		ClientIp string `json:",omitzero"`
	}

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

		// Extended information for specific information required
		// by specific payment providers.
		ExtInfo any `json:",omitzero"`

		// Buyer information.
		//
		// Note: some payment providers require this parameter information.
		// If not needed, the driver implementation should ignore it.
		Buyer Buyer `json:",omitzero"`

		// Whether to enable profit sharing when creating a payment.
		//
		// Note: For WeChat Pay, if profit sharing is needed, it requires
		// that the profit sharing flag must be set when creating the payment.
		Share bool `json:",omitzero"`
	}

	CancelPaymentRequest struct {
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

		// Unique refund ID corresponding to the payment provider side.
		ChannelRefundId string `json:",omitzero"`

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
		RefundId:  r.RefundId,
		PaymentId: r.PaymentId,

		ChannelData:      r.ChannelData,
		ChannelPaymentId: r.ChannelPaymentId,
	}
}

const (
	CallbackTypeRefund  CallbackType = "Refund"
	CallbackTypePayment CallbackType = "Payment"
)

type (
	CallbackType string

	CallbackRequest struct {
		// It will be empty if the payment channel provider, such as PayPal,
		// uses the webhook callback based on the event. Or, it will be set
		// to the specific callback type, such as weixin or wechat.
		Type CallbackType

		Request *http.Request
	}

	CallbackResponse struct {
		// The type of the callback.
		Type CallbackType

		// The parsed Refund or Payment information by the callback type.
		RefundInfo  *RefundInfo
		PaymentInfo *PaymentInfo
	}
)

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

func (md Metadata) WithCurrencies(currencies ...string) Metadata {
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

func EncodeChannelData[T any](channelData T) (channelDataStr string) {
	channelDataStr, _ = jsonx.MarshalString(channelData)
	if channelDataStr == "{}" {
		channelDataStr = ""
	}
	return
}

func DecodeChannelData[T any](channelDataStr string) (channelData T) {
	if channelDataStr == "" || channelDataStr == "{}" {
		return
	}

	_ = jsonx.UnmarshalString(channelDataStr, &channelData)
	return
}

type Driver interface {
	// Metadata returns the metadata of the payment channel driver.
	Metadata() Metadata

	// CreatePayment creates a payment and returns the pay link information, such as qrcode url.
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (info PayLinkInfo, err error)

	// QueryPayment queries the payment status information.
	QueryPayment(ctx context.Context, req QueryPaymentRequest) (info PaymentInfo, ok bool, err error)

	// CancelPayment cancels the unpaid payment.
	//
	// If has paid, return driver.ErrPaid.
	// If the payment has been canceled or closed, return nil.
	CancelPayment(ctx context.Context, req CancelPaymentRequest) (err error)

	// RefundPayment refunds the paid payment.
	//
	// If the payment has been fully refunded, return driver.ErrPaymentRefundedFully.
	// If the balance is insufficient, return driver.ErrBalanceInsufficient.
	// If it's not allowed to refund the payment, return driver.ErrUnallowed.
	RefundPayment(ctx context.Context, req CreateRefundRequest) (info RefundInfo, err error)

	// QueryRefund queries the refund status information.
	QueryRefund(ctx context.Context, req QueryRefundRequest) (info RefundInfo, ok bool, err error)

	// ParseCallbackRequest parses the callback request and returns the parsed information.
	ParseCallbackRequest(ctx context.Context, req CallbackRequest) (resp CallbackResponse, err error)

	// SendCallbackResponse sends the callback response.
	//
	// If err is nil, the response will be sent successfully.
	// If err is not nil, the response will be sent with the error.
	SendCallbackResponse(ctx context.Context, rw http.ResponseWriter, err error)
}
