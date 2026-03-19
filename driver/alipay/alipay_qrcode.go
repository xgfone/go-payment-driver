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
	"cmp"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/smartwalle/alipay/v3"
	"github.com/xgfone/go-payment-driver/builder"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-toolkit/jsonx"
	"github.com/xgfone/go-toolkit/timex"
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

func (d *QrcodeDriver) CreateTrade(ctx context.Context, r driver.CreateTradeRequest) (info driver.LinkInfo, err error) {
	rsp, err := d.client.TradePreCreate(ctx, alipay.TradePreCreate{
		// DiscountableAmount:"",
		Trade: alipay.Trade{
			NotifyURL:   r.CallbackUrl,
			OutTradeNo:  r.TradeNo,
			Subject:     r.TradeDesc,
			TotalAmount: driver.FormatPercentAmount(r.TradeAmount),
			ProductCode: "QR_CODE_OFFLINE",
			TimeExpire:  r.ExipredAt().Format(time.DateTime),
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

func (d *QrcodeDriver) QueryTrade(ctx context.Context, query driver.QueryTradeRequest) (info driver.TradeInfo, ok bool, err error) {
	rsp, err := d.client.TradeQuery(ctx, alipay.TradeQuery{
		OutTradeNo:   query.TradeNo,
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
				info.TradeNo = query.TradeNo
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
// If the trade has been canceled, return nil.
func (d *QrcodeDriver) CancelTrade(ctx context.Context, query driver.CancelTradeRequest) (err error) {
	rsp, err := d.client.TradeClose(ctx, alipay.TradeClose{
		OutTradeNo: query.TradeNo,
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

// If the trade has been fully refunded, return ErrTradeRefundedFully.
// If the balance is insufficient, return ErrBalanceInsufficient.
// If it's not allowed to refund the trade, return ErrUnallowed.
func (d *QrcodeDriver) RefundTrade(ctx context.Context, r driver.RefundTradeRequest) (info driver.RefundInfo, err error) {
	rsp, err := d.client.TradeRefund(ctx, alipay.TradeRefund{
		OutTradeNo:   r.TradeNo,
		RefundAmount: driver.FormatPercentAmount(r.RefundAmount),
		RefundReason: r.RefundReason,
		OutRequestNo: r.RefundNo,

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

	info = d.parseRefundInfo(rsp)
	if info.RefundNo == "" {
		info.RefundNo = r.RefundNo
	}
	if info.RefundAmount == 0 {
		info.RefundAmount = r.RefundAmount
	}
	if info.TradeAmount == 0 {
		info.TradeAmount = r.TradeAmount
	}

	return
}

func (d *QrcodeDriver) QueryRefund(ctx context.Context, query driver.QueryRefundRequest) (info driver.RefundInfo, ok bool, err error) {
	rsp, err := d.client.TradeFastPayRefundQuery(ctx, alipay.TradeFastPayRefundQuery{
		OutTradeNo:   query.TradeNo,
		OutRequestNo: query.RefundNo,
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

func (d *QrcodeDriver) ParseTradeCallbackRequest(ctx context.Context, r *http.Request) (info driver.TradeInfo, err error) {
	if err = r.ParseForm(); err != nil {
		return
	}

	n, err := d.client.DecodeNotification(ctx, r.Form)
	if err != nil {
		return
	}

	info = d.parseNotification(n)
	return
}

func (d *QrcodeDriver) SendTradeCallbackResponse(ctx context.Context, w http.ResponseWriter, err error) {
	d.sendResponse(ctx, w, err)
}

func (d *QrcodeDriver) ParseRefundCallbackRequest(ctx context.Context, r *http.Request) (info driver.RefundInfo, err error) {
	err = errors.New("unimplemented")
	return
}

func (d *QrcodeDriver) SendRefundCallbackResponse(ctx context.Context, w http.ResponseWriter, err error) {
	d.sendResponse(ctx, w, err)
}

func (d *QrcodeDriver) sendResponse(_ context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		alipay.AckNotification(w)
	} else {
		w.WriteHeader(500)
	}
}

/// ----------------------------------------------------------------------- ///

func (d *QrcodeDriver) parseOrderQuery(rsp *alipay.TradeQueryRsp) (info driver.TradeInfo) {
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
		PayAmount:   rsp.PayAmount,
		PayCurrency: rsp.PayCurrency,

		PointAmount:     rsp.PointAmount,
		ReceiptAmount:   rsp.ReceiptAmount,
		InvoiceAmount:   rsp.InvoiceAmount,
		DiscountAmount:  rsp.DiscountAmount,
		MdiscountAmount: rsp.MdiscountAmount,
		BuyerUserType:   rsp.BuyerUserType,
		BuyerLogonId:    rsp.BuyerLogonId,
		BuyerUserId:     rsp.BuyerUserId,
		BuyerOpenId:     rsp.BuyerOpenId,

		SettleAmount:    rsp.SettleAmount,
		SettleCurrency:  rsp.SettleCurrency,
		SettleTransRate: rsp.SettleTransRate,

		ChargeInfoList: infos,
	}

	info.TradeNo = rsp.OutTradeNo
	info.TradeAmount, _ = driver.ParsePercentAmount(rsp.TotalAmount)
	info.TradeCurrency = rsp.TransCurrency

	info.ChannelTradeNo = rsp.TradeNo
	info.ChannelStatus = string(rsp.TradeStatus)
	info.ChannelData, _ = jsonx.MarshalStringWithCap(data, 512)

	info.PaidCurrency = info.TradeCurrency
	info.PaidAmount, _ = driver.ParsePercentAmount(cmp.Or(rsp.BuyerPayAmount, rsp.ReceiptAmount))
	if rsp.BuyerOpenId != "" {
		info.PayerId = rsp.BuyerOpenId
	} else {
		info.PayerId = rsp.BuyerUserId
	}
	if rsp.SendPayDate != "" {
		info.PaidAt, _ = time.ParseInLocation(time.DateTime, rsp.SendPayDate, time.Local)
	}

	switch rsp.TradeStatus {
	case alipay.TradeStatusWaitBuyerPay:
		info.TaskStatus = driver.TaskStatusProcessing

	case alipay.TradeStatusClosed:
		if info.PaidAt.IsZero() {
			info.TaskStatus = driver.TaskStatusClosed
		} else {
			info.TaskStatus = driver.TaskStatusSuccess
			info.IsRefunded = true
			if info.PaidAmount == 0 {
				info.PaidAmount = info.TradeAmount
			}
		}

	case alipay.TradeStatusSuccess, alipay.TradeStatusFinished:
		info.TaskStatus = driver.TaskStatusSuccess

	default:
		info.TaskStatus = driver.TaskStatusUnknown
	}

	return
}

func (d *QrcodeDriver) parseNotification(rsp *alipay.Notification) (info driver.TradeInfo) {
	data := ChannelData{
		PointAmount:   rsp.PointAmount,
		ReceiptAmount: rsp.ReceiptAmount,
		InvoiceAmount: rsp.InvoiceAmount,
		BuyerLogonId:  rsp.BuyerLogonId,
		BuyerOpenId:   rsp.BuyerOpenId,
		BuyerUserId:   rsp.BuyerId,
	}

	info.TradeNo = rsp.OutTradeNo
	info.TradeAmount, _ = driver.ParsePercentAmount(rsp.TotalAmount)
	info.TradeCurrency = ""

	info.ChannelTradeNo = rsp.TradeNo
	info.ChannelStatus = string(rsp.TradeStatus)
	info.ChannelData, _ = jsonx.MarshalStringWithCap(data, 256)

	info.PaidCurrency = info.TradeCurrency
	info.PaidAmount, _ = driver.ParsePercentAmount(cmp.Or(rsp.BuyerPayAmount, rsp.ReceiptAmount))
	if rsp.BuyerId != "" {
		info.PayerId = rsp.BuyerId
	} else if rsp.BuyerOpenId != "" {
		info.PayerId = rsp.BuyerOpenId
	} else {
		info.PayerId = rsp.BuyerLogonId
	}
	if rsp.GmtPayment != "" {
		info.PaidAt, _ = time.ParseInLocation(time.DateTime, rsp.GmtPayment, time.Local)
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

func (d *QrcodeDriver) parseRefundInfo(rsp *alipay.TradeRefundRsp) (info driver.RefundInfo) {
	var items []RefundDetailItem
	if len(rsp.RefundDetailItemList) > 0 {
		items = make([]RefundDetailItem, len(rsp.RefundDetailItemList))
		for i, item := range rsp.RefundDetailItemList {
			items[i] = RefundDetailItem{
				Amount:      item.Amount,
				RealAmount:  item.RealAmount,
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
	info.TradeNo = rsp.OutTradeNo
	// info.TradeAmount = 0

	// info.RefundNo = ""
	// info.RefundReason = ""
	// info.RefundAmount = 0

	info.ChannelTradeNo = rsp.TradeNo
	// info.ChannelRefundNo = ""
	// info.ChannelStatus = ""
	info.ChannelData, _ = jsonx.MarshalStringWithCap(data, 1024)

	if rsp.FundChange == "Y" {
		info.RefundedAt = timex.Now()
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
				Amount:      item.Amount,
				RealAmount:  item.RealAmount,
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
	info.TradeNo = rsp.OutTradeNo
	info.TradeAmount, _ = driver.ParsePercentAmount(rsp.TotalAmount)

	info.RefundNo = rsp.OutRequestNo
	// info.RefundReason = ""
	info.RefundAmount, _ = driver.ParsePercentAmount(rsp.RefundAmount)

	// info.ChannelRefundNo = ""
	info.ChannelTradeNo = rsp.TradeNo
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
