package aitrainingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const cycleEntity = "RetrainingCycle"

type cycleRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCycles(p Params) repositories.RetrainingCycleRepository {
	return &cycleRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aitraining-retraining-cycle-repository"),
	}
}

func (r *cycleRepository) Create(
	ctx context.Context,
	entity *aitraining.RetrainingCycle,
) (*aitraining.RetrainingCycle, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aitraining.RetrainingCycle, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, errortypes.NewBusinessError(
					"A retraining cycle is already exporting, waiting for a trainer, or training",
				).WithInternal(err)
			}
			r.l.Error("failed to create retraining cycle", zap.Error(err))

			return nil, fmt.Errorf("create retraining cycle: %w", err)
		}

		return entity, nil
	})
}

func (r *cycleRepository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*aitraining.RetrainingCycle, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aitraining.RetrainingCycle, error) {
		entity := new(aitraining.RetrainingCycle)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(buncolgen.RetrainingCycleColumns.ID.Eq(), id).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, cycleEntity)
		}

		return entity, nil
	})
}

func (r *cycleRepository) List(
	ctx context.Context,
	req repositories.ListRetrainingCyclesRequest,
) ([]*aitraining.RetrainingCycle, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aitraining.RetrainingCycle, error) {
		cols := buncolgen.RetrainingCycleColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultListLimit
		}
		limit = min(limit, maxListLimit)

		entities := make([]*aitraining.RetrainingCycle, 0, limit)
		query := r.db.DBForContext(ctx).NewSelect().Model(&entities)
		if len(req.Statuses) > 0 {
			query = query.Where(cols.Status.In(), bun.List(req.Statuses))
		}
		err := query.
			Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list retraining cycles", zap.Error(err))

			return nil, fmt.Errorf("list retraining cycles: %w", err)
		}

		return entities, nil
	})
}

func (r *cycleRepository) Update(
	ctx context.Context,
	entity *aitraining.RetrainingCycle,
) (*aitraining.RetrainingCycle, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aitraining.RetrainingCycle, error) {
		cols := buncolgen.RetrainingCycleColumns
		previous := entity.Version

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Where(cols.ID.Eq(), entity.ID).
			Where(cols.Version.Eq(), previous).
			ExcludeColumn(cols.ID.Bare(), cols.CreatedAt.Bare(), cols.Version.Bare()).
			Set(cols.Version.Inc(1)).
			Exec(ctx)
		if err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, errortypes.NewBusinessError(
					"A retraining cycle is already exporting, waiting for a trainer, or training",
				).WithInternal(err)
			}
			r.l.Error("failed to update retraining cycle", zap.Error(err))

			return nil, fmt.Errorf("update retraining cycle: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, cycleEntity, entity.ID.String()); err != nil {
			return nil, err
		}
		entity.Version = previous + 1

		return entity, nil
	})
}
