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

package alipay

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/smartwalle/alipay/v3"
	"github.com/xgfone/go-payment-driver/builder"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-toolkit/jsonx"
)

// https://opendocs.alipay.com/open/05osux
// https://opendocs.alipay.com/open/8ad49e4a_alipay.trade.precreate

func init() {
	registerBuilder("qrcode", driver.LinkTypeCodeUrl, func(b builder.Builder, c Config) (driver.Driver, error) {
		driver, err := newDriver(c, b)
		return &QrcodeDriver{_Driver: driver}, err
	})
}

type QrcodeDriver struct{ _Driver }

func (d *QrcodeDriver) CreatePayment(ctx context.Context, req driver.CreatePaymentRequest) (info driver.PayLinkInfo, err error) {
	if err = d.CheckCreateTradeRequest(&req); err != nil {
		return
	}

	totalAmount, err := d.FormatMinorToMajor(req.PaymentAmount)
	if err != nil {
		return
	}

	rsp, err := d.client.TradePreCreate(ctx, alipay.TradePreCreate{
		// DiscountableAmount:"",
		Trade: alipay.Trade{
			NotifyURL:   req.CallbackUrl,
			OutTradeNo:  req.PaymentId,
			Subject:     req.PaymentDesc,
			TotalAmount: totalAmount,
			ProductCode: "QR_CODE_OFFLINE",
			TimeExpire:  req.GetExipredAt().Format(time.DateTime),
			// SellerId:        "",
			// Body:            "",
			// MerchantOrderNo: "",
		},
	})
	switch {
	case err != nil:
		return

	case rsp.IsFailure():
		err = ToError(rsp.Error)
		return
	}

	info.PayLink = rsp.QRCode
	return
}

func (d *QrcodeDriver) QueryPayment(ctx context.Context, req driver.QueryPaymentRequest) (info driver.PaymentInfo, ok bool, err error) {
	rsp, err := d.client.TradeQuery(ctx, alipay.TradeQuery{
		OutTradeNo:   req.PaymentId,
		QueryOptions: []string{"voucher_detail_list"},
	})
	switch {
	case err != nil:
		return

	case rsp.IsFailure():
		if rsp.Error.Code == "40004" {
			switch rsp.Error.SubCode {
			case "ACQ.TRADE_NOT_EXIST", "TRADE_NOT_EXIST":
				ok = true
				info.PaymentId = req.PaymentId
				info.TaskStatus = driver.TaskStatusProcessing
				return
			}
		}
		err = ToError(rsp.Error)
		return
	}

	ok = true
	info = d.parseOrderQuery(rsp)
	return
}

// If has paid, return ErrPaid
// If the payment has been canceled, return nil.
func (d *QrcodeDriver) CancelPayment(ctx context.Context, req driver.CancelPaymentRequest) (err error) {
	rsp, err := d.client.TradeClose(ctx, alipay.TradeClose{
		OutTradeNo: req.PaymentId,
	})
	switch {
	case err != nil:
		return

	case rsp.IsFailure():
		if rsp.Error.Code == "40004" {
			switch rsp.Error.SubCode {
			case "ACQ.TRADE_NOT_EXIST", "TRADE_NOT_EXIST":
				return
			}
		}
		if strings.Contains(rsp.Msg, "已支付") || strings.Contains(rsp.SubMsg, "已支付") {
			err = driver.ErrPaid
		} else {
			err = ToError(rsp.Error)
		}
		return
	}

	return
}

// If the payment has been fully refunded, return ErrPaymentRefundedFully.
// If the balance is insufficient, return ErrBalanceInsufficient.
// If it's not allowed to refund the payment, return ErrUnallowed.
func (d *QrcodeDriver) RefundPayment(ctx context.Context, req driver.CreateRefundRequest) (info driver.RefundInfo, err error) {
	if err = d.CheckRefundTradeRequest(&req); err != nil {
		return
	}

	refundAmount, err := d.FormatMinorToMajor(req.RefundAmount)
	if err != nil {
		return
	}

	rsp, err := d.client.TradeRefund(ctx, alipay.TradeRefund{
		OutTradeNo:   req.PaymentId,
		RefundAmount: refundAmount,
		RefundReason: req.RefundReason,
		OutRequestNo: req.RefundId,

		QueryOptions: []string{"refund_detail_item_list"},
	})

	switch {
	case err != nil:
		return

	case rsp.IsFailure():
		switch rsp.Error.SubCode {
		case "ACQ.TRADE_NOT_ALLOW_REFUND", "ACQ.TRADE_HAS_FINISHED", "ACQ.ONLINE_TRADE_VOUCHER_NOT_ALLOW_REFUND":
			err = driver.ErrUnallowed.WithError(ToError(rsp.Error))

		case "ACQ.SELLER_BALANCE_NOT_ENOUGH":
			err = driver.ErrBalanceInsufficient

		default:
			err = ToError(rsp.Error)
		}
		return
	}

	info = d.parseRefundInfo(rsp, &req)
	return
}

