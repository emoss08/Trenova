package extractionevalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	resultEntity        = "ExtractionEvalResult"
	defaultPendingLimit = 100
	maxPendingLimit     = 500
)

type resultRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewResults(p Params) repositories.ExtractionEvalResultRepository {
	return &resultRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractioneval-result-repository"),
	}
}

func (r *resultRepository) GetByID(
	ctx context.Context,
	req repositories.GetExtractionEvalResultRequest,
) (*extractioneval.ExtractionResult, error) {
	entity := new(extractioneval.ExtractionResult)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionResultScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ExtractionResultColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, resultEntity)
	}

	return entity, nil
}

func (r *resultRepository) Save(
	ctx context.Context,
	entity *extractioneval.ExtractionResult,
) (*extractioneval.ExtractionResult, error) {
	cols := buncolgen.ExtractionResultColumns
	previous := entity.Version

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ExtractionResultScopeTenantUpdate(uq, pagination.TenantInfo{
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
			cols.RunID.Bare(),
			cols.CaseID.Bare(),
			cols.Ordinal.Bare(),
			cols.CreatedAt.Bare(),
			cols.Version.Bare(),
		).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to save extraction eval result", zap.Error(err))

		return nil, fmt.Errorf("save extraction eval result: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, resultEntity, entity.ID.String()); err != nil {
		return nil, err
	}
	entity.Version = previous + 1

	return entity, nil
}

func (r *resultRepository) ListPending(
	ctx context.Context,
	req repositories.ListPendingExtractionEvalResultsRequest,
) ([]*extractioneval.ExtractionResult, error) {
	cols := buncolgen.ExtractionResultColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPendingLimit
	}
	limit = min(limit, maxPendingLimit)

	entities := make([]*extractioneval.ExtractionResult, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(
			cols.ID.Bare(),
			cols.OrganizationID.Bare(),
			cols.BusinessUnitID.Bare(),
			cols.RunID.Bare(),
			cols.CaseID.Bare(),
			cols.Ordinal.Bare(),
			cols.Status.Bare(),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionResultScopeTenant(sq, req.TenantInfo).
				Where(cols.RunID.Eq(), req.RunID).
				Where(cols.Status.Eq(), extractioneval.ResultStatusPending).
				Where(cols.Ordinal.Gt(), req.AfterOrdinal)
		}).
		Order(cols.Ordinal.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list pending extraction eval results", zap.Error(err))

		return nil, fmt.Errorf("list pending extraction eval results: %w", err)
	}

	return entities, nil
}

func (r *resultRepository) ListByRun(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
) ([]*extractioneval.ExtractionResult, error) {
	cols := buncolgen.ExtractionResultColumns
	entities := make([]*extractioneval.ExtractionResult, 0, extractioneval.DefaultCaseLimit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionResultScopeTenant(sq, tenant).
				Where(cols.RunID.Eq(), runID)
		}).
		Order(cols.Ordinal.OrderAsc()).
		Limit(extractioneval.MaxCaseLimit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list extraction eval results", zap.Error(err))

		return nil, fmt.Errorf("list extraction eval results: %w", err)
	}

	return entities, nil
}

func (r *resultRepository) SkipPending(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
) (int64, error) {
	cols := buncolgen.ExtractionResultColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*extractioneval.ExtractionResult)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ExtractionResultScopeTenantUpdate(uq, tenant).
				Where(cols.RunID.Eq(), runID).
				Where(cols.Status.Eq(), extractioneval.ResultStatusPending)
		}).
		Set(cols.Status.Set(), extractioneval.ResultStatusSkipped).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to skip pending extraction eval results", zap.Error(err))

		return 0, fmt.Errorf("skip pending extraction eval results: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("skip pending extraction eval results rows: %w", err)
	}

	return rows, nil
}

func (r *resultRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListExtractionEvalResultConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionResult], error) {
	dba := r.db.DBForContext(ctx)
	cols := buncolgen.ExtractionResultColumns
	scope := func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(cols.RunID.Eq(), req.RunID)
	}

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*extractioneval.ExtractionResult)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.ExtractionResultTable.Alias,
					req.Filter,
					(*extractioneval.ExtractionResult)(nil),
				)

				return scope(sq.Apply(buncolgen.ExtractionResultApplyTenant(req.Filter.TenantInfo)))
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count extraction eval results", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*extractioneval.ExtractionResult]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*extractioneval.ExtractionResult) *bun.SelectQuery {
			q := dba.NewSelect().Model(entities)
			if len(req.Columns) > 0 {
				q = q.Column(req.Columns...)
			}

			return scope(q)
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.ExtractionResultTable.Alias,
				req.Filter,
				req.Cursor,
				(*extractioneval.ExtractionResult)(nil),
			)
		},
	})
	if err != nil {
		r.l.Error("failed to list extraction eval results", zap.Error(err))

		return nil, err
	}

	return result, nil
}
