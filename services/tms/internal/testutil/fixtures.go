package testutil

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type RoleBuilder struct {
	role *rbac.Role
}

func NewRoleBuilder() *RoleBuilder {
	return &RoleBuilder{
		role: &rbac.Role{
			Name:           "Test Role",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions:    []*rbac.ResourcePermission{},
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

func (b *RoleBuilder) WithParentRoleIDs(ids ...pulid.ID) *RoleBuilder {
	b.role.ParentRoleIDs = ids
	return b
}

func (b *RoleBuilder) WithPermission(resource string, ops ...permission.Operation) *RoleBuilder {
	rp := &rbac.ResourcePermission{
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

func (b *RoleBuilder) WithPermissions(perms ...*rbac.ResourcePermission) *RoleBuilder {
	b.role.Permissions = append(b.role.Permissions, perms...)
	return b
}

func (b *RoleBuilder) Build() *rbac.Role {
	return b.role
}

type ResourcePermissionBuilder struct {
	rp *rbac.ResourcePermission
}

func NewResourcePermissionBuilder() *ResourcePermissionBuilder {
	return &ResourcePermissionBuilder{
		rp: &rbac.ResourcePermission{
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

func (b *ResourcePermissionBuilder) Build() *rbac.ResourcePermission {
	return b.rp
}

type UserRoleAssignmentBuilder struct {
	assignment *rbac.UserRoleAssignment
}

func NewUserRoleAssignmentBuilder() *UserRoleAssignmentBuilder {
	return &UserRoleAssignmentBuilder{
		assignment: &rbac.UserRoleAssignment{
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

func (b *UserRoleAssignmentBuilder) Build() *rbac.UserRoleAssignment {
	return b.assignment
}

type EffectivePermissionsBuilder struct {
	ep *permissionports.EffectivePermissions
}

func NewEffectivePermissionsBuilder() *EffectivePermissionsBuilder {
	return &EffectivePermissionsBuilder{
		ep: &permissionports.EffectivePermissions{
			MaxSensitivity: permission.SensitivityInternal,
			Resources:      make(map[string]permissionports.EffectiveResourcePermission),
			Roles:          []permissionports.RoleSummary{},
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
	b.ep.Resources[resource] = permissionports.EffectiveResourcePermission{
		Operations: ops,
		DataScope:  scope,
	}
	return b
}

func (b *EffectivePermissionsBuilder) WithRole(
	id pulid.ID,
	name string,
) *EffectivePermissionsBuilder {
	b.ep.Roles = append(b.ep.Roles, permissionports.RoleSummary{
		ID:   id,
		Name: name,
	})
	return b
}

func (b *EffectivePermissionsBuilder) Build() *permissionports.EffectivePermissions {
	return b.ep
}

type CachedPermissionsBuilder struct {
	cp *permissionports.CachedPermissions
}

func NewCachedPermissionsBuilder() *CachedPermissionsBuilder {
	return &CachedPermissionsBuilder{
		cp: &permissionports.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityInternal),
			Resources:      make(map[string]*permissionports.CachedResourcePermission),
		},
	}
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
	b.cp.Resources[resource] = &permissionports.CachedResourcePermission{
		Operations: strOps,
		DataScope:  string(scope),
	}
	return b
}

func (b *CachedPermissionsBuilder) Build() *permissionports.CachedPermissions {
	return b.cp
}
