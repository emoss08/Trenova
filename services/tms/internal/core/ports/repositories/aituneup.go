package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAITuneUpRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type DecideAITuneUpRequest struct {
	TenantInfo     pagination.TenantInfo
	ID             pulid.ID
	Version        int64
	Status         aituneup.Status
	DismissedUntil *int64
	DecidedByID    pulid.ID
	DecidedAt      int64
}

type ReopenAITuneUpRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
}

type AITuneUpRepository interface {
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*aituneup.TuneUp, error)
	GetByID(ctx context.Context, req GetAITuneUpRequest) (*aituneup.TuneUp, error)
	Save(ctx context.Context, tenantInfo pagination.TenantInfo, plan *aituneup.Plan) error
	Decide(ctx context.Context, req *DecideAITuneUpRequest) (*aituneup.TuneUp, error)
	Reopen(ctx context.Context, req *ReopenAITuneUpRequest) (*aituneup.TuneUp, error)
}
