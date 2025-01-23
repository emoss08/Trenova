package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type PermissionRepository interface {
	GetUserPermissions(ctx context.Context, userID pulid.ID) ([]*permission.Permission, error)
	GetUserRoles(ctx context.Context, userID pulid.ID) ([]*string, error)
	GetRolesAndPermissions(ctx context.Context, userID pulid.ID) (*permission.RolesAndPermissions, error)
}

type PermissionCacheRepository interface {
	GetUserRoles(ctx context.Context, userID pulid.ID) ([]*string, error)
	SetUserRoles(ctx context.Context, userID pulid.ID, roles []*string) error
	GetUserPermissions(ctx context.Context, userID pulid.ID) ([]*permission.Permission, error)
	SetUserPermissions(ctx context.Context, userID pulid.ID, permissions []*permission.Permission) error
	InvalidateUserRoles(ctx context.Context, userID pulid.ID) error
	InvalidateUserPermissions(ctx context.Context, userID pulid.ID) error
	InvalidateAllUserData(ctx context.Context, userID pulid.ID) error
}
