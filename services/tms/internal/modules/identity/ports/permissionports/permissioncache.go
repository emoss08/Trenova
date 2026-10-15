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
	AgentRoleID    string                               `json:"agentRoleId,omitempty"`
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
	Epoch(ctx context.Context, orgID pulid.ID) (int64, error)
	InvalidateOrganization(ctx context.Context, orgID pulid.ID) error
}

type PermissionChangeNotifier interface {
	OrganizationChanged(ctx context.Context, tenant pagination.TenantInfo)
	UserChanged(ctx context.Context, tenant pagination.TenantInfo, userID pulid.ID)
}
