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

package factory

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

type OrganizationFactory struct {
	db *bun.DB
}

func NewOrganizationFactory(db *bun.DB) *OrganizationFactory {
	return &OrganizationFactory{db: db}
}

func (o *OrganizationFactory) MustCreateOrganization(ctx context.Context) (*models.Organization, error) {
	bu, err := NewBusinessUnitFactory(o.db).CreateBusinessUnit(ctx)
	if err != nil {
		return nil, err
	}

	state, err := NewStateFactory(o.db).CreateUSState(ctx)
	if err != nil {
		return nil, err
	}

	org := &models.Organization{
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

	// Check if the organization already exists
	err = o.db.NewSelect().Model(org).Where("name = ?", org.Name).Scan(ctx)
	if err == nil {
		return org, nil
	}

	_, err = o.db.NewInsert().Model(org).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}
