// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package services

import (
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type LoginRequest struct {
	EmailAddress string `json:"emailAddress"`
	Password     string `json:"password"`
}

func (lr *LoginRequest) Validate() error {
	return validation.ValidateStruct(lr,
		validation.
			Field(
				&lr.EmailAddress,
				validation.Required.Error("Email address is required. Please try again."),
				validation.Length(1, 255).
					Error("Email address must be between 1 and 255 characters. Please try again."),
			),
		validation.
			Field(&lr.Password,
				validation.Required.Error("Password is required. Please try again."),
			),
	)
}

type LoginResponse struct {
	User      *user.User `json:"user"`
	ExpiresAt int64      `json:"expiresAt"`
	SessionID string     `json:"sessionId"`
}

type Session struct {
	UserID    pulid.ID `json:"userId"`
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
	ExpiresAt int64    `json:"expiresAt"`
}
