package extractionshadowrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	resultEntity       = "ExtractionShadowResult"
	defaultScoredLimit = 2000
	maxScoredLimit     = 10000
	defaultPurgeLimit  = 1000
	maxPurgeLimit      = 10000
)

type resultRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewResults(p Params) repositories.ExtractionShadowResultRepository {
	return &resultRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionshadow-result-repository"),
	}
}

func uniqueExtraction() string {
	cols := buncolgen.ShadowResultColumns

	return "CONFLICT (" + strings.Join([]string{
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.DocumentID.Bare(),
		cols.ExtractedAt.Bare(),
	}, ", ") + ") DO NOTHING"
}

func (r *resultRepository) Create(
	ctx context.Context,
	entity *extractionshadow.ShadowResult,
) (*extractionshadow.ShadowResult, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowResult, bool, error) {
		res, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On(uniqueExtraction()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to create extraction shadow result",
				zap.String("documentId", entity.DocumentID.String()),
				zap.Error(err),
			)

			return nil, false, fmt.Errorf("create extraction shadow result: %w", err)
		}

		inserted, err := res.RowsAffected()
		if err != nil {
			return nil, false, fmt.Errorf("create extraction shadow result rows: %w", err)
		}
		if inserted > 0 {
			return entity, true, nil
		}

		existing, err := r.GetByExtraction(
			ctx,
			repositories.GetExtractionShadowResultByExtractionRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				DocumentID:  entity.DocumentID,
				ExtractedAt: entity.ExtractedAt,
			},
		)
		if err != nil {
			return nil, false, err
		}

		return existing, false, nil
	})
}

func (r *resultRepository) GetByID(
	ctx context.Context,
	req repositories.GetExtractionShadowResultRequest,
) (*extractionshadow.ShadowResult, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowResult, error) {
		entity := new(extractionshadow.ShadowResult)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ShadowResultScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.ShadowResultColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, resultEntity)
		}

		return entity, nil
	})
}

func (r *resultRepository) GetByExtraction(
	ctx context.Context,
	req repositories.GetExtractionShadowResultByExtractionRequest,
) (*extractionshadow.ShadowResult, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowResult, error) {
		cols := buncolgen.ShadowResultColumns
		entity := new(extractionshadow.ShadowResult)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ShadowResultScopeTenant(sq, req.TenantInfo).
					Where(cols.DocumentID.Eq(), req.DocumentID).
					Where(cols.ExtractedAt.Eq(), req.ExtractedAt)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, resultEntity)
		}

		return entity, nil
	})
}

func (r *resultRepository) Save(
	ctx context.Context,
	entity *extractionshadow.ShadowResult,
) (*extractionshadow.ShadowResult, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowResult, error) {
		cols := buncolgen.ShadowResultColumns
		previous := entity.Version

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ShadowResultScopeTenantUpdate(uq, pagination.TenantInfo{
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
				cols.DocumentID.Bare(),
				cols.ExtractedAt.Bare(),
				cols.CreatedAt.Bare(),
				cols.Version.Bare(),
			).
			Set(cols.Version.Inc(1)).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to save extraction shadow result", zap.Error(err))

			return nil, fmt.Errorf("save extraction shadow result: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, resultEntity, entity.ID.String()); err != nil {
			return nil, err
		}
		entity.Version = previous + 1

		return entity, nil
	})
}

func (r *resultRepository) CountCreatedSince(
	ctx context.Context,
	tenant pagination.TenantInfo,
	since int64,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*extractionshadow.ShadowResult)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ShadowResultScopeTenant(sq, tenant).
					Where(buncolgen.ShadowResultColumns.CreatedAt.Gte(), since)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count extraction shadow results", zap.Error(err))

			return 0, fmt.Errorf("count extraction shadow results: %w", err)
		}

		return count, nil
	})
}

func (r *resultRepository) ListScored(
	ctx context.Context,
	req *repositories.ListScoredExtractionShadowResultsRequest,
) ([]*extractionshadow.ShadowResult, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*extractionshadow.ShadowResult, error) {
		cols := buncolgen.ShadowResultColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultScoredLimit
		}
		limit = min(limit, maxScoredLimit)

		entities := make([]*extractionshadow.ShadowResult, 0, min(limit, defaultScoredLimit))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Column(
				cols.ID.Bare(),
				cols.ProviderID.Bare(),
				cols.ProviderName.Bare(),
				cols.ServedModel.Bare(),
				cols.Verdict.Bare(),
				cols.FieldResults.Bare(),
				cols.ScoredCount.Bare(),
				cols.CorrectCount.Bare(),
				cols.CorrectedCount.Bare(),
				cols.MissedCount.Bare(),
				cols.BaselineFieldResults.Bare(),
				cols.BaselineScoredCount.Bare(),
				cols.BaselineCorrectCount.Bare(),
				cols.BaselineCorrectedCount.Bare(),
				cols.BaselineMissedCount.Bare(),
				cols.ScoredAt.Bare(),
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = buncolgen.ShadowResultScopeTenant(sq, req.TenantInfo).
					Where(cols.Status.Eq(), extractionshadow.ResultStatusCompleted).
					Where(cols.ScoredAt.Gte(), req.Since)
				if req.ProviderID.IsNotNil() {
					sq = sq.Where(cols.ProviderID.Eq(), req.ProviderID)
				}

				return sq
			}).
			Order(cols.ScoredAt.OrderDesc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list scored extraction shadow results", zap.Error(err))

			return nil, fmt.Errorf("list scored extraction shadow results: %w", err)
		}

		return entities, nil
	})
}

