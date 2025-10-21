package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type USStatesSeed struct {
	seedhelpers.BaseSeed
}

func NewUSStatesSeed() *USStatesSeed {
	seed := &USStatesSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"USStates",
		"1.0.0",
		"Creates all US states required for the application",
		[]common.Environment{
			common.EnvProduction,
			common.EnvStaging,
			common.EnvDevelopment,
			common.EnvTest,
		},
	)
	return seed
}

func (s *USStatesSeed) Run(ctx context.Context, db *bun.DB) error {
	var count int
	err := db.NewSelect().
		Model((*usstate.UsState)(nil)).
		ColumnExpr("count(*)").
		Scan(ctx, &count)
	if err != nil {
		return err
	}

	if count > 0 {
		seedhelpers.LogSuccess("US states already exist, skipping")
		return nil
	}

	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			states := []usstate.UsState{
				{
					Name:         "Alabama",
					Abbreviation: "AL",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Alaska",
					Abbreviation: "AK",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Arizona",
					Abbreviation: "AZ",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Arkansas",
					Abbreviation: "AR",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "California",
					Abbreviation: "CA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Colorado",
					Abbreviation: "CO",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Connecticut",
					Abbreviation: "CT",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Delaware",
					Abbreviation: "DE",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "District of Columbia",
					Abbreviation: "DC",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Florida",
					Abbreviation: "FL",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Georgia",
					Abbreviation: "GA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Hawaii",
					Abbreviation: "HI",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Idaho",
					Abbreviation: "ID",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Illinois",
					Abbreviation: "IL",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Indiana",
					Abbreviation: "IN",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Iowa",
					Abbreviation: "IA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Kansas",
					Abbreviation: "KS",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Kentucky",
					Abbreviation: "KY",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Louisiana",
					Abbreviation: "LA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Maine",
					Abbreviation: "ME",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Maryland",
					Abbreviation: "MD",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Massachusetts",
					Abbreviation: "MA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Michigan",
					Abbreviation: "MI",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Minnesota",
					Abbreviation: "MN",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Mississippi",
					Abbreviation: "MS",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Missouri",
					Abbreviation: "MO",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Montana",
					Abbreviation: "MT",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Nebraska",
					Abbreviation: "NE",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Nevada",
					Abbreviation: "NV",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "New Hampshire",
					Abbreviation: "NH",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "New Jersey",
					Abbreviation: "NJ",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "New Mexico",
					Abbreviation: "NM",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "New York",
					Abbreviation: "NY",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "North Carolina",
					Abbreviation: "NC",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "North Dakota",
					Abbreviation: "ND",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Ohio",
					Abbreviation: "OH",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Oklahoma",
					Abbreviation: "OK",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Oregon",
					Abbreviation: "OR",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Pennsylvania",
					Abbreviation: "PA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Rhode Island",
					Abbreviation: "RI",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "South Carolina",
					Abbreviation: "SC",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "South Dakota",
					Abbreviation: "SD",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Tennessee",
					Abbreviation: "TN",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Texas",
					Abbreviation: "TX",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Utah",
					Abbreviation: "UT",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Vermont",
					Abbreviation: "VT",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Virginia",
					Abbreviation: "VA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Washington",
					Abbreviation: "WA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "West Virginia",
					Abbreviation: "WV",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Wisconsin",
					Abbreviation: "WI",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
				{
					Name:         "Wyoming",
					Abbreviation: "WY",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
			}

			if _, err := tx.NewInsert().Model(&states).Exec(ctx); err != nil {
				return err
			}

			seedhelpers.LogSuccess("Created US states", "- 51 states created")
			return nil
		},
	)
}
