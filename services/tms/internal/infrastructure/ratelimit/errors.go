package ratelimit

import "errors"

var (
	ErrEmptyKey       = errors.New("ratelimit: request key must not be empty")
	ErrInvalidPolicy  = errors.New("ratelimit: policy rate, burst and period must be positive")
	ErrStoreDenied    = errors.New("ratelimit: store unavailable and failure mode is deny")
	ErrUnknownStore   = errors.New("ratelimit: unknown store")
	ErrMissingClient  = errors.New("ratelimit: redis store requires a redis client")
	ErrScriptResponse = errors.New("ratelimit: unexpected script response")
)
