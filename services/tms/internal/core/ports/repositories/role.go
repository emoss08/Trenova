package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
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

type ListRolesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetRoleByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type GetUserRoleAssignmentsByIDRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	RoleAssignmentID pulid.ID              `json:"roleAssignmentID"`
	UserID           pulid.ID              `json:"userId"` // UserID is the user ID to get the role assignments for
	ExpandRoles      bool                  `json:"expandRoles"`
}

type RoleRepository interface {
	List(
		ctx context.Context,
		req *ListRolesRequest,
	) (*pagination.ListResult[*permission.Role], error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*permission.Role], error)
	Create(ctx context.Context, role *permission.Role) error
	Update(ctx context.Context, role *permission.Role) error
	GetByID(ctx context.Context, req GetRoleByIDRequest) (*permission.Role, error)
	GetRolesWithInheritance(ctx context.Context, roleIDs []pulid.ID) ([]*permission.Role, error)
	GetUsersWithRole(ctx context.Context, roleID pulid.ID) ([]ImpactedUser, error)
	GetUserRoleAssignments(
		ctx context.Context,
		userID, orgID pulid.ID,
	) ([]*permission.UserRoleAssignment, error)
	HasBusinessUnitAdminAccess(ctx context.Context, userID, orgID pulid.ID) (bool, error)
	CreateAssignment(ctx context.Context, assignment *permission.UserRoleAssignment) error
	DeleteAssignment(ctx context.Context, assignmentID pulid.ID) error
	CreateResourcePermission(ctx context.Context, rp *permission.ResourcePermission) error
	UpdateResourcePermission(ctx context.Context, rp *permission.ResourcePermission) error
	DeleteResourcePermission(ctx context.Context, resourceID pulid.ID) error
	GetResourcePermissionsByRoleID(
		ctx context.Context,
		roleID pulid.ID,
	) ([]*permission.ResourcePermission, error)
}
