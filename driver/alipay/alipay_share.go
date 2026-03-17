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

	"github.com/xgfone/go-payment-driver/share"
)

// https://opendocs.alipay.com/open/009yj8

const (
	AccountTypeUserId    share.AccountType = "UserId"
	AccountTypeOpenId    share.AccountType = "OpenId"
	AccountTypeLoginName share.AccountType = "LoginName"
)

func init() {
	share.RegisterReceiverAccountTypeValidator(Type, func(t share.AccountType) (err error) {
		switch t {
		case AccountTypeUserId, AccountTypeOpenId, AccountTypeLoginName:
		default:
			err = fmt.Errorf("invalid account type: not in [%s, %s, %s]",
				AccountTypeUserId, AccountTypeOpenId, AccountTypeLoginName)
		}
		return
	})
}

var (
	_AccountTypeUserId    = "userId"
	_AccountTypeOpenId    = "openId"
	_AccountTypeLoginName = "loginName"
)

func fmtAccountType(rtype share.AccountType) (_type string, err error) {
	switch rtype {
	case AccountTypeUserId:
		_type = _AccountTypeUserId

	case AccountTypeOpenId:
		_type = _AccountTypeOpenId

	case AccountTypeLoginName:
		_type = _AccountTypeLoginName

	default:
		switch _type = string(rtype); _type {
		case _AccountTypeUserId, _AccountTypeOpenId, _AccountTypeLoginName:
		default:
			err = fmt.Errorf("invalid receiver type: %s", _type)
		}
	}

	return
}

func unfmtAccountType(_type string) (atype share.AccountType) {
	switch _type {
	case _AccountTypeUserId:
		atype = AccountTypeUserId

	case _AccountTypeOpenId:
		atype = AccountTypeOpenId

	case _AccountTypeLoginName:
		atype = AccountTypeLoginName

	default:
		atype = share.AccountType(_type)
	}

	return
}
