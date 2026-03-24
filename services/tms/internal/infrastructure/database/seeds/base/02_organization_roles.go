package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
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
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
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
		Where("username = ?", "admin").
		Scan(ctx); err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	now := timeutils.NowUnix()

	for _, org := range orgs {
		adminRole := &permission.Role{
			ID:             pulid.MustNew("rol_"),
			BusinessUnitID: org.BusinessUnitID,
			OrganizationID: org.ID,
			Name:           "Organization Administrator",
			Description:    "Full access to all resources within the organization",
			MaxSensitivity: permission.SensitivityConfidential,
			IsSystem:       true,
			IsOrgAdmin:     true,
			CreatedBy:      adminUser.ID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if _, err := tx.NewInsert().Model(adminRole).Exec(ctx); err != nil {
			return fmt.Errorf("create admin role for org %s: %w", org.Name, err)
		}

		if err := s.createAdminPermissions(ctx, tx, adminRole.ID, now); err != nil {
			return fmt.Errorf("create admin permissions for org %s: %w", org.Name, err)
		}

		assignment := &permission.UserRoleAssignment{
			ID:             pulid.MustNew("ura_"),
			UserID:         adminUser.ID,
			OrganizationID: org.ID,
			RoleID:         adminRole.ID,
			AssignedBy:     adminUser.ID,
			AssignedAt:     now,
		}

		if _, err := tx.NewInsert().Model(assignment).Exec(ctx); err != nil {
			return fmt.Errorf("assign admin role for org %s: %w", org.Name, err)
		}
	}

	return nil
}

func (s *OrganizationRolesSeed) createAdminPermissions(
	ctx context.Context,
	tx bun.Tx,
	roleID pulid.ID,
	now int64,
) error {
	resources := s.registry.All()

	for _, res := range resources {
		ops := make([]permission.Operation, 0, len(res.Operations))
		for _, op := range res.Operations {
			ops = append(ops, op.Operation)
		}

		perm := &permission.ResourcePermission{
			ID:         pulid.MustNew("rp_"),
			RoleID:     roleID,
			Resource:   res.Resource,
			Operations: ops,
			DataScope:  permission.DataScopeOrganization,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if _, err := tx.NewInsert().Model(perm).Exec(ctx); err != nil {
			return fmt.Errorf("create permission for resource %s: %w", res.Resource, err)
		}
	}

	return nil
}
