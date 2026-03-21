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
	"errors"
	"fmt"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-payment-driver/share"
	"github.com/xgfone/go-toolkit/runtimex"
)

// https://pay.wechatpay.cn/doc/v3/merchant/4012067962
// https://pay.wechatpay.cn/doc/v3/merchant/4012524936 ApplyShare

var _ share.Sharer = &_Driver{}

// If the balance is insufficient, return driver.ErrBalanceInsufficient.
func (d *_Driver) ApplyShare(ctx context.Context, req share.ApplyShareRequest) (info share.ShareInfo, err error) {
	if len(req.ShareReceivers) == 0 {
		return share.ShareInfo{}, errors.New("no share receivers")
	}

	receivers := make([]profitsharing.CreateOrderReceiver, len(req.ShareReceivers))
	for i, r := range req.ShareReceivers {
		rtype, err := fmtAccountType(r.Receiver.AccountType)
		if err != nil {
			return share.ShareInfo{}, err
		}

		receivers[i] = profitsharing.CreateOrderReceiver{
			Name: nil, // For Personal, optional. For Merchant, required??
			Type: rtype,

			Amount:      &r.ShareAmount,
			Account:     &r.Receiver.Account,
			Description: &r.ShareDesc,
		}
	}

	svc := profitsharing.OrdersApiService{Client: d.client}
	resp, result, err := svc.CreateOrder(ctx, profitsharing.CreateOrderRequest{
		Appid:     &d.config.Appid,
		Receivers: receivers,

		OutOrderNo:      &req.ShareId,
		TransactionId:   &req.ChannelPaymentId,
		UnfreezeUnsplit: &req.UnfreezeUnsplit,
	})

	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}

	if err != nil {
		if e, ok := err.(*core.APIError); ok {
			switch e.Code {
			case "NOT_ENOUGH":
				err = driver.ErrBalanceInsufficient.WithMessage(e.Message)

			default:
				err = fmt.Errorf("%s: %s", e.Code, e.Message)
			}
		}
		return
	}

	info = d.parseShareInfo(resp)
	return
}

func (d *_Driver) QueryShare(ctx context.Context, req share.QueryShareRequest) (info share.ShareInfo, ok bool, err error) {
	svc := profitsharing.OrdersApiService{Client: d.client}
	resp, result, err := svc.QueryOrder(ctx, profitsharing.QueryOrderRequest{
		OutOrderNo:    &req.ShareId,
		TransactionId: &req.ChannelPaymentId,
	})
	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}

	if err != nil {
		if e, ok := err.(*core.APIError); ok {
			switch {
			case e.StatusCode == 404 && e.Code == "RESOURCE_NOT_EXISTS":
				err = nil

			default:
				err = fmt.Errorf("%s: %s", e.Code, e.Message)
			}
		}
		return
	}

	info = d.parseShareInfo(resp)
	ok = true
	return
}

// If the return account is equal to the share account, return share.ErrReturnAccountIsShareAccount.
func (d *_Driver) ReturnShare(ctx context.Context, req share.ReturnShareRequest) (info share.ReturnInfo, err error) {
	if req.ReturnAccount == d.config.Mchid {
		err = share.ErrReturnAccountIsShareAccount
		return
	}

	svc := profitsharing.ReturnOrdersApiService{Client: d.client}
	resp, result, err := svc.CreateReturnOrder(ctx, profitsharing.CreateReturnOrderRequest{
		Amount:      &req.ReturnAmount,
		OrderId:     &req.ChannelShareId,
		OutOrderNo:  &req.ShareId,
		OutReturnNo: &req.ReturnId,
		Description: &req.ReturnDesc,
		ReturnMchid: &req.ReturnAccount,
	})
	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}

	if err != nil {
		if e, ok := err.(*core.APIError); ok {
			err = fmt.Errorf("%s: %s", e.Code, e.Message)
		}
		return
	}

	info = d.parseShareReturnInfo(resp)
	return
}

func (d *_Driver) QueryReturn(ctx context.Context, req share.QueryReturnRequest) (info share.ReturnInfo, ok bool, err error) {
	svc := profitsharing.ReturnOrdersApiService{Client: d.client}
	resp, result, err := svc.QueryReturnOrder(ctx, profitsharing.QueryReturnOrderRequest{
		OutOrderNo:  &req.ShareId,
		OutReturnNo: &req.ReturnId,
	})
	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}

	if err != nil {
		if e, ok := err.(*core.APIError); ok {
			switch {
			case e.StatusCode == 404 && e.Code == "RESOURCE_NOT_EXISTS":
				err = nil

			default:
				err = fmt.Errorf("%s: %s", e.Code, e.Message)
			}
		}
		return
	}

	info = d.parseShareReturnInfo(resp)
	ok = true
	return
}

