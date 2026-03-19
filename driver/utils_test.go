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

import "testing"

func TestFormatPercentAmount(t *testing.T) {
	if s := FormatPercentAmount(1); s != "0.01" {
		t.Errorf("FormatPercentAmount(1) = %s, want 0.01", s)
	}
	if s := FormatPercentAmount(10); s != "0.10" {
		t.Errorf("FormatPercentAmount(100) = %s, want 0.10", s)
	}
	if s := FormatPercentAmount(100); s != "1.00" {
		t.Errorf("FormatPercentAmount(100) = %s, want 1.00", s)
	}
	if s := FormatPercentAmount(1234); s != "12.34" {
		t.Errorf("FormatPercentAmount(1234) = %s, want 12.34", s)
	}
}

func TestParsePercentAmount(t *testing.T) {
	if v, err := ParsePercentAmount("0.01"); v != 1 || err != nil {
		t.Errorf("ParsePercentAmount(0.01) = %d, want 1, err: %v", v, err)
	}
	if v, err := ParsePercentAmount("0.1"); v != 1 || err != nil {
		t.Errorf("ParsePercentAmount(0.1) = %d, want 1, err: %v", v, err)
	}
	if v, err := ParsePercentAmount("1.00"); v != 100 || err != nil {
		t.Errorf("ParsePercentAmount(1.00) = %d, want 100, err: %v", v, err)
	}
	if v, err := ParsePercentAmount("12.34"); v != 1234 || err != nil {
		t.Errorf("ParsePercentAmount(12.34) = %d, want 1234, err: %v", v, err)
	}
}
