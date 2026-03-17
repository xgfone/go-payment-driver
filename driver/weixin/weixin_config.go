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

	"github.com/xgfone/go-payment-driver/builder"
	"github.com/xgfone/go-toolkit/jsonx"
	"github.com/xgfone/go-toolkit/validation"
)

const Type = "weixin"

type Config struct {
	Mchid  string `validate:"required"`                   // The merchant id
	Appid  string `validate:"required"`                   // The weixin appid
	Apikey string `validate:"required" json:",omitempty"` // The API v3 key of the merchant
	Prikey string `validate:"required" json:",omitempty"` // The PEM private key of the merchant certificate
	Certsn string `validate:"required" json:",omitempty"` // The serial number of the merchant certificate

	// New: Optional
	PubKeyId string `json:",omitempty"` // The ID of the PEM public key of weixin
	PubKey   string `json:",omitempty"` // The PEM public key of weixin

	H5Type string `json:",omitempty"` // Wap, iOS, Android
}

func (c *Config) init() (err error) {
	if c.H5Type == "" {
		c.H5Type = "Wap"
	}
	return
}

func (c *Config) Parse(conf string) (err error) {
	if err = jsonx.UnmarshalString(conf, c); err == nil {
		err = validation.Validate(context.Background(), c)
	}
	return
}

func (c *Config) Driver(b builder.Builder) (_Driver, error) {
	return newDriver(*c, b)
}

func (c *Config) Desensitize() {
	*c = Config{Mchid: c.Mchid, Appid: c.Appid}
}

/// ------------------------------------------------------------------------ ///

type (
	Promotion struct {
		Name     string `json:",omitempty"`
		Type     string `json:",omitempty"`
		Scope    string `json:",omitempty"`
		StockId  string `json:",omitempty"`
		CouponId string `json:",omitempty"`
		Currency string `json:",omitempty"`
		Amount   int64  `json:",omitempty"`

		WechatpayContribute int64 `json:",omitempty"`
		MerchantContribute  int64 `json:",omitempty"`
		OtherContribute     int64 `json:",omitempty"`
	}

	ChannelData struct {
		// For Pay
		BankType   string      `json:",omitempty"`
		Promotions []Promotion `json:",omitzero"`

		// For Refund
		UserAccount string `json:",omitempty"`
	}
)
