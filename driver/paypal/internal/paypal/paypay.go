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

package paypal

type Link struct {
	Rel    string `json:"rel"`
	Href   string `json:"href"`
	Method string `json:"method"`
}

type Amount struct {
	Currency string `json:"currency_code,omitempty"`
	Amount   string `json:"value,omitempty"`
}

type OrderResponse struct {
	Id            string              `json:"id"`
	Status        string              `json:"status"`
	Payer         OrderPayer          `json:"payer,omitempty"`
	PurchaseUnits []OrderPurchaseUnit `json:"purchase_units,omitempty"`
	Links         []Link              `json:"links,omitempty"`
}

type OrderPayer struct {
	PayerId      string `json:"payer_id,omitempty"`
	EmailAddress string `json:"email_address,omitempty"`
}

type OrderPurchaseUnit struct {
	ReferenceId string        `json:"reference_id,omitempty"`
	CustomId    string        `json:"custom_id,omitempty"`
	Amount      Amount        `json:"amount,omitzero"`
	Payments    OrderPayments `json:"payments,omitempty"`
}

type OrderPayments struct {
	Captures []CaptureResource `json:"captures,omitempty"`
}

type CaptureResource struct {
	Id         string `json:"id"`
	Status     string `json:"status"`
	CustomId   string `json:"custom_id,omitempty"`
	InvoiceId  string `json:"invoice_id,omitempty"`
	Amount     Amount `json:"amount,omitzero"`
	CreateTime string `json:"create_time,omitempty"`
	UpdateTime string `json:"update_time,omitempty"`

	StatusDetails struct {
		Reason string `json:"reason,omitempty"`
	} `json:"status_details,omitempty"`

	SupplementaryData struct {
		RelatedIds struct {
			OrderId string `json:"order_id,omitempty"`
		} `json:"related_ids,omitempty"`
	} `json:"supplementary_data,omitempty"`
}

type RefundResource struct {
	Id          string `json:"id"`
	Status      string `json:"status"`
	CustomId    string `json:"custom_id,omitempty"`
	InvoiceId   string `json:"invoice_id,omitempty"`
	NoteToPayer string `json:"note_to_payer,omitempty"`
	Amount      Amount `json:"amount,omitempty"`
	CreateTime  string `json:"create_time,omitempty"`
	UpdateTime  string `json:"update_time,omitempty"`

	StatusDetails struct {
		Reason string `json:"reason,omitempty"`
	} `json:"status_details,omitempty"`
}
