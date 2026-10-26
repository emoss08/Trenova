package fuelpurchaserepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"go.uber.org/zap"
)

const (
	defaultDeskRowLimit = 5
	maxHeldCardRows     = 200
	maxRecentPurchases  = 50
)

func NewDeskCardReader(p Params) fuelpurchaseports.FuelDeskCardReader {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fuel-desk-card-reader"),
	}
}

func sumFilters(req *fuelpurchaseports.SumFuelPurchasesRequest) []keel.Pred {
	fp := fuelpurchasedb.FuelPurchases

	return []keel.Pred{
		when(!req.TractorID.IsNil(), fp.TractorID.Eq(req.TractorID)),
		when(!req.FuelCardID.IsNil(), fp.FuelCardID.Eq(req.FuelCardID)),
		when(!req.JurisdictionID.IsNil(), fp.JurisdictionID.Eq(req.JurisdictionID)),
		when(len(req.FuelTypes) > 0, fp.FuelType.In(req.FuelTypes...)),
		when(req.From > 0, fp.PurchasedAt.Gte(req.From)),
		when(req.To > 0, fp.PurchasedAt.Lt(req.To)),
	}
}

func (r *repository) SumPurchases(
	ctx context.Context,
	req *fuelpurchaseports.SumFuelPurchasesRequest,
) ([]*fuelpurchaseports.FuelSpendRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*fuelpurchaseports.FuelSpendRow, error) {
			fp := fuelpurchasedb.FuelPurchases
			total := fp.TotalAmountMinor.Sum()
			rows, err := keel.Into[fuelpurchaseports.FuelSpendRow](
				ctx,
				r.db.Keel(ctx),
				fp.Pick(
					fp.CurrencyCode,
					keel.Count(),
					fp.Gallons.Sum(),
					keel.NumericOf(fp.Gallons.Sum().Filter(fp.TaxPaid.IsFalse())).Coalesce(0),
					total,
					fp.PurchasedAt.Max(),
				).
					Where(sumFilters(req)...).
					Group(fp.CurrencyCode).
					Order(total.Desc()),
			)
			if err != nil {
				r.l.Error("failed to total fuel purchases", zap.Error(err))
				return nil, fmt.Errorf("total fuel purchases: %w", err)
			}

			return pointers(rows), nil
		},
	)
}

func (r *repository) SumGallonsByJurisdiction(
	ctx context.Context,
	req *fuelpurchaseports.SumFuelPurchasesRequest,
) ([]*fuelpurchaseports.FuelJurisdictionRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*fuelpurchaseports.FuelJurisdictionRow, error) {
			fp := fuelpurchasedb.FuelPurchases
			gallons := fp.Gallons.Sum()
			rows, err := keel.Into[fuelpurchaseports.FuelJurisdictionRow](
				ctx,
				r.db.Keel(ctx),
				fp.Pick(fp.JurisdictionID, gallons, keel.Count()).
					Where(sumFilters(req)...).
					Group(fp.JurisdictionID).
					Order(gallons.Desc(), fp.JurisdictionID.Asc()),
			)
			if err != nil {
				r.l.Error("failed to total fuel gallons by jurisdiction", zap.Error(err))
				return nil, fmt.Errorf("total fuel gallons by jurisdiction: %w", err)
			}

			return pointers(rows), nil
		},
	)
}

func (r *repository) GetPreviousFill(
	ctx context.Context,
	req *fuelpurchaseports.GetPreviousFillRequest,
) (*fuelpurchase.FuelPurchase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelPurchase, error) {
		fp := fuelpurchasedb.FuelPurchases
		current := req.Purchase
		entity, err := fp.Where(
			fp.TractorID.Eq(current.TractorID),
			fp.ID.Ne(current.ID),
			fp.Odometer.IsNotNull(),
			fp.FuelType.In(domaintypes.IFTAFuelTypes()...),
			keel.Tuple(fp.PurchasedAt, fp.ID).
				Lt(keel.Tuple(current.PurchasedAt, current.ID.String())),
		).Order(fp.PurchasedAt.Desc(), fp.ID.Desc()).First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
			r.l.Error("failed to read the previous fill", zap.Error(err))
			return nil, fmt.Errorf("read the previous fill: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) GetPurchaseImportRow(
	ctx context.Context,
	req *fuelpurchaseports.GetPurchaseImportRowRequest,
) (*fuelpurchase.ImportRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelpurchase.ImportRow, error) {
		ir := fuelpurchasedb.ImportRows
		row, err := ir.Where(
			ir.FuelPurchaseID.Eq(req.PurchaseID),
			ir.Status.Eq(fuelpurchase.ImportRowStatusCommitted),
		).Load(ir.Batch()).Order(ir.CreatedAt.Desc()).First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
			r.l.Error("failed to read a purchase's import row", zap.Error(err))
			return nil, fmt.Errorf("read a purchase's import row: %w", err)
		}

		return row, nil
	})
}

func (r *repository) ListHeldCardRows(
	ctx context.Context,
	req *fuelpurchaseports.ListHeldCardRowsRequest,
) ([]*fuelpurchase.ImportRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelpurchase.ImportRow, error) {
		ir := fuelpurchasedb.ImportRows
		b := fuelpurchasedb.ImportBatches
		openBatches := b.Pick(b.ID).Where(keel.Or(
			keel.And(
				b.Origin.Eq(fuelpurchase.ImportOriginFeed),
				b.Status.Ne(fuelpurchase.ImportStatusDiscarded),
			),
			keel.And(
				b.Origin.Eq(fuelpurchase.ImportOriginUpload),
				b.Status.In(fuelpurchase.ImportStatusPending, fuelpurchase.ImportStatusParsed),
			),
		))

		rows, err := ir.Where(
			ir.ResolvedFuelCardID.Eq(req.CardID),
			ir.FuelPurchaseID.IsNull(),
			ir.Status.In(fuelpurchase.ImportRowStatusNew, fuelpurchase.ImportRowStatusError),
			ir.ImportBatchID.InQuery(openBatches),
		).
			Load(ir.Batch()).
			Order(ir.CreatedAt.Desc(), ir.RowNumber.Asc()).
			Limit(limitOr(req.Limit, maxHeldCardRows, maxHeldCardRows)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list a card's held rows", zap.Error(err))
			return nil, fmt.Errorf("list a card's held rows: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ListRecentCardPurchases(
	ctx context.Context,
	req *fuelpurchaseports.ListRecentCardPurchasesRequest,
) ([]*fuelpurchase.FuelPurchase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelpurchase.FuelPurchase, error) {
		fp := fuelpurchasedb.FuelPurchases
		purchases, err := fp.Where(fp.FuelCardID.Eq(req.CardID)).
			Load(fp.Jurisdiction()).
			Order(fp.PurchasedAt.Desc(), fp.ID.Desc()).
			Limit(limitOr(req.Limit, defaultDeskRowLimit, maxRecentPurchases)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list a card's recent purchases", zap.Error(err))
			return nil, fmt.Errorf("list a card's recent purchases: %w", err)
		}

		return purchases, nil
	})
}
