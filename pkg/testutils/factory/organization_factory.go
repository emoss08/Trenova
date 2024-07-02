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

	_, err = o.db.NewInsert().Model(org).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}
