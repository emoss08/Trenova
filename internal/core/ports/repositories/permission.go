package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type RolesQueryOptions struct {
	IncludeChildren    bool `query:"includeChildren"    default:"false" json:"includeChildren"`
	IncludeParent      bool `query:"includeParent"      default:"false" json:"includeParent"`
	IncludePermissions bool `query:"includePermissions" default:"false" json:"includePermissions"`
}

type ListRolesRequest struct {
	Filter       *ports.LimitOffsetQueryOptions `query:"filter"       json:"filter"`
	QueryOptions RolesQueryOptions              `query:"queryOptions" json:"queryOptions"`
}

type GetRoleByIDRequest struct {
	RoleID       pulid.ID
	OrgID        pulid.ID
	BuID         pulid.ID
	UserID       pulid.ID
	QueryOptions RolesQueryOptions `query:"queryOptions" json:"queryOptions"`
}

type PermissionRepository interface {
	ListRoles(
		ctx context.Context,
		opts *ListRolesRequest,
	) (*ports.ListResult[*permission.Role], error)
	GetRoleByID(
		ctx context.Context,
		req *GetRoleByIDRequest,
	) (*permission.Role, error)
	GetUserPermissions(ctx context.Context, userID pulid.ID) ([]*permission.Permission, error)
	GetUserRoles(ctx context.Context, userID pulid.ID) ([]*string, error)
	GetRolesAndPermissions(
		ctx context.Context,
		userID pulid.ID,
	) (*permission.RolesAndPermissions, error)
}

type PermissionCacheRepository interface {
	GetUserRoles(ctx context.Context, userID pulid.ID) ([]*string, error)
	SetUserRoles(ctx context.Context, userID pulid.ID, roles []*string) error
	GetUserPermissions(ctx context.Context, userID pulid.ID) ([]*permission.Permission, error)
	SetUserPermissions(
		ctx context.Context,
		userID pulid.ID,
		permissions []*permission.Permission,
	) error
	InvalidateUserRoles(ctx context.Context, userID pulid.ID) error
	InvalidateUserPermissions(ctx context.Context, userID pulid.ID) error
	InvalidateAllUserData(ctx context.Context, userID pulid.ID) error
}
