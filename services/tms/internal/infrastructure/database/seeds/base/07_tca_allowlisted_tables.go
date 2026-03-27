package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type TCAAllowlistedTablesSeed struct {
	seedhelpers.BaseSeed
}

func NewTCAAllowlistedTablesSeed() *TCAAllowlistedTablesSeed {
	seed := &TCAAllowlistedTablesSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"TCAAllowlistedTables",
		"1.0.0",
		"Seeds the default set of tables eligible for Table Change Alert subscriptions",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

type allowlistEntry struct {
	tableName   string
	displayName string
}

var defaultAllowlistedTables = []allowlistEntry{
	{"shipments", "Shipments"},
	{"customers", "Customers"},
	{"workers", "Workers"},
	{"tractors", "Tractors"},
	{"trailers", "Trailers"},
}

func (s *TCAAllowlistedTablesSeed) Run(ctx context.Context, tx bun.Tx) error {
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
					return fmt.Errorf("get default organization: %w", err)
				}
			}

			count, err := tx.NewSelect().
				Model((*tablechangealert.TCAAllowlistedTable)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing allowlisted tables: %w", err)
			}

			if count > 0 {
				return nil
			}

			entities := make([]*tablechangealert.TCAAllowlistedTable, 0, len(defaultAllowlistedTables))
			for _, entry := range defaultAllowlistedTables {
				entities = append(entities, &tablechangealert.TCAAllowlistedTable{
					ID:             pulid.MustNew("tcaw_"),
					OrganizationID: org.ID,
					BusinessUnitID: org.BusinessUnitID,
					TableName:      entry.tableName,
					DisplayName:    entry.displayName,
					Enabled:        true,
				})
			}

			if _, err = tx.NewInsert().Model(&entities).Exec(ctx); err != nil {
				return fmt.Errorf("insert allowlisted tables: %w", err)
			}

			return nil
		},
	)
}
