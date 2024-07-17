// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
