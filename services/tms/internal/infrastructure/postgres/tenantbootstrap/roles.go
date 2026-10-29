package tenantbootstrap

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	AdministratorRoleName        = "Organization Administrator"
	administratorRoleDescription = "Full access to all resources within the organization"
)

type AdminRoleParams struct {
	Scope     tenantscope.Scope
	CreatedBy pulid.ID
	Registry  *permissioncatalog.Registry
}

func CreateAdminRole(
	ctx context.Context,
	db *keel.DB,
	params AdminRoleParams,
) (*rbac.Role, error) {
	now := params.Scope.NowUnix()
	role := &rbac.Role{
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

	if err := permissiondb.Roles.Create(ctx, db, role); err != nil {
		return nil, fmt.Errorf("create admin role: %w", err)
	}

	registry := params.Registry
	if registry == nil {
		registry = permissioncatalog.NewRegistry()
	}

	grants := planAdminGrants(registry, role.ID, nil, now).add
	if err := permissiondb.ResourcePermissions.CreateMany(ctx, db, grants); err != nil {
		return nil, fmt.Errorf("grant the admin role every resource: %w", err)
	}

	return role, nil
}

type RoleAssignmentParams struct {
	UserID         pulid.ID
	OrganizationID pulid.ID
	RoleID         pulid.ID
	AssignedBy     pulid.ID
	AssignedAt     int64
}

func AssignRole(ctx context.Context, db *keel.DB, params RoleAssignmentParams) error {
	assignment := &rbac.UserRoleAssignment{
		ID:             pulid.MustNew("ura_"),
		UserID:         params.UserID,
		OrganizationID: params.OrganizationID,
		RoleID:         params.RoleID,
		AssignedBy:     params.AssignedBy,
		AssignedAt:     params.AssignedAt,
	}

	if err := permissiondb.UserRoleAssignments.Create(ctx, db, assignment); err != nil {
		return fmt.Errorf("assign role %s to user %s: %w", params.RoleID, params.UserID, err)
	}

	return nil
}
