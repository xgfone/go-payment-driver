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

import "fmt"

func ExampleCurrency_FormatMinorToMajor() {
	c := &Currency{
		Name:      "Chinese Yuan",
		Code:      "CNY",
		Symbol:    "¥",
		MinorUnit: 2,
	}

	s, err := c.FormatMinorToMajor(12345)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(s)
	// Output:
	// 123.45
}

func ExampleCurrency_ParseMajorToMinor() {
	c := &Currency{
		Name:      "Chinese Yuan",
		Code:      "CNY",
		Symbol:    "¥",
		MinorUnit: 2,
	}

	minor, err := c.ParseMajorToMinor("123.45")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(minor)
	// Output:
	// 12345
}

func ExampleCurrency_ParseMajorToMinor_leadingDecimalPoint() {
	c := &Currency{
		Name:      "Chinese Yuan",
		Code:      "CNY",
		Symbol:    "¥",
		MinorUnit: 2,
	}

	minor, err := c.ParseMajorToMinor(".5")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(minor)
	// Output:
	// 50
}

func ExampleCurrency_ParseMajorToMinor_negativeAmount() {
	c := &Currency{
		Name:      "Chinese Yuan",
		Code:      "CNY",
		Symbol:    "¥",
		MinorUnit: 2,
	}

	minor, err := c.ParseMajorToMinor("-1.23")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(minor)
	// Output:
	// -123
}

func ExampleCurrency_FormatMinorToMajor_zeroDecimalCurrency() {
	c := &Currency{
		Name:      "Japanese Yen",
		Code:      "JPY",
		Symbol:    "¥",
		MinorUnit: 0,
	}

	s, err := c.FormatMinorToMajor(500)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(s)
	// Output:
	// 500
}

func ExampleCurrency_ParseMajorToMinor_zeroDecimalCurrency() {
	c := &Currency{
		Name:      "Japanese Yen",
		Code:      "JPY",
		Symbol:    "¥",
		MinorUnit: 0,
	}

	minor, err := c.ParseMajorToMinor("500")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(minor)
	// Output:
	// 500
}

func ExampleCurrency_ParseMajorToMinor_threeDecimalCurrency() {
	c := &Currency{
		Name:      "Kuwaiti Dinar",
		Code:      "KWD",
		Symbol:    "KD",
		MinorUnit: 3,
	}

	minor, err := c.ParseMajorToMinor("1.234")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(minor)
	// Output:
	// 1234
}

func ExampleCurrency_ParseMajorToMinor_tooManyDecimalPlaces() {
	c := &Currency{
		Name:      "Chinese Yuan",
		Code:      "CNY",
		Symbol:    "¥",
		MinorUnit: 2,
	}

	_, err := c.ParseMajorToMinor("1.234")
	fmt.Println(err != nil)

	// Output:
	// true
}

func ExampleCurrency_roundTrip() {
	c := &Currency{
		Name:      "Chinese Yuan",
		Code:      "CNY",
		Symbol:    "¥",
		MinorUnit: 2,
	}

	minor, err := c.ParseMajorToMinor("19.9")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	major, err := c.FormatMinorToMajor(minor)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(minor)
	fmt.Println(major)

	// Output:
	// 1990
	// 19.90
}
