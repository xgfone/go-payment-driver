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
	"errors"
	"fmt"
	"time"

	"github.com/smartwalle/alipay/v3"
	"github.com/xgfone/go-currency"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-payment-driver/share"
	"github.com/xgfone/go-toolkit/random"
	"github.com/xgfone/go-toolkit/timex"
)

var _ share.Sharer = &_Driver{}

// If the balance is insufficient, return driver.ErrBalanceInsufficient.
func (d *_Driver) ApplyShare(ctx context.Context, req share.ApplyShareRequest) (info share.ShareInfo, err error) {
	if !d.config.ShareEnabled {
		err = errors.New("not configure share account")
		return
	}

	_req := alipay.NewPayload("alipay.trade.order.settle")
	_req.AddBizField("out_request_no", req.ShareId)
	_req.AddBizField("trade_no", req.ChannelPaymentId)

	type (
		ExtendParams struct {
			RoyaltyFinish string `json:"royalty_finish"`
		}

		RoyaltyParameters struct {
			Desc   string `json:"desc"`
			Amount string `json:"amount"`

			TransOut     string `json:"trans_out,omitempty"`
			TransOutType string `json:"trans_out_type,omitempty"` // userId or loginName

			TransInType string `json:"trans_in_type"` // userId or loginName or cardAliasNo or openId
			TransIn     string `json:"trans_in"`

			// TransInName string `json:"trans_in_name,omitempty"`
		}
	)

	if len(req.ShareReceivers) > 0 {
		receivers := make([]RoyaltyParameters, len(req.ShareReceivers))
		for i, r := range req.ShareReceivers {
			inatype, err := fmtAccountType(r.Receiver.AccountType)
			if err != nil {
				return share.ShareInfo{}, err
			}

			outatype, err := fmtAccountType(share.AccountType(d.config.ShareAccountType))
			if err != nil {
				return share.ShareInfo{}, err
			}

			amount, err := currency.CNY.FormatMinorToMajor(r.ShareAmount)
			if err != nil {
				return share.ShareInfo{}, err
			}

			receivers[i] = RoyaltyParameters{
				Desc:   r.ShareDesc,
				Amount: amount,

				TransOut:     d.config.ShareAccount,
				TransOutType: outatype,

				TransInType: inatype,
				TransIn:     r.Receiver.Account,
			}
		}

		_req.AddBizField("royalty_parameters", receivers)
	}

	if req.UnfreezeUnsplit {
		_req.AddBizField("extend_params", ExtendParams{RoyaltyFinish: "true"})
	}
	if d.config.ShareAsync {
		_req.AddBizField("royalty_mode", "async")
		info.ShareStatus = share.StatusProcessing
	} else {
		info.ShareStatus = share.StatusFinished
	}

	var rsp struct {
		alipay.Error

		TradeNo  string `json:"trade_no"`
		SettleNo string `json:"settle_no"`
	}

	err = d.client.Request(ctx, _req, &rsp)
	switch {
	case err != nil:
		return

	case rsp.IsFailure():
		switch rsp.Error.SubCode {
		case "ACQ.TXN_RESULT_ACCOUNT_BALANCE_NOT_ENOUGH":
			err = driver.ErrBalanceInsufficient

		default:
			err = ToError(rsp.Error)
		}
		return
	}

	info.ShareId = req.ShareId
	info.ChannelShareId = rsp.SettleNo
	info.ChannelPaymentId = rsp.TradeNo

	if info.ShareStatus == share.StatusFinished {
		req := share.QueryShareRequest{
			ShareId:          info.ShareId,
			PaymentId:        req.PaymentId,
			ChannelPaymentId: info.ChannelPaymentId,
		}
		_info, ok, err := d.QueryShare(ctx, req)
		if err != nil || !ok {
			info.ShareStatus = share.StatusProcessing
		} else {
			_info.ChannelShareId = info.ChannelShareId
			info = _info
		}
	}

	return
}

