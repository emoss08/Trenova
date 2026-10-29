package permissionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ImpactedUser struct {
	UserID         pulid.ID `json:"userId"`
	UserName       string   `json:"userName"`
	OrganizationID pulid.ID `json:"organizationId"`
	OrgName        string   `json:"orgName"`
	AssignmentType string   `json:"assignmentType"`
}

// PermittedUser is an active user who holds an operation on a resource in an
// organization, through any role they are assigned or that role's parents.
type PermittedUser struct {
	UserID        pulid.ID
	Name          string
	Username      string
	EmailAddress  string
	ProfilePicURL string
	Locale        string
}

type ListUsersWithPermissionRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Resource       permission.Resource
	Operation      permission.Operation
	// Now excludes expired role assignments.
	Now int64
}

type ListRolesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListRoleConnectionRequest struct {
	Filter      *pagination.QueryOptions `json:"filter"`
	Cursor      pagination.CursorInfo    `json:"-"`
	RoleColumns []string                 `json:"-"`
}

type DeleteRoleAssignmentRequest struct {
	AssignmentID   pulid.ID
	OrganizationID pulid.ID
}

type DeleteResourcePermissionRequest struct {
	PermissionID pulid.ID
	RoleID       pulid.ID
}

type GetRoleByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type GetRolesByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	RoleIDs    []pulid.ID            `json:"roleIds"`
}

type GetUserRoleAssignmentsByIDRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	RoleAssignmentID pulid.ID              `json:"roleAssignmentID"`
	UserID           pulid.ID              `json:"userId"` // UserID is the user ID to get the role assignments for
	ExpandRoles      bool                  `json:"expandRoles"`
}

type DeleteRoleRequest struct {
	RoleID         pulid.ID
	OrganizationID pulid.ID
}

type OrphanedAssignment struct {
	UserID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type RoleRepository interface {
	List(
		ctx context.Context,
		req *ListRolesRequest,
	) (*pagination.ListResult[*rbac.Role], error)
	ListConnection(
		ctx context.Context,
		req *ListRoleConnectionRequest,
	) (*pagination.CursorListResult[*rbac.Role], error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*rbac.Role], error)
	Create(ctx context.Context, role *rbac.Role) error
	Update(ctx context.Context, role *rbac.Role) error
	Delete(ctx context.Context, req DeleteRoleRequest) error
	GetByID(ctx context.Context, req GetRoleByIDRequest) (*rbac.Role, error)
	GetByIDs(ctx context.Context, req GetRolesByIDsRequest) ([]*rbac.Role, error)
	GetRolesWithInheritance(ctx context.Context, roleIDs []pulid.ID) ([]*rbac.Role, error)
	GetUsersWithRole(ctx context.Context, roleID pulid.ID) ([]ImpactedUser, error)
	ListUsersWithPermission(
		ctx context.Context,
		req ListUsersWithPermissionRequest,
	) ([]PermittedUser, error)
	GetUserRoleAssignments(
		ctx context.Context,
		userID, orgID pulid.ID,
	) ([]*rbac.UserRoleAssignment, error)
	CreateAssignment(ctx context.Context, assignment *rbac.UserRoleAssignment) error
	DeleteAssignment(
		ctx context.Context,
		req DeleteRoleAssignmentRequest,
	) (*rbac.UserRoleAssignment, error)
	DeleteUserAssignments(
		ctx context.Context,
		userID, orgID pulid.ID,
	) ([]*rbac.UserRoleAssignment, error)
	ListOrphanedAssignments(
		ctx context.Context,
		now int64,
		limit int,
	) ([]OrphanedAssignment, error)
	CreateResourcePermission(ctx context.Context, rp *rbac.ResourcePermission) error
	UpdateResourcePermission(ctx context.Context, rp *rbac.ResourcePermission) error
	DeleteResourcePermission(ctx context.Context, req DeleteResourcePermissionRequest) error
	GetResourcePermissionsByRoleID(
		ctx context.Context,
		roleID pulid.ID,
	) ([]*rbac.ResourcePermission, error)
}
