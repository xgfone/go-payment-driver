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

package driver

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePercentAmount parses an amount string as an integer using hundredfold conversion,
// for example, from yuan to cent (1 yuan = 100 cents).
func ParsePercentAmount(s string) (cent int64, err error) {
	yuanstr, centstr, _ := strings.Cut(s, ".")

	var yuan int64
	if yuanstr != "" {
		yuan, err = strconv.ParseInt(yuanstr, 10, 64)
		if err != nil {
			return
		}
	}

	if centstr != "" {
		cent, err = strconv.ParseInt(centstr, 10, 64)
		if err != nil {
			return
		}
	}

	cent += yuan * 100
	return
}

// FormatPercentAmount formats an integer as an amount string using hundredfold conversion,
// for example, from cent to yuan (100 cents = 1 yuan).
func FormatPercentAmount(cent int64) (yuan string) {
	return fmt.Sprintf("%d.%02d", cent/100, cent%100)
}
