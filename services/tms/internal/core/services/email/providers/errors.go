package providers

import "errors"

var (
	ErrResendAPIKeyRequired = errors.New("resend API key is required")
	ErrSMTPHostRequired     = errors.New("smtp host is required")
)
