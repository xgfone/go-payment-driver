package paypal

import "errors"

const Type = "paypal"

type Config struct {
	ClientId     string
	ClientSecret string

	// PayPal webhook id, not secret.
	// Used by /v1/notifications/verify-webhook-signature.
	WebhookId string

	// For PayPal Checkout payer return.
	ReturnUrl string
	CancelUrl string

	// Optional experience fields.
	BrandName  string
	Locale     string
	UserAction string // default: PAY_NOW

	Currencies []string

	Sandbox bool
}

func (c *Config) init() (err error) {
	if c.UserAction == "" {
		c.UserAction = "PAY_NOW"
	}

	if len(c.Currencies) == 0 {
		return errors.New("missing Currencies")
	}

	return
}

type ChannelData struct {
	CaptureId string `json:",omitempty"`
}
