package ailogrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AILogRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ai-log-repository"),
	}
}

func (r *repository) Create(ctx context.Context, entity *ailog.Log) (*ailog.Log, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}
