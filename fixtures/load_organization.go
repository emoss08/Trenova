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
