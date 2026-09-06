package workerdrugalcoholrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultTestPageSize  = 200
	defaultDrawPageSize  = 50
	defaultSweepPageSize = 500
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

func New(p Params) repositories.WorkerDrugAlcoholRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-drug-alcohol-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListTests(
	ctx context.Context,
	req *repositories.ListWorkerDOTTestsRequest,
) ([]*worker.WorkerDOTTest, error) {
	cols := buncolgen.WorkerDOTTestColumns
	entities := make([]*worker.WorkerDOTTest, 0, 16)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerDOTTestScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if len(req.TestTypes) > 0 {
				sq = sq.Where(cols.TestType.In(), bun.In(req.TestTypes))
			}
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.In(req.Statuses))
			}
			if req.OpenOnly {
				sq = sq.Where(cols.Status.In(), bun.In([]worker.DOTTestStatus{
					worker.DOTTestStatusScheduled,
					worker.DOTTestStatusCollected,
					worker.DOTTestStatusAwaitingResult,
				}))
			}
			if req.Since > 0 {
				sq = sq.Where(cols.CollectedAt.Gte(), req.Since)
			}
			return sq
		}).
		OrderExpr(cols.CollectedAt.Qualified() + " DESC NULLS LAST").
		Order(cols.CreatedAt.OrderDesc()).
		Limit(limitOr(req.Limit, defaultTestPageSize))

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerDOTTestRelations.Worker)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerDOTTestRelations.Document)
	}
	if req.IncludeActors {
		q = q.Relation(buncolgen.WorkerDOTTestRelations.OrderedBy).
			Relation(buncolgen.WorkerDOTTestRelations.RecordedBy)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list dot tests", zap.Error(err))
		return nil, fmt.Errorf("list dot tests: %w", err)
	}

	return entities, nil
}

func (r *repository) GetTestByID(
	ctx context.Context,
	req *repositories.GetWorkerDOTTestByIDRequest,
) (*worker.WorkerDOTTest, error) {
	entity := new(worker.WorkerDOTTest)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerDOTTestScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerDOTTestColumns.ID.Eq(), req.ID)
		})

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerDOTTestRelations.Worker)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerDOTTestRelations.Document)
	}
	if req.IncludeActors {
		q = q.Relation(buncolgen.WorkerDOTTestRelations.OrderedBy).
			Relation(buncolgen.WorkerDOTTestRelations.RecordedBy)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerDOTTest")
	}

	return entity, nil
}

func (r *repository) CreateTest(
	ctx context.Context,
	entity *worker.WorkerDOTTest,
) (*worker.WorkerDOTTest, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"drawEntryId",
				errortypes.ErrDuplicate,
				"This random selection already has a test recorded against it",
			)
		}
		r.l.Error("failed to create dot test", zap.Error(err))
		return nil, fmt.Errorf("create dot test: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateTest(
	ctx context.Context,
	entity *worker.WorkerDOTTest,
) (*worker.WorkerDOTTest, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerDOTTestColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update dot test", zap.Error(err))
		return nil, fmt.Errorf("update dot test: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "WorkerDOTTest", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListViolations(
	ctx context.Context,
	req *repositories.ListWorkerDOTViolationsRequest,
) ([]*worker.WorkerDOTViolation, error) {
	cols := buncolgen.WorkerDOTViolationColumns
	entities := make([]*worker.WorkerDOTViolation, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerDOTViolationScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.UnresolvedOnly {
				sq = sq.Where(cols.Status.Ne(), worker.DOTViolationStatusResolved)
			}
			return sq
		}).
		Order(cols.OccurredAt.OrderDesc())

	if req.IncludeTests {
		q = q.Relation(buncolgen.WorkerDOTViolationRelations.SourceTest).
			Relation(buncolgen.WorkerDOTViolationRelations.RTDTest)
	}
	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerDOTViolationRelations.Worker)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list dot violations", zap.Error(err))
		return nil, fmt.Errorf("list dot violations: %w", err)
	}

	return entities, nil
}

func (r *repository) GetViolationByID(
	ctx context.Context,
	req *repositories.GetWorkerDOTViolationByIDRequest,
) (*worker.WorkerDOTViolation, error) {
	entity := new(worker.WorkerDOTViolation)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerDOTViolationScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerDOTViolationColumns.ID.Eq(), req.ID)
		})
	if req.IncludeTests {
		q = q.Relation(buncolgen.WorkerDOTViolationRelations.SourceTest).
			Relation(buncolgen.WorkerDOTViolationRelations.RTDTest)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerDOTViolation")
	}

	return entity, nil
}

