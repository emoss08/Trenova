package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// LocationCategorySeed Creates location category data
type LocationCategorySeed struct {
	seedhelpers.BaseSeed
}

// NewLocationCategorySeed creates a new location_category seed
func NewLocationCategorySeed() *LocationCategorySeed {
	seed := &LocationCategorySeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"LocationCategory",
		"1.0.0",
		"Creates location category data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	// Development seeds typically depend on base seeds
	seed.SetDependencies("USStates", "AdminAccount", "Permissions", "HazmatExpiration")

	return seed
}

// Run executes the seed
func (s *LocationCategorySeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			var count int
			err := db.NewSelect().
				Model((*location.LocationCategory)(nil)).
				ColumnExpr("count(*)").
				Scan(ctx, &count)
			if err != nil {
				return err
			}

			if count > 0 {
				seedhelpers.LogSuccess("Location categories already exist, skipping")
				return nil
			}

			// Get default organization and business unit
			defaultOrg, err := seedCtx.GetDefaultOrganization()
			if err != nil {
				return fmt.Errorf("get default organization: %w", err)
			}

			defaultBU, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return fmt.Errorf("get default business unit: %w", err)
			}

			locationCategories := []*location.LocationCategory{
				{
					BusinessUnitID:      defaultBU.ID,
					OrganizationID:      defaultOrg.ID,
					Name:                "Company Terminal",
					Description:         "Primary fleet operations hub with secure parking and driver facilities",
					Type:                location.CategoryTerminal,
					FacilityType:        location.FacilityTypeCrossDock,
					Color:               "#2E7D32",
					HasSecureParking:    true,
					RequiresAppointment: true,
					AllowsOvernight:     true,
					HasRestroom:         true,
				},
				{
					BusinessUnitID:      defaultBU.ID,
					OrganizationID:      defaultOrg.ID,
					Name:                "Customer Warehouse",
					Description:         "Customer warehouse for storing and distributing goods",
					Type:                location.CategoryWarehouse,
					FacilityType:        location.FacilityTypeStorageWarehouse,
					Color:               "#1976D2",
					HasSecureParking:    false,
					RequiresAppointment: true,
					AllowsOvernight:     false,
					HasRestroom:         true,
				},
				{
					BusinessUnitID:      defaultBU.ID,
					OrganizationID:      defaultOrg.ID,
					Name:                "Regional Distribution Center",
					Description:         "Large-scale distribution facility with full service capabilities",
					Type:                location.CategoryDistributionCenter,
					FacilityType:        location.FacilityTypeCrossDock,
					Color:               "#7B1FA2",
					HasSecureParking:    true,
					RequiresAppointment: true,
					AllowsOvernight:     true,
					HasRestroom:         true,
				},
				{
					BusinessUnitID:      defaultBU.ID,
					OrganizationID:      defaultOrg.ID,
					Name:                "Cold Storage Facility",
					Description:         "Temperature-controlled storage and distribution",
					Type:                location.CategoryWarehouse,
					FacilityType:        location.FacilityTypeColdStorage,
					Color:               "#0097A7",
					HasSecureParking:    true,
					RequiresAppointment: true,
					AllowsOvernight:     true,
					HasRestroom:         true,
				},
				{
					BusinessUnitID:      defaultBU.ID,
					OrganizationID:      defaultOrg.ID,
					Name:                "Truck Stop",
					Description:         "Driver services and rest facility",
					Type:                location.CategoryTruckStop,
					Color:               "#FF5722",
					HasSecureParking:    true,
					RequiresAppointment: true,
					AllowsOvernight:     false,
					HasRestroom:         true,
				},
				{
					BusinessUnitID:      defaultBU.ID,
					OrganizationID:      defaultOrg.ID,
					Name:                "Hazmat Facility",
					Description:         "Specialized facility for hazardous materials handling",
					Type:                location.CategoryWarehouse,
					FacilityType:        location.FacilityTypeHazmatFacility,
					Color:               "#F44336",
					HasSecureParking:    true,
					RequiresAppointment: true,
					AllowsOvernight:     false,
					HasRestroom:         true,
				},
			}

			if _, err := tx.NewInsert().Model(&locationCategories).Exec(ctx); err != nil {
				return fmt.Errorf("failed to bulk insert location categories: %w", err)
			}

			seedhelpers.LogSuccess("Created location_category fixtures",
				"- 5 location categories created",
			)

			return nil
		},
	)
}
