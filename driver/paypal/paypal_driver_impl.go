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

package paypal

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xgfone/go-payment-driver/currency"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-payment-driver/driver/paypal/internal/paypal"
	"github.com/xgfone/go-toolkit/jsonx"
)

type Driver struct {
	_Driver
}

func (d *Driver) CreatePayment(ctx context.Context, req driver.CreatePaymentRequest) (info driver.PayLinkInfo, err error) {
	if !d.metadata.CurrencyIsSupported(req.PaymentCurrency) {
		return info, driver.ErrUnsupported.WithErrorf("unsupported currency '%s'", req.PaymentCurrency)
	}

	amount, err := currency.FormatMinorToMajor(req.PaymentAmount, req.PaymentCurrency)
	if err != nil {
		return
	}

	type (
		ExperienceContext struct {
			ReturnUrl  string `json:"return_url"`
			CancelUrl  string `json:"cancel_url"`
			UserAction string `json:"user_action"`
			BrandName  string `json:"brand_name,omitempty"`
			Locale     string `json:"locale,omitempty"`
		}

		PayPalSource struct {
			ExperienceContext ExperienceContext `json:"experience_context"`
		}

		PaymentSource struct {
			PayPal PayPalSource `json:"paypal"`
		}

		PurchaseUnit struct {
			Description string        `json:"description,omitempty"`
			ReferenceId string        `json:"reference_id,omitempty"`
			CustomId    string        `json:"custom_id,omitempty"`
			Amount      paypal.Amount `json:"amount"`
		}

		CreateRequest struct {
			Intent        string         `json:"intent"`
			PurchaseUnits []PurchaseUnit `json:"purchase_units"`
			PaymentSource PaymentSource  `json:"payment_source"`
		}
	)

	_req := CreateRequest{
		Intent: "CAPTURE",
		PurchaseUnits: []PurchaseUnit{
			{
				Description: req.PaymentDesc,
				ReferenceId: req.PaymentId,
				CustomId:    req.PaymentId,
				Amount: paypal.Amount{
					Currency: req.PaymentCurrency,
					Amount:   amount,
				},
			},
		},
		PaymentSource: PaymentSource{
			PayPal: PayPalSource{
				ExperienceContext: ExperienceContext{
					ReturnUrl:  d.config.ReturnUrl,
					CancelUrl:  d.config.CancelUrl,
					UserAction: d.config.UserAction,
					BrandName:  d.config.BrandName,
					Locale:     d.config.Locale,
				},
			},
		},
	}

	header := map[string]string{
		"PayPal-Request-Id": req.PaymentId,
		"Prefer":            "return=representation",
	}

	var _resp paypal.OrderResponse
	_, _, err = d.doJSON(ctx, http.MethodPost, "/v2/checkout/orders", header, _req, &_resp)
	if err != nil {
		return info, err
	}

	approveURL := ""
	for _, l := range _resp.Links {
		if l.Rel == "approve" || l.Rel == "payer-action" {
			approveURL = l.Href
			break
		}
	}
	if approveURL == "" {
		return info, errors.New("paypal approve link not found")
	}

	info.PayLink = approveURL
	info.ChannelPaymentId = _resp.Id
	return
}

func (d *Driver) QueryPayment(ctx context.Context, req driver.QueryPaymentRequest) (info driver.PaymentInfo, ok bool, err error) {
	var resp paypal.OrderResponse
	_, _, err = d.doJSON(ctx, http.MethodGet, "/v2/checkout/orders/"+req.ChannelPaymentId, nil, nil, &resp)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") {
			return info, false, nil
		}
		return info, false, err
	}

	return d.orderToPaymentInfo(resp, req.PaymentId), true, nil
}

func (d *Driver) CancelPayment(ctx context.Context, req driver.CancelPaymentRequest) error {
	pi, ok, err := d.QueryPayment(ctx, driver.QueryPaymentRequest{
		OpenId:           req.OpenId,
		PaymentId:        req.PaymentId,
		ChannelPaymentId: req.ChannelPaymentId,
		ChannelData:      req.ChannelData,
	})
	if err != nil || !ok {
		return err
	}

	switch pi.TaskStatus {
	case driver.TaskStatusSuccess:
		return driver.ErrPaid

	case driver.TaskStatusClosed, driver.TaskStatusFailure:
		return nil

	default:
		const s = "paypal orders v2 does not provide a generic cancel endpoint; let the order expire naturally"
		return driver.ErrUnallowed.WithReason(s)
	}
}

