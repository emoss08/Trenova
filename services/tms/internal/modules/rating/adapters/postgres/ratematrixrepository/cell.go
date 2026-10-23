package ratematrixrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/ratematrixcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) ListCells(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixCellsRequest,
) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
			cols := ratematrixcol.RateMatrixCellColumns
			entities := make([]*ratematrix.RateMatrixCell, 0, req.Filter.Pagination.SafeLimit())

			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return ratematrixcol.RateMatrixCellScopeTenant(sq, req.TenantInfo).
						Where(cols.RateMatrixID.Eq(), req.RateMatrixID).
						Where(cols.VersionNumber.Eq(), req.VersionNumber)
				}).
				Order(
					cols.D0Key.OrderAsc(),
					cols.D1Key.OrderAsc(),
					cols.D2Key.OrderAsc(),
					cols.D3Key.OrderAsc(),
					cols.D0Min.OrderAsc(),
					cols.D1Min.OrderAsc(),
					cols.D2Min.OrderAsc(),
					cols.D3Min.OrderAsc(),
					cols.ID.OrderAsc(),
				).
				Limit(int64(req.Filter.Pagination.SafeLimit())).
				Offset(int64(req.Filter.Pagination.SafeOffset())).
				ScanAndCount(ctx)
			if err != nil {
				r.l.Error("failed to list rate matrix cells",
					zap.String("matrixId", req.RateMatrixID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return &pagination.ListResult[*ratematrix.RateMatrixCell]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) ReplaceCells(
	ctx context.Context,
	req *ratematrixports.ReplaceRateMatrixCellsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := r.db.WithTx(ctx, kernelports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
			if dErr := r.deleteVersionCells(
				c,
				req.TenantInfo,
				req.RateMatrixID,
				ratematrix.DraftVersion,
			); dErr != nil {
				return dErr
			}

			for _, cell := range req.Cells {
				if cell == nil {
					continue
				}
				cell.ID = pulid.Nil
				cell.RateMatrixID = req.RateMatrixID
				cell.VersionNumber = ratematrix.DraftVersion
				cell.OrganizationID = req.TenantInfo.OrgID
				cell.BusinessUnitID = req.TenantInfo.BuID
			}

			return r.insertCellBatches(c, req.Cells)
		})
		if err != nil {
			r.l.Error("failed to replace rate matrix cells",
				zap.String("matrixId", req.RateMatrixID.String()),
				zap.Int("cells", len(req.Cells)),
				zap.Error(err),
			)
			return dberror.MapRetryableTransactionError(err, busyMessage)
		}

		return nil
	})
}

func (r *repository) deleteVersionCells(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	matrixID pulid.ID,
	versionNumber int64,
) error {
	cols := ratematrixcol.RateMatrixCellColumns
	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*ratematrix.RateMatrixCell)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return ratematrixcol.RateMatrixCellScopeTenantDelete(dq, tenantInfo).
				Where(cols.RateMatrixID.Eq(), matrixID).
				Where(cols.VersionNumber.Eq(), versionNumber)
		}).
		Exec(ctx)
	return err
}

func (r *repository) insertCellBatches(
	ctx context.Context,
	cells []*ratematrix.RateMatrixCell,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		kept := make([]*ratematrix.RateMatrixCell, 0, len(cells))
		for _, cell := range cells {
			if cell != nil {
				kept = append(kept, cell)
			}
		}
		for start := 0; start < len(kept); start += cellInsertBatch {
			batch := kept[start:min(start+cellInsertBatch, len(kept))]
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

func (r *repository) CopyVersion(
	ctx context.Context,
	req *ratematrixports.CopyVersionRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		dimensionCols := ratematrixcol.RateMatrixDimensionColumns
		dimensions := copyStatement{
			table:    ratematrixcol.RateMatrixDimensionTable,
			columns:  ratematrixcol.RateMatrixDimensionInsertableColumns,
			idPrefix: "rmd_",
			id:       dimensionCols.ID,
			version:  dimensionCols.VersionNumber,
			matrix:   dimensionCols.RateMatrixID,
			org:      dimensionCols.OrganizationID,
			bu:       dimensionCols.BusinessUnitID,
			created:  dimensionCols.CreatedAt,
			updated:  dimensionCols.UpdatedAt,
		}
		cellCols := ratematrixcol.RateMatrixCellColumns
		cells := copyStatement{
			table:    ratematrixcol.RateMatrixCellTable,
			columns:  ratematrixcol.RateMatrixCellInsertableColumns,
			idPrefix: "rmc_",
			id:       cellCols.ID,
			version:  cellCols.VersionNumber,
			matrix:   cellCols.RateMatrixID,
			org:      cellCols.OrganizationID,
			bu:       cellCols.BusinessUnitID,
			created:  cellCols.CreatedAt,
			updated:  cellCols.UpdatedAt,
		}

		var copied int64
		err := r.db.WithTx(ctx, kernelports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
			if _, dErr := r.db.DBForContext(c).
				NewDelete().
				Model((*ratematrix.RateMatrixDimension)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return ratematrixcol.RateMatrixDimensionScopeTenantDelete(dq, req.TenantInfo).
						Where(dimensionCols.RateMatrixID.Eq(), req.RateMatrixID).
						Where(dimensionCols.VersionNumber.Eq(), req.To)
				}).
				Exec(c); dErr != nil {
				return dErr
			}
			if dErr := r.deleteVersionCells(c, req.TenantInfo, req.RateMatrixID, req.To); dErr != nil {
				return dErr
			}
			if _, cErr := r.copyRows(c, &dimensions, req); cErr != nil {
				return cErr
			}
			count, cErr := r.copyRows(c, &cells, req)
			copied = count
			return cErr
		})
		if err != nil {
			r.l.Error("failed to copy rate matrix version",
				zap.String("matrixId", req.RateMatrixID.String()),
				zap.Int64("from", req.From),
				zap.Int64("to", req.To),
				zap.Error(err),
			)
			return 0, dberror.MapRetryableTransactionError(err, busyMessage)
		}
		return copied, nil
	})
}

