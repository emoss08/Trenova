package ratematrixrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix/ratematrixdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	copiedIDSQL  = "? || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 26))"
	copiedNowSQL = "extract(epoch from current_timestamp)::bigint"
)

func cellsQuery(
	req *ratematrixports.ListRateMatrixCellsRequest,
) *keel.Query[ratematrix.RateMatrixCell] {
	c := ratematrixdb.RateMatrixCells

	return c.Where(c.RateMatrixID.Eq(req.RateMatrixID), c.VersionNumber.Eq(req.VersionNumber))
}

func (r *repository) ListCells(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixCellsRequest,
) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
			c := ratematrixdb.RateMatrixCells
			page := req.Filter.Pagination
			rows := cellsQuery(req).
				Order(
					c.D0Key.Asc(), c.D1Key.Asc(), c.D2Key.Asc(), c.D3Key.Asc(),
					c.D0Min.Asc(), c.D1Min.Asc(), c.D2Min.Asc(), c.D3Min.Asc(),
					c.ID.Asc(),
				).
				Limit(page.SafeLimit()).
				Offset(page.SafeOffset())

			var items []*ratematrix.RateMatrixCell
			var total int64
			err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.ListInto(rows, &items),
				keel.CountInto(cellsQuery(req), &total),
			)
			if err != nil {
				r.l.Error("failed to list rate matrix cells",
					zap.String("matrixId", req.RateMatrixID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return &pagination.ListResult[*ratematrix.RateMatrixCell]{
				Items: items,
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
		db := r.db.Keel(ctx)
		err := deleteVersionCells(ctx, db, req.RateMatrixID, ratematrix.DraftVersion)
		if err == nil {
			_, err = ratematrixdb.RateMatrixCells.Copy(ctx, db, draftCells(req))
		}
		if err != nil {
			r.l.Error("failed to replace rate matrix cells",
				zap.String("matrixId", req.RateMatrixID.String()),
				zap.Int("cells", len(req.Cells)),
				zap.Error(err),
			)
			return mapRateMatrixError(err, req.RateMatrixID.String())
		}

		return nil
	})
}

func draftCells(req *ratematrixports.ReplaceRateMatrixCellsRequest) []*ratematrix.RateMatrixCell {
	kept := make([]*ratematrix.RateMatrixCell, 0, len(req.Cells))
	for _, cell := range req.Cells {
		if cell == nil {
			continue
		}
		cell.ID = pulid.Nil
		cell.RateMatrixID = req.RateMatrixID
		cell.VersionNumber = ratematrix.DraftVersion
		cell.OrganizationID = req.TenantInfo.OrgID
		cell.BusinessUnitID = req.TenantInfo.BuID
		kept = append(kept, cell)
	}

	return kept
}

func deleteVersionCells(
	ctx context.Context,
	db *keel.DB,
	matrixID pulid.ID,
	versionNumber int64,
) error {
	c := ratematrixdb.RateMatrixCells
	_, err := c.Where(c.RateMatrixID.Eq(matrixID), c.VersionNumber.Eq(versionNumber)).
		Delete(ctx, db)

	return err
}

func (r *repository) CopyVersion(
	ctx context.Context,
	req *ratematrixports.CopyVersionRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		copied, err := copyVersion(ctx, r.db.Keel(ctx), req)
		if err != nil {
			r.l.Error("failed to copy rate matrix version",
				zap.String("matrixId", req.RateMatrixID.String()),
				zap.Int64("from", req.From),
				zap.Int64("to", req.To),
				zap.Error(err),
			)
			return 0, mapRateMatrixError(err, req.RateMatrixID.String())
		}

		return copied, nil
	})
}

func copyVersion(
	ctx context.Context,
	db *keel.DB,
	req *ratematrixports.CopyVersionRequest,
) (int64, error) {
	d := ratematrixdb.RateMatrixDimensions
	if _, err := d.Where(d.RateMatrixID.Eq(req.RateMatrixID), d.VersionNumber.Eq(req.To)).
		Delete(ctx, db); err != nil {
		return 0, err
	}
	if err := deleteVersionCells(ctx, db, req.RateMatrixID, req.To); err != nil {
		return 0, err
	}

	if _, err := copyDimensions(ctx, db, req); err != nil {
		return 0, err
	}

	return copyCells(ctx, db, req)
}

func copiedID(prefix string) keel.Expr {
	return keel.Raw(copiedIDSQL, prefix)
}

func copiedNow() keel.Expr {
	return keel.Raw(copiedNowSQL)
}

