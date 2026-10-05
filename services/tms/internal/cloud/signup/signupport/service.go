package signupport

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

type CloudSignupRequest struct {
	Name           string `json:"name"`
	EmailAddress   string `json:"emailAddress"`
	Password       string `json:"password"`
	CompanyName    string `json:"companyName"`
	AcceptTerms    bool   `json:"acceptTerms"`
	TurnstileToken string `json:"turnstileToken"`
	Website        string `json:"website"`
}

type CloudSignupResendRequest struct {
	EmailAddress   string `json:"emailAddress"`
	TurnstileToken string `json:"turnstileToken"`
}

type CloudSignupVerifyRequest struct {
	Token string `json:"token"`
}

type CloudSignupAccepted struct {
	Status string `json:"status"`
}

type CloudSignupService interface {
	Enabled() bool
	Signup(ctx context.Context, req *CloudSignupRequest) (*CloudSignupAccepted, error)
	Resend(ctx context.Context, req *CloudSignupResendRequest) (*CloudSignupAccepted, error)
	Verify(ctx context.Context, req *CloudSignupVerifyRequest) (*services.LoginResponse, error)
}
