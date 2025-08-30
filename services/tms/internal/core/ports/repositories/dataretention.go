package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

type DataRetentionRepository interface {
	List(ctx context.Context) (*ports.ListResult[*organization.DataRetention], error)
}

type DataRetentionCacheRepository interface {
	List(ctx context.Context) (*ports.ListResult[*organization.DataRetention], error)
	SetList(ctx context.Context, entities []*organization.DataRetention) error
	InvalidateAll(ctx context.Context) error
	GetByID(ctx context.Context, entityID pulid.ID) (*organization.DataRetention, error)
	Set(ctx context.Context, entity *organization.DataRetention) error
	Invalidate(ctx context.Context, entityID pulid.ID) error
}
