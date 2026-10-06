package base

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
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

func (s *TCAAllowlistedTablesSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := defaultOrganization(ctx, sc)
			if err != nil || org == nil {
				return err
			}

			_, err = tenantbootstrap.CreateTCAAllowlist(ctx, tx, tenantbootstrap.Scope{
				OrganizationID: org.ID,
				BusinessUnitID: org.BusinessUnitID,
				Record:         seedRecorder(sc, s.Name()),
			})

			return err
		},
	)
}
