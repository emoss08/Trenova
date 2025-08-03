/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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

type CreateRoleRequest struct {
	Role           *permission.Role
	PermissionIDs  []pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
}

type UpdateRoleRequest struct {
	Role           *permission.Role
	PermissionIDs  []pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
}

type DeleteRoleRequest struct {
	RoleID         pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
}

type ListPermissionsRequest struct {
	Filter         *ports.LimitOffsetQueryOptions
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
}

type PermissionRepository interface {
	// Role operations
	ListRoles(
		ctx context.Context,
		opts *ListRolesRequest,
	) (*ports.ListResult[*permission.Role], error)
	GetRoleByID(
		ctx context.Context,
		req *GetRoleByIDRequest,
	) (*permission.Role, error)
	CreateRole(
		ctx context.Context,
		req *CreateRoleRequest,
	) (*permission.Role, error)
	UpdateRole(
		ctx context.Context,
		req *UpdateRoleRequest,
	) (*permission.Role, error)
	DeleteRole(
		ctx context.Context,
		req *DeleteRoleRequest,
	) error

	// Permission operations
	ListPermissions(
		ctx context.Context,
		req *ListPermissionsRequest,
	) (*ports.ListResult[*permission.Permission], error)

	// User permission operations
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
