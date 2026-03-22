package paypal

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

	Sandbox bool
}

func (c *Config) init() (err error) {
	if c.UserAction == "" {
		c.UserAction = "PAY_NOW"
	}

	return
}

type ChannelData struct {
	CaptureId string `json:",omitempty"`
}
