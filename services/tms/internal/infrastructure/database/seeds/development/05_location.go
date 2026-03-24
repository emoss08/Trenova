package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type LocationSeed struct {
	seedhelpers.BaseSeed
}

func NewLocationSeed() *LocationSeed {
	seed := &LocationSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"Location",
		"1.0.0",
		"Creates location data for development",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedLocationCategory, seedhelpers.SeedUSStates)

	return seed
}

type locationDef struct {
	code         string
	name         string
	description  string
	addressLine1 string
	city         string
	postalCode   string
	stateAbbr    string
	catName      string
}

func (s *LocationSeed) Run(ctx context.Context, tx bun.Tx) error {
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
				Model((*location.Location)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing locations: %w", err)
			}

			if count > 0 {
				return nil
			}

			var categories []locationcategory.LocationCategory
			if err = tx.NewSelect().
				Model(&categories).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Scan(ctx); err != nil {
				return fmt.Errorf("get location categories: %w", err)
			}

			catByName := make(map[string]pulid.ID, len(categories))
			for _, c := range categories {
				catByName[c.Name] = c.ID
			}

			defs := []locationDef{
				{
					code:         "TERM-LA",
					name:         "Los Angeles Terminal",
					description:  "Primary terminal in Los Angeles metro area",
					addressLine1: "1200 E 7th St",
					city:         "Los Angeles",
					postalCode:   "90021",
					stateAbbr:    "CA",
					catName:      "Main Terminal",
				},
				{
					code:         "WH-DAL",
					name:         "Dallas Warehouse",
					description:  "Central warehouse serving the Dallas-Fort Worth region",
					addressLine1: "4500 Singleton Blvd",
					city:         "Dallas",
					postalCode:   "75212",
					stateAbbr:    "TX",
					catName:      "Central Warehouse",
				},
				{
					code:         "DC-CHI",
					name:         "Chicago Distribution Center",
					description:  "East distribution center for Midwest operations",
					addressLine1: "3901 S Ashland Ave",
					city:         "Chicago",
					postalCode:   "60609",
					stateAbbr:    "IL",
					catName:      "East Distribution Center",
				},
				{
					code:         "COLD-PHX",
					name:         "Phoenix Cold Storage",
					description:  "Temperature-controlled storage in Phoenix",
					addressLine1: "2020 S 7th Ave",
					city:         "Phoenix",
					postalCode:   "85003",
					stateAbbr:    "AZ",
					catName:      "Cold Storage Hub",
				},
				{
					code:         "CUST-DEN",
					name:         "Denver Drop Point",
					description:  "Customer delivery location in Denver",
					addressLine1: "1701 Wynkoop St",
					city:         "Denver",
					postalCode:   "80202",
					stateAbbr:    "CO",
					catName:      "Customer Drop Point",
				},
				{
					code:         "MAINT-01",
					name:         "Fleet Maintenance Yard",
					description:  "Primary maintenance facility",
					addressLine1: "8900 NW 33rd St",
					city:         "Miami",
					postalCode:   "33172",
					stateAbbr:    "FL",
					catName:      "Fleet Maintenance Yard",
				},
			}

			now := timeutils.NowUnix()

			for _, d := range defs {
				catID, ok := catByName[d.catName]
				if !ok {
					sc.Logger().
						Info("Skipping location %s: category %s not found", d.name, d.catName)
					continue
				}

				state, stateErr := sc.GetState(ctx, d.stateAbbr)
				if stateErr != nil {
					return fmt.Errorf("get state %s: %w", d.stateAbbr, stateErr)
				}

				locID := pulid.MustNew("loc_")

				l := &location.Location{
					ID:                 locID,
					BusinessUnitID:     org.BusinessUnitID,
					OrganizationID:     org.ID,
					LocationCategoryID: catID,
					StateID:            state.ID,
					Status:             domaintypes.StatusActive,
					Code:               d.code,
					Name:               d.name,
					Description:        d.description,
					AddressLine1:       d.addressLine1,
					City:               d.city,
					PostalCode:         d.postalCode,
					CreatedAt:          now,
					UpdatedAt:          now,
				}

				if _, err = tx.NewInsert().Model(l).Exec(ctx); err != nil {
					return fmt.Errorf("insert location %s: %w", d.name, err)
				}

				if err = sc.TrackCreated(ctx, "locations", locID, s.Name()); err != nil {
					return fmt.Errorf("track location: %w", err)
				}

				sc.Logger().Info("Created location %s (%s)", d.name, d.code)
			}

			return nil
		},
	)
}

func (s *LocationSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *LocationSeed) CanRollback() bool {
	return true
}
