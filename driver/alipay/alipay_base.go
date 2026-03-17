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
	"fmt"
	"strconv"

	"github.com/smartwalle/alipay/v3"
)

const Type = "alipay"

type Error struct {
	Code string
	Msg  string

	SubCode string
	SubMsg  string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.Code, e.SubCode, e.SubMsg)
}

func (e *Error) From(err *alipay.Error) {
	*e = Error{
		Code: string(err.Code),
		Msg:  err.Msg,

		SubCode: err.SubCode,
		SubMsg:  err.SubMsg,
	}
}

func ToError(err error) error {
	var _e Error
	switch e := err.(type) {
	case alipay.Error:
		_e.From(&e)

	case *alipay.Error:
		_e.From(e)

	default:
		return err
	}

	return _e
}

func parseAmount(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func formatAmount(v int64) string {
	return fmt.Sprintf("%d.%02d", v/100, v%100)
}

type (
	SubFeeDetail struct {
		ChargeFee         string `json:",omitzero"`
		SwitchFeeRate     string `json:",omitzero"`
		OriginalChargeFee string `json:",omitzero"`
	}

	ChargeInfo struct {
		ChargeFee  string `json:",omitzero"`
		ChargeType string `json:",omitzero"`

		SwitchFeeRate     string `json:",omitzero"`
		OriginalChargeFee string `json:",omitzero"`

		IsRatingOnTradeReceiver string `json:",omitzero"`
		IsRatingOnSwitch        string `json:",omitzero"`

		SubFeeDetailList []SubFeeDetail `json:",omitzero"`
	}

	ChannelData struct {
		/// PayOrder
		PayAmount   string `json:",omitzero"`
		PayCurrency string `json:",omitzero"`

		PointAmount     string `json:",omitzero"`
		ReceiptAmount   string `json:",omitzero"`
		InvoiceAmount   string `json:",omitzero"`
		DiscountAmount  string `json:",omitzero"`
		MdiscountAmount string `json:",omitzero"`
		BuyerUserType   string `json:",omitzero"`
		BuyerLogonId    string `json:",omitzero"`
		BuyerUserId     string `json:",omitzero"`
		BuyerOpenId     string `json:",omitzero"`

		SettleAmount    string `json:",omitzero"`
		SettleCurrency  string `json:",omitzero"`
		SettleTransRate string `json:",omitzero"`

		ChargeInfoList []ChargeInfo `json:",omitzero"`

		/// RefundOrder
		SendBackFee string `json:",omitzero"`
		FundChange  string `json:",omitzero"`

		RefundDetailItems []RefundDetailItem `json:",omitzero"`
		RefundChargeInfos []RefundChargeInfo `json:",omitzero"`
	}

	RefundDetailItem struct {
		Amount      string `json:",omitzero"`
		RealAmount  string `json:",omitzero"`
		FundChannel string `json:",omitzero"`
		FundType    string `json:",omitzero"`
	}

	RefundChargeInfo struct {
		ChargeType      string               `json:",omitzero"`
		SwitchFeeRate   string               `json:",omitzero"`
		RefundChargeFee string               `json:",omitzero"`
		SubFeeDetails   []RefundSubFeeDetail `json:",omitzero"`
	}

	RefundSubFeeDetail struct {
		SwitchFeeRate   string `json:",omitzero"`
		RefundChargeFee string `json:",omitzero"`
	}
)
