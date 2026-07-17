package reportrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func definitionVisibilityScope(
	q *bun.SelectQuery,
	req *repositories.ListReportDefinitionConnectionRequest,
) *bun.SelectQuery {
	if req.ViewerID.IsNil() {
		return q
	}
	cols := buncolgen.ReportDefinitionColumns
	return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			Where(cols.Visibility.Eq(), report.VisibilityShared).
			WhereOr(cols.OwnerID.Eq(), req.ViewerID)
	})
}

func (r *definitionRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListReportDefinitionConnectionRequest,
) (*pagination.CursorListResult[*report.ReportDefinition], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*report.ReportDefinition)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.ReportDefinitionTable.Alias,
				req.Filter,
				(*report.ReportDefinition)(nil),
			)
			return definitionVisibilityScope(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count report definitions", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*report.ReportDefinition]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*report.ReportDefinition) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.ReportDefinitionTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.ReportDefinitionTable.Alias,
					req.Filter,
					req.Cursor,
					(*report.ReportDefinition)(nil),
				)
				if applyErr != nil {
					return nil, applyErr
				}
				return definitionVisibilityScope(sq, req), nil
			},
		})
	if err != nil {
		log.Error("failed to list report definitions", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func runConnectionScope(
	q *bun.SelectQuery,
	req *repositories.ListReportRunConnectionRequest,
) *bun.SelectQuery {
	cols := buncolgen.ReportRunColumns

	if !req.DefinitionID.IsNil() {
		q = q.Where(cols.DefinitionID.Eq(), req.DefinitionID)
	}
	if !req.RequestedBy.IsNil() {
		q = q.Where(cols.RequestedByID.Eq(), req.RequestedBy)
	}
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	return q
}

func (r *runRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListReportRunConnectionRequest,
) (*pagination.CursorListResult[*report.ReportRun], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*report.ReportRun)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.ReportRunTable.Alias,
				req.Filter,
				(*report.ReportRun)(nil),
			)
			return runConnectionScope(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count report runs", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*report.ReportRun]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*report.ReportRun) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.ReportRunTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.ReportRunTable.Alias,
					req.Filter,
					req.Cursor,
					(*report.ReportRun)(nil),
				)
				if applyErr != nil {
					return nil, applyErr
				}
				return runConnectionScope(sq, req), nil
			},
		})
	if err != nil {
		log.Error("failed to list report runs", zap.Error(err))
		return nil, err
	}

	return result, nil
}
