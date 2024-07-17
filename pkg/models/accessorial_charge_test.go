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

package models_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccessorialCharge_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ac := &models.AccessorialCharge{
			Status:         property.StatusActive,
			OrganizationID: uuid.New(),
			BusinessUnitID: uuid.New(),
			Code:           "CODE",
			Method:         "Distance",
			Description:    "Test Accessorial Charge",
		}

		err := ac.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		ac := &models.AccessorialCharge{
			Status:         property.StatusActive,
			OrganizationID: uuid.New(),
			BusinessUnitID: uuid.New(),
		}

		err := ac.Validate()
		require.Error(t, err)
	})
}