func (d *Driver) RefundPayment(ctx context.Context, req driver.CreateRefundRequest) (info driver.RefundInfo, err error) {
	if !currency.IsSupported(req.PaymentCurrency) {
		return info, driver.ErrUnsupported.WithReasonf("unsupported currency: %s", req.PaymentCurrency)
	}

	channeldata := driver.DecodeChannelData[ChannelData](req.ChannelData)
	if channeldata.CaptureId == "" {
		pay, ok, err := d.QueryPayment(ctx, driver.QueryPaymentRequest{
			OpenId:           req.OpenId,
			PaymentId:        req.PaymentId,
			ChannelPaymentId: req.ChannelPaymentId,
			ChannelData:      req.ChannelData,
		})
		if err != nil {
			return info, err
		}
		if !ok {
			return info, driver.ErrBadRequest.WithReason("payment not found")
		}
		if pay.TaskStatus != driver.TaskStatusSuccess {
			return info, driver.ErrUnallowed.WithReason("paypal payment is not captured successfully")
		}

		channeldata = driver.DecodeChannelData[ChannelData](pay.ChannelData)
		if channeldata.CaptureId == "" {
			return info, driver.ErrUnallowed.WithReason("paypal capture id not found")
		}
	}

	amount, err := currency.FormatMinorToMajor(req.RefundAmount, req.PaymentCurrency)
	if err != nil {
		return info, driver.ErrBadRequest.WithReason(err.Error())
	}

	type RequestBody struct {
		CustomId    string `json:"custom_id"`
		InvoiceId   string `json:"invoice_id"`
		NoteToPayer string `json:"note_to_payer,omitempty"`

		Amount paypal.Amount `json:"amount"`
	}

	_req := RequestBody{
		CustomId:    req.RefundId,
		InvoiceId:   req.RefundId,
		NoteToPayer: req.RefundReason,

		Amount: paypal.Amount{
			Currency: strings.ToUpper(req.PaymentCurrency),
			Amount:   amount,
		},
	}

	path := fmt.Sprintf("/v2/payments/captures/%s/refund", channeldata.CaptureId)
	header := map[string]string{
		"PayPal-Request-Id": req.RefundId,
		"Prefer":            "return=representation",
	}

	var _resp paypal.RefundResource
	_, raw, err := d.doJSON(ctx, http.MethodPost, path, header, _req, &_resp)
	if err != nil {
		lower := strings.ToLower(string(raw))
		switch {
		case strings.Contains(lower, "refund amount is greater than"):
			return info, driver.ErrPaymentRefundedFully

		case strings.Contains(lower, "insufficient"):
			return info, driver.ErrBalanceInsufficient

		default:
			return info, err
		}
	}

	return d.refundToRefundInfo(req.PaymentId, req.RefundId, _resp, req.ChannelData), nil
}

func (d *Driver) QueryRefund(ctx context.Context, req driver.QueryRefundRequest) (info driver.RefundInfo, ok bool, err error) {
	var _resp paypal.RefundResource
	path := fmt.Sprintf("/v2/payments/refunds/%s", req.ChannelRefundId)
	_, _, err = d.doJSON(ctx, http.MethodGet, path, nil, nil, &_resp)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") {
			return info, false, nil
		}
		return info, false, err
	}

	return d.refundToRefundInfo(req.PaymentId, req.RefundId, _resp, req.ChannelData), true, nil
}

