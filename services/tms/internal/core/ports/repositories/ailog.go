package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ListAILogRequest struct {
	Filter      *pagination.QueryOptions `form:"filter"`
	IncludeUser bool                     `form:"includeUser"`
}

type AILogRepository interface {
	Insert(ctx context.Context, log *ailog.AILog) error
	List(
		ctx context.Context,
		opts *ListAILogRequest,
	) (*pagination.ListResult[*ailog.AILog], error)
}
