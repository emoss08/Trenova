package ratematrixversionrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/ratematrixcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const maxAxisRanges = 20_000

type axisColumns struct {
	key buncolgen.Column
	to  buncolgen.Column
}

func columnsForAxis(position int16) (axisColumns, bool) {
	cols := ratematrixcol.RateMatrixCellColumns

	switch position {
	case 0:
		return axisColumns{key: cols.D0Key, to: cols.D0KeyTo}, true
	case 1:
		return axisColumns{key: cols.D1Key, to: cols.D1KeyTo}, true
	case 2:
		return axisColumns{key: cols.D2Key, to: cols.D2KeyTo}, true
	case 3:
		return axisColumns{key: cols.D3Key, to: cols.D3KeyTo}, true
	default:
		return axisColumns{}, false
	}
}

func scopeVersionCells(
	sq *bun.SelectQuery,
	req *ratematrixports.ListAxisKeysRequest,
) *bun.SelectQuery {
	cols := ratematrixcol.RateMatrixCellColumns
	return ratematrixcol.RateMatrixCellScopeTenant(sq, req.TenantInfo).
		Where(cols.RateMatrixID.Eq(), req.Key.RateMatrixID).
		Where(cols.VersionNumber.Eq(), req.Key.VersionNumber)
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
				sq = scopeVersionCells(sq, &ratematrixports.ListAxisKeysRequest{
					TenantInfo: req.TenantInfo,
					Key:        req.Key,
				})
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
			r.l.Error("failed to load rate table cells",
				zap.String("matrixId", req.Key.RateMatrixID.String()),
				zap.Int64("versionNumber", req.Key.VersionNumber),
				zap.Error(err),
			)
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ListTableCells(
	ctx context.Context,
	req *ratematrixports.ListTableCellsRequest,
) ([]*ratematrix.RateMatrixCell, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrixCell, error) {
		if len(req.Keys) == 0 {
			return nil, nil
		}
		matrixIDs, numbers := splitKeys(req.Keys)
		cols := ratematrixcol.RateMatrixCellColumns
		candidates := make([]*ratematrix.RateMatrixCell, 0, len(req.Keys)*16)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&candidates).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixCellScopeTenant(sq, req.TenantInfo).
					Where(cols.RateMatrixID.In(), bun.List(matrixIDs)).
					Where(cols.VersionNumber.In(), bun.List(numbers))
			}).
			Order(cols.RateMatrixID.OrderAsc(), cols.VersionNumber.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list rate table cells", zap.Error(err))
			return nil, err
		}

		wanted := make(map[ratematrix.VersionKey]struct{}, len(req.Keys))
		for _, key := range req.Keys {
			wanted[key] = struct{}{}
		}
		cells := candidates[:0]
		for _, cell := range candidates {
			key := ratematrix.VersionKey{
				RateMatrixID:  cell.RateMatrixID,
				VersionNumber: cell.VersionNumber,
			}
			if _, ok := wanted[key]; ok {
				cells = append(cells, cell)
			}
		}
		return cells, nil
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
		keys := make([]string, 0, 32)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*ratematrix.RateMatrixCell)(nil)).
			ColumnExpr("DISTINCT "+columns.key.Qualified()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeVersionCells(sq, req)
			}).
			Scan(ctx, &keys)
		if err != nil {
			r.l.Error("failed to list rate table axis keys", zap.Error(err))
			return nil, err
		}

		return keys, nil
	})
}

func (r *repository) ListAxisRanges(
	ctx context.Context,
	req *ratematrixports.ListAxisKeysRequest,
) ([]ratematrix.PostalRange, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]ratematrix.PostalRange, error) {
		columns, ok := columnsForAxis(req.Position)
		if !ok {
			return nil, nil
		}

		type rangeRow struct {
			From string `bun:"range_from"`
			To   string `bun:"range_to"`
		}
		rows := make([]rangeRow, 0, 32)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*ratematrix.RateMatrixCell)(nil)).
			ColumnExpr("DISTINCT "+columns.key.As("range_from")).
			ColumnExpr(columns.to.As("range_to")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeVersionCells(sq, req)
			}).
			Limit(maxAxisRanges).
			Scan(ctx, &rows)
		if err != nil {
			r.l.Error("failed to list rate table axis ranges", zap.Error(err))
			return nil, err
		}

		ranges := make([]ratematrix.PostalRange, 0, len(rows))
		for _, row := range rows {
			ranges = append(ranges, ratematrix.PostalRange{From: row.From, To: row.To})
		}
		return ranges, nil
	})
}