func (d *_Driver) DeleteShareReceiver(ctx context.Context, r share.Receiver) (err error) {
	_type, err := fmtAccountType(r.AccountType)
	if err != nil {
		return
	}

	svc := profitsharing.ReceiversApiService{Client: d.client}
	_, result, err := svc.DeleteReceiver(ctx, profitsharing.DeleteReceiverRequest{
		Appid:   &d.config.Appid,
		Account: &r.Account,

		Type: (*profitsharing.ReceiverType)(_type),
	})

	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}

	if err != nil {
		if e, ok := err.(*core.APIError); ok {
			err = fmt.Errorf("%s: %s", e.Code, e.Message)
		}
		return
	}

	return
}

func (d *_Driver) AddShareReceiver(ctx context.Context, r share.Receiver) (err error) {
	rtype, custom, err := fmtRelationType(r.RelationType)
	if err != nil {
		return
	}

	_type, err := fmtAccountType(r.AccountType)
	if err != nil {
		return
	}

	svc := profitsharing.ReceiversApiService{Client: d.client}
	_, result, err := svc.AddReceiver(ctx, profitsharing.AddReceiverRequest{
		Appid: &d.config.Appid,

		Name: &r.Name,
		Type: (*profitsharing.ReceiverType)(_type),

		Account:        &r.Account,
		RelationType:   rtype,
		CustomRelation: custom,
	})

	if result != nil && result.Response != nil {
		result.Response.Body.Close()
	}

	if err != nil {
		if e, ok := err.(*core.APIError); ok {
			err = fmt.Errorf("%s: %s", e.Code, e.Message)
		}
		return
	}

	return
}

/// ----------------------------------------------------------------------- ///

func (d *_Driver) parseShareInfo(resp *profitsharing.OrdersEntity) (info share.ShareInfo) {
	if resp == nil {
		return
	}

	info.ShareId = runtimex.Indirect(resp.OutOrderNo)
	info.ChannelShareId = runtimex.Indirect(resp.OrderId)
	info.ChannelPaymentId = runtimex.Indirect(resp.TransactionId)

	switch s := runtimex.Indirect(resp.State); s {
	case "PROCESSING":
		info.ShareStatus = share.StatusProcessing

	case "FINISHED":
		info.ShareStatus = share.StatusFinished

	default:
		info.ShareStatus = share.Status(s)
	}

	info.ShareRecords = make([]share.ShareRecord, len(resp.Receivers))
	for i, r := range resp.Receivers {
		var result share.Result
		switch s := runtimex.Indirect(r.Result); s {
		case "PENDING":
			result = share.ResultProcess

		case "SUCCESS":
			result = share.ResultSuccess

		case "CLOSED":
			result = share.ResultFailure

		default:
			result = share.Result(s)
		}

		account := runtimex.Indirect(r.Account)
		info.ShareRecords[i] = share.ShareRecord{
			FailReason:  string(runtimex.Indirect(r.FailReason)),
			ShareResult: result,

			CreatedAt:  runtimex.Indirect(r.CreateTime),
			FinishedAt: runtimex.Indirect(r.FinishTime),

			IsInitiator:     account == d.config.Mchid,
			ChannelDetailId: runtimex.Indirect(r.DetailId),

			ShareReceiver: share.ShareReceiver{
				ShareAmount: runtimex.Indirect(r.Amount),
				ShareDesc:   runtimex.Indirect(r.Description),
				Receiver: share.AccountInfo{
					AccountType: unfmtAccountType(string(*r.Type)),
					Account:     account,
				},
			},

			Sender: share.AccountInfo{
				AccountType: AccountTypeMerchant,
				Account:     d.config.Mchid,
			},
		}
	}

	return
}

func (d *_Driver) parseShareReturnInfo(resp *profitsharing.ReturnOrdersEntity) (info share.ReturnInfo) {
	if resp == nil {
		return
	}

	info.ShareId = runtimex.Indirect(resp.OutOrderNo)
	info.ReturnId = runtimex.Indirect(resp.OutReturnNo)
	info.PaymentId = "" // Keep empty

	info.ChannelShareId = runtimex.Indirect(resp.OrderId)
	info.ChannelReturnId = runtimex.Indirect(resp.ReturnId)

	info.ReturnAmount = runtimex.Indirect(resp.Amount)
	info.ReturnAccount = runtimex.Indirect(resp.ReturnMchid)
	info.ReturnDesc = runtimex.Indirect(resp.Description)

	info.CreatedAt = runtimex.Indirect(resp.CreateTime)
	info.FinishedAt = runtimex.Indirect(resp.FinishTime)

	info.FailReason = string(runtimex.Indirect(resp.FailReason))
	switch s := runtimex.Indirect(resp.Result); s {
	case "PENDING":
		info.ReturnResult = share.ResultProcess

	case "SUCCESS":
		info.ReturnResult = share.ResultSuccess

	case "CLOSED":
		info.ReturnResult = share.ResultFailure

	default:
		info.ReturnResult = share.Result(s)
	}

	return
}