func (d *QrcodeDriver) QueryRefund(ctx context.Context, query driver.QueryRefundRequest) (info driver.RefundInfo, ok bool, err error) {
	rsp, err := d.client.TradeFastPayRefundQuery(ctx, alipay.TradeFastPayRefundQuery{
		OutTradeNo:   query.PaymentId,
		OutRequestNo: query.RefundId,
		QueryOptions: []string{"refund_detail_item_list", "gmt_refund_pay"},
	})

	switch {
	case err != nil:
		return

	case rsp.IsFailure():
		if rsp.Error.Code == "40004" {
			switch rsp.Error.SubCode {
			case "ACQ.TRADE_NOT_EXIST", "TRADE_NOT_EXIST":
				return
			}
		}
		err = ToError(rsp.Error)
		return
	}

	ok = true
	info = d.parseRefundQuery(rsp)
	return
}

func (d *QrcodeDriver) SendCallbackResponse(_ context.Context, rw http.ResponseWriter, err error) {
	if err == nil {
		alipay.AckNotification(rw)
	} else {
		rw.WriteHeader(500)
	}
}

func (d *QrcodeDriver) ParseCallbackRequest(ctx context.Context, req driver.CallbackRequest) (cbinfo driver.CallbackInfo, err error) {
	cbinfo.Type = req.Type
	if req.Type == driver.CallbackTypePayment {
		if err = req.Request.ParseForm(); err != nil {
			return
		}

		var notification *alipay.Notification
		notification, err = d.client.DecodeNotification(ctx, req.Request.Form)
		if err != nil {
			return
		}

		info := d.parseNotification(notification)
		cbinfo.PaymentInfo = &info
	}
	return
}

/// ----------------------------------------------------------------------- ///