type positionColumns struct {
	key     buncolgen.Column
	keyTo   buncolgen.Column
	minimum buncolgen.Column
}

func columnsAt(position int16) positionColumns {
	cols := ratematrixcol.RateMatrixCellColumns
	switch position {
	case 0:
		return positionColumns{key: cols.D0Key, keyTo: cols.D0KeyTo, minimum: cols.D0Min}
	case 1:
		return positionColumns{key: cols.D1Key, keyTo: cols.D1KeyTo, minimum: cols.D1Min}
	case 2:
		return positionColumns{key: cols.D2Key, keyTo: cols.D2KeyTo, minimum: cols.D2Min}
	default:
		return positionColumns{key: cols.D3Key, keyTo: cols.D3KeyTo, minimum: cols.D3Min}
	}
}

func misfitAt(position int16, dimension *ratematrix.RateMatrixDimension) string {
	columns := columnsAt(position)
	switch {
	case dimension == nil:
		return buncolgen.Expr(
			"({0} <> '' OR {1} <> '' OR {2} IS NOT NULL)",
			columns.key, columns.keyTo, columns.minimum,
		)
	case dimension.IsPostalRange():
		return buncolgen.Expr(
			"({0} = '' OR {1} = '' OR {2} IS NOT NULL)",
			columns.key, columns.keyTo, columns.minimum,
		)
	case dimension.MatchMode == ratematrix.MatchModeExact:
		return buncolgen.Expr(
			"({0} = '' OR {1} <> '' OR {2} IS NOT NULL)",
			columns.key, columns.keyTo, columns.minimum,
		)
	default:
		return buncolgen.Expr(
			"({0} <> '' OR {1} <> '' OR {2} IS NULL)",
			columns.key, columns.keyTo, columns.minimum,
		)
	}
}

func (r *repository) CountMisfitCells(
	ctx context.Context,
	req *ratematrixports.CountMisfitCellsRequest,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		byPosition := make(map[int16]*ratematrix.RateMatrixDimension, len(req.Dimensions))
		for _, dimension := range req.Dimensions {
			if dimension != nil {
				byPosition[dimension.Position] = dimension
			}
		}
		cols := ratematrixcol.RateMatrixCellColumns
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*ratematrix.RateMatrixCell)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixCellScopeTenant(sq, req.TenantInfo).
					Where(cols.RateMatrixID.Eq(), req.RateMatrixID).
					Where(cols.VersionNumber.Eq(), req.VersionNumber)
			}).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				for position := range int16(ratematrix.MaxDimensions) {
					sq = sq.WhereOr(misfitAt(position, byPosition[position]))
				}
				return sq
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count rate matrix cells that do not fit their axes",
				zap.String("matrixId", req.RateMatrixID.String()),
				zap.Error(err),
			)
			return 0, err
		}
		return int64(count), nil
	})
}

type copyStatement struct {
	table    buncolgen.TableInfo
	columns  []string
	idPrefix string
	id       buncolgen.Column
	version  buncolgen.Column
	matrix   buncolgen.Column
	org      buncolgen.Column
	bu       buncolgen.Column
	created  buncolgen.Column
	updated  buncolgen.Column
}

func (r *repository) copyRows(
	ctx context.Context,
	statement *copyStatement,
	req *ratematrixports.CopyVersionRequest,
) (int64, error) {
	targets := make([]string, 0, len(statement.columns))
	sources := make([]string, 0, len(statement.columns))
	args := make([]any, 0, 8)
	for _, column := range statement.columns {
		targets = append(targets, column)
		switch column {
		case statement.id.Name:
			sources = append(sources,
				"? || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 26))")
			args = append(args, statement.idPrefix)
		case statement.version.Name:
			sources = append(sources, "?")
			args = append(args, req.To)
		case statement.created.Name, statement.updated.Name:
			sources = append(sources, "extract(epoch from current_timestamp)::bigint")
		default:
			sources = append(sources, column)
		}
	}
	args = append(
		args,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		req.RateMatrixID,
		req.From,
	)

	query := "INSERT INTO " + statement.table.Name +
		" (" + strings.Join(targets, ", ") + ") SELECT " + strings.Join(sources, ", ") +
		" FROM " + statement.table.Name +
		" WHERE " + statement.org.Name + " = ? AND " + statement.bu.Name + " = ? AND " +
		statement.matrix.Name + " = ? AND " + statement.version.Name + " = ?"

	result, err := r.db.DBForContext(ctx).NewRaw(query, args...).Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
