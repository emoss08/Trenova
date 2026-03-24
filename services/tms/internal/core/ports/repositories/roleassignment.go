package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListRoleAssignmentsRequest struct {
	Filter      *pagination.QueryOptions `json:"filter"`
	ExpandRoles bool                     `json:"expandRoles"`
}
type GetRoleAssignmentByIDRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	RoleAssignmentID pulid.ID              `json:"roleAssignmentID"`
	ExpandRoles      bool                  `json:"expandRoles"`
}

type RoleAssignmentRepository interface {
	List(
		ctx context.Context,
		req *ListRoleAssignmentsRequest,
	) (*pagination.ListResult[*permission.UserRoleAssignment], error)
	GetByID(
		ctx context.Context,
		req GetRoleAssignmentByIDRequest,
	) (*permission.UserRoleAssignment, error)
}
