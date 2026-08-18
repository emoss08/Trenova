package ratematrixrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// axisColumns maps a dimension's position to the cell columns that hold it.
// The mapping is positional by design: it is what lets four axes live in fixed
// columns that one composite index can serve.
type axisColumns struct {
	key     buncolgen.Column
	minimum buncolgen.Column
	maximum buncolgen.Column
}

func columnsForAxis(position int16) (axisColumns, bool) {
	cols := buncolgen.RateMatrixCellColumns

	switch position {
	case 0:
		return axisColumns{key: cols.D0Key, minimum: cols.D0Min, maximum: cols.D0Max}, true
	case 1:
		return axisColumns{key: cols.D1Key, minimum: cols.D1Min, maximum: cols.D1Max}, true
	case 2:
		return axisColumns{key: cols.D2Key, minimum: cols.D2Min, maximum: cols.D2Max}, true
	case 3:
		return axisColumns{key: cols.D3Key, minimum: cols.D3Min, maximum: cols.D3Max}, true
	default:
		return axisColumns{}, false
	}
}

// LookupCells fetches only the cells that could price one lookup.
//
// Exact axes narrow by the candidate keys — several, because a place can belong
// to more than one zone. Banded axes narrow to the bands that contain the
// quantity, which is the half-open comparison the domain then re-applies when
// it picks the winner. Nothing here loads a whole matrix: a class tariff runs
// to hundreds of thousands of cells, and rating happens on every shipment
// write.
func (r *repository) LookupCells(
	ctx context.Context,
	req *repositories.LookupRateMatrixCellsRequest,
) ([]*ratematrix.RateMatrixCell, error) {
	log := r.l.With(
		zap.String("operation", "LookupCells"),
		zap.String("matrixId", req.RateMatrixID.String()),
	)

	cols := buncolgen.RateMatrixCellColumns
	entities := make([]*ratematrix.RateMatrixCell, 0, len(req.Axes)*2)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.RateMatrixCellScopeTenant(sq, req.TenantInfo).
				Where(cols.RateMatrixID.Eq(), req.RateMatrixID)

			for _, axis := range req.Axes {
				sq = applyAxisFilter(sq, axis)
			}

			return sq
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to look up rate matrix cells", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func applyAxisFilter(
	q *bun.SelectQuery,
	axis repositories.MatrixAxisQuery,
) *bun.SelectQuery {
	columns, ok := columnsForAxis(axis.Position)
	if !ok {
		return q
	}

	if len(axis.Keys) > 0 {
		return q.Where(columns.key.In(), bun.List(axis.Keys))
	}

	if !axis.Quantity.Valid {
		return q
	}

	// Bands are half open on the upper bound: a quantity sitting exactly on a
	// boundary belongs to the band above. An open bound means the band runs to
	// infinity in that direction.
	return q.
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(columns.minimum.IsNull()).
				WhereOr(columns.minimum.Lte(), axis.Quantity.Decimal)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(columns.maximum.IsNull()).
				WhereOr(columns.maximum.Gt(), axis.Quantity.Decimal)
		})
}

func (r *repository) ListCells(
	ctx context.Context,
	req *repositories.ListRateMatrixCellsRequest,
) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
	log := r.l.With(
		zap.String("operation", "ListCells"),
		zap.String("matrixId", req.RateMatrixID.String()),
	)

	cols := buncolgen.RateMatrixCellColumns
	entities := make([]*ratematrix.RateMatrixCell, 0, req.Filter.Pagination.SafeLimit())

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateMatrixCellScopeTenant(sq, req.TenantInfo).
				Where(cols.RateMatrixID.Eq(), req.RateMatrixID)
		}).
		Order(cols.D0Key.OrderAsc()).
		Order(cols.D1Key.OrderAsc()).
		Order(cols.D2Min.OrderAsc()).
		Order(cols.D3Min.OrderAsc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to list rate matrix cells", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*ratematrix.RateMatrixCell]{
		Items: entities,
		Total: total,
	}, nil
}

// ReplaceCells swaps a matrix's entire cell set in one transaction.
//
// Rate sheets arrive whole and are loaded whole, so replacing beats diffing:
// there is no identity in a cell worth preserving, and a diff would have to be
// certain it had matched every axis exactly to avoid leaving a stale row behind
// still pricing an intersection the new sheet dropped.
//
// The insert is batched because a class tariff is large enough that one
// statement would hold locks far longer than the work deserves.
func (r *repository) ReplaceCells(
	ctx context.Context,
	req *repositories.ReplaceRateMatrixCellsRequest,
) error {
	log := r.l.With(
		zap.String("operation", "ReplaceCells"),
		zap.String("matrixId", req.RateMatrixID.String()),
		zap.Int("cells", len(req.Cells)),
	)

	cols := buncolgen.RateMatrixCellColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*ratematrix.RateMatrixCell)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.RateMatrixCellScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.RateMatrixID.Eq(), req.RateMatrixID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		for _, cell := range req.Cells {
			if cell == nil {
				continue
			}
			cell.ID = pulid.Nil
			cell.RateMatrixID = req.RateMatrixID
			cell.OrganizationID = req.TenantInfo.OrgID
			cell.BusinessUnitID = req.TenantInfo.BuID
		}

		return r.insertCellBatches(c, req.Cells)
	})
	if err != nil {
		log.Error("failed to replace rate matrix cells", zap.Error(err))
		return dberror.MapRetryableTransactionError(
			err,
			"Rate matrix is busy. Retry the request.",
		)
	}

	return nil
}

func (r *repository) insertCellBatches(
	ctx context.Context,
	cells []*ratematrix.RateMatrixCell,
) error {
	for start := 0; start < len(cells); start += cellInsertBatch {
		end := min(start+cellInsertBatch, len(cells))

		batch := cells[start:end]
		if len(batch) == 0 {
			continue
		}

		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&batch).
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
