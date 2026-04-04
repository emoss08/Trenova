package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
)

type AILogRepository interface {
	Create(ctx context.Context, entity *ailog.Log) (*ailog.Log, error)
}
