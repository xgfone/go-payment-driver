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

package currency

import (
	"fmt"
	"strings"
)

var _currencies = make(map[string]*Currency, 32)

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// Register registers a currency into the package-level registry.
//
// It panics if the given currency definition is invalid.
// It also panics if the same currency code is registered more than once.
func Register(code string, minorUnit int8, symbol, name string) {
	code = normalizeCode(code)

	if _, exists := _currencies[code]; exists {
		panic(fmt.Errorf("currency %s already registered", code))
	}

	currency := Currency{
		Name:      name,
		Code:      code,
		Symbol:    symbol,
		MinorUnit: minorUnit,
	}

	if err := currency.Validate(); err != nil {
		panic(err)
	}

	_currencies[code] = &currency
}

// Get returns the registered currency by code.
//
// It returns nil if the currency is not found.
func Get(code string) *Currency {
	return _currencies[normalizeCode(code)]
}

// IsSupported reports whether the given currency code is supported.
func IsSupported(code string) bool {
	_, ok := _currencies[normalizeCode(code)]
	return ok
}
