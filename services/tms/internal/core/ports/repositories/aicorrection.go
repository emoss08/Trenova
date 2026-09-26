package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PurgeAICorrectionsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

type GetAICorrectionRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListAICorrectionConnectionRequest struct {
	Filter  *pagination.QueryOptions
	Cursor  pagination.CursorInfo
	Columns []string
}

type ListAICorrectionsForAccuracyRequest struct {
	TenantInfo pagination.TenantInfo
	Task       aicorrection.Task
	Since      int64
	Limit      int
}

type AICorrectionRepository interface {
	Upsert(ctx context.Context, entity *aicorrection.Correction) (*aicorrection.Correction, error)
	GetByID(ctx context.Context, req GetAICorrectionRequest) (*aicorrection.Correction, error)
	ListConnection(
		ctx context.Context,
		req *ListAICorrectionConnectionRequest,
	) (*pagination.CursorListResult[*aicorrection.Correction], error)
	ListForAccuracy(
		ctx context.Context,
		req ListAICorrectionsForAccuracyRequest,
	) ([]*aicorrection.Correction, error)
	PurgeBefore(ctx context.Context, req PurgeAICorrectionsRequest) (int64, error)
}