func (d *QrcodeDriver) parseOrderQuery(rsp *alipay.TradeQueryRsp) (info driver.PaymentInfo) {
	var infos []ChargeInfo
	if len(rsp.ChargeInfoList) > 0 {
		infos = make([]ChargeInfo, 0, len(rsp.ChargeInfoList))
		for i, v := range rsp.ChargeInfoList {
			var details []SubFeeDetail
			if len(v.SubFeeDetailList) > 0 {
				details = make([]SubFeeDetail, len(v.SubFeeDetailList))
				for j, d := range v.SubFeeDetailList {
					details[j] = SubFeeDetail{
						ChargeFee:         d.ChargeFee,
						SwitchFeeRate:     d.SwitchFeeRate,
						OriginalChargeFee: d.OriginalChargeFee,
					}
				}
			}

			infos[i] = ChargeInfo{
				ChargeFee:  v.ChargeFee,
				ChargeType: v.ChargeType,

				SwitchFeeRate:     v.SwitchFeeRate,
				OriginalChargeFee: v.OriginalChargeFee,

				IsRatingOnTradeReceiver: v.IsRatingOnTradeReceiver,
				IsRatingOnSwitch:        v.IsRatingOnSwitch,

				SubFeeDetailList: details,
			}
		}
	}

	data := ChannelData{
		PayAmount:   fixAmount(rsp.PayAmount),
		PayCurrency: rsp.PayCurrency,

		PointAmount:     fixAmount(rsp.PointAmount),
		ReceiptAmount:   fixAmount(rsp.ReceiptAmount),
		InvoiceAmount:   fixAmount(rsp.InvoiceAmount),
		DiscountAmount:  fixAmount(rsp.DiscountAmount),
		MdiscountAmount: fixAmount(rsp.MdiscountAmount),
		BuyerUserType:   rsp.BuyerUserType,
		BuyerLogonId:    rsp.BuyerLogonId,
		BuyerUserId:     rsp.BuyerUserId,
		BuyerOpenId:     rsp.BuyerOpenId,

		SettleAmount:    fixAmount(rsp.SettleAmount),
		SettleCurrency:  rsp.SettleCurrency,
		SettleTransRate: rsp.SettleTransRate,

		ChargeInfoList: infos,
	}

	info.PaymentId = rsp.OutTradeNo

	info.ChannelPaymentId = rsp.TradeNo
	info.ChannelStatus = string(rsp.TradeStatus)
	info.ChannelData, _ = jsonx.MarshalStringWithCap(data, 512)

	info.PayerPaidCurrency = d.Currency().Code
	info.PayerPaidAmount, _ = d.ParseMajorToMinor(rsp.BuyerPayAmount)
	if rsp.BuyerOpenId != "" {
		info.PayerId = rsp.BuyerOpenId
	} else {
		info.PayerId = rsp.BuyerUserId
	}
	if rsp.SendPayDate != "" {
		info.PayerPaidAt, _ = time.ParseInLocation(time.DateTime, rsp.SendPayDate, time.Local)
	}

	switch rsp.TradeStatus {
	case alipay.TradeStatusWaitBuyerPay:
		info.TaskStatus = driver.TaskStatusProcessing

	case alipay.TradeStatusClosed:
		if info.PayerPaidAt.IsZero() {
			info.TaskStatus = driver.TaskStatusClosed
		} else {
			info.TaskStatus = driver.TaskStatusSuccess
			info.IsRefunded = true
		}

	case alipay.TradeStatusSuccess, alipay.TradeStatusFinished:
		info.TaskStatus = driver.TaskStatusSuccess

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	return
}

func (d *QrcodeDriver) parseNotification(rsp *alipay.Notification) (info driver.PaymentInfo) {
	data := ChannelData{
		PointAmount:   fixAmount(rsp.PointAmount),
		ReceiptAmount: fixAmount(rsp.ReceiptAmount),
		InvoiceAmount: fixAmount(rsp.InvoiceAmount),
		BuyerLogonId:  rsp.BuyerLogonId,
		BuyerOpenId:   rsp.BuyerOpenId,
		BuyerUserId:   rsp.BuyerId,
	}

	info.PaymentId = rsp.OutTradeNo

	info.ChannelPaymentId = rsp.TradeNo
	info.ChannelStatus = string(rsp.TradeStatus)
	info.ChannelData, _ = jsonx.MarshalStringWithCap(data, 256)

	info.PayerPaidCurrency = d.Currency().Code
	info.PayerPaidAmount, _ = d.ParseMajorToMinor(rsp.BuyerPayAmount)
	if rsp.BuyerId != "" {
		info.PayerId = rsp.BuyerId
	} else if rsp.BuyerOpenId != "" {
		info.PayerId = rsp.BuyerOpenId
	} else {
		info.PayerId = rsp.BuyerLogonId
	}
	if rsp.GmtPayment != "" {
		info.PayerPaidAt, _ = time.ParseInLocation(time.DateTime, rsp.GmtPayment, time.Local)
	}

	switch rsp.TradeStatus {
	case alipay.TradeStatusWaitBuyerPay:
		info.TaskStatus = driver.TaskStatusProcessing

	case alipay.TradeStatusClosed:
		info.TaskStatus = driver.TaskStatusClosed

	case alipay.TradeStatusSuccess, alipay.TradeStatusFinished:
		info.TaskStatus = driver.TaskStatusSuccess

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	return
}

func (d *QrcodeDriver) parseRefundInfo(rsp *alipay.TradeRefundRsp, req *driver.CreateRefundRequest) (info driver.RefundInfo) {
	var items []RefundDetailItem
	if len(rsp.RefundDetailItemList) > 0 {
		items = make([]RefundDetailItem, len(rsp.RefundDetailItemList))
		for i, item := range rsp.RefundDetailItemList {
			items[i] = RefundDetailItem{
				Amount:      fixAmount(item.Amount),
				RealAmount:  fixAmount(item.RealAmount),
				FundChannel: item.FundChannel,
				FundType:    item.FundType,
			}
		}
	}

	var infos []RefundChargeInfo
	if len(rsp.RefundChargeInfoList) > 0 {
		infos = make([]RefundChargeInfo, len(rsp.RefundChargeInfoList))
		for i, info := range rsp.RefundChargeInfoList {
			var details []RefundSubFeeDetail
			if len(info.RefundSubFeeDetailList) > 0 {
				details = make([]RefundSubFeeDetail, len(info.RefundSubFeeDetailList))
				for j, detail := range info.RefundSubFeeDetailList {
					details[j] = RefundSubFeeDetail{
						SwitchFeeRate:   detail.SwitchFeeRate,
						RefundChargeFee: detail.RefundChargeFee,
					}
				}
			}

			infos[i] = RefundChargeInfo{
				ChargeType:      info.ChargeType,
				SwitchFeeRate:   info.SwitchFeeRate,
				RefundChargeFee: info.RefundChargeFee,
				SubFeeDetails:   details,
			}
		}
	}

	data := ChannelData{
		BuyerLogonId: rsp.BuyerLogonId,

		SendBackFee: rsp.SendBackFee,
		FundChange:  rsp.FundChange,

		RefundDetailItems: items,
		RefundChargeInfos: infos,
	}

	/// ---------------------------------
	info.PaymentId = rsp.OutTradeNo
	info.RefundId = req.RefundId

	info.RefundReason = "" // Keep empty

	// info.ChannelRefundId = ""
	info.ChannelStatus = "" // Keep empty
	info.ChannelData, _ = jsonx.MarshalStringWithCap(data, 1024)

	info.RefundedAt = time.Time{} // Keep ZERO

	if rsp.FundChange == "Y" {
		info.TaskStatus = driver.TaskStatusSuccess
	} else {
		info.TaskStatus = driver.TaskStatusProcessing
	}

	return
}

func (d *QrcodeDriver) parseRefundQuery(rsp *alipay.TradeFastPayRefundQueryRsp) (info driver.RefundInfo) {
	var items []RefundDetailItem
	if len(rsp.RefundDetailItemList) > 0 {
		items = make([]RefundDetailItem, len(rsp.RefundDetailItemList))
		for i, item := range rsp.RefundDetailItemList {
			items[i] = RefundDetailItem{
				Amount:      fixAmount(item.Amount),
				RealAmount:  fixAmount(item.RealAmount),
				FundChannel: item.FundChannel,
				FundType:    item.FundType,
			}
		}
	}

	var infos []RefundChargeInfo
	if len(rsp.RefundChargeInfoList) > 0 {
		infos = make([]RefundChargeInfo, len(rsp.RefundChargeInfoList))
		for i, info := range rsp.RefundChargeInfoList {
			var details []RefundSubFeeDetail
			if len(info.RefundSubFeeDetailList) > 0 {
				details = make([]RefundSubFeeDetail, len(info.RefundSubFeeDetailList))
				for j, detail := range info.RefundSubFeeDetailList {
					details[j] = RefundSubFeeDetail{
						SwitchFeeRate:   detail.SwitchFeeRate,
						RefundChargeFee: detail.RefundChargeFee,
					}
				}
			}

			infos[i] = RefundChargeInfo{
				ChargeType:      info.ChargeType,
				SwitchFeeRate:   info.SwitchFeeRate,
				RefundChargeFee: info.RefundChargeFee,
				SubFeeDetails:   details,
			}
		}
	}

	data := ChannelData{
		SendBackFee: rsp.SendBackFee,

		RefundDetailItems: items,
		RefundChargeInfos: infos,

		// RefundRoyaltys: rsp.RefundRoyaltys,
	}

	/// ---------------------------------
	info.PaymentId = rsp.OutTradeNo

	info.RefundId = rsp.OutRequestNo
	info.RefundReason = "" // Keep empty

	// info.ChannelRefundId = ""
	info.ChannelStatus = rsp.RefundStatus
	info.ChannelData, _ = jsonx.MarshalStringWithCap(data, 1024)

	info.RefundedAt, _ = time.ParseInLocation(time.DateTime, rsp.GMTRefundPay, time.Local)

	switch info.ChannelStatus {
	case "REFUND_SUCCESS":
		info.TaskStatus = driver.TaskStatusSuccess

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	return
}
