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

package fixtures

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

func loadOrganization(ctx context.Context, db *bun.DB, bu *models.BusinessUnit) (*models.Organization, error) {
	state := new(models.UsState)

	err := db.NewSelect().Model(state).Where("abbreviation = ?", "NC").Scan(ctx)
	if err != nil {
		return nil, err
	}

	org := new(models.Organization)

	// check if the organization already exists.
	exists, err := db.NewSelect().Model(org).
		Where("name = ?", "Trenova Logistics").
		Where("scac_code = ?", "TRLS").
		Exists(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		org = &models.Organization{
			Name:           "Trenova Logistics",
			ScacCode:       "TRLS",
			DOTNumber:      "123456",
			BusinessUnitID: bu.ID,
			BusinessUnit:   bu,
			City:           "Charlotte",
			StateID:        state.ID,
			State:          state,
			PostalCode:     "28202",
			Timezone:       "America/New_York",
		}

		_, err = db.NewInsert().Model(org).Exec(ctx)
		if err != nil {
			return nil, err
		}
	}

	err = db.NewSelect().Model(org).
		Where("name = ?", "Trenova Logistics").
		Where("scac_code = ?", "TRLS").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}
