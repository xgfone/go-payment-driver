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

func init() {
	// Common 0-decimal currencies
	Register("JPY", 0, "¥", "Japanese Yen")
	Register("KRW", 0, "₩", "South Korean Won")
	Register("VND", 0, "₫", "Vietnamese Dong")

	// Common 2-decimal currencies
	Register("CNY", 2, "¥", "Chinese Yuan")
	Register("USD", 2, "$", "US Dollar")
	Register("EUR", 2, "€", "Euro")
	Register("GBP", 2, "£", "Pound Sterling")
	Register("HKD", 2, "HK$", "Hong Kong Dollar")
	Register("SGD", 2, "S$", "Singapore Dollar")
	Register("AUD", 2, "A$", "Australian Dollar")
	Register("CAD", 2, "C$", "Canadian Dollar")
	Register("NZD", 2, "NZ$", "New Zealand Dollar")
	Register("CHF", 2, "CHF", "Swiss Franc")
	Register("INR", 2, "₹", "Indian Rupee")
	Register("THB", 2, "฿", "Thai Baht")
	Register("MYR", 2, "RM", "Malaysian Ringgit")
	Register("IDR", 2, "Rp", "Indonesian Rupiah")
	Register("PHP", 2, "₱", "Philippine Peso")
	Register("TWD", 2, "NT$", "New Taiwan Dollar")
	Register("AED", 2, "AED", "UAE Dirham")
	Register("SAR", 2, "SAR", "Saudi Riyal")
	Register("BRL", 2, "R$", "Brazilian Real")
	Register("MXN", 2, "$", "Mexican Peso")
	Register("TRY", 2, "₺", "Turkish Lira")
	Register("RUB", 2, "₽", "Russian Ruble")

	// Common 3-decimal currencies
	Register("BHD", 3, "BD", "Bahraini Dinar")
	Register("KWD", 3, "KD", "Kuwaiti Dinar")
}
