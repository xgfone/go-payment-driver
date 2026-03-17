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
	"errors"
	"fmt"
	"strings"

	"github.com/smartwalle/alipay/v3"
	"github.com/xgfone/go-payment-driver/builder"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-toolkit/timex"
)

var _channels = []string{Type}

func registerBuilder(scene string, newf func(_Driver) driver.Driver) {
	_type := fmt.Sprintf("%s_%s", Type, scene)
	builder.Register(builder.New[*Config](newf, driver.Metadata{
		Type:     _type,
		PayScene: scene,
		Provider: Type,
		Channels: _channels,
	}))
}

type _Driver struct {
	config   Config
	client   *alipay.Client
	metadata driver.Metadata
}

func (d *_Driver) initSignatureForCert() (err error) {
	switch {
	case d.config.AppCertPubKey == "":
		return errors.New("missing AppCertPubKey")

	case d.config.AlipayCertPubKey == "":
		return errors.New("missing AlipayCertPubKey")

	case d.config.AlipayRootCert == "":
		return errors.New("missing AlipayRootCert")
	}

	d.config.AppCertPubKey = strings.ReplaceAll(d.config.AppCertPubKey, `\n`, "\n")
	if err = d.client.LoadAppCertPublicKey(d.config.AppCertPubKey); err != nil {
		err = fmt.Errorf("fail to load app cert public key: %w", err)
		return
	}

	d.config.AlipayCertPubKey = strings.ReplaceAll(d.config.AlipayCertPubKey, `\n`, "\n")
	if err = d.client.LoadAlipayCertPublicKey(d.config.AlipayCertPubKey); err != nil {
		err = fmt.Errorf("fail to load alipay cert public key: %w", err)
		return
	}

	d.config.AlipayRootCert = strings.ReplaceAll(d.config.AlipayRootCert, `\n`, "\n")
	if err = d.client.LoadAliPayRootCert(d.config.AlipayRootCert); err != nil {
		err = fmt.Errorf("fail to load alipay root cert: %w", err)
		return
	}

	return
}

func (d *_Driver) initSignatureForPubKey() (err error) {
	if d.config.AlipayPubKey == "" {
		return errors.New("missing AlipayPubKey")
	}

	d.config.AlipayPubKey = strings.ReplaceAll(d.config.AlipayPubKey, `\n`, "\n")
	if err = d.client.LoadAliPayPublicKey(d.config.AlipayPubKey); err != nil {
		err = fmt.Errorf("fail to load alipay public key: %w", err)
	}
	return
}

func newDriver(c Config, b builder.Builder) (d _Driver, err error) {
	d.metadata = b.Metadata()
	d.config = c

	// 1. Initialize the alipay client
	options := make([]alipay.OptionFunc, 0, 1)
	options = append(options, alipay.WithTimeLocation(timex.Location))
	d.client, err = alipay.New(c.Appid, c.Privatekey, !c.IsTest, options...)
	if err != nil {
		err = fmt.Errorf("fail to create alipay client: %w", err)
		return
	}

	// 2. TODO:

	// 3. Initialize the alipay encrypt key
	if c.EncryptKey != "" {
		if err = d.client.SetEncryptKey(c.EncryptKey); err != nil {
			err = fmt.Errorf("fail to set encrypt key: %w", err)
			return
		}
	}

	// 4. Initialize the alipay signature method
	switch c.SignMethod {
	case "":
		d.config.SignMethod = SignMethodCert
		err = d.initSignatureForCert()

	case SignMethodCert:
		err = d.initSignatureForCert()

	case SignMethodPubkey:
		err = d.initSignatureForPubKey()

	default:
		err = fmt.Errorf("unsupported SignMethod")
	}

	// 5. Initialize the share information
	d.config.ShareEnabled = d.config.ShareAccount != "" && d.config.ShareAccountType != ""

	return
}

func (d *_Driver) Metadata() driver.Metadata {
	return d.metadata
}

func (d *_Driver) LinkInfo(paylink string) driver.LinkInfo {
	return d.metadata.LinkType.LinkInfo(paylink)
}
