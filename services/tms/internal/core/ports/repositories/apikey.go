package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type APIKeyUsageMetadata struct {
	LastUsedAt        int64
	LastUsedIP        string
	LastUsedUserAgent string
}

type ListAPIKeysRequest struct {
	Filter *pagination.QueryOptions
}

type APIKeyRepository interface {
	List(ctx context.Context, req *ListAPIKeysRequest) (*pagination.ListResult[*apikey.Key], error)
	GetByID(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) (*apikey.Key, error)
	GetByPrefix(ctx context.Context, prefix string) (*apikey.Key, error)
	Create(ctx context.Context, key *apikey.Key) error
	CreateWithPermissions(
		ctx context.Context,
		key *apikey.Key,
		permissions []*apikey.Permission,
	) error
	Update(ctx context.Context, key *apikey.Key) error
	UpdateWithPermissions(
		ctx context.Context,
		key *apikey.Key,
		permissions []*apikey.Permission,
	) error
	ReplacePermissions(
		ctx context.Context,
		key *apikey.Key,
		permissions []*apikey.Permission,
	) error
	CountActiveByCreator(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (int, error)
	UpdateUsage(ctx context.Context, id pulid.ID, metadata APIKeyUsageMetadata) error
	IncrementDailyUsage(
		ctx context.Context,
		id, orgID, buID pulid.ID,
		date time.Time,
		count int64,
	) error
}
