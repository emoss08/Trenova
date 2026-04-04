package testutil

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type RoleBuilder struct {
	role *permission.Role
}

func NewRoleBuilder() *RoleBuilder {
	return &RoleBuilder{
		role: &permission.Role{
			Name:           "Test Role",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions:    []*permission.ResourcePermission{},
			CreatedAt:      timeutils.NowUnix(),
			UpdatedAt:      timeutils.NowUnix(),
		},
	}
}

func (b *RoleBuilder) WithID(id pulid.ID) *RoleBuilder {
	b.role.ID = id
	return b
}

func (b *RoleBuilder) WithName(name string) *RoleBuilder {
	b.role.Name = name
	return b
}

func (b *RoleBuilder) WithOrganizationID(orgID pulid.ID) *RoleBuilder {
	b.role.OrganizationID = orgID
	return b
}

func (b *RoleBuilder) WithDescription(desc string) *RoleBuilder {
	b.role.Description = desc
	return b
}

func (b *RoleBuilder) WithMaxSensitivity(s permission.FieldSensitivity) *RoleBuilder {
	b.role.MaxSensitivity = s
	return b
}

func (b *RoleBuilder) WithIsSystem(isSystem bool) *RoleBuilder {
	b.role.IsSystem = isSystem
	return b
}

func (b *RoleBuilder) WithIsOrgAdmin(isOrgAdmin bool) *RoleBuilder {
	b.role.IsOrgAdmin = isOrgAdmin
	return b
}

func (b *RoleBuilder) WithParentRoleIDs(ids ...pulid.ID) *RoleBuilder {
	b.role.ParentRoleIDs = ids
	return b
}

func (b *RoleBuilder) WithPermission(resource string, ops ...permission.Operation) *RoleBuilder {
	rp := &permission.ResourcePermission{
		ID:         pulid.MustNew("rp_"),
		Resource:   resource,
		Operations: ops,
		DataScope:  permission.DataScopeOrganization,
		CreatedAt:  timeutils.NowUnix(),
		UpdatedAt:  timeutils.NowUnix(),
	}
	b.role.Permissions = append(b.role.Permissions, rp)
	return b
}

func (b *RoleBuilder) WithPermissions(perms ...*permission.ResourcePermission) *RoleBuilder {
	b.role.Permissions = append(b.role.Permissions, perms...)
	return b
}

func (b *RoleBuilder) Build() *permission.Role {
	return b.role
}

type ResourcePermissionBuilder struct {
	rp *permission.ResourcePermission
}

func NewResourcePermissionBuilder() *ResourcePermissionBuilder {
	return &ResourcePermissionBuilder{
		rp: &permission.ResourcePermission{
			ID:         pulid.MustNew("rp_"),
			Resource:   "shipment",
			Operations: []permission.Operation{permission.OpRead},
			DataScope:  permission.DataScopeOrganization,
			CreatedAt:  timeutils.NowUnix(),
			UpdatedAt:  timeutils.NowUnix(),
		},
	}
}

func (b *ResourcePermissionBuilder) WithID(id pulid.ID) *ResourcePermissionBuilder {
	b.rp.ID = id
	return b
}

func (b *ResourcePermissionBuilder) WithRoleID(roleID pulid.ID) *ResourcePermissionBuilder {
	b.rp.RoleID = roleID
	return b
}

func (b *ResourcePermissionBuilder) WithResource(resource string) *ResourcePermissionBuilder {
	b.rp.Resource = resource
	return b
}

func (b *ResourcePermissionBuilder) WithOperations(
	ops ...permission.Operation,
) *ResourcePermissionBuilder {
	b.rp.Operations = ops
	return b
}

func (b *ResourcePermissionBuilder) WithDataScope(
	scope permission.DataScope,
) *ResourcePermissionBuilder {
	b.rp.DataScope = scope
	return b
}

func (b *ResourcePermissionBuilder) Build() *permission.ResourcePermission {
	return b.rp
}

type UserRoleAssignmentBuilder struct {
	assignment *permission.UserRoleAssignment
}

func NewUserRoleAssignmentBuilder() *UserRoleAssignmentBuilder {
	return &UserRoleAssignmentBuilder{
		assignment: &permission.UserRoleAssignment{
			ID:         pulid.MustNew("ura_"),
			AssignedAt: timeutils.NowUnix(),
		},
	}
}