// GetOpenViolation returns the unresolved violation for a worker, or nil when
// there is none. A missing row is the ordinary case, so it is not an error.
func (r *repository) GetOpenViolation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerDOTViolation, error) {
	cols := buncolgen.WorkerDOTViolationColumns
	entity := new(worker.WorkerDOTViolation)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerDOTViolationScopeTenant(sq, tenantInfo).
				Where(cols.WorkerID.Eq(), workerID).
				Where(cols.Status.Ne(), worker.DOTViolationStatusResolved)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.l.Error("failed to read open dot violation", zap.Error(err))
		return nil, fmt.Errorf("read open dot violation: %w", err)
	}

	return entity, nil
}

func (r *repository) CreateViolation(
	ctx context.Context,
	entity *worker.WorkerDOTViolation,
) (*worker.WorkerDOTViolation, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"workerId",
				errortypes.ErrDuplicate,
				"This worker already has an unresolved violation; record the new one against it",
			)
		}
		r.l.Error("failed to create dot violation", zap.Error(err))
		return nil, fmt.Errorf("create dot violation: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateViolation(
	ctx context.Context,
	entity *worker.WorkerDOTViolation,
) (*worker.WorkerDOTViolation, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerDOTViolationColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update dot violation", zap.Error(err))
		return nil, fmt.Errorf("update dot violation: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"WorkerDOTViolation",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListQueries(
	ctx context.Context,
	req *repositories.ListClearinghouseQueriesRequest,
) ([]*worker.WorkerClearinghouseQuery, error) {
	cols := buncolgen.WorkerClearinghouseQueryColumns
	entities := make([]*worker.WorkerClearinghouseQuery, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerClearinghouseQueryScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.PendingOnly {
				sq = sq.Where(cols.Result.Eq(), worker.ClearinghouseResultPending)
			}
			return sq
		}).
		Order(cols.RequestedAt.OrderDesc())

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerClearinghouseQueryRelations.Worker)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list clearinghouse queries", zap.Error(err))
		return nil, fmt.Errorf("list clearinghouse queries: %w", err)
	}

	return entities, nil
}

func (r *repository) GetQueryByID(
	ctx context.Context,
	req *repositories.GetClearinghouseQueryByIDRequest,
) (*worker.WorkerClearinghouseQuery, error) {
	entity := new(worker.WorkerClearinghouseQuery)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerClearinghouseQueryScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerClearinghouseQueryColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerClearinghouseQuery")
	}

	return entity, nil
}

func (r *repository) CreateQuery(
	ctx context.Context,
	entity *worker.WorkerClearinghouseQuery,
) (*worker.WorkerClearinghouseQuery, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create clearinghouse query", zap.Error(err))
		return nil, fmt.Errorf("create clearinghouse query: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateQuery(
	ctx context.Context,
	entity *worker.WorkerClearinghouseQuery,
) (*worker.WorkerClearinghouseQuery, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerClearinghouseQueryColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update clearinghouse query", zap.Error(err))
		return nil, fmt.Errorf("update clearinghouse query: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"WorkerClearinghouseQuery",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) applyPoolFilters(
	q *bun.SelectQuery,
	req *repositories.ListDOTRandomPoolsRequest,
) *bun.SelectQuery {
	if req.Status != "" {
		q = q.Where(buncolgen.DOTRandomPoolColumns.Status.Eq(), req.Status)
	}
	return q
}

func (r *repository) ListPools(
	ctx context.Context,
	req *repositories.ListDOTRandomPoolsRequest,
) (*pagination.CursorListResult[*worker.DOTRandomPool], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*worker.DOTRandomPool)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.DOTRandomPoolTable.Alias,
					req.Filter,
					(*worker.DOTRandomPool)(nil),
				)
				return r.applyPoolFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count random pools", zap.Error(err))
			return nil, fmt.Errorf("count random pools: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*worker.DOTRandomPool]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(items *[]*worker.DOTRandomPool) *bun.SelectQuery {
			return dba.NewSelect().
				Model(items).
				ColumnExpr(buncolgen.DOTRandomPoolTable.All())
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.DOTRandomPoolTable.Alias,
				req.Filter,
				req.Cursor,
				(*worker.DOTRandomPool)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			return r.applyPoolFilters(sq, req), nil
		},
	})
	if err != nil {
		r.l.Error("failed to list random pools", zap.Error(err))
		return nil, fmt.Errorf("list random pools: %w", err)
	}

	return result, nil
}

