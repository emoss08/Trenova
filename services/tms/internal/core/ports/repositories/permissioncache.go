package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

const PermissionCacheVariantAllRoles = "all"

type CachedResourcePermission struct {
	Operations []string `json:"operations"`
	DataScope  string   `json:"dataScope"`
}

type CachedPermissions struct {
	MaxSensitivity string                               `json:"maxSensitivity"`
	Resources      map[string]*CachedResourcePermission `json:"resources"`
	Checksum       string                               `json:"checksum"`
	ExpiresAt      int64                                `json:"expiresAt"`
}

type PermissionCacheKey struct {
	UserID  pulid.ID
	OrgID   pulid.ID
	Variant string
}

type PermissionCacheRepository interface {
	Get(ctx context.Context, key PermissionCacheKey) (*CachedPermissions, error)
	Set(
		ctx context.Context,
		key PermissionCacheKey,
		perms *CachedPermissions,
		ttl time.Duration,
	) error
	Delete(ctx context.Context, userID, orgID pulid.ID) error
	InvalidateByRole(ctx context.Context, roleID pulid.ID, roleRepo RoleRepository) error
	InvalidateOrganization(ctx context.Context, orgID pulid.ID) error
}
