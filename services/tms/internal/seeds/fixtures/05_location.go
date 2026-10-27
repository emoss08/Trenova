package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/internal/seeds/reference"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type LocationSeed struct {
}

var Location = Seeds.Add(seed.Seed{
	Name:        "location",
	Kind:        seed.Fixture,
	Description: "Creates location data for development",
	After:       []*seed.Seed{LocationCategory, reference.USStates},
	Run:         (&LocationSeed{}).Run,
})

type locationDef struct {
	code         string
	name         string
	description  string
	addressLine1 string
	city         string
	postalCode   string
	stateAbbr    string
	catName      string
	// lat/lng place the location on a map. Without them nothing in the seed
	// plots, so map charts and the shipment map have no data to demonstrate.
	lat float64
	lng float64
}

func (s *LocationSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	lc := locationcategorydb.LocationCategories
	categories, err := lc.AllTenants().Where(
		lc.OrganizationID.Eq(org.ID),
		lc.BusinessUnitID.Eq(org.BusinessUnitID),
	).All(ctx, db)
	if err != nil {
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
			lat:          34.0522,
			lng:          -118.2437,
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
			lat:          32.7767,
			lng:          -96.7970,
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
			lat:          41.8781,
			lng:          -87.6298,
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
			lat:          33.4484,
			lng:          -112.0740,
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
			lat:          39.7392,
			lng:          -104.9903,
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
			continue
		}

		state, stateErr := stateByAbbreviation(ctx, db, d.stateAbbr)
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
			GeofenceType:       geofence.TypeAuto,
		}

		if d.lat != 0 && d.lng != 0 {
			lat, lng := d.lat, d.lng
			l.Latitude = &lat
			l.Longitude = &lng
			l.IsGeocoded = true
		}

		if err = locationdb.Locations.Create(ctx, db, l); err != nil {
			return fmt.Errorf("insert location %s: %w", d.name, err)
		}

	}

	return nil
}
