package services

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type AuthService interface {
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	Logout(ctx context.Context) error
	GetSession(ctx context.Context)
}

type CheckEmailRequest struct {
	EmailAddress string `json:"emailAddress"`
}

func (r *CheckEmailRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(
			&r.EmailAddress,
			validation.Required.Error("Email address is required"),
			is.EmailFormat.Error("Invalid email format. Please try again"),
		),
	)
}
