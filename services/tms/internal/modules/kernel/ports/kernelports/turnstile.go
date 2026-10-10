package kernelports

import (
	"context"
	"errors"
)

const TurnstileActionSignup = "signup"

const TurnstileActionSignupResend = "signup_resend"

var ErrTurnstileRejected = errors.New("turnstile challenge was not passed")

var ErrTurnstileUnavailable = errors.New("turnstile verification is unavailable")

type TurnstileVerification struct {
	Token          string
	RemoteIP       string
	ExpectedAction string
}

type TurnstileVerifier interface {
	Enabled() bool
	Verify(ctx context.Context, req *TurnstileVerification) error
}
