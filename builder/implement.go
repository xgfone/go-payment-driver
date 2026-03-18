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
	"fmt"
	"reflect"

	"github.com/xgfone/go-payment-driver/driver"
)

type DriverConfig[Driver any] interface {
	Parse(string) error
	Driver(Builder) (Driver, error)
}

type DriverNewer[Driver any] func(Driver) driver.Driver

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
func New[Config DriverConfig[Driver], Driver any](newDriver DriverNewer[Driver], metadata driver.Metadata) Builder {
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

	ctype := reflect.TypeFor[Config]()
	if ctype.Kind() != reflect.Pointer {
		panic(fmt.Errorf("builder typed '%s' expects config type is a pointer", metadata.Type))
	}

	ctype = ctype.Elem()
	newconfig := func() Config { return reflect.New(ctype).Interface().(Config) }

	return &_Builder[Config, Driver]{
		metadata:  metadata,
		newConfig: newconfig,
		newDriver: newDriver,
	}
}

type _Builder[Config DriverConfig[Driver], Driver any] struct {
	metadata  driver.Metadata
	newConfig func() Config
	newDriver func(Driver) driver.Driver
}

func (b *_Builder[Config, Driver]) Metadata() driver.Metadata {
	return b.metadata
}

func (b *_Builder[Config, Driver]) ParseConfig(conf string) (any, error) {
	config := b.newConfig()
	err := config.Parse(conf)
	return config, err
}

func (b *_Builder[Config, Driver]) BuildDriver(conf any) (driver.Driver, error) {
	config, ok := conf.(Config)
	if !ok {
		return nil, fmt.Errorf("expects config type '%T', but got '%T'", config, conf)
	}

	driver, err := config.Driver(b)
	if err != nil {
		return nil, err
	}

	return b.newDriver(driver), nil
}