func (r *repository) GetPoolByID(
	ctx context.Context,
	req *repositories.GetDOTRandomPoolByIDRequest,
) (*worker.DOTRandomPool, error) {
	entity := new(worker.DOTRandomPool)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DOTRandomPoolScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DOTRandomPoolColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DOTRandomPool")
	}

	return entity, nil
}

// GetDefaultPool returns the pool a draw runs against when none is named. An
// organisation with no default returns nil rather than an error: the caller
// decides whether that is a problem.
func (r *repository) GetDefaultPool(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.DOTRandomPool, error) {
	cols := buncolgen.DOTRandomPoolColumns
	entity := new(worker.DOTRandomPool)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DOTRandomPoolScopeTenant(sq, tenantInfo).
				Where(cols.IsDefault.Eq(), true)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.l.Error("failed to read default random pool", zap.Error(err))
		return nil, fmt.Errorf("read default random pool: %w", err)
	}

	return entity, nil
}

func (r *repository) PoolCodeExists(
	ctx context.Context,
	req *repositories.DOTRandomPoolCodeExistsRequest,
) (bool, error) {
	cols := buncolgen.DOTRandomPoolColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.DOTRandomPool)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DOTRandomPoolScopeTenant(sq, req.TenantInfo).
				Where("LOWER("+cols.Code.String()+") = ?", strings.ToLower(req.Code))
		})
	if !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExcludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		r.l.Error("failed to check random pool code", zap.Error(err))
		return false, fmt.Errorf("check random pool code: %w", err)
	}

	return exists, nil
}

func (r *repository) CreatePool(
	ctx context.Context,
	entity *worker.DOTRandomPool,
) (*worker.DOTRandomPool, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"code",
				errortypes.ErrDuplicate,
				"A pool with this code already exists",
			)
		}
		r.l.Error("failed to create random pool", zap.Error(err))
		return nil, fmt.Errorf("create random pool: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdatePool(
	ctx context.Context,
	entity *worker.DOTRandomPool,
) (*worker.DOTRandomPool, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.DOTRandomPoolColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"code",
				errortypes.ErrDuplicate,
				"A pool with this code already exists",
			)
		}
		r.l.Error("failed to update random pool", zap.Error(err))
		return nil, fmt.Errorf("update random pool: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "DOTRandomPool", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// ClearDefaultPool drops the default flag from every other pool, so promoting
// one never trips the partial unique index.
func (r *repository) ClearDefaultPool(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exceptID pulid.ID,
) error {
	cols := buncolgen.DOTRandomPoolColumns
	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.DOTRandomPool)(nil)).
		Set(cols.IsDefault.Bare() + " = FALSE").
		Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DOTRandomPoolScopeTenantUpdate(uq, tenantInfo).
				Where(cols.IsDefault.Eq(), true)
		})
	if !exceptID.IsNil() {
		q = q.Where(cols.ID.Ne(), exceptID)
	}

	if _, err := q.Exec(ctx); err != nil {
		r.l.Error("failed to clear default random pool", zap.Error(err))
		return fmt.Errorf("clear default random pool: %w", err)
	}

	return nil
}

func (r *repository) ListDraws(
	ctx context.Context,
	req *repositories.ListDOTRandomDrawsRequest,
) ([]*worker.DOTRandomDraw, error) {
	cols := buncolgen.DOTRandomDrawColumns
	entities := make([]*worker.DOTRandomDraw, 0, 16)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.DOTRandomDrawScopeTenant(sq, req.TenantInfo)
			if !req.PoolID.IsNil() {
				sq = sq.Where(cols.PoolID.Eq(), req.PoolID)
			}
			return sq
		}).
		Order(cols.DrawnAt.OrderDesc()).
		Limit(limitOr(req.Limit, defaultDrawPageSize))

	if req.IncludePool {
		q = q.Relation(buncolgen.DOTRandomDrawRelations.Pool)
	}
	if req.IncludeActors {
		q = q.Relation(buncolgen.DOTRandomDrawRelations.DrawnBy)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list random draws", zap.Error(err))
		return nil, fmt.Errorf("list random draws: %w", err)
	}

	return entities, nil
}

