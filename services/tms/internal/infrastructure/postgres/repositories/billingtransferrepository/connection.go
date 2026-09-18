package billingtransferrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func runItemScope(
	q *bun.SelectQuery,
	req *repositories.ListBillingTransferRunItemsRequest,
) *bun.SelectQuery {
	cols := buncolgen.BillingTransferRunItemColumns

	q = q.Where(cols.RunID.Eq(), req.RunID)
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}

	return q
}

// ListItems pages one run's report. The COUNT only runs when the caller asked
// for a total, because the CSV export walks every page and has no use for it.
func (r *repository) ListItems(
	ctx context.Context,
	req *repositories.ListBillingTransferRunItemsRequest,
) (*pagination.CursorListResult[*billingtransfer.BillingTransferRunItem], error) {
	log := r.l.With(zap.String("operation", "ListItems"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*billingtransfer.BillingTransferRunItem)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.BillingTransferRunItemTable.Alias,
					req.Filter,
					(*billingtransfer.BillingTransferRunItem)(nil),
				)
				return runItemScope(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count billing transfer run items", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*billingtransfer.BillingTransferRunItem]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*billingtransfer.BillingTransferRunItem) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.BillingTransferRunItemTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.BillingTransferRunItemTable.Alias,
					req.Filter,
					req.Cursor,
					(*billingtransfer.BillingTransferRunItem)(nil),
				)
				if applyErr != nil {
					return nil, applyErr
				}
				return runItemScope(sq, req), nil
			},
		})
	if err != nil {
		log.Error("failed to list billing transfer run items", zap.Error(err))
		return nil, err
	}

	return result, nil
}
