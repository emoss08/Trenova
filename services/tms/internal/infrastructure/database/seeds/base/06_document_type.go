package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type DocumentTypeSeed struct {
	seedhelpers.BaseSeed
}

func NewDocumentTypeSeed() *DocumentTypeSeed {
	seed := &DocumentTypeSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DocumentType",
		"1.0.0",
		"Creates default system document types for transportation operations",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

func (s *DocumentTypeSeed) Run(ctx context.Context, tx bun.Tx) error {
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

			exists, err := tenantbootstrap.HasSystemDocumentTypes(ctx, tx, org.ID, org.BusinessUnitID)
			if err != nil {
				return err
			}

			if exists {
				return nil
			}

			created, err := tenantbootstrap.CreateDocumentTypes(ctx, tx, tenantbootstrap.Scope{
				OrganizationID: org.ID,
				BusinessUnitID: org.BusinessUnitID,
				Record:         seedRecorder(sc, s.Name()),
			})
			if err != nil {
				return fmt.Errorf("create system document types: %w", err)
			}

			seedhelpers.LogSuccess(
				"Created system document type fixtures",
				fmt.Sprintf("- Created %d system document types", created),
			)

			return nil
		},
	)
}

func (s *DocumentTypeSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *DocumentTypeSeed) CanRollback() bool {
	return true
}