func (r *repository) GetDrawByID(
	ctx context.Context,
	req *repositories.GetDOTRandomDrawByIDRequest,
) (*worker.DOTRandomDraw, error) {
	entity := new(worker.DOTRandomDraw)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DOTRandomDrawScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DOTRandomDrawColumns.ID.Eq(), req.ID)
		})

	if req.IncludePool {
		q = q.Relation(buncolgen.DOTRandomDrawRelations.Pool)
	}
	if req.IncludeEntries {
		q = q.Relation(
			buncolgen.DOTRandomDrawRelations.Entries,
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = sq.Order(buncolgen.DOTRandomDrawEntryColumns.Rank.OrderAsc())
				if req.IncludeWorkers {
					sq = sq.Relation(buncolgen.DOTRandomDrawEntryRelations.Worker)
				}
				return sq
			},
		)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "DOTRandomDraw")
	}

	return entity, nil
}

// GetDrawByPeriod finds the round already run for a period, or nil when there
// is none. Cancelled rounds do not count: a period can be re-drawn after one is
// voided.
func (r *repository) GetDrawByPeriod(
	ctx context.Context,
	req *repositories.GetDOTRandomDrawByPeriodRequest,
) (*worker.DOTRandomDraw, error) {
	cols := buncolgen.DOTRandomDrawColumns
	entity := new(worker.DOTRandomDraw)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DOTRandomDrawScopeTenant(sq, req.TenantInfo).
				Where(cols.PoolID.Eq(), req.PoolID).
				Where(cols.PeriodKey.Eq(), req.PeriodKey).
				Where(cols.Status.Ne(), worker.RandomDrawStatusCancelled)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.l.Error("failed to read random draw by period", zap.Error(err))
		return nil, fmt.Errorf("read random draw by period: %w", err)
	}

	return entity, nil
}

func (r *repository) CreateDrawWithEntries(
	ctx context.Context,
	draw *worker.DOTRandomDraw,
	entries []*worker.DOTRandomDrawEntry,
) (*worker.DOTRandomDraw, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(draw).Returning("*").Exec(txCtx); iErr != nil {
			return iErr
		}
		if len(entries) == 0 {
			return nil
		}
		for _, entry := range entries {
			entry.DrawID = draw.ID
			entry.OrganizationID = draw.OrganizationID
			entry.BusinessUnitID = draw.BusinessUnitID
		}
		_, iErr := tx.NewInsert().Model(&entries).Exec(txCtx)
		return iErr
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"periodKey",
				errortypes.ErrDuplicate,
				"This pool has already been drawn for that period",
			)
		}
		r.l.Error("failed to create random draw", zap.Error(err))
		return nil, fmt.Errorf("create random draw: %w", err)
	}

	draw.Entries = entries

	return draw, nil
}

func (r *repository) UpdateDraw(
	ctx context.Context,
	entity *worker.DOTRandomDraw,
) (*worker.DOTRandomDraw, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.DOTRandomDrawColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update random draw", zap.Error(err))
		return nil, fmt.Errorf("update random draw: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "DOTRandomDraw", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// ListPoolCandidates returns the workers eligible to be drawn: active drivers
// of the types the pool covers who are not already prohibited. Drawing a name
// that cannot be tested would put the round under target for no reason.
func (r *repository) ListPoolCandidates(
	ctx context.Context,
	req *repositories.ListPoolCandidatesRequest,
) ([]pulid.ID, error) {
	workerCols := buncolgen.WorkerColumns
	profileCols := buncolgen.WorkerProfileColumns

	ids := make([]pulid.ID, 0, 64)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		Column(workerCols.ID.Bare()).
		Join("JOIN worker_profiles AS wrkp ON wrkp.worker_id = wrk.id"+
			" AND wrkp.organization_id = wrk.organization_id"+
			" AND wrkp.business_unit_id = wrk.business_unit_id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerScopeTenant(sq, req.TenantInfo).
				Where(workerCols.Status.Eq(), "Active").
				Where(profileCols.DrugAlcoholStatus.Ne(), worker.DrugAlcoholProhibited)
			if len(req.DriverTypes) > 0 {
				sq = sq.Where(workerCols.DriverType.In(), bun.In(req.DriverTypes))
			}
			return sq
		}).
		Order(workerCols.ID.OrderAsc())

	if err := q.Scan(ctx, &ids); err != nil {
		r.l.Error("failed to list pool candidates", zap.Error(err))
		return nil, fmt.Errorf("list pool candidates: %w", err)
	}

	return ids, nil
}

func (r *repository) ListDrawEntries(
	ctx context.Context,
	req *repositories.ListDOTRandomDrawEntriesRequest,
) ([]*worker.DOTRandomDrawEntry, error) {
	cols := buncolgen.DOTRandomDrawEntryColumns
	entities := make([]*worker.DOTRandomDrawEntry, 0, 32)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.DOTRandomDrawEntryScopeTenant(sq, req.TenantInfo)
			if !req.DrawID.IsNil() {
				sq = sq.Where(cols.DrawID.Eq(), req.DrawID)
			}
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.OutstandingOnly {
				sq = sq.Where(cols.Status.In(), bun.In([]worker.RandomEntryStatus{
					worker.RandomEntrySelected,
					worker.RandomEntryNotified,
				}))
			}
			return sq
		}).
		Order(cols.Rank.OrderAsc())

	if req.IncludeWorker {
		q = q.Relation(buncolgen.DOTRandomDrawEntryRelations.Worker)
	}
	if req.IncludeDraw {
		q = q.Relation(buncolgen.DOTRandomDrawEntryRelations.Draw)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list draw entries", zap.Error(err))
		return nil, fmt.Errorf("list draw entries: %w", err)
	}

	return entities, nil
}

