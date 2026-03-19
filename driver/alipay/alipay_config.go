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

const (
	SignMethodCert   = "cert"
	SignMethodPubkey = "pubkey"
)

type Config struct {
	Appid string `validate:"required"`

	// 在【控制台】->【（应用的）开发设置】->【接口加签方式（密钥/证书）】中进行配置时，
	// 支付宝会自动生成 CSR 文件及应用公私钥，其中私钥文件中存储的内容，即为该值。
	Privatekey string `json:",omitempty" validate:"required"`

	// 在【控制台】->【（应用的）开发设置】->【接口内容加签方式】中点击查看，
	// 然后复制 AES: 后面的内容，即为该值。
	EncryptKey string `json:",omitempty"` // Optional

	// Optional: SignMethodCert or SignMethodPubkey
	// Default: cert (SignMethodCert)
	SignMethod string `default:"cert" validate:"oneof(\"cert\",\"pubkey\")"`

	// 在【控制台】->【（应用的）开发设置】->【接口加签方式（密钥/证书）】中，
	// 点击查看，然后下载【应用公钥证书】、【支付宝公钥证书】、【支付宝根证书】三个文件。

	// SignMethodCert
	AppCertPubKey    string `json:",omitempty"`
	AlipayCertPubKey string `json:",omitempty"`
	AlipayRootCert   string `json:",omitempty"`

	// SignMethodPubkey
	AlipayPubKey string `json:",omitempty"`

	// Share Info
	ShareAccountType string `json:",omitempty"` // userId or loginName
	ShareAccount     string `json:",omitempty"` // 2088xxx or email
	ShareEnabled     bool   `json:"-"`
	ShareAsync       bool

	IsTest bool `json:",omitempty"`
}

func (c *Config) Desensitize() {
	c.Privatekey = ""
	c.EncryptKey = ""
	c.AppCertPubKey = ""
	c.AlipayCertPubKey = ""
	c.AlipayRootCert = ""
	c.AlipayPubKey = ""
}
