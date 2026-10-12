package webhooks

import "errors"

var (
	ErrWebhookIDRequired      = errors.New("webhook id is required")
	ErrWebhookNameRequired    = errors.New("webhook name is required")
	ErrWebhookNameTooLong     = errors.New("webhook name must be at most 255 characters")
	ErrWebhookURLRequired     = errors.New("webhook url is required")
	ErrWebhookURLTooLong      = errors.New("webhook url must be at most 2047 characters")
	ErrWebhookURLInvalid      = errors.New("webhook url must be an absolute http or https URL")
	ErrWebhookVersionInvalid  = errors.New("webhook version is invalid")
	ErrWebhookEventInvalid    = errors.New("webhook event type is invalid")
	ErrCustomHeadersTooMany   = errors.New("webhook custom headers must contain at most 5 entries")
	ErrCustomHeaderInvalid    = errors.New("webhook custom header key and value are required")
	ErrCustomHeaderKeyTooLong = errors.New(
		"webhook custom header key must be at most 100 characters",
	)
	ErrListLimitInvalid = errors.New("webhooks limit must be between 1 and 512")
)
