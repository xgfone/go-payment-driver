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
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/xgfone/go-payment-driver/builder"
	"github.com/xgfone/go-payment-driver/driver"
)

// https://pay.wechatpay.cn/doc/v3/merchant/4012791832  Introduction
// https://pay.wechatpay.cn/doc/v3/merchant/4012791834  CreateOrder

func init() {
	registerBuilder("h5", "", func(b builder.Builder, c Config) (driver.Driver, error) {
		d, err := newDriver(c, b)
		return &H5Driver{_Driver: d}, err
	})
}

type H5Driver struct{ _Driver }

func (d *H5Driver) CreatePayment(ctx context.Context, req driver.CreatePaymentRequest) (info driver.PayLinkInfo, err error) {
	// If passing OpenId, we think that it is from WeChat MiniProgram or Browser.
	if req.Buyer.OpenId != "" {
		return (*JsapiDriver)(d).CreatePayment(ctx, req)
	}

	if err = d.CheckCreatePaymentRequest(&req); err != nil {
		return
	}

	if req.Buyer.ClientIp == "" {
		err = driver.ErrBadRequest.WithReason("missing ClientIp")
		return
	}

	if d.config.H5Type == "" {
		d.config.H5Type = "Wap"
	}

	expiretime := req.GetExipredAt()
	svc := h5.H5ApiService{Client: d.client}
	resp, result, err := svc.Prepay(ctx, h5.PrepayRequest{
		Appid: core.String(d.config.Appid), // 公众号ID
		Mchid: core.String(d.config.Mchid), // 直连商户号

		Description: &req.PaymentDesc, // 商品描述
		OutTradeNo:  &req.PaymentId,   // 商户订单号
		TimeExpire:  &expiretime,      // 订单失效时间，格式为rfc3339格式
		NotifyUrl:   &req.CallbackUrl, // 必须为直接可访问的URL: 1. HTTPS；2. 不允许携带查询串

		Attach: nil, // 附加数据
		Amount: &h5.Amount{
			Total:    &req.PaymentAmount,   // 订单总金额，单位为分
			Currency: &req.PaymentCurrency, // CNY：人民币，境内商户号仅支持人民币。
		},

		SettleInfo: &h5.SettleInfo{ProfitSharing: &req.Share},
		SceneInfo: &h5.SceneInfo{
			PayerClientIp: &req.Buyer.ClientIp,
			H5Info: &h5.H5Info{
				Type: &d.config.H5Type,
			},
		},
	})
	if result != nil && result.Response != nil {
		defer result.Response.Body.Close()
	}

	if resp != nil {
		info.PayLink = *resp.H5Url
	}
	return
}

func (d *H5Driver) QueryPayment(ctx context.Context, req driver.QueryPaymentRequest) (info driver.PaymentInfo, ok bool, err error) {
	svc := h5.H5ApiService{Client: d.client}
	resp, result, err := svc.QueryOrderByOutTradeNo(ctx, h5.QueryOrderByOutTradeNoRequest{
		OutTradeNo: &req.PaymentId,
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
	ok = true
	return
}

// If has paid, return ErrPaid
// If the payment has been canceled, return nil.
func (d *H5Driver) CancelPayment(ctx context.Context, req driver.CancelPaymentRequest) (err error) {
	svc := h5.H5ApiService{Client: d.client}
	result, err := svc.CloseOrder(ctx, h5.CloseOrderRequest{
		OutTradeNo: &req.PaymentId,
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
