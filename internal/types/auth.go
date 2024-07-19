// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
