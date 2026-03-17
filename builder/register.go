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

	"github.com/xgfone/go-payment-driver/driver"
)

var _builders = make(map[string]Builder, 8)

// Register registers a builder to the global builder registry.
func Register(builder Builder) {
	_type := builder.Metadata().Type
	if _, ok := _builders[_type]; ok {
		panic(fmt.Errorf("builder typed '%s' has been registered", _type))
	}
	_builders[_type] = builder
}

// Get returns the builder registered with the given type.
func Get(_type string) Builder {
	return _builders[_type]
}

// ParseConfig parses the configuration string and returns the config object.
// If the config object implements the interface{ Desensitize() },
// it will be called to desensitize the config.
func ParseConfig(_type, conf string) (any, error) {
	builder, ok := _builders[_type]
	if !ok {
		return nil, fmt.Errorf("not found builder typed '%s'", _type)
	}

	config, err := builder.ParseConfig(conf)
	if err != nil {
		return nil, err
	}

	type Desensitizer interface{ Desensitize() }
	if d, ok := config.(Desensitizer); ok {
		d.Desensitize()
	}

	return config, nil
}

// BuildDriver builds the payment channel driver from the configuration string.
func BuildDriver(_type, conf string) (driver.Driver, error) {
	builder, ok := _builders[_type]
	if !ok {
		return nil, fmt.Errorf("not found builder typed '%s'", _type)
	}

	config, err := builder.ParseConfig(conf)
	if err != nil {
		return nil, err
	}

	return builder.BuildDriver(config)
}
