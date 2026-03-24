package webhooks

import "errors"

var (
	ErrWebhookIDRequired   = errors.New("webhook id is required")
	ErrWebhookNameRequired = errors.New("webhook name is required")
	ErrWebhookURLRequired  = errors.New("webhook url is required")
	ErrListLimitInvalid    = errors.New("webhooks limit must be between 1 and 512")
)
