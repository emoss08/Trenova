package extractionevalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	runEntity         = "ExtractionEvalRun"
	defaultPurgeLimit = 500
	maxPurgeLimit     = 5000
	maxRecentRuns     = 50
)

type runRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRuns(p Params) repositories.ExtractionEvalRunRepository {
	return &runRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractioneval-run-repository"),
	}
}

func (r *runRepository) Create(
	ctx context.Context,
	run *extractioneval.ExtractionRun,
	results []*extractioneval.ExtractionResult,
) (*extractioneval.ExtractionRun, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(run).Returning("*").Exec(txCtx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return errortypes.NewBusinessError(
					"An extraction evaluation is already running; wait for it to finish or cancel it",
				).WithInternal(err)
			}

			return fmt.Errorf("create extraction eval run: %w", err)
		}
		if len(results) == 0 {
			return nil
		}
		for _, result := range results {
			result.RunID = run.ID
			result.OrganizationID = run.OrganizationID
			result.BusinessUnitID = run.BusinessUnitID
		}
		if _, err := tx.NewInsert().Model(&results).Exec(txCtx); err != nil {
			return fmt.Errorf("create extraction eval results: %w", err)
		}

		return nil
	})
	if err != nil {
		r.l.Error("failed to create extraction eval run", zap.Error(err))

		return nil, err
	}

	return run, nil
}

func (r *runRepository) GetByID(
	ctx context.Context,
	req repositories.GetExtractionEvalRunRequest,
) (*extractioneval.ExtractionRun, error) {
	entity := new(extractioneval.ExtractionRun)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionRunScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ExtractionRunColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, runEntity)
	}

	return entity, nil
}

func (r *runRepository) Update(
	ctx context.Context,
	entity *extractioneval.ExtractionRun,
) (*extractioneval.ExtractionRun, error) {
	cols := buncolgen.ExtractionRunColumns
	previous := entity.Version

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ExtractionRunScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), previous)
		}).
		ExcludeColumn(
			cols.ID.Bare(),
			cols.OrganizationID.Bare(),
			cols.BusinessUnitID.Bare(),
			cols.CreatedAt.Bare(),
			cols.Version.Bare(),
		).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update extraction eval run", zap.Error(err))

		return nil, fmt.Errorf("update extraction eval run: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, runEntity, entity.ID.String()); err != nil {
		return nil, err
	}
	entity.Version = previous + 1

	return entity, nil
}

func (r *runRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListExtractionEvalRunConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionRun], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*extractioneval.ExtractionRun)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.ExtractionRunTable.Alias,
					req.Filter,
					(*extractioneval.ExtractionRun)(nil),
				)

				return sq.Apply(buncolgen.ExtractionRunApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count extraction eval runs", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*extractioneval.ExtractionRun]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*extractioneval.ExtractionRun) *bun.SelectQuery {
			q := dba.NewSelect().Model(entities)
			if len(req.Columns) > 0 {
				q = q.Column(req.Columns...)
			}

			return q
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.ExtractionRunTable.Alias,
				req.Filter,
				req.Cursor,
				(*extractioneval.ExtractionRun)(nil),
			)
		},
	})
	if err != nil {
		r.l.Error("failed to list extraction eval runs", zap.Error(err))

		return nil, err
	}

	return result, nil
}

func (r *runRepository) ListRecentFinished(
	ctx context.Context,
	tenant pagination.TenantInfo,
	task aicorrection.Task,
	limit int,
) ([]*extractioneval.ExtractionRun, error) {
	cols := buncolgen.ExtractionRunColumns
	if limit <= 0 || limit > maxRecentRuns {
		limit = maxRecentRuns
	}

	entities := make([]*extractioneval.ExtractionRun, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionRunScopeTenant(sq, tenant).
				Where(cols.Task.Eq(), task).
				Where(cols.Status.In(), bun.List([]extractioneval.RunStatus{
					extractioneval.RunStatusCompleted,
					extractioneval.RunStatusBudgetStopped,
					extractioneval.RunStatusCanceled,
				})).
				Where(cols.CasesCompleted.Gt(), 0)
		}).
		Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list recent extraction eval runs", zap.Error(err))

		return nil, fmt.Errorf("list recent extraction eval runs: %w", err)
	}

	return entities, nil
}

func (r *runRepository) PurgeBefore(
	ctx context.Context,
	req repositories.PurgeExtractionEvalRunsRequest,
) (int64, error) {
	cols := buncolgen.ExtractionRunColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPurgeLimit
	}
	limit = min(limit, maxPurgeLimit)

	dba := r.db.DBForContext(ctx)
	expired := dba.NewSelect().
		Model((*extractioneval.ExtractionRun)(nil)).
		Column(cols.ID.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionRunScopeTenant(sq, req.TenantInfo).
				Where(cols.CreatedAt.Lt(), req.Before).
				Where(cols.Status.NotIn(), bun.List([]extractioneval.RunStatus{
					extractioneval.RunStatusQueued,
					extractioneval.RunStatusRunning,
				}))
		}).
		OrderExpr(cols.CreatedAt.OrderAsc()).
		Limit(limit)

	res, err := dba.NewDelete().
		Model((*extractioneval.ExtractionRun)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ExtractionRunScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.In(), expired)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to purge expired extraction eval runs", zap.Error(err))

		return 0, fmt.Errorf("purge expired extraction eval runs: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge expired extraction eval runs rows: %w", err)
	}

	return rows, nil
}
