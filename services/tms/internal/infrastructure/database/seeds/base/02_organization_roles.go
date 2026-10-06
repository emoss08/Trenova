package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type OrganizationRolesSeed struct {
	seedhelpers.BaseSeed
	registry *permission.Registry
}

func NewOrganizationRolesSeed() *OrganizationRolesSeed {
	seed := &OrganizationRolesSeed{
		registry: permission.NewRegistry(),
	}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"OrganizationRoles",
		"1.0.0",
		"Creates organization admin roles and assigns admin user",
		[]common.Environment{
			common.EnvDevelopment, common.EnvTest,
		},
	)
	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

func (s *OrganizationRolesSeed) Run(ctx context.Context, tx bun.Tx) error {
	var orgs []tenant.Organization
	if err := tx.NewSelect().Model(&orgs).Order("created_at ASC").Scan(ctx); err != nil {
		return fmt.Errorf("get organizations: %w", err)
	}

	if len(orgs) == 0 {
		return fmt.Errorf("no organizations found")
	}

	var adminUser tenant.User
	if err := tx.NewSelect().
		Model(&adminUser).
		Where("username = ?", coreAdminUsername).
		Scan(ctx); err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	now := timeutils.NowUnix()

	for _, org := range orgs {
		adminRole, err := tenantbootstrap.CreateAdminRole(ctx, tx, tenantbootstrap.AdminRoleParams{
			Scope: tenantbootstrap.Scope{
				OrganizationID: org.ID,
				BusinessUnitID: org.BusinessUnitID,
				Now:            now,
			},
			CreatedBy: adminUser.ID,
			Registry:  s.registry,
		})
		if err != nil {
			return fmt.Errorf("create admin role for org %s: %w", org.Name, err)
		}

		if err = tenantbootstrap.AssignRole(ctx, tx, tenantbootstrap.RoleAssignmentParams{
			UserID:         adminUser.ID,
			OrganizationID: org.ID,
			RoleID:         adminRole.ID,
			AssignedBy:     adminUser.ID,
			AssignedAt:     now,
		}); err != nil {
			return fmt.Errorf("assign admin role in org %s: %w", org.Name, err)
		}
	}

	return nil
}
