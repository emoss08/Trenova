/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type FieldPermissionCheck struct {
	Allowed bool
	Error   error
}

type FieldAccess struct {
	CanModify bool
	CanView   bool
	Errors    []error
}

type PermissionContext struct {
	UserID     pulid.ID
	OrgID      pulid.ID
	BuID       pulid.ID
	Roles      []*string
	Time       time.Time
	CustomData map[string]any
}

type PermissionCheckResult struct {
	Allowed bool
	Error   error
}

type PermissionCheck struct {
	UserID         pulid.ID
	Resource       permission.Resource
	Action         permission.Action
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	ResourceID     pulid.ID       // Optional specific resource ID
	Field          string         // Optional field name for field-level checks
	CustomData     map[string]any // Additional context data
}

type CreateRoleRequest struct {
	Name           string
	Description    string
	RoleType       permission.RoleType
	Priority       int
	ParentRoleID   *pulid.ID
	PermissionIDs  []pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	UserID         pulid.ID // For permission checking
}

type UpdateRoleRequest struct {
	ID             pulid.ID
	Name           string
	Description    string
	RoleType       permission.RoleType
	Priority       int
	ParentRoleID   *pulid.ID
	PermissionIDs  []pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	UserID         pulid.ID // For permission checking
}

type DeleteRoleRequest struct {
	ID             pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	UserID         pulid.ID // For permission checking
}

type ListPermissionsRequest struct {
	Filter *ports.LimitOffsetQueryOptions
	BuID   pulid.ID
	OrgID  pulid.ID
	UserID pulid.ID
}

type PermissionService interface {
	// Permission read operations
	List(
		ctx context.Context,
		req *ListPermissionsRequest,
	) (*ports.ListResult[*permission.Permission], error)

	// Permission checking methods
	CheckFieldModification(
		ctx context.Context,
		userID pulid.ID,
		resource permission.Resource,
		field string,
	) FieldPermissionCheck
	HasPermission(ctx context.Context, check *PermissionCheck) (PermissionCheckResult, error)
	HasAnyPermissions(ctx context.Context, checks []*PermissionCheck) (PermissionCheckResult, error)
	HasFieldPermission(ctx context.Context, check *PermissionCheck) (PermissionCheckResult, error)
	HasAllPermissions(ctx context.Context, checks []*PermissionCheck) (PermissionCheckResult, error)
	HasAnyFieldPermissions(
		ctx context.Context,
		fields []string,
		check *PermissionCheck,
	) (PermissionCheckResult, error)
	HasAllFieldPermissions(
		ctx context.Context,
		fields []string,
		check *PermissionCheck,
	) (PermissionCheckResult, error)
	HasScopedPermission(
		ctx context.Context,
		check *PermissionCheck,
		requiredScope permission.Scope,
	) (PermissionCheckResult, error)
	HasDependentPermissions(
		ctx context.Context,
		check *PermissionCheck,
	) (PermissionCheckResult, error)
	HasTemporalPermission(
		ctx context.Context,
		check *PermissionCheck,
	) (PermissionCheckResult, error)
	CheckFieldAccess(
		ctx context.Context,
		userID pulid.ID,
		resource permission.Resource,
		field string,
	) FieldAccess
	CheckFieldView(
		ctx context.Context,
		userID pulid.ID,
		resource permission.Resource,
		field string,
	) FieldPermissionCheck
}
