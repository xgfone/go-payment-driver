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

package weixin

import (
	"context"
	"strings"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-toolkit/jsonx"
)

// https://pay.wechatpay.cn/doc/v3/merchant/4012062524  Introduction
// https://pay.wechatpay.cn/doc/v3/merchant/4012791856  CreateOrder

func init() {
	registerBuilder("jsapi", func(d _Driver) driver.Driver {
		return &JsapiDriver{_Driver: d}
	})
}

type JsapiDriver struct{ _Driver }

func (d *JsapiDriver) CreateTrade(ctx context.Context, r driver.CreateTradeRequest) (info driver.LinkInfo, err error) {
	if err = d.CheckCreateTradeRequest(&r); err != nil {
		return
	}

	if r.OpenId == "" {
		err = driver.ErrBadRequest.WithReason("missing OpenId")
		return
	}

	expiretime := d.ExpireTime(r.Timeout)
	svc := jsapi.JsapiApiService{Client: d.client}
	resp, result, err := svc.Prepay(ctx, jsapi.PrepayRequest{
		Appid: core.String(d.config.Appid), // 公众号ID
		Mchid: core.String(d.config.Mchid), // 直连商户号

		Description: &r.TradeDesc,   // 商品描述
		OutTradeNo:  &r.TradeNo,     // 商户订单号
		TimeExpire:  &expiretime,    // 订单失效时间，格式为rfc3339格式
		NotifyUrl:   &r.CallbackUrl, // 必须为直接可访问的URL: 1. HTTPS；2. 不允许携带查询串

		Attach: nil, // 附加数据
		Amount: &jsapi.Amount{
			Total:    &r.TradeAmount,   // 订单总金额，单位为分
			Currency: &r.TradeCurrency, // CNY：人民币，境内商户号仅支持人民币。
		},

		Payer:      &jsapi.Payer{Openid: &r.OpenId},
		SettleInfo: &jsapi.SettleInfo{ProfitSharing: &r.Share},
	})
	if result != nil && result.Response != nil {
		defer result.Response.Body.Close()
	}

	if err != nil || resp == nil {
		return
	}

	prepayid := *resp.PrepayId
	sign, err := d.getSign(prepayid)
	if err != nil {
		return info, err
	}

	type LinkInfo struct {
		Appid   string `json:"appId"`
		Package string `json:"package"`

		Signature
	}

	paylink, err := jsonx.MarshalStringWithCap(LinkInfo{
		Appid:   d.config.Appid,
		Package: "prepay_id=" + prepayid,

		Signature: sign,
	}, 640)
	if err != nil {
		return
	}

	info = d.LinkInfo(paylink)
	return
}

func (d *JsapiDriver) QueryTrade(ctx context.Context, query driver.QueryTradeRequest) (info driver.TradeInfo, ok bool, err error) {
	svc := jsapi.JsapiApiService{Client: d.client}
	resp, result, err := svc.QueryOrderByOutTradeNo(ctx, jsapi.QueryOrderByOutTradeNoRequest{
		OutTradeNo: &query.TradeNo,
		Mchid:      core.String(d.config.Mchid),
	})

	if result != nil && result.Response != nil {
		defer result.Response.Body.Close()
	}

	switch e := err.(type) {
	case nil:
	case *core.APIError:
		if e.StatusCode == 404 {
			err = nil
		}
		return
	default:
		return
	}

	info = d.parsePayRequest(resp)
	if info.TradeNo == "" {
		info.TradeNo = query.TradeNo
	}
	ok = true
	return
}

// If has paid, return ErrPaid
// If the trade has been canceled, return nil.
func (d *JsapiDriver) CancelTrade(ctx context.Context, query driver.CancelTradeRequest) (err error) {
	svc := jsapi.JsapiApiService{Client: d.client}
	result, err := svc.CloseOrder(ctx, jsapi.CloseOrderRequest{
		OutTradeNo: &query.TradeNo,
		Mchid:      core.String(d.config.Mchid),
	})
	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}

	if err != nil {
		if e, ok := err.(*core.APIError); ok {
			if e.StatusCode == 400 && strings.Contains(e.Message, "已支付") {
				return driver.ErrPaid
			}
		}
	}

	return
}