func (r *repository) GetDrawEntryByID(
	ctx context.Context,
	req *repositories.GetDOTRandomDrawEntryByIDRequest,
) (*worker.DOTRandomDrawEntry, error) {
	entity := new(worker.DOTRandomDrawEntry)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DOTRandomDrawEntryScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DOTRandomDrawEntryColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DOTRandomDrawEntry")
	}

	return entity, nil
}

func (r *repository) UpdateDrawEntry(
	ctx context.Context,
	entity *worker.DOTRandomDrawEntry,
) (*worker.DOTRandomDrawEntry, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.DOTRandomDrawEntryColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update draw entry", zap.Error(err))
		return nil, fmt.Errorf("update draw entry: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"DOTRandomDrawEntry",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateProfileRollup(
	ctx context.Context,
	req *repositories.UpdateDrugAlcoholRollupRequest,
) error {
	cols := buncolgen.WorkerProfileColumns

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.WorkerProfile)(nil)).
		Set(cols.DrugAlcoholStatus.Set(), req.Rollup.Status).
		Set(cols.ReturnToDutyStatus.Set(), req.Rollup.ReturnToDuty).
		Set(cols.LastClearinghouseQueryAt.Set(), req.Rollup.LastClearinghouseQueryAt).
		Set(cols.NextClearinghouseQueryDue.Set(), req.Rollup.NextClearinghouseQueryDue).
		Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.WorkerProfileScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update drug and alcohol rollup", zap.Error(err))
		return fmt.Errorf("update drug and alcohol rollup: %w", err)
	}

	return nil
}

// ListWorkersWithClearinghouseDue finds the drivers whose annual query falls
// due on or before the horizon, plus those who have never been queried at all —
// a driver nobody has ever asked about is the one most worth chasing.
func (r *repository) ListWorkersWithClearinghouseDue(
	ctx context.Context,
	req *repositories.ListWorkersWithClearinghouseDueRequest,
) ([]repositories.WorkerTenantRef, error) {
	workerCols := buncolgen.WorkerColumns
	profileCols := buncolgen.WorkerProfileColumns

	refs := make([]repositories.WorkerTenantRef, 0, 64)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		ColumnExpr("wrk.id AS worker_id").
		ColumnExpr("wrk.organization_id AS organization_id").
		ColumnExpr("wrk.business_unit_id AS business_unit_id").
		Join("JOIN worker_profiles AS wrkp ON wrkp.worker_id = wrk.id"+
			" AND wrkp.organization_id = wrk.organization_id"+
			" AND wrkp.business_unit_id = wrk.business_unit_id").
		Where(workerCols.Status.Eq(), "Active").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(profileCols.NextClearinghouseQueryDue.Lte(), req.Until).
				WhereOr(profileCols.NextClearinghouseQueryDue.IsNull())
		}).
		Order(workerCols.ID.OrderAsc()).
		Limit(limitOr(req.Limit, defaultSweepPageSize)).
		Scan(ctx, &refs)
	if err != nil {
		r.l.Error("failed to list workers with clearinghouse due", zap.Error(err))
		return nil, fmt.Errorf("list workers with clearinghouse due: %w", err)
	}

	return refs, nil
}
