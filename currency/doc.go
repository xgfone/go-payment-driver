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

// Package currency provides helpers for converting monetary amounts
// between major units and minor units for a limited set of decimal
// precisions commonly used by payment systems.
//
// # Design
//
// This package uses the following model:
//
//   - Major unit: the user-facing amount representation, such as
//     "12.34" CNY, "9.99" USD, or "500" JPY.
//   - Minor unit: the smallest currency unit represented as an integer,
//     such as fen for CNY, cent for USD, or yen itself for JPY.
//
// Examples:
//
//   - CNY with MinorUnit=2:
//     "12.34" major units <=> 1234 minor units
//   - USD with MinorUnit=2:
//     "9.99" major units <=> 999 minor units
//   - JPY with MinorUnit=0:
//     "500" major units <=> 500 minor units
//   - KWD with MinorUnit=3:
//     "1.234" major units <=> 1234 minor units
//
// # Supported MinorUnit values
//
// For simplicity, this package supports only MinorUnit values in the
// range 0..3.
//
//   - 0: zero-decimal currencies
//   - 1: one decimal place
//   - 2: two decimal places
//   - 3: three decimal places
//
// Any other MinorUnit is considered unsupported and results in an error.
//
// # Formatting
//
// Currency.FormatMinorToMajor converts an integer minor-unit amount into
// a normalized major-unit decimal string.
//
// The formatted result always uses exactly MinorUnit fractional digits
// when MinorUnit > 0.
//
// Examples:
//
//   - MinorUnit=2, 123  => "1.23"
//   - MinorUnit=2, 120  => "1.20"
//   - MinorUnit=2, -123 => "-1.23"
//   - MinorUnit=0, 500  => "500"
//
// # Parsing
//
// Currency.ParseMajorToMinor converts a major-unit decimal string into
// an integer minor-unit amount.
//
// Parsing is strict:
//
//   - Leading and trailing spaces are ignored.
//   - A leading "+" or "-" sign is allowed.
//   - Inputs like ".5" and "1." are allowed.
//   - Fractional digits must not exceed MinorUnit.
//   - If fractional digits are fewer than MinorUnit, they are padded
//     on the right.
//   - Invalid formats and out-of-range values return an error.
//
// Examples:
//
//   - MinorUnit=2, "1"     => 100
//   - MinorUnit=2, "1.2"   => 120
//   - MinorUnit=2, "1.23"  => 123
//   - MinorUnit=2, ".5"    => 50
//   - MinorUnit=2, "-1.23" => -123
//   - MinorUnit=2, "1.234" => error
//
// # Zero-decimal currencies
//
// When MinorUnit=0, only integer major-unit inputs are accepted.
//
// Examples:
//
//   - "500"  => 500
//   - "500." => 500
//   - "500.0" => error
//
// # Error handling
//
// This package is intended to behave like a reusable library:
//
//   - It does not panic on normal input errors.
//   - Unsupported MinorUnit values return an error.
//   - Invalid amount formats return an error.
//   - Values outside int64 range return an error.
//
// # Naming
//
// The package uses the term MinorUnit to represent the decimal precision
// rule for a currency, for example:
//
//   - CNY => MinorUnit=2
//   - USD => MinorUnit=2
//   - JPY => MinorUnit=0
//   - KWD => MinorUnit=3
package currency
