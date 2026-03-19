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

package builder

import (
	"context"
	"fmt"

	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-toolkit/validation"
)

type DriverNewer[Config any] func(Builder, Config) (driver.Driver, error)

// New returns the new builder of a payment channel driver.
//
// Metadata:
//
//	Provider: Required
//	PayScene: Required
//	Type:     Optional, defaults to "${Provider}_${PayScene}"
//	LinkType: Optional, defaults to Type
//	Channels: Optional, defaults to [Provider]
//	Currencies: Optional, defaults to ["CNY"]
func New[Config any](newDriver DriverNewer[Config], metadata driver.Metadata) Builder {
	if metadata.Provider == "" {
		panic("Metadata.Provider must not be empty")
	}
	if metadata.PayScene == "" {
		panic("Metadata.PayScene must not be empty")
	}

	if metadata.Type == "" {
		metadata.Type = fmt.Sprintf("%s_%s", metadata.Provider, metadata.PayScene)
	}

	if metadata.LinkType == "" {
		metadata.LinkType = driver.LinkType(metadata.Type)
	}

	if len(metadata.Channels) == 0 {
		metadata.Channels = []string{metadata.Provider}
	}

	if len(metadata.Currencies) == 0 {
		metadata.Currencies = []string{"CNY"}
	}

	return &_Builder[Config]{
		metadata:  metadata,
		newDriver: newDriver,
	}
}

type _Builder[Config any] struct {
	metadata  driver.Metadata
	newDriver DriverNewer[Config]
}

func (b *_Builder[Config]) Metadata() driver.Metadata {
	return b.metadata
}

func (b *_Builder[Config]) ParseConfig(conf string) (any, error) {
	var err error
	var config Config
	if v, ok := any(config).(interface{ Bind(string) error }); ok {
		err = v.Bind(conf)
	} else {
		err = validation.BindJSONString(context.Background(), conf, &config)
	}
	return config, err
}

func (b *_Builder[Config]) BuildDriver(conf any) (driver.Driver, error) {
	config, ok := conf.(Config)
	if !ok {
		return nil, fmt.Errorf("expects config type '%T', but got '%T'", config, conf)
	}
	return b.newDriver(b, config)
}
