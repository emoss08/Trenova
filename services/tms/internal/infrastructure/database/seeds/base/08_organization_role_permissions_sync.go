package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const organizationAdministratorRoleName = "Organization Administrator"

type OrganizationRolePermissionsSyncSeed struct {
	seedhelpers.BaseSeed
	registry *permission.Registry
}

func NewOrganizationRolePermissionsSyncSeed() *OrganizationRolePermissionsSyncSeed {
	seed := &OrganizationRolePermissionsSyncSeed{
		registry: permission.NewRegistry(),
	}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"OrganizationRolePermissionsSync",
		"1.0.0",
		"Synchronizes organization administrator role permissions with registered resources",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)
	seed.SetDependencies(seedhelpers.SeedOrganizationRoles)

	return seed
}

func (s *OrganizationRolePermissionsSyncSeed) Run(ctx context.Context, tx bun.Tx) error {
	var roles []permission.Role
	if err := tx.NewSelect().
		Model(&roles).
		Column("id").
		Where("is_system = ?", true).
		Where("name = ?", organizationAdministratorRoleName).
		Scan(ctx); err != nil {
		return fmt.Errorf("get organization administrator roles: %w", err)
	}

	definitions := s.registry.All()
	for _, role := range roles {
		if err := s.syncRolePermissions(ctx, tx, role.ID, definitions); err != nil {
			return fmt.Errorf("sync permissions for role %s: %w", role.ID, err)
		}
	}

	return nil
}

func (s *OrganizationRolePermissionsSyncSeed) syncRolePermissions(
	ctx context.Context,
	tx bun.Tx,
	roleID pulid.ID,
	definitions []*permission.ResourceDefinition,
) error {
	existing, err := s.existingPermissions(ctx, tx, roleID)
	if err != nil {
		return err
	}

	missingPermissions := make([]*permission.ResourcePermission, 0, len(definitions))
	for _, def := range definitions {
		requiredOps := operationsForDefinition(def)
		existingPermission, exists := existing[def.Resource]
		if exists {
			mergedOps, changed := mergeOperations(existingPermission.Operations, requiredOps)
			if !changed {
				continue
			}

			existingPermission.Operations = mergedOps
			if err := s.updatePermissionOperations(ctx, tx, existingPermission); err != nil {
				return err
			}
			continue
		}

		missingPermissions = append(missingPermissions, &permission.ResourcePermission{
			RoleID:     roleID,
			Resource:   def.Resource,
			Operations: requiredOps,
			DataScope:  permission.DataScopeOrganization,
		})
	}

	if len(missingPermissions) == 0 {
		return nil
	}

	if _, err = tx.NewInsert().Model(&missingPermissions).Exec(ctx); err != nil {
		return fmt.Errorf("insert missing resource permissions: %w", err)
	}

	return nil
}

func (s *OrganizationRolePermissionsSyncSeed) existingPermissions(
	ctx context.Context,
	tx bun.Tx,
	roleID pulid.ID,
) (map[string]*permission.ResourcePermission, error) {
	var permissions []permission.ResourcePermission
	if err := tx.NewSelect().
		Model(&permissions).
		Column("id", "resource", "operations").
		Where("role_id = ?", roleID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get existing resource permissions: %w", err)
	}

	existing := make(map[string]*permission.ResourcePermission, len(permissions))
	for i := range permissions {
		existing[permissions[i].Resource] = &permissions[i]
	}

	return existing, nil
}

func (s *OrganizationRolePermissionsSyncSeed) updatePermissionOperations(
	ctx context.Context,
	tx bun.Tx,
	perm *permission.ResourcePermission,
) error {
	if _, err := tx.NewUpdate().
		Model(perm).
		Column("operations").
		WherePK().
		Exec(ctx); err != nil {
		return fmt.Errorf("update operations for resource %s: %w", perm.Resource, err)
	}

	return nil
}

func operationsForDefinition(def *permission.ResourceDefinition) []permission.Operation {
	ops := make([]permission.Operation, len(def.Operations))
	for i, op := range def.Operations {
		ops[i] = op.Operation
	}

	return ops
}

func mergeOperations(
	current []permission.Operation,
	required []permission.Operation,
) ([]permission.Operation, bool) {
	seen := make(map[permission.Operation]bool, len(current)+len(required))
	merged := make([]permission.Operation, 0, len(current)+len(required))
	for _, op := range current {
		if seen[op] {
			continue
		}
		seen[op] = true
		merged = append(merged, op)
	}

	changed := false
	for _, op := range required {
		if seen[op] {
			continue
		}
		seen[op] = true
		merged = append(merged, op)
		changed = true
	}

	return merged, changed
}
