package permissionports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/pagination"
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
	AgentIDs       []string                             `json:"agentIds"`
	Checksum       string                               `json:"checksum"`
	ExpiresAt      int64                                `json:"expiresAt"`
	Epoch          int64                                `json:"epoch"`
	UserEpoch      int64                                `json:"userEpoch"`
	AgentRoleID    string                               `json:"agentRoleId,omitempty"`
}

type CacheEpoch struct {
	Organization int64
	User         int64
}

func (e CacheEpoch) Stamp(perms *CachedPermissions) {
	perms.Epoch = e.Organization
	perms.UserEpoch = e.User
}

func (e CacheEpoch) Current(perms *CachedPermissions) bool {
	return perms.Epoch == e.Organization && perms.UserEpoch == e.User
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
	Epoch(ctx context.Context, userID, orgID pulid.ID) (CacheEpoch, error)
	InvalidateOrganization(ctx context.Context, orgID pulid.ID) error
}

type PermissionChangeNotifier interface {
	OrganizationChanged(ctx context.Context, tenant pagination.TenantInfo)
	UserChanged(ctx context.Context, tenant pagination.TenantInfo, userID pulid.ID)
}