func (r *resultRepository) TotalsByStatus(
	ctx context.Context,
	req repositories.TotalExtractionShadowResultsRequest,
) ([]repositories.ExtractionShadowStatusTotal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]repositories.ExtractionShadowStatusTotal, error) {
		cols := buncolgen.ShadowResultColumns
		totals := make(
			[]repositories.ExtractionShadowStatusTotal,
			0,
			len(extractionshadow.AllResultStatuses()),
		)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*extractionshadow.ShadowResult)(nil)).
			Column(cols.Status.Bare()).
			ColumnExpr(buncolgen.Count("count")).
			ColumnExpr(cols.CostUSD.Expr("COALESCE(SUM({}), 0) AS cost_usd")).
			ColumnExpr(cols.LatencyMs.Expr("COALESCE(SUM({}) FILTER (WHERE {} > 0), 0) AS latency_ms_sum")).
			ColumnExpr(buncolgen.CountFilter("timed", cols.LatencyMs.Expr("{} > 0"))).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = buncolgen.ShadowResultScopeTenant(sq, req.TenantInfo).
					Where(cols.CreatedAt.Gte(), req.Since)
				if req.ProviderID.IsNotNil() {
					sq = sq.Where(cols.ProviderID.Eq(), req.ProviderID)
				}

				return sq
			}).
			Group(cols.Status.Bare()).
			Scan(ctx, &totals)
		if err != nil {
			r.l.Error("failed to total extraction shadow results", zap.Error(err))

			return nil, fmt.Errorf("total extraction shadow results: %w", err)
		}

		return totals, nil
	})
}

func (r *resultRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListExtractionShadowResultConnectionRequest,
) (*pagination.CursorListResult[*extractionshadow.ShadowResult], error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagination.CursorListResult[*extractionshadow.ShadowResult], error) {
		dba := r.db.DBForContext(ctx)

		var totalCount *int
		if req.Cursor.IncludeTotalCount {
			total, err := dba.NewSelect().
				Model((*extractionshadow.ShadowResult)(nil)).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = querybuilder.ApplyFiltersWithoutSort(
						sq,
						buncolgen.ShadowResultTable.Alias,
						req.Filter,
						(*extractionshadow.ShadowResult)(nil),
					)

					return sq.Apply(buncolgen.ShadowResultApplyTenant(req.Filter.TenantInfo))
				}).
				Count(ctx)
			if err != nil {
				r.l.Error("failed to count extraction shadow results", zap.Error(err))

				return nil, err
			}
			totalCount = &total
		}

		result, err := dbhelper.CursorList(
			ctx,
			dbhelper.CursorListParams[*extractionshadow.ShadowResult]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(entities *[]*extractionshadow.ShadowResult) *bun.SelectQuery {
					q := dba.NewSelect().Model(entities)
					if len(req.Columns) > 0 {
						q = q.Column(req.Columns...)
					}

					return q
				},
				Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
					return querybuilder.ApplyCursorFilters(
						sq,
						buncolgen.ShadowResultTable.Alias,
						req.Filter,
						req.Cursor,
						(*extractionshadow.ShadowResult)(nil),
					)
				},
			},
		)
		if err != nil {
			r.l.Error("failed to list extraction shadow results", zap.Error(err))

			return nil, err
		}

		return result, nil
	})
}

func (r *resultRepository) PurgeBefore(
	ctx context.Context,
	req repositories.PurgeExtractionShadowResultsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		cols := buncolgen.ShadowResultColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultPurgeLimit
		}
		limit = min(limit, maxPurgeLimit)

		dba := r.db.DBForContext(ctx)
		expired := dba.NewSelect().
			Model((*extractionshadow.ShadowResult)(nil)).
			Column(cols.ID.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ShadowResultScopeTenant(sq, req.TenantInfo).
					Where(cols.CreatedAt.Lt(), req.Before)
			}).
			OrderExpr(cols.CreatedAt.OrderAsc()).
			Limit(limit)

		res, err := dba.NewDelete().
			Model((*extractionshadow.ShadowResult)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ShadowResultScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.In(), expired)
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to purge extraction shadow results", zap.Error(err))

			return 0, fmt.Errorf("purge extraction shadow results: %w", err)
		}

		purged, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("purge extraction shadow results rows: %w", err)
		}

		return purged, nil
	})
}
