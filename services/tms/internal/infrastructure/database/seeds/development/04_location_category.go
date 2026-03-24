package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type LocationCategorySeed struct {
	seedhelpers.BaseSeed
}

func NewLocationCategorySeed() *LocationCategorySeed {
	seed := &LocationCategorySeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"LocationCategory",
		"1.0.0",
		"Creates location category data for development",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedTestOrganizations)

	return seed
}

type locationCategoryDef struct {
	name         string
	description  string
	catType      locationcategory.Category
	facilityType locationcategory.FacilityType
	color        string
}

func (s *LocationCategorySeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetOrganization("default_org")
			if err != nil {
				org, err = sc.GetDefaultOrganization(ctx)
				if err != nil {
					return fmt.Errorf("get organization: %w", err)
				}
			}

			count, err := tx.NewSelect().
				Model((*locationcategory.LocationCategory)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing location categories: %w", err)
			}

			if count > 0 {
				return nil
			}

			now := timeutils.NowUnix()

			defs := []locationCategoryDef{
				{
					name:        "Main Terminal",
					description: "Primary terminal facility for receiving and distributing goods",
					catType:     locationcategory.CategoryTerminal,
					color:       "#3b82f6",
				},
				{
					name:         "Central Warehouse",
					description:  "Main storage and distribution warehouse",
					catType:      locationcategory.CategoryWarehouse,
					facilityType: locationcategory.FacilityTypeStorageWarehouse,
					color:        "#10b981",
				},
				{
					name:         "Cold Storage Hub",
					description:  "Temperature-controlled storage facility",
					catType:      locationcategory.CategoryWarehouse,
					facilityType: locationcategory.FacilityTypeColdStorage,
					color:        "#0ea5e9",
				},
				{
					name:        "East Distribution Center",
					description: "Regional distribution center for eastern operations",
					catType:     locationcategory.CategoryDistributionCenter,
					color:       "#8b5cf6",
				},
				{
					name:        "Customer Drop Point",
					description: "Standard customer delivery location",
					catType:     locationcategory.CategoryCustomerLocation,
					color:       "#ec4899",
				},
				{
					name:        "Fleet Maintenance Yard",
					description: "Primary maintenance and repair facility",
					catType:     locationcategory.CategoryMaintenanceFacility,
					color:       "#ef4444",
				},
			}

			categories := make([]*locationcategory.LocationCategory, 0, len(defs))
			for _, d := range defs {
				categories = append(categories, &locationcategory.LocationCategory{
					ID:             pulid.MustNew("lc_"),
					BusinessUnitID: org.BusinessUnitID,
					OrganizationID: org.ID,
					Name:           d.name,
					Description:    d.description,
					Type:           d.catType,
					FacilityType:   d.facilityType,
					Color:          d.color,
					CreatedAt:      now,
					UpdatedAt:      now,
				})
			}

			if _, err = tx.NewInsert().Model(&categories).Exec(ctx); err != nil {
				return fmt.Errorf("insert location categories: %w", err)
			}

			return nil
		},
	)
}
