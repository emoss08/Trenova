package ratematrixversionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix/ratematrixdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"go.uber.org/zap"
)

const maxAxisRanges = 20_000

type axisColumns struct {
	key keel.StringCol
	to  keel.StringCol
}

func columnsForAxis(position int16) (axisColumns, bool) {
	c := ratematrixdb.RateMatrixCells

	switch position {
	case 0:
		return axisColumns{key: c.D0Key, to: c.D0KeyTo}, true
	case 1:
		return axisColumns{key: c.D1Key, to: c.D1KeyTo}, true
	case 2:
		return axisColumns{key: c.D2Key, to: c.D2KeyTo}, true
	case 3:
		return axisColumns{key: c.D3Key, to: c.D3KeyTo}, true
	default:
		return axisColumns{}, false
	}
}

func versionCells(key ratematrix.VersionKey) *keel.Query[*ratematrix.RateMatrixCell] {
	c := ratematrixdb.RateMatrixCells

	return c.Where(c.RateMatrixID.Eq(key.RateMatrixID), c.VersionNumber.Eq(key.VersionNumber))
}

func tableCellsQuery(
	req *ratematrixports.LoadTableCellsRequest,
) *keel.Query[*ratematrix.RateMatrixCell] {
	q := versionCells(req.Key)
	for position, keys := range req.Keys {
		if columns, ok := columnsForAxis(position); ok {
			q = q.Where(columns.key.In(keys...))
		}
	}

	return q.Order(ratematrixdb.RateMatrixCells.ID.Asc())
}

func (r *repository) LoadTableCells(
	ctx context.Context,
	req *ratematrixports.LoadTableCellsRequest,
) ([]*ratematrix.RateMatrixCell, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrixCell, error) {
		entities, err := tableCellsQuery(req).All(ctx, r.db.Keel(ctx))
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
		keys := uniqueKeys(req.Keys)
		if len(keys) == 0 {
			return nil, nil
		}

		c := ratematrixdb.RateMatrixCells
		wanted := versionKeys(keys)
		cells, err := c.With(wanted).
			Join(wanted, matchesKey(wanted, c.RateMatrixID.Col, c.VersionNumber.Col)...).
			Order(c.RateMatrixID.Asc(), c.VersionNumber.Asc(), c.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list rate table cells", zap.Error(err))
			return nil, err
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

		keys, err := keel.Into[string](ctx, r.db.Keel(ctx),
			versionCells(req.Key).Group(columns.key).Pick(columns.key))
		if err != nil {
			r.l.Error("failed to list rate table axis keys", zap.Error(err))
			return nil, err
		}

		return keys, nil
	})
}

type rangeRow struct {
	From string
	To   string
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

		rows, err := keel.Into[rangeRow](ctx, r.db.Keel(ctx), versionCells(req.Key).
			Group(columns.key, columns.to).
			Pick(columns.key, columns.to).
			Limit(maxAxisRanges))
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
