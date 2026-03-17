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
	"fmt"

	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"github.com/xgfone/go-payment-driver/share"
)

const (
	AccountTypeMerchant share.AccountType = "Merchant"
	AccountTypePersonal share.AccountType = "Personal"
)

func init() {
	share.RegisterReceiverAccountTypeValidator(Type, func(t share.AccountType) (err error) {
		switch t {
		case AccountTypeMerchant, AccountTypePersonal:
		default:
			err = fmt.Errorf("invalid account type: not in [%s, %s]",
				AccountTypeMerchant, AccountTypePersonal)
		}
		return
	})
}

func _ptr[T any](v T) *T { return &v }

var (
	_AccountTypeMerchant = "MERCHANT_ID"
	_AccountTypePersonal = "PERSONAL_OPENID"

	_AccountTypeMerchantPtr = _ptr(_AccountTypeMerchant)
	_AccountTypePersonalPtr = _ptr(_AccountTypePersonal)
)

func fmtAccountType(rtype share.AccountType) (_type *string, err error) {
	switch rtype {
	case AccountTypeMerchant:
		_type = _AccountTypeMerchantPtr

	case AccountTypePersonal:
		_type = _AccountTypePersonalPtr

	default:
		err = fmt.Errorf("invalid receiver type: %s", rtype)
	}

	return
}

func unfmtAccountType(_type string) (rtype share.AccountType) {
	switch _type {
	case _AccountTypeMerchant:
		rtype = AccountTypeMerchant

	case _AccountTypePersonal:
		rtype = AccountTypePersonal

	default:
		rtype = share.AccountType(_type)
	}

	return
}

var (
	_ReceiverRelationTypeDistributor = _ptr[profitsharing.ReceiverRelationType]("DISTRIBUTOR")
	_ReceiverRelationTypeSupplier    = _ptr[profitsharing.ReceiverRelationType]("SUPPLIER")
	_ReceiverRelationTypeBrand       = _ptr[profitsharing.ReceiverRelationType]("BRAND")

	_ReceiverRelationTypePartner = _ptr[profitsharing.ReceiverRelationType]("PARTNER")
	_ReceiverRelationTypeStore   = _ptr[profitsharing.ReceiverRelationType]("STORE")
	_ReceiverRelationTypeStaff   = _ptr[profitsharing.ReceiverRelationType]("STAFF")
	_ReceiverRelationTypeUser    = _ptr[profitsharing.ReceiverRelationType]("USER")

	_ReceiverRelationTypeCustom = _ptr[profitsharing.ReceiverRelationType]("CUSTOM")
)

func fmtRelationType(rt share.RelationType) (_type *profitsharing.ReceiverRelationType, _custom *string, err error) {
	switch rt {
	case share.RelationTypeDistributor:
		_type = _ReceiverRelationTypeDistributor

	case share.RelationTypeSupplier:
		_type = _ReceiverRelationTypeSupplier

	case share.RelationTypeBrand:
		_type = _ReceiverRelationTypeBrand

	case share.RelationTypePartner:
		_type = _ReceiverRelationTypePartner

	case share.RelationTypeStore:
		_type = _ReceiverRelationTypeStore

	case share.RelationTypeStaff:
		_type = _ReceiverRelationTypeStaff

	case share.RelationTypeUser:
		_type = _ReceiverRelationTypeUser

	default:
		if len(rt) > 10 {
			err = fmt.Errorf("too longer receiver relation type: %s", rt)
		} else {
			_type = _ReceiverRelationTypeCustom
			_custom = (*string)(&rt)
		}
	}
	return
}