func (d *Driver) ParsePaymentCallbackRequest(ctx context.Context, req *http.Request) (info driver.PaymentInfo, err error) {
	event, raw, err := d.parseAndVerifyWebhook(ctx, req)
	if err != nil {
		return info, err
	}

	switch event.EventType {
	case "CHECKOUT.ORDER.APPROVED":
		var res struct {
			Id string `json:"id"`
		}
		if err = json.Unmarshal(raw, &res); err != nil {
			return info, err
		}

		ord, err := d.captureOrder(ctx, res.Id)
		if err != nil {
			return info, err
		}

		return d.orderToPaymentInfo(ord, ""), nil

	case "CHECKOUT.PAYMENT-APPROVAL.REVERSED":
		var res struct {
			Id string `json:"id"`
		}

		_ = jsonx.UnmarshalBytes(raw, &res)
		return driver.PaymentInfo{
			ChannelPaymentId: res.Id,
			ChannelStatus:    "REVERSED",
			FailReason:       "payment approval reversed before capture",
			TaskStatus:       driver.TaskStatusFailure,
		}, nil

	case "PAYMENT.CAPTURE.COMPLETED", "PAYMENT.CAPTURE.PENDING", "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.REFUNDED", "PAYMENT.CAPTURE.REVERSED":
		var cap paypal.CaptureResource
		if err = jsonx.UnmarshalBytes(raw, &cap); err != nil {
			return info, err
		}

		// cd := ChannelData{CaptureId: cap.Id}
		// cd.OrderId = cap.SupplementaryData.RelatedIds.OrderId
		return d.captureToPaymentInfo(cap.SupplementaryData.RelatedIds.OrderId, cap), nil
	}

	return info, driver.ErrUnsupported.WithReasonf("unsupported paypal payment webhook event: %s", event.EventType)
}

func (d *Driver) ParseRefundCallbackRequest(ctx context.Context, req *http.Request) (info driver.RefundInfo, err error) {
	event, raw, err := d.parseAndVerifyWebhook(ctx, req)
	if err != nil {
		return info, err
	}

	switch event.EventType {
	case "PAYMENT.REFUND.PENDING", "PAYMENT.REFUND.FAILED":
		var rf paypal.RefundResource
		if err = json.Unmarshal(raw, &rf); err != nil {
			return info, err
		}
		// TODO
		return d.refundToRefundInfo("", "", rf, ""), nil

	case "PAYMENT.CAPTURE.REFUNDED":
		var cap paypal.CaptureResource
		if err = jsonx.UnmarshalBytes(raw, &cap); err != nil {
			return info, err
		}

		cd := ChannelData{CaptureId: cap.Id}

		// cd.OrderID =
		return driver.RefundInfo{
			PaymentId:        cmp.Or(cap.CustomId, cap.InvoiceId),
			ChannelPaymentId: cap.SupplementaryData.RelatedIds.OrderId,
			ChannelStatus:    cap.Status,
			ChannelData:      driver.EncodeChannelData(cd),
			TaskStatus:       driver.TaskStatusSuccess,
		}, nil
	}

	return info, driver.ErrUnsupported.WithReasonf("unsupported paypal refund webhook event: %s", event.EventType)
}

func (d *Driver) SendPaymentCallbackResponse(ctx context.Context, rw http.ResponseWriter, err error) {
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte(`{"received":true}`))
}

func (d *Driver) SendRefundCallbackResponse(ctx context.Context, rw http.ResponseWriter, err error) {
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte(`{"received":true}`))
}

/// ----------------------------------------------------------------------- ///

type webhookEvent struct {
	Id        string          `json:"id"`
	EventType string          `json:"event_type"`
	Resource  json.RawMessage `json:"resource"`
}

func (d *Driver) parseAndVerifyWebhook(ctx context.Context, req *http.Request) (event webhookEvent, raw json.RawMessage, err error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return event, raw, err
	}

	if err = d.verifyWebhookSignature(ctx, req.Header, body); err != nil {
		return event, raw, err
	}

	if err = json.Unmarshal(body, &event); err != nil {
		return event, raw, err
	}

	return event, event.Resource, nil
}

func (d *Driver) verifyWebhookSignature(ctx context.Context, h http.Header, body []byte) error {
	var eventObj map[string]any
	if err := json.Unmarshal(body, &eventObj); err != nil {
		return err
	}

	payload := map[string]any{
		"auth_algo":         h.Get("PAYPAL-AUTH-ALGO"),
		"cert_url":          h.Get("PAYPAL-CERT-URL"),
		"transmission_id":   h.Get("PAYPAL-TRANSMISSION-ID"),
		"transmission_sig":  h.Get("PAYPAL-TRANSMISSION-SIG"),
		"transmission_time": h.Get("PAYPAL-TRANSMISSION-TIME"),
		"webhook_id":        d.config.WebhookId,
		"webhook_event":     eventObj,
	}

	var resp struct {
		VerificationStatus string `json:"verification_status"`
	}
	_, _, err := d.doJSON(ctx, http.MethodPost, "/v1/notifications/verify-webhook-signature", nil, payload, &resp)
	if err != nil {
		return err
	}

	if !strings.EqualFold(resp.VerificationStatus, "SUCCESS") {
		return driver.ErrBadRequest.WithReason("invalid paypal webhook signature")
	}

	return nil
}

