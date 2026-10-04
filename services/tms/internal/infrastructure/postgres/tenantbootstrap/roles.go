package tenantbootstrap

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	AdministratorRoleName        = "Organization Administrator"
	administratorRoleDescription = "Full access to all resources within the organization"
)

type AdminRoleParams struct {
	Scope     Scope
	CreatedBy pulid.ID
	Registry  *permission.Registry
}

func CreateAdminRole(
	ctx context.Context,
	db bun.IDB,
	params AdminRoleParams,
) (*permission.Role, error) {
	now := params.Scope.now()
	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		BusinessUnitID: params.Scope.BusinessUnitID,
		OrganizationID: params.Scope.OrganizationID,
		Name:           AdministratorRoleName,
		Description:    administratorRoleDescription,
		MaxSensitivity: permission.SensitivityConfidential,
		IsSystem:       true,
		CreatedBy:      params.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if _, err := db.NewInsert().Model(role).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create admin role: %w", err)
	}
	if err := params.Scope.record(ctx, "roles", role.ID); err != nil {
		return nil, err
	}

	registry := params.Registry
	if registry == nil {
		registry = permission.NewRegistry()
	}

	if err := createAdminPermissions(ctx, db, params.Scope, role.ID, registry); err != nil {
		return nil, err
	}

	return role, nil
}

func createAdminPermissions(
	ctx context.Context,
	db bun.IDB,
	scope Scope,
	roleID pulid.ID,
	registry *permission.Registry,
) error {
	now := scope.now()

	for _, res := range registry.All() {
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

		if _, err := db.NewInsert().Model(perm).Exec(ctx); err != nil {
			return fmt.Errorf("create permission for resource %s: %w", res.Resource, err)
		}
		if err := scope.record(ctx, "resource_permissions", perm.ID); err != nil {
			return err
		}
	}

	return nil
}

type RoleAssignmentParams struct {
	UserID         pulid.ID
	OrganizationID pulid.ID
	RoleID         pulid.ID
	AssignedBy     pulid.ID
	AssignedAt     int64
	Record         Recorder
}

func AssignRole(ctx context.Context, db bun.IDB, params RoleAssignmentParams) error {
	assignment := &permission.UserRoleAssignment{
		ID:             pulid.MustNew("ura_"),
		UserID:         params.UserID,
		OrganizationID: params.OrganizationID,
		RoleID:         params.RoleID,
		AssignedBy:     params.AssignedBy,
		AssignedAt:     params.AssignedAt,
	}

	if _, err := db.NewInsert().Model(assignment).Exec(ctx); err != nil {
		return fmt.Errorf("assign role %s to user %s: %w", params.RoleID, params.UserID, err)
	}

	if params.Record == nil {
		return nil
	}

	return params.Record(ctx, "user_role_assignments", assignment.ID)
}
