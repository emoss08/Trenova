package services

import (
	"context"
	"errors"
)

const (
	TurnstileActionSignup       = "signup"
	TurnstileActionSignupResend = "signup_resend"
)

var (
	ErrTurnstileRejected    = errors.New("turnstile challenge was not passed")
	ErrTurnstileUnavailable = errors.New("turnstile verification is unavailable")
)

type TurnstileVerification struct {
	Token          string
	RemoteIP       string
	ExpectedAction string
	IdempotencyKey string
}

type TurnstileVerifier interface {
	Enabled() bool
	Verify(ctx context.Context, req *TurnstileVerification) error
}
