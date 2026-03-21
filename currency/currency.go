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
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	MinMinorUnit int8 = 0
	MaxMinorUnit int8 = 3
)

var (
	ErrUnsupportedCurrency  = errors.New("unsupported currency code")
	ErrUnsupportedMinorUnit = errors.New("unsupported minor unit")
	ErrTooManyDecimalPlaces = errors.New("too many decimal places")
	ErrInvalidAmountFormat  = errors.New("invalid amount format")
	ErrAmountOutOfRange     = errors.New("amount out of int64 range")
)

var (
	CNY = Currency{Code: "CNY", MinorUnit: 2, Symbol: "¥", Name: "Chinese Yuan"}
	USD = Currency{Code: "USD", MinorUnit: 2, Symbol: "$", Name: "US Dollar"}
	EUR = Currency{Code: "EUR", MinorUnit: 2, Symbol: "€", Name: "Euro"}
)

type Currency struct {
	Name      string
	Code      string
	Symbol    string
	MinorUnit int8
}

func (c Currency) Validate() error {
	if c.Code == "" {
		return errors.New("Currency: Code is empty")
	}

	if c.MinorUnit < MinMinorUnit || c.MinorUnit > MaxMinorUnit {
		return fmt.Errorf("%w: %d for currency %s", ErrUnsupportedMinorUnit, c.MinorUnit, c.Code)
	}

	return nil
}

func (c Currency) scale() (uint64, error) {
	if err := c.Validate(); err != nil {
		return 0, err
	}

	switch c.MinorUnit {
	case 0:
		return 1, nil

	case 1:
		return 10, nil

	case 2:
		return 100, nil

	case 3:
		return 1000, nil

	default:
		return 0, fmt.Errorf("%w: %d for currency %s", ErrUnsupportedMinorUnit, c.MinorUnit, c.Code)
	}
}

// FormatMinorToMajor converts an integer amount in minor units to a major-unit string.
// Examples:
//
//	MinorUnit=2, 123  => "1.23"
//	MinorUnit=2, -123 => "-1.23"
//	MinorUnit=0, 500  => "500"
func (c Currency) FormatMinorToMajor(minorAmount int64) (string, error) {
	scale, err := c.scale()
	if err != nil {
		return "", err
	}

	if c.MinorUnit == 0 {
		return strconv.FormatInt(minorAmount, 10), nil
	}

	var sign string
	if minorAmount < 0 {
		sign = "-"
	}

	mag := absInt64ToUint64(minorAmount)

	intPart := mag / scale
	fracPart := mag % scale

	s := fmt.Sprintf("%s%d.%0*d", sign, intPart, int(c.MinorUnit), fracPart)
	return s, nil
}

// ParseMajorToMinor converts a major-unit decimal string into an integer amount in minor units.
// Strict mode:
//   - accepts: "1", "1.2", "1.23", ".5", "1.", "-1.23"
//   - rejects values with more fractional digits than MinorUnit allows
//
// Examples:
//
//	MinorUnit=2, "1"     => 100
//	MinorUnit=2, "1.2"   => 120
//	MinorUnit=2, ".5"    => 50
//	MinorUnit=2, "-1.23" => -123
//	MinorUnit=2, "1.234" => error
func (c Currency) ParseMajorToMinor(majorAmount string) (int64, error) {
	if err := c.Validate(); err != nil {
		return 0, err
	}
	return parseMajorAmount(majorAmount, c.MinorUnit)
}

func parseMajorAmount(majorAmount string, minorUnit int8) (int64, error) {
	s := strings.TrimSpace(majorAmount)
	if s == "" {
		return 0, fmt.Errorf("%w: empty amount", ErrInvalidAmountFormat)
	}

	negative := false
	switch s[0] {
	case '-':
		negative = true
		s = s[1:]

	case '+':
		s = s[1:]
	}

	if s == "" {
		return 0, fmt.Errorf("%w: sign without digits", ErrInvalidAmountFormat)
	}

	if strings.Count(s, ".") > 1 {
		return 0, fmt.Errorf("%w: multiple decimal points", ErrInvalidAmountFormat)
	}

	before, after, _ := strings.Cut(s, ".")

	// Allow ".5" => "0.5"
	if before == "" {
		before = "0"
	}

	if !allDigits(before) {
		return 0, fmt.Errorf("%w: invalid integer part %q", ErrInvalidAmountFormat, before)
	}
	if after != "" && !allDigits(after) {
		return 0, fmt.Errorf("%w: invalid fractional part %q", ErrInvalidAmountFormat, after)
	}

	if len(after) > int(minorUnit) {
		return 0, fmt.Errorf("%w: got %d, want <= %d", ErrTooManyDecimalPlaces, len(after), minorUnit)
	}

	// Right-pad fractional digits to the currency precision.
	// e.g. MinorUnit=2: "1.2" => "1.20"
	if len(after) < int(minorUnit) {
		after += strings.Repeat("0", int(minorUnit)-len(after))
	}

	intPart, err := parseUint64Digits(before)
	if err != nil {
		return 0, err
	}

	var fracPart uint64
	if after != "" {
		fracPart, err = parseUint64Digits(after)
		if err != nil {
			return 0, err
		}
	}

	scale := pow10u(minorUnit)
	if intPart > math.MaxUint64/scale {
		return 0, ErrAmountOutOfRange
	}
	totalMag := intPart * scale

	if fracPart > math.MaxUint64-totalMag {
		return 0, ErrAmountOutOfRange
	}
	totalMag += fracPart

	if negative {
		// allow MinInt64 magnitude
		maxNegMag := uint64(math.MaxInt64) + 1
		if totalMag > maxNegMag {
			return 0, ErrAmountOutOfRange
		}
		if totalMag == maxNegMag {
			return math.MinInt64, nil
		}
		return -int64(totalMag), nil
	}

	if totalMag > uint64(math.MaxInt64) {
		return 0, ErrAmountOutOfRange
	}
	return int64(totalMag), nil
}

func parseUint64Digits(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}

	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return 0, ErrAmountOutOfRange
		}
		return 0, fmt.Errorf("%w: %q", ErrInvalidAmountFormat, s)
	}

	return v, nil
}

func allDigits(s string) bool {
	if s == "" {
		return true
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func pow10u(n int8) uint64 {
	switch n {
	case 0:
		return 1

	case 1:
		return 10

	case 2:
		return 100

	case 3:
		return 1000

	default:
		return 0
	}
}

func absInt64ToUint64(v int64) uint64 {
	if v >= 0 {
		return uint64(v)
	}

	// Safe abs for MinInt64
	return uint64(-(v + 1)) + 1
}