func (d *Driver) orderToPaymentInfo(ord paypal.OrderResponse, fallbackPaymentId string) driver.PaymentInfo {
	if len(ord.PurchaseUnits) > 0 && len(ord.PurchaseUnits[0].Payments.Captures) > 0 {
		pu := ord.PurchaseUnits[0]
		if pu.ReferenceId != "" && fallbackPaymentId == "" {
			fallbackPaymentId = pu.ReferenceId
		}
		if pu.CustomId != "" && fallbackPaymentId == "" {
			fallbackPaymentId = pu.CustomId
		}

		cap := pu.Payments.Captures[0]
		return d.captureToPaymentInfo(ord.Id, cap)
	}

	info := driver.PaymentInfo{
		PaymentId: fallbackPaymentId,

		ChannelPaymentId: ord.Id,
		ChannelStatus:    ord.Status,

		PayerId: cmp.Or(ord.Payer.PayerId, ord.Payer.EmailAddress),
	}

	switch strings.ToUpper(ord.Status) {
	case "COMPLETED":
		info.TaskStatus = driver.TaskStatusSuccess

	case "VOIDED":
		info.TaskStatus = driver.TaskStatusClosed

	case "APPROVED", "CREATED", "PAYER_ACTION_REQUIRED", "SAVED":
		info.TaskStatus = driver.TaskStatusProcessing

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	return info
}

func (d *Driver) captureToPaymentInfo(orderId string, cap paypal.CaptureResource) driver.PaymentInfo {
	channeldata := driver.EncodeChannelData(ChannelData{CaptureId: cap.Id})

	info := driver.PaymentInfo{
		PaymentId: cmp.Or(cap.CustomId, cap.InvoiceId),

		ChannelPaymentId: orderId,
		ChannelStatus:    cap.Status,
		ChannelData:      channeldata,

		FailReason: cap.StatusDetails.Reason,
	}

	switch strings.ToUpper(cap.Status) {
	case "COMPLETED", "PARTIALLY_REFUNDED", "REFUNDED":
		info.TaskStatus = driver.TaskStatusSuccess

	case "PENDING":
		info.TaskStatus = driver.TaskStatusProcessing

	case "DECLINED", "FAILED":
		info.TaskStatus = driver.TaskStatusFailure

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	info.PayerPaidCurrency = strings.ToUpper(cap.Amount.Currency)
	if cap.Amount.Amount != "" {
		info.PayerPaidAmount, _ = currency.ParseMajorToMinor(cap.Amount.Amount, cap.Amount.Currency)
	}

	if ts := cmp.Or(cap.UpdateTime, cap.CreateTime); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			info.PayerPaidAt = t
		}
	}

	if cap.Status == "REFUNDED" || cap.Status == "PARTIALLY_REFUNDED" {
		info.IsRefunded = true
	}

	return info
}

func (d *Driver) refundToRefundInfo(paymentId, refundId string, resp paypal.RefundResource, cd string) driver.RefundInfo {
	info := driver.RefundInfo{
		PaymentId: paymentId,
		RefundId:  refundId,

		FailReason: resp.StatusDetails.Reason,

		ChannelRefundId: resp.Id,
		ChannelStatus:   resp.Status,
		ChannelData:     cd,
	}

	if resp.NoteToPayer != "" {
		info.RefundReason = resp.NoteToPayer
	}

	if ts := cmp.Or(resp.UpdateTime, resp.CreateTime); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			info.RefundedAt = t
		}
	}

	switch strings.ToUpper(resp.Status) {
	case "COMPLETED":
		info.TaskStatus = driver.TaskStatusSuccess

	case "PENDING":
		info.TaskStatus = driver.TaskStatusProcessing

	case "FAILED", "CANCELLED":
		info.TaskStatus = driver.TaskStatusFailure

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	return info
}

func (d *Driver) captureOrder(ctx context.Context, orderID string) (paypal.OrderResponse, error) {
	path := fmt.Sprintf("/v2/checkout/orders/%s/capture", orderID)
	header := map[string]string{
		"PayPal-Request-Id": orderID + "-capture",
		"Prefer":            "return=representation",
	}

	var out paypal.OrderResponse
	_, _, err := d.doJSON(ctx, http.MethodPost, path, header, map[string]any{}, &out)
	return out, err
}
