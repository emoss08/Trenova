package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetDataRetentionRequest struct {
	UserID pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
}

type DataRetentionRepository interface {
	List(ctx context.Context) (*pagination.ListResult[*tenant.DataRetention], error)
	Get(ctx context.Context, req GetDataRetentionRequest) (*tenant.DataRetention, error)
	Update(
		ctx context.Context,
		entity *tenant.DataRetention,
	) (*tenant.DataRetention, error)
}

type DataRetentionCacheRepository interface {
	List(ctx context.Context) (*pagination.ListResult[*tenant.DataRetention], error)
	SetList(ctx context.Context, entities []*tenant.DataRetention) error
	InvalidateAll(ctx context.Context) error
	Get(ctx context.Context, req GetDataRetentionRequest) (*tenant.DataRetention, error)
	Set(ctx context.Context, entity *tenant.DataRetention) error
	Invalidate(ctx context.Context, req GetDataRetentionRequest) error
}
