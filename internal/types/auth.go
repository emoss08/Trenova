package types

import (
	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type CheckEmailRequest struct {
	EmailAddress string `json:"emailAddress" validate:"required,email"`
}

func (cer CheckEmailRequest) Validate() error {
	return validation.ValidateStruct(&cer,
		validation.Field(&cer.EmailAddress, validation.Required, is.Email),
	)
}

type CheckEmailResponse struct {
	Exists        bool            `json:"exists"`
	AccountStatus property.Status `json:"accountStatus"`
	Message       string          `json:"message"`
}

type LoginRequest struct {
	EmailAddress string `json:"emailAddress" validate:"required"`
	Password     string `json:"password" validate:"required"`
}

func (lr LoginRequest) Validate() error {
	return validation.ValidateStruct(&lr,
		validation.Field(&lr.EmailAddress, validation.Required, is.Email),
		validation.Field(&lr.Password, validation.Required),
	)
}
