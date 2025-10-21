package ailogrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
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

func NewRepository(p Params) repositories.AILogRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ailog-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAILogRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"ailog",
		req.Filter,
		(*ailog.AILog)(nil),
	)

	if req.IncludeUser {
		q = q.Relation("User")
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAILogRequest,
) (*pagination.ListResult[*ailog.AILog], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*ailog.AILog, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan ai logs", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*ailog.AILog]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Insert(ctx context.Context, aiLog *ailog.AILog) error {
	log := r.l.With(zap.String("operation", "Insert"))

	db, err := r.db.DB(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewInsert().Model(aiLog).Exec(ctx)
	if err != nil {
		log.Error("failed to insert ai log", zap.Error(err))
		return err
	}

	return err
}
