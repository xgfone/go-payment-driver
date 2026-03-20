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
	"math"
	"testing"
)

func TestCurrency_Validate(t *testing.T) {
	t.Run("valid currency", func(t *testing.T) {
		c := Currency{Code: "CNY", MinorUnit: 2}
		if err := c.Validate(); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("unsupported minor unit negative", func(t *testing.T) {
		c := Currency{Code: "XXX", MinorUnit: -1}
		err := c.Validate()
		if !errors.Is(err, ErrUnsupportedMinorUnit) {
			t.Fatalf("expected ErrUnsupportedMinorUnit, got %v", err)
		}
	})

	t.Run("unsupported minor unit too large", func(t *testing.T) {
		c := Currency{Code: "XXX", MinorUnit: 4}
		err := c.Validate()
		if !errors.Is(err, ErrUnsupportedMinorUnit) {
			t.Fatalf("expected ErrUnsupportedMinorUnit, got %v", err)
		}
	})
}

func TestCurrency_FormatMinorToMajor(t *testing.T) {
	tests := []struct {
		name      string
		currency  Currency
		input     int64
		want      string
		wantErrIs error
	}{
		{
			name:     "minor unit 0 positive",
			currency: Currency{Code: "JPY", MinorUnit: 0},
			input:    500,
			want:     "500",
		},
		{
			name:     "minor unit 0 negative",
			currency: Currency{Code: "JPY", MinorUnit: 0},
			input:    -500,
			want:     "-500",
		},
		{
			name:     "minor unit 1 positive",
			currency: Currency{Code: "X1", MinorUnit: 1},
			input:    123,
			want:     "12.3",
		},
		{
			name:     "minor unit 1 negative small",
			currency: Currency{Code: "X1", MinorUnit: 1},
			input:    -5,
			want:     "-0.5",
		},
		{
			name:     "minor unit 2 positive",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    123,
			want:     "1.23",
		},
		{
			name:     "minor unit 2 zero padding",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    5,
			want:     "0.05",
		},
		{
			name:     "minor unit 2 negative",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    -123,
			want:     "-1.23",
		},
		{
			name:     "minor unit 3 positive",
			currency: Currency{Code: "KWD", MinorUnit: 3},
			input:    1234,
			want:     "1.234",
		},
		{
			name:     "minor unit 3 zero padding",
			currency: Currency{Code: "KWD", MinorUnit: 3},
			input:    5,
			want:     "0.005",
		},
		{
			name:     "minor unit 3 negative",
			currency: Currency{Code: "KWD", MinorUnit: 3},
			input:    -1234,
			want:     "-1.234",
		},
		{
			name:     "min int64 with minor unit 2",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    math.MinInt64,
			want:     "-92233720368547758.08",
		},
		{
			name:      "unsupported minor unit",
			currency:  Currency{Code: "BAD", MinorUnit: 4},
			input:     100,
			wantErrIs: ErrUnsupportedMinorUnit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.currency.FormatMinorToMajor(tt.input)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCurrency_ParseMajorToMinor(t *testing.T) {
	tests := []struct {
		name      string
		currency  Currency
		input     string
		want      int64
		wantErrIs error
	}{
		{
			name:     "minor unit 0 integer",
			currency: Currency{Code: "JPY", MinorUnit: 0},
			input:    "123",
			want:     123,
		},
		{
			name:     "minor unit 0 with trailing dot",
			currency: Currency{Code: "JPY", MinorUnit: 0},
			input:    "123.",
			want:     123,
		},
		{
			name:      "minor unit 0 rejects decimal digits",
			currency:  Currency{Code: "JPY", MinorUnit: 0},
			input:     "123.0",
			wantErrIs: ErrTooManyDecimalPlaces,
		},
		{
			name:     "minor unit 1 integer",
			currency: Currency{Code: "X1", MinorUnit: 1},
			input:    "12",
			want:     120,
		},
		{
			name:     "minor unit 1 decimal",
			currency: Currency{Code: "X1", MinorUnit: 1},
			input:    "12.3",
			want:     123,
		},
		{
			name:     "minor unit 2 integer",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "1",
			want:     100,
		},
		{
			name:     "minor unit 2 one decimal place padded",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "1.2",
			want:     120,
		},
		{
			name:     "minor unit 2 exact decimal places",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "1.23",
			want:     123,
		},
		{
			name:     "minor unit 2 leading decimal point",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    ".5",
			want:     50,
		},
		{
			name:     "minor unit 2 trailing decimal point",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "1.",
			want:     100,
		},
		{
			name:     "minor unit 2 negative",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "-1.23",
			want:     -123,
		},
		{
			name:     "minor unit 2 positive sign",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "+1.23",
			want:     123,
		},
		{
			name:     "minor unit 2 trims spaces",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "  1.23  ",
			want:     123,
		},
		{
			name:     "minor unit 3 exact decimal places",
			currency: Currency{Code: "KWD", MinorUnit: 3},
			input:    "1.234",
			want:     1234,
		},
		{
			name:     "minor unit 3 padded",
			currency: Currency{Code: "KWD", MinorUnit: 3},
			input:    "1.2",
			want:     1200,
		},
		{
			name:      "too many decimal places",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "1.234",
			wantErrIs: ErrTooManyDecimalPlaces,
		},
		{
			name:      "empty amount",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "",
			wantErrIs: ErrInvalidAmountFormat,
		},
		{
			name:      "spaces only",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "   ",
			wantErrIs: ErrInvalidAmountFormat,
		},
		{
			name:      "sign only",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "-",
			wantErrIs: ErrInvalidAmountFormat,
		},
		{
			name:      "multiple dots",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "1.2.3",
			wantErrIs: ErrInvalidAmountFormat,
		},
		{
			name:      "invalid integer part",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "a.23",
			wantErrIs: ErrInvalidAmountFormat,
		},
		{
			name:      "invalid fractional part",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "1.ab",
			wantErrIs: ErrInvalidAmountFormat,
		},
		{
			name:      "unsupported minor unit",
			currency:  Currency{Code: "BAD", MinorUnit: 4},
			input:     "1.23",
			wantErrIs: ErrUnsupportedMinorUnit,
		},
		{
			name:      "overflow positive",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "92233720368547758.08",
			wantErrIs: ErrAmountOutOfRange,
		},
		{
			name:     "max int64 boundary",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "92233720368547758.07",
			want:     math.MaxInt64,
		},
		{
			name:     "min int64 boundary",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "-92233720368547758.08",
			want:     math.MinInt64,
		},
		{
			name:      "overflow negative beyond min int64",
			currency:  Currency{Code: "CNY", MinorUnit: 2},
			input:     "-92233720368547758.09",
			wantErrIs: ErrAmountOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.currency.ParseMajorToMinor(tt.input)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCurrency_ParseMajorToMinor_And_FormatMinorToMajor_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		currency Currency
		input    string
		wantFmt  string
	}{
		{
			name:     "minor unit 0",
			currency: Currency{Code: "JPY", MinorUnit: 0},
			input:    "123",
			wantFmt:  "123",
		},
		{
			name:     "minor unit 1",
			currency: Currency{Code: "X1", MinorUnit: 1},
			input:    "12.3",
			wantFmt:  "12.3",
		},
		{
			name:     "minor unit 2 padded from one decimal",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "1.2",
			wantFmt:  "1.20",
		},
		{
			name:     "minor unit 2 exact",
			currency: Currency{Code: "CNY", MinorUnit: 2},
			input:    "-1.23",
			wantFmt:  "-1.23",
		},
		{
			name:     "minor unit 3 exact",
			currency: Currency{Code: "KWD", MinorUnit: 3},
			input:    "1.234",
			wantFmt:  "1.234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minor, err := tt.currency.ParseMajorToMinor(tt.input)
			if err != nil {
				t.Fatalf("parse unexpected error: %v", err)
			}

			got, err := tt.currency.FormatMinorToMajor(minor)
			if err != nil {
				t.Fatalf("format unexpected error: %v", err)
			}

			if got != tt.wantFmt {
				t.Fatalf("got %q, want %q", got, tt.wantFmt)
			}
		})
	}
}