func (d *_Driver) QueryShare(ctx context.Context, req share.QueryShareRequest) (info share.ShareInfo, ok bool, err error) {
	_req := alipay.NewPayload("alipay.trade.order.settle.query")
	_req.AddBizField("out_request_no", req.ShareId)
	_req.AddBizField("trade_no", req.ChannelPaymentId)

	type Detail struct {
		alipay.Error

		ErrorCode string `json:"error_code"`
		ErrorDesc string `json:"error_desc"`

		OperationType string `json:"operation_type"` // replenish, replenish_refund, transfer, transfer_refund
		ExecutedAt    string `json:"execute_dt"`     // 2021-07-30 12:00:00
		Amount        string `json:"amount"`
		State         string `json:"state"` // PROCESSING, SUCCESS, FAIL

		DetailId string `json:"detail_id"`

		TransOutType   string `json:"trans_out_type"` // userId, loginName, secondMerchantID
		TransOutOpenId string `json:"trans_out_open_id"`
		TransOut       string `json:"trans_out"`

		TransIn       string `json:"trans_in"`
		TransInType   string `json:"trans_in_type"` // userId, loginName, secondMerchantID
		TransInOpenId string `json:"trans_in_open_id"`
	}

	var rsp struct {
		alipay.Error

		OperatedAt   string   `json:"operation_dt"`
		OutRequestNo string   `json:"out_request_no"`
		ShareDetails []Detail `json:"royalty_detail_list"`
	}

	err = d.client.Request(ctx, _req, &rsp)
	switch {
	case err != nil:
		return

	case rsp.IsFailure():
		// TODO: check NotFound
		err = ToError(rsp.Error)
		return
	}

	info.ShareId = rsp.OutRequestNo
	info.ShareStatus = share.StatusFinished
	info.ChannelPaymentId = req.ChannelPaymentId

	optime, _ := time.ParseInLocation(time.DateTime, rsp.OperatedAt, time.Local)
	info.ShareRecords = make([]share.ShareRecord, len(rsp.ShareDetails))
	for i, sd := range rsp.ShareDetails {
		exectime, _ := time.ParseInLocation(time.DateTime, sd.ExecutedAt, time.Local)

		inaccount := sd.TransIn
		if inaccount == "" {
			inaccount = sd.TransInOpenId
		}

		outaccount := sd.TransOut
		if outaccount == "" {
			outaccount = sd.TransOutOpenId
		}

		var failureReason string
		var result share.Result
		switch sd.State {
		case "PROCESSING":
			result = share.ResultProcess
			info.ShareStatus = share.StatusProcessing

		case "SUCCESS":
			result = share.ResultSuccess

		case "FAIL":
			result = share.ResultFailure

		default:
			result = share.ResultFailure
		}

		if sd.ErrorCode != "" {
			failureReason = fmt.Sprintf("%s: %s", sd.ErrorCode, sd.ErrorDesc)
		}

		shareAmount, _ := currency.CNY.ParseMajorToMinor(sd.Amount)
		info.ShareRecords[i] = share.ShareRecord{
			ShareReceiver: share.ShareReceiver{
				ShareAmount: shareAmount,
				ShareDesc:   "",

				Receiver: share.AccountInfo{
					Account:     inaccount,
					AccountType: unfmtAccountType(sd.TransInType),
				},
			},

			Sender: share.AccountInfo{
				Account:     outaccount,
				AccountType: unfmtAccountType(sd.TransOutType),
			},

			IsInitiator:     outaccount == d.config.ShareAccount,
			ChannelDetailId: sd.DetailId,

			CreatedAt:  optime,
			FinishedAt: exectime,

			ShareResult: result,
			FailReason:  failureReason,
		}
	}

	return
}

// If the return account is equal to the share account, return share.ErrReturnAccountIsShareAccount.
func (d *_Driver) ReturnShare(ctx context.Context, req share.ReturnShareRequest) (info share.ReturnInfo, err error) {
	now := timex.Now()
	info = share.ReturnInfo{
		ShareId:  req.ShareId,
		ReturnId: req.ReturnId,

		ChannelShareId:  req.ChannelShareId,
		ChannelReturnId: "",

		ReturnAmount:  req.ReturnAmount,
		ReturnAccount: req.ReturnAccount,
		ReturnDesc:    req.ReturnDesc,

		ReturnResult: share.ResultSuccess,
		FailReason:   "",

		CreatedAt:  now,
		FinishedAt: now,
	}

	// TODO:
	return
}

func (d *_Driver) QueryReturn(ctx context.Context, req share.QueryReturnRequest) (info share.ReturnInfo, ok bool, err error) {
	now := timex.Now()
	info = share.ReturnInfo{
		ShareId:  req.ShareId,
		ReturnId: req.ReturnId,

		ReturnResult: share.ResultSuccess,
		FailReason:   "",

		CreatedAt:  now,
		FinishedAt: now,
	}

	// TODO:
	return
}

func (d *_Driver) DeleteShareReceiver(ctx context.Context, r share.Receiver) (err error) {
	// https://opendocs.alipay.com/open/3613f4e1_alipay.trade.royalty.relation.unbind
	return d.addOrDeleteReceiver(ctx, "alipay.trade.royalty.relation.unbind", r)
}

func (d *_Driver) AddShareReceiver(ctx context.Context, r share.Receiver) (err error) {
	// https://opendocs.alipay.com/open/c21931d6_alipay.trade.royalty.relation.bind
	return d.addOrDeleteReceiver(ctx, "alipay.trade.royalty.relation.bind", r)
}

func (d *_Driver) addOrDeleteReceiver(ctx context.Context, action string, r share.Receiver) (err error) {
	type Receiver struct {
		Name string `json:"name,omitempty"`
		Type string `json:"type,omitempty"`
		Memo string `json:"memo,omitempty"`

		Account string `json:"account,omitempty"`

		// LoginName     string `json:"login_name,omitempty"`
		// BindLoginName string `json:"bind_login_name,omitempty"`
		// AccountOpenId string `json:"account_open_id,omitempty"`
	}

	atype, err := fmtAccountType(r.AccountType)
	if err != nil {
		return
	}

	// TODO:
	reqno := random.String(28, random.DefaultCharset)

	memo := string(r.RelationType)
	req := alipay.NewPayload(action)
	req.AddBizField("out_request_no", reqno)
	req.AddBizField("receiver_list", []Receiver{{
		Name: r.Name,
		Type: atype,
		Memo: memo,

		Account: r.Account,
	}})

	var rsp struct {
		alipay.Error

		ResultCode string `json:"result_code"`
	}

	err = d.client.Request(ctx, req, &rsp)
	switch {
	case err != nil:

	case rsp.IsFailure():
		err = ToError(rsp.Error)

	case rsp.ResultCode != "SUCCESS":
		err = errors.New(rsp.ResultCode)
	}

	return
}
