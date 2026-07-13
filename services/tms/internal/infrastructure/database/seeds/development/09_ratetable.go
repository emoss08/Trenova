package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type RateTableSeed struct {
	seedhelpers.BaseSeed
}

func NewRateTableSeed() *RateTableSeed {
	seed := &RateTableSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"RateTable",
		"1.0.0",
		"Creates sample rate lookup tables for development",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedTestOrganizations)

	return seed
}

type rateTableEntryDef struct {
	matchKey  string
	rangeMin  string
	rangeMax  string
	value     string
	sortOrder int32
}

type rateTableDef struct {
	name        string
	key         string
	description string
	lookupType  ratetable.LookupType
	entries     []rateTableEntryDef
}

func nullDecimalFromString(s string) decimal.NullDecimal {
	if s == "" {
		return decimal.NullDecimal{}
	}

	return decimal.NullDecimal{Decimal: decimal.RequireFromString(s), Valid: true}
}

func (s *RateTableSeed) Run(ctx context.Context, tx bun.Tx) error {
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
				Model((*ratetable.RateTable)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing rate tables: %w", err)
			}

			if count > 0 {
				return nil
			}

			defs := []rateTableDef{
				{
					name:        "Fuel Surcharge",
					key:         "fuel_surcharge",
					description: "Fuel surcharge percentage by national average diesel price",
					lookupType:  ratetable.LookupTypeRange,
					entries: []rateTableEntryDef{
						{rangeMin: "0", rangeMax: "3", value: "0", sortOrder: 0},
						{rangeMin: "3", rangeMax: "3.5", value: "0.12", sortOrder: 1},
						{rangeMin: "3.5", rangeMax: "4", value: "0.18", sortOrder: 2},
						{rangeMin: "4", value: "0.25", sortOrder: 3},
					},
				},
				{
					name:        "Lane Rates",
					key:         "lane_rate",
					description: "Flat lane rates by origin-destination pair",
					lookupType:  ratetable.LookupTypeExact,
					entries: []rateTableEntryDef{
						{matchKey: "ATL-MIA", value: "1450", sortOrder: 0},
						{matchKey: "ATL-JAX", value: "980", sortOrder: 1},
					},
				},
			}

			for _, def := range defs {
				tbl := &ratetable.RateTable{
					ID:             pulid.MustNew("rt_"),
					OrganizationID: org.ID,
					BusinessUnitID: org.BusinessUnitID,
					Name:           def.name,
					Key:            def.key,
					Description:    def.description,
					LookupType:     def.lookupType,
					Active:         true,
				}

				if _, err = tx.NewInsert().Model(tbl).Exec(ctx); err != nil {
					return fmt.Errorf("insert rate table %s: %w", def.key, err)
				}

				entries := make([]*ratetable.RateTableEntry, 0, len(def.entries))
				for _, entryDef := range def.entries {
					entry := &ratetable.RateTableEntry{
						ID:             pulid.MustNew("rte_"),
						OrganizationID: org.ID,
						BusinessUnitID: org.BusinessUnitID,
						RateTableID:    tbl.ID,
						RangeMin:       nullDecimalFromString(entryDef.rangeMin),
						RangeMax:       nullDecimalFromString(entryDef.rangeMax),
						Value:          decimal.RequireFromString(entryDef.value),
						SortOrder:      entryDef.sortOrder,
					}
					if entryDef.matchKey != "" {
						matchKey := entryDef.matchKey
						entry.MatchKey = &matchKey
					}
					entries = append(entries, entry)
				}

				if _, err = tx.NewInsert().Model(&entries).Exec(ctx); err != nil {
					return fmt.Errorf("insert rate table entries %s: %w", def.key, err)
				}

				if err = sc.TrackCreated(ctx, "rate_tables", tbl.ID, s.Name()); err != nil {
					return fmt.Errorf("track rate table: %w", err)
				}
			}

			return nil
		},
	)
}

func (s *RateTableSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *RateTableSeed) CanRollback() bool {
	return true
}