func copyDimensions(
	ctx context.Context,
	db *keel.DB,
	req *ratematrixports.CopyVersionRequest,
) (int64, error) {
	d := ratematrixdb.RateMatrixDimensions
	src := d.Where(d.RateMatrixID.Eq(req.RateMatrixID), d.VersionNumber.Eq(req.From)).Pick(
		copiedID(d.TableRef().Table.IDPrefix), d.RateMatrixID, versionNumber(req.To), d.Position,
		d.Kind, d.MatchMode, d.Label, d.KeyNormalization, d.RangeOverflow,
		copiedNow(), copiedNow(),
	)

	return d.InsertFrom(ctx, db, src,
		d.ID, d.RateMatrixID, d.VersionNumber, d.Position,
		d.Kind, d.MatchMode, d.Label, d.KeyNormalization, d.RangeOverflow,
		d.CreatedAt, d.UpdatedAt,
	)
}

func copyCells(
	ctx context.Context,
	db *keel.DB,
	req *ratematrixports.CopyVersionRequest,
) (int64, error) {
	c := ratematrixdb.RateMatrixCells
	src := c.Where(c.RateMatrixID.Eq(req.RateMatrixID), c.VersionNumber.Eq(req.From)).Pick(
		copiedID(c.TableRef().Table.IDPrefix), c.RateMatrixID, versionNumber(req.To),
		c.D0Key, c.D1Key, c.D2Key, c.D3Key,
		c.D0KeyTo, c.D1KeyTo, c.D2KeyTo, c.D3KeyTo,
		c.D0Min, c.D0Max, c.D1Min, c.D1Max, c.D2Min, c.D2Max, c.D3Min, c.D3Max,
		c.Value, c.MinCharge, c.DeficitEligible,
		copiedNow(), copiedNow(),
	)

	return c.InsertFrom(ctx, db, src,
		c.ID, c.RateMatrixID, c.VersionNumber,
		c.D0Key, c.D1Key, c.D2Key, c.D3Key,
		c.D0KeyTo, c.D1KeyTo, c.D2KeyTo, c.D3KeyTo,
		c.D0Min, c.D0Max, c.D1Min, c.D1Max, c.D2Min, c.D2Max, c.D3Min, c.D3Max,
		c.Value, c.MinCharge, c.DeficitEligible,
		c.CreatedAt, c.UpdatedAt,
	)
}

type positionColumns struct {
	key     keel.StringCol
	keyTo   keel.StringCol
	minimum keel.Col
}

func columnsAt(position int16) positionColumns {
	c := ratematrixdb.RateMatrixCells
	switch position {
	case 0:
		return positionColumns{key: c.D0Key, keyTo: c.D0KeyTo, minimum: c.D0Min.Col}
	case 1:
		return positionColumns{key: c.D1Key, keyTo: c.D1KeyTo, minimum: c.D1Min.Col}
	case 2:
		return positionColumns{key: c.D2Key, keyTo: c.D2KeyTo, minimum: c.D2Min.Col}
	default:
		return positionColumns{key: c.D3Key, keyTo: c.D3KeyTo, minimum: c.D3Min.Col}
	}
}

func misfitAt(position int16, dimension *ratematrix.RateMatrixDimension) keel.Pred {
	columns := columnsAt(position)
	switch {
	case dimension == nil:
		return keel.Or(columns.key.Ne(""), columns.keyTo.Ne(""), columns.minimum.IsNotNull())
	case dimension.IsPostalRange():
		return keel.Or(columns.key.Eq(""), columns.keyTo.Eq(""), columns.minimum.IsNotNull())
	case dimension.MatchMode == ratematrix.MatchModeExact:
		return keel.Or(columns.key.Eq(""), columns.keyTo.Ne(""), columns.minimum.IsNotNull())
	default:
		return keel.Or(columns.key.Ne(""), columns.keyTo.Ne(""), columns.minimum.IsNull())
	}
}

func misfitQuery(
	req *ratematrixports.CountMisfitCellsRequest,
) *keel.Query[ratematrix.RateMatrixCell] {
	byPosition := make(map[int16]*ratematrix.RateMatrixDimension, len(req.Dimensions))
	for _, dimension := range req.Dimensions {
		if dimension != nil {
			byPosition[dimension.Position] = dimension
		}
	}

	misfits := make([]keel.Pred, 0, ratematrix.MaxDimensions)
	for position := range int16(ratematrix.MaxDimensions) {
		misfits = append(misfits, misfitAt(position, byPosition[position]))
	}

	c := ratematrixdb.RateMatrixCells

	return c.Where(
		c.RateMatrixID.Eq(req.RateMatrixID),
		c.VersionNumber.Eq(req.VersionNumber),
		keel.Or(misfits...),
	)
}

func (r *repository) CountMisfitCells(
	ctx context.Context,
	req *ratematrixports.CountMisfitCellsRequest,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		count, err := misfitQuery(req).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count rate matrix cells that do not fit their axes",
				zap.String("matrixId", req.RateMatrixID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return count, nil
	})
}

func versionNumber(n int64) keel.Expr {
	return keel.Raw("?::bigint", n)
}
