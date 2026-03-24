package domain

import "errors"

var (
	ErrSinkNotFound       = errors.New("sink not found")
	ErrSinkAlreadyExists  = errors.New("sink already registered")
	ErrSinkInitFailed     = errors.New("sink initialization failed")
	ErrSinkProcessFailed  = errors.New("sink processing failed")
	ErrSinkShutdownFailed = errors.New("sink shutdown failed")
	ErrWALConnectionLost  = errors.New("WAL connection lost")
	ErrInvalidEvent       = errors.New("invalid CDC event")
	ErrCircuitOpen        = errors.New("circuit breaker is open")
)
