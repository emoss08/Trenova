package ratematrixrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/ratematrixcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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
	cols := ratematrixcol.RateMatrixCellColumns

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

func (r *repository) LoadTableCells(
	ctx context.Context,
	req *ratematrixports.LoadTableCellsRequest,
) ([]*ratematrix.RateMatrixCell, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrixCell, error) {
		cols := ratematrixcol.RateMatrixCellColumns
		entities := make([]*ratematrix.RateMatrixCell, 0, len(req.Keys)*4)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = ratematrixcol.RateMatrixCellScopeTenant(sq, req.TenantInfo).
					Where(cols.RateMatrixID.Eq(), req.RateMatrixID)
				for position, keys := range req.Keys {
					columns, ok := columnsForAxis(position)
					if !ok {
						continue
					}
					sq = sq.Where(columns.key.In(), bun.List(keys))
				}
				return sq
			}).
			Order(cols.ID.OrderAsc()).
			Scan(ctx)
		if err != nil {
			r.l.With(
				zap.String("operation", "LoadTableCells"),
				zap.String("matrixId", req.RateMatrixID.String()),
			).Error("failed to load rate table cells", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ListAxisKeys(
	ctx context.Context,
	req *ratematrixports.ListAxisKeysRequest,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		columns, ok := columnsForAxis(req.Position)
		if !ok {
			return nil, nil
		}
		cols := ratematrixcol.RateMatrixCellColumns
		keys := make([]string, 0, 32)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*ratematrix.RateMatrixCell)(nil)).
			ColumnExpr("DISTINCT "+columns.key.Qualified()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixCellScopeTenant(sq, req.TenantInfo).
					Where(cols.RateMatrixID.Eq(), req.RateMatrixID)
			}).
			Scan(ctx, &keys)
		if err != nil {
			r.l.With(zap.String("operation", "ListAxisKeys")).
				Error("failed to list rate table axis keys", zap.Error(err))
			return nil, err
		}

		return keys, nil
	})
}

func (r *repository) ListTableCells(
	ctx context.Context,
	req *ratematrixports.ListTableCellsRequest,
) ([]*ratematrix.RateMatrixCell, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrixCell, error) {
		if len(req.RateMatrixIDs) == 0 {
			return nil, nil
		}
		cols := ratematrixcol.RateMatrixCellColumns
		entities := make([]*ratematrix.RateMatrixCell, 0, len(req.RateMatrixIDs)*16)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixCellScopeTenant(sq, req.TenantInfo).
					Where(cols.RateMatrixID.In(), bun.List(req.RateMatrixIDs))
			}).
			Order(cols.RateMatrixID.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx)
		if err != nil {
			r.l.With(zap.String("operation", "ListTableCells")).
				Error("failed to list rate table cells", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ListCells(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixCellsRequest,
) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
			log := r.l.With(
				zap.String("operation", "ListCells"),
				zap.String("matrixId", req.RateMatrixID.String()),
			)

			cols := ratematrixcol.RateMatrixCellColumns
			entities := make([]*ratematrix.RateMatrixCell, 0, req.Filter.Pagination.SafeLimit())

			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return ratematrixcol.RateMatrixCellScopeTenant(sq, req.TenantInfo).
						Where(cols.RateMatrixID.Eq(), req.RateMatrixID)
				}).
				Order(cols.D0Key.OrderAsc()).
				Order(cols.D1Key.OrderAsc()).
				Order(cols.D2Min.OrderAsc()).
				Order(cols.D3Min.OrderAsc()).
				Limit(int64(req.Filter.Pagination.SafeLimit())).
				Offset(int64(req.Filter.Pagination.SafeOffset())).
				ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to list rate matrix cells", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*ratematrix.RateMatrixCell]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
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
	req *ratematrixports.ReplaceRateMatrixCellsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		log := r.l.With(
			zap.String("operation", "ReplaceCells"),
			zap.String("matrixId", req.RateMatrixID.String()),
			zap.Int("cells", len(req.Cells)),
		)

		cols := ratematrixcol.RateMatrixCellColumns

		err := r.db.WithTx(ctx, kernelports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
			if _, dErr := r.db.DBForContext(c).
				NewDelete().
				Model((*ratematrix.RateMatrixCell)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return ratematrixcol.RateMatrixCellScopeTenantDelete(dq, req.TenantInfo).
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

			if iErr := r.insertCellBatches(c, req.Cells); iErr != nil {
				return iErr
			}

			return r.bumpMatrixVersion(c, req)
		})
		if err != nil {
			log.Error("failed to replace rate matrix cells", zap.Error(err))
			return dberror.MapRetryableTransactionError(
				err,
				"Rate matrix is busy. Retry the request.",
			)
		}

		return nil
	})
}

func (r *repository) insertCellBatches(
	ctx context.Context,
	cells []*ratematrix.RateMatrixCell,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
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
	})
}

// GetLookupData reads every matrix a formula's lookup() or lookup2() call
// could address.
//
// The axis-count restriction (one or two) is applied in the database, not
// after the load: a class tariff with four axes and a hundred thousand cells
// must never ride along just to be discarded. What survives the filter is
// lookup-table sized — dozens to hundreds of cells — which is why loading it
// whole is affordable here and nowhere else.
func (r *repository) ListTables(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*ratematrix.RateMatrix, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrix, error) {
		matrixCols := ratematrixcol.RateMatrixColumns
		matrices := make([]*ratematrix.RateMatrix, 0, 8)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&matrices).
			Relation(buncolgen.Rel(ratematrixcol.RateMatrixRelations.Dimensions), orderDimensions).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixScopeTenant(sq, tenantInfo).
					Where(matrixCols.Status.Eq(), domaintypes.StatusActive)
			}).
			Order(matrixCols.Code.OrderAsc()).
			Scan(ctx)
		if err != nil {
			r.l.With(zap.String("operation", "ListTables")).
				Error("failed to list rate tables", zap.Error(err))
			return nil, err
		}

		tables := matrices[:0]
		for _, matrix := range matrices {
			if count := len(matrix.Dimensions); count >= 1 && count <= ratematrix.MaxDimensions {
				tables = append(tables, matrix)
			}
		}

		return tables, nil
	})
}

// bumpMatrixVersion records that the matrix's content changed even though no
// header field did, so lookup stamps and optimistic locks both see a new
// sheet as a new matrix.
func (r *repository) bumpMatrixVersion(
	ctx context.Context,
	req *ratematrixports.ReplaceRateMatrixCellsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		matrixCols := ratematrixcol.RateMatrixColumns

		_, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*ratematrix.RateMatrix)(nil)).
			Set("? = ? + 1", bun.Ident(matrixCols.Version.Name), bun.Ident(matrixCols.Version.Name)).
			Set(matrixCols.UpdatedAt.Set(), timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return ratematrixcol.RateMatrixScopeTenantUpdate(uq, req.TenantInfo).
					Where(matrixCols.ID.Eq(), req.RateMatrixID)
			}).
			Exec(ctx)

		return err
	})
}
