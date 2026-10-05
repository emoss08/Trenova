package base

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type SystemAccountSeed struct {
	seedhelpers.BaseSeed
}

func NewSystemAccountSeed() *SystemAccountSeed {
	seed := &SystemAccountSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"SystemAccount",
		"1.0.0",
		"Creates system account data",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

func (s *SystemAccountSeed) Run(ctx context.Context, tx bun.Tx) error {
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

			var password string
			if cfg := sc.Config(); cfg != nil {
				password = cfg.System.SystemUserPassword
			}

			_, _, err = tenantbootstrap.EnsureSystemUser(ctx, tx, tenantbootstrap.SystemUserParams{
				Scope: tenantbootstrap.Scope{
					OrganizationID: org.ID,
					BusinessUnitID: org.BusinessUnitID,
					Record:         seedRecorder(sc, s.Name()),
				},
				Password: password,
			})

			return err
		},
	)
}

func (s *SystemAccountSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *SystemAccountSeed) CanRollback() bool {
	return true
}
