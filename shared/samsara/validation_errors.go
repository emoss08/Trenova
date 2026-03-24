package samsara

import "errors"

var (
	ErrTokenRequired            = errors.New("samsara token is required")
	ErrRetryMaxAttemptsTooHigh  = errors.New("samsara retry maxAttempts must be <= 10")
	ErrRetryBackoffOrderInvalid = errors.New(
		"samsara retry initialBackoff must be <= maxBackoff",
	)
	ErrBaseURLOverrideEmpty           = errors.New("samsara base URL override cannot be empty")
	ErrRetryMaxAttemptsInvalid        = errors.New("samsara retry maxAttempts must be > 0")
	ErrTimeoutInvalid                 = errors.New("samsara timeout must be > 0")
	ErrRetryInitialBackoffInvalid     = errors.New("samsara retry initialBackoff must be > 0")
	ErrRetryMaxBackoffInvalid         = errors.New("samsara retry maxBackoff must be > 0")
	ErrCustomHTTPClientTransportEmpty = errors.New(
		"samsara custom http client must have a transport configured",
	)
)
