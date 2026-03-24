package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
			org, err := sc.GetOrganization("default_org")
			if err != nil {
				org, err = sc.GetDefaultOrganization(ctx)
				if err != nil {
					return fmt.Errorf("get default organization: %w", err)
				}
			}

			exists, err := tx.NewSelect().
				Model((*tenant.User)(nil)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Where("usr.current_organization_id = ?", org.ID).
						Where("usr.business_unit_id = ?", org.BusinessUnitID).
						Where("usr.email_address = ?", "system@trenova.app")
				}).
				Exists(ctx)
			if err != nil {
				return fmt.Errorf("check system user exists: %w", err)
			}

			if exists {
				return nil
			}

			cfg := sc.Config()
			if cfg == nil || cfg.System.SystemUserPassword == "" {
				return fmt.Errorf(
					"system user password must be set in config (system.systemUserPassword)",
				)
			}

			_, err = sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
				OrganizationID: org.ID,
				BusinessUnitID: org.BusinessUnitID,
				Name:           "System Account",
				Username:       "system",
				Email:          "system@trenova.app",
				Password:       cfg.System.SystemUserPassword,
				Status:         domaintypes.StatusActive,
				Timezone:       "America/Los_Angeles",
			}, s.Name())

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
