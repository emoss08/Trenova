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
