package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	Upsert(
		ctx context.Context,
		entity *tenant.DataRetention,
	) (*tenant.DataRetention, error)
}
