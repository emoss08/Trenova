// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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