func (b *UserRoleAssignmentBuilder) WithID(id pulid.ID) *UserRoleAssignmentBuilder {
	b.assignment.ID = id
	return b
}

func (b *UserRoleAssignmentBuilder) WithUserID(userID pulid.ID) *UserRoleAssignmentBuilder {
	b.assignment.UserID = userID
	return b
}

func (b *UserRoleAssignmentBuilder) WithRoleID(roleID pulid.ID) *UserRoleAssignmentBuilder {
	b.assignment.RoleID = roleID
	return b
}

func (b *UserRoleAssignmentBuilder) WithOrganizationID(orgID pulid.ID) *UserRoleAssignmentBuilder {
	b.assignment.OrganizationID = orgID
	return b
}

func (b *UserRoleAssignmentBuilder) WithExpiresAt(expiresAt int64) *UserRoleAssignmentBuilder {
	b.assignment.ExpiresAt = &expiresAt
	return b
}

func (b *UserRoleAssignmentBuilder) Build() *permission.UserRoleAssignment {
	return b.assignment
}

type EffectivePermissionsBuilder struct {
	ep *services.EffectivePermissions
}

func NewEffectivePermissionsBuilder() *EffectivePermissionsBuilder {
	return &EffectivePermissionsBuilder{
		ep: &services.EffectivePermissions{
			MaxSensitivity: permission.SensitivityInternal,
			Resources:      make(map[string]services.EffectiveResourcePermission),
			Roles:          []services.RoleSummary{},
		},
	}
}

func (b *EffectivePermissionsBuilder) WithUserID(userID pulid.ID) *EffectivePermissionsBuilder {
	b.ep.UserID = userID
	return b
}

func (b *EffectivePermissionsBuilder) WithOrganizationID(
	orgID pulid.ID,
) *EffectivePermissionsBuilder {
	b.ep.OrganizationID = orgID
	return b
}

func (b *EffectivePermissionsBuilder) WithMaxSensitivity(
	s permission.FieldSensitivity,
) *EffectivePermissionsBuilder {
	b.ep.MaxSensitivity = s
	return b
}

func (b *EffectivePermissionsBuilder) WithResource(
	resource string,
	ops []permission.Operation,
	scope permission.DataScope,
) *EffectivePermissionsBuilder {
	b.ep.Resources[resource] = services.EffectiveResourcePermission{
		Operations: ops,
		DataScope:  scope,
	}
	return b
}

func (b *EffectivePermissionsBuilder) WithRole(
	id pulid.ID,
	name string,
	isOrgAdmin bool,
) *EffectivePermissionsBuilder {
	b.ep.Roles = append(b.ep.Roles, services.RoleSummary{
		ID:         id,
		Name:       name,
		IsOrgAdmin: isOrgAdmin,
	})
	return b
}

func (b *EffectivePermissionsBuilder) Build() *services.EffectivePermissions {
	return b.ep
}

type CachedPermissionsBuilder struct {
	cp *repositories.CachedPermissions
}

func NewCachedPermissionsBuilder() *CachedPermissionsBuilder {
	return &CachedPermissionsBuilder{
		cp: &repositories.CachedPermissions{
			IsOrgAdmin:     false,
			MaxSensitivity: string(permission.SensitivityInternal),
			Resources:      make(map[string]*repositories.CachedResourcePermission),
		},
	}
}

func (b *CachedPermissionsBuilder) WithIsOrgAdmin(isOrgAdmin bool) *CachedPermissionsBuilder {
	b.cp.IsOrgAdmin = isOrgAdmin
	return b
}

func (b *CachedPermissionsBuilder) WithMaxSensitivity(
	s permission.FieldSensitivity,
) *CachedPermissionsBuilder {
	b.cp.MaxSensitivity = string(s)
	return b
}

func (b *CachedPermissionsBuilder) WithResource(
	resource string,
	ops []permission.Operation,
	scope permission.DataScope,
) *CachedPermissionsBuilder {
	strOps := make([]string, len(ops))
	for i, op := range ops {
		strOps[i] = string(op)
	}
	b.cp.Resources[resource] = &repositories.CachedResourcePermission{
		Operations: strOps,
		DataScope:  string(scope),
	}
	return b
}

func (b *CachedPermissionsBuilder) Build() *repositories.CachedPermissions {
	return b.cp
}
