package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

type DataRetentionRepository interface {
	List(ctx context.Context) (*ports.ListResult[*organization.DataRetention], error)
	Get(ctx context.Context, req GetDataRetentionRequest) (*organization.DataRetention, error)
	Update(
		ctx context.Context,
		entity *organization.DataRetention,
	) (*organization.DataRetention, error)
}

type DataRetentionCacheRepository interface {
	List(ctx context.Context) (*ports.ListResult[*organization.DataRetention], error)
	SetList(ctx context.Context, entities []*organization.DataRetention) error
	InvalidateAll(ctx context.Context) error
	Get(ctx context.Context, req GetDataRetentionRequest) (*organization.DataRetention, error)
	Set(ctx context.Context, entity *organization.DataRetention) error
	Invalidate(ctx context.Context, req GetDataRetentionRequest) error
}

type GetDataRetentionRequest struct {
	UserID pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
}
