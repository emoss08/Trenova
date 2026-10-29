package masterdatafixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type LocationCategorySeed struct {
}

type locationCategoryDef struct {
	name         string
	description  string
	catType      locationcategory.Category
	facilityType locationcategory.FacilityType
	color        string
}

func (s *LocationCategorySeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
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

	if err = locationcategorydb.LocationCategories.CreateMany(ctx, db, categories); err != nil {
		return fmt.Errorf("insert location categories: %w", err)
	}

	return nil
}
