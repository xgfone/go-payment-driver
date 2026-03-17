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

// Package builder provides the builder interface and implementations.
package builder

import "github.com/xgfone/go-payment-driver/driver"

// Builder is used to build the payment channel driver.
type Builder interface {
	// Metadata returns the metadata of the payment channel driver.
	Metadata() driver.Metadata

	// ParseConfig parses the configuration string into the config object.
	ParseConfig(conf string) (config any, err error)

	// BuildDriver builds the payment channel driver from the config object.
	BuildDriver(conf any) (driver.Driver, error)
}
