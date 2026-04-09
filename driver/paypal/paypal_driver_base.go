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

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xgfone/go-payment-driver/builder"
	"github.com/xgfone/go-payment-driver/driver"
	"github.com/xgfone/go-toolkit/httpx"
	"github.com/xgfone/go-toolkit/jsonx"
	"github.com/xgfone/go-toolkit/timex"
)

func registerBuilder(scene string, linktype driver.LinkType, newf builder.DriverNewer[Config]) {
	metadata := driver.NewMetadata(Type, scene).WithLinkType(linktype)
	builder.Register(builder.New(newf, metadata))
}

func newDriver(b builder.Builder, c Config) (d *_Driver, err error) {
	if err = c.init(); err != nil {
		return
	}

	d = &_Driver{metadata: b.Metadata(), config: c}
	d.metadata.Currencies = c.Currencies

	if c.Sandbox {
		d.baseurl = "https://api-m.sandbox.paypal.com"
	} else {
		d.baseurl = "https://api-m.paypal.com"
	}

	auth := c.ClientId + ":" + c.ClientSecret
	d.auth = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	return
}

type _Driver struct {
	metadata driver.Metadata
	config   Config

	baseurl string

	auth  string
	mutex sync.Mutex
	token string
	etime time.Time
}

func (d *_Driver) Metadata() driver.Metadata {
	return d.metadata
}

func (d *_Driver) getAccessToken(ctx context.Context) (string, error) {
	d.mutex.Lock()
	if d.token != "" && timex.Now().Before(d.etime.Add(-30*time.Second)) {
		tok := d.token
		d.mutex.Unlock()
		return tok, nil
	}
	d.mutex.Unlock()

	url := d.baseurl + "/v1/oauth2/token"
	body := strings.NewReader("grant_type=client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", d.auth)

	resp, err := httpx.GetClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("paypal oauth failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", errors.New("paypal oauth failed: empty access_token")
	}

	d.mutex.Lock()
	d.token = out.AccessToken
	d.etime = timex.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	d.mutex.Unlock()

	return out.AccessToken, nil
}

func (d *_Driver) doJSON(ctx context.Context, method, path string, headers map[string]string, in, out any) (
	statuscode int, raw []byte, err error) {
	var body io.Reader
	if in != nil {
		b, err := jsonx.MarshalBytes(in)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(b)
	}

	token, err := d.getAccessToken(ctx)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, d.baseurl+path, body)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := httpx.GetClient().Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	raw, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, raw, errors.New(string(raw))
	}

	if out != nil && len(raw) > 0 {
		if err := jsonx.UnmarshalBytes(raw, out); err != nil {
			return resp.StatusCode, raw, err
		}
	}

	return resp.StatusCode, raw, nil
}

type webhookEvent struct {
	Id        string          `json:"id"`
	EventType string          `json:"event_type"`
	Resource  json.RawMessage `json:"resource"`
}

func (d *Driver) parseAndVerifyWebhook(ctx context.Context, req *http.Request) (event webhookEvent, raw json.RawMessage, err error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return event, raw, err
	}

	if err = d.verifyWebhookSignature(ctx, req.Header, body); err != nil {
		return event, raw, err
	}

	if err = json.Unmarshal(body, &event); err != nil {
		return event, raw, err
	}

	return event, event.Resource, nil
}

func (d *Driver) verifyWebhookSignature(ctx context.Context, h http.Header, body []byte) error {
	payload := map[string]any{
		"auth_algo":         h.Get("Paypal-Auth-Algo"),
		"cert_url":          h.Get("Paypal-Cert-Url"),
		"transmission_id":   h.Get("Paypal-Transmission-Id"),
		"transmission_sig":  h.Get("Paypal-Transmission-Sig"),
		"transmission_time": h.Get("Paypal-Transmission-Time"),
		"webhook_id":        d.config.WebhookId,
		"webhook_event":     json.RawMessage(body),
	}

	var resp struct {
		VerificationStatus string `json:"verification_status"`
	}
	_, _, err := d.doJSON(ctx, http.MethodPost, "/v1/notifications/verify-webhook-signature", nil, payload, &resp)
	if err != nil {
		return err
	}

	if !strings.EqualFold(resp.VerificationStatus, "SUCCESS") {
		return driver.ErrBadRequest.WithReasonf("invalid paypal webhook signature: %s", resp.VerificationStatus)
	}

	return nil
}
