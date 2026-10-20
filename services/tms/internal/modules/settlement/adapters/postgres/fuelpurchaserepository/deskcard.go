package fuelpurchaserepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/fuelpurchasecol"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/uptrace/bun"
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

func scopeSums(
	sq *bun.SelectQuery,
	req *fuelpurchaseports.SumFuelPurchasesRequest,
) *bun.SelectQuery {
	cols := fuelpurchasecol.FuelPurchaseColumns

	sq = fuelpurchasecol.FuelPurchaseScopeTenant(sq, req.TenantInfo)
	if !req.TractorID.IsNil() {
		sq = sq.Where(cols.TractorID.Eq(), req.TractorID)
	}
	if !req.FuelCardID.IsNil() {
		sq = sq.Where(cols.FuelCardID.Eq(), req.FuelCardID)
	}
	if !req.JurisdictionID.IsNil() {
		sq = sq.Where(cols.JurisdictionID.Eq(), req.JurisdictionID)
	}
	if len(req.FuelTypes) > 0 {
		sq = sq.Where(cols.FuelType.In(), bun.List(req.FuelTypes))
	}
	if req.From > 0 {
		sq = sq.Where(cols.PurchasedAt.Gte(), req.From)
	}
	if req.To > 0 {
		sq = sq.Where(cols.PurchasedAt.Lt(), req.To)
	}

	return sq
}

func (r *repository) SumPurchases(
	ctx context.Context,
	req *fuelpurchaseports.SumFuelPurchasesRequest,
) ([]*fuelpurchaseports.FuelSpendRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*fuelpurchaseports.FuelSpendRow, error) {
			cols := fuelpurchasecol.FuelPurchaseColumns
			rows := make([]*fuelpurchaseports.FuelSpendRow, 0, 2)

			err := r.db.DBForContext(ctx).
				NewSelect().
				Model((*fuelpurchase.FuelPurchase)(nil)).
				ColumnExpr(cols.CurrencyCode.As("currency_code")).
				ColumnExpr(buncolgen.Count("purchase_count")).
				ColumnExpr(buncolgen.Sum(cols.Gallons, "gallons")).
				ColumnExpr(buncolgen.Expr(
					"COALESCE(SUM({0}) FILTER (WHERE NOT {1}), 0) AS untaxed_gallons",
					cols.Gallons,
					cols.TaxPaid,
				)).
				ColumnExpr(buncolgen.Sum(cols.TotalAmountMinor, "total_amount_minor")).
				ColumnExpr(buncolgen.Max(cols.PurchasedAt, "last_purchased_at")).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return scopeSums(sq, req)
				}).
				GroupExpr(cols.CurrencyCode.Qualified()).
				OrderExpr("total_amount_minor DESC").
				Scan(ctx, &rows)
			if err != nil {
				r.l.Error("failed to total fuel purchases", zap.Error(err))
				return nil, fmt.Errorf("total fuel purchases: %w", err)
			}

			return rows, nil
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
			cols := fuelpurchasecol.FuelPurchaseColumns
			rows := make([]*fuelpurchaseports.FuelJurisdictionRow, 0, 8)

			err := r.db.DBForContext(ctx).
				NewSelect().
				Model((*fuelpurchase.FuelPurchase)(nil)).
				ColumnExpr(cols.JurisdictionID.As("jurisdiction_id")).
				ColumnExpr(buncolgen.Sum(cols.Gallons, "gallons")).
				ColumnExpr(buncolgen.Count("purchase_count")).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return scopeSums(sq, req)
				}).
				GroupExpr(cols.JurisdictionID.Qualified()).
				OrderExpr("gallons DESC").
				OrderExpr(cols.JurisdictionID.OrderAsc()).
				Scan(ctx, &rows)
			if err != nil {
				r.l.Error("failed to total fuel gallons by jurisdiction", zap.Error(err))
				return nil, fmt.Errorf("total fuel gallons by jurisdiction: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) GetPreviousFill(
	ctx context.Context,
	req *fuelpurchaseports.GetPreviousFillRequest,
) (*fuelpurchase.FuelPurchase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelPurchase, error) {
		cols := fuelpurchasecol.FuelPurchaseColumns
		current := req.Purchase
		entity := new(fuelpurchase.FuelPurchase)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return fuelpurchasecol.FuelPurchaseScopeTenant(sq, req.TenantInfo).
					Where(cols.TractorID.Eq(), current.TractorID).
					Where(cols.ID.NotEq(), current.ID).
					Where(cols.Odometer.IsNotNull()).
					Where(cols.FuelType.In(), bun.List(domaintypes.IFTAFuelTypes())).
					Where(
						buncolgen.Expr("({0}, {1}) < (?, ?)", cols.PurchasedAt, cols.ID),
						current.PurchasedAt,
						current.ID,
					)
			}).
			Order(cols.PurchasedAt.OrderDesc()).
			Order(cols.ID.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
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
		cols := fuelpurchasecol.ImportRowColumns
		row := new(fuelpurchase.ImportRow)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(row).
			Relation(fuelpurchasecol.ImportRowRelations.Batch).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return fuelpurchasecol.ImportRowScopeTenant(sq, req.TenantInfo).
					Where(cols.FuelPurchaseID.Eq(), req.PurchaseID).
					Where(cols.Status.Eq(), fuelpurchase.ImportRowStatusCommitted)
			}).
			Order(cols.CreatedAt.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
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
		cols := fuelpurchasecol.ImportRowColumns
		batchCols := fuelpurchasecol.ImportBatchColumns
		dba := r.db.DBForContext(ctx)
		limit := limitOr(req.Limit, maxHeldCardRows, maxHeldCardRows)

		openBatches := dba.NewSelect().
			Model((*fuelpurchase.ImportBatch)(nil)).
			Column(batchCols.ID.String()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return fuelpurchasecol.ImportBatchScopeTenant(sq, req.TenantInfo).
					WhereGroup(" AND ", func(open *bun.SelectQuery) *bun.SelectQuery {
						return open.
							WhereGroup(" OR ", func(feed *bun.SelectQuery) *bun.SelectQuery {
								return feed.
									Where(batchCols.Origin.Eq(), fuelpurchase.ImportOriginFeed).
									Where(batchCols.Status.NotEq(), fuelpurchase.ImportStatusDiscarded)
							}).
							WhereGroup(" OR ", func(upload *bun.SelectQuery) *bun.SelectQuery {
								return upload.
									Where(batchCols.Origin.Eq(), fuelpurchase.ImportOriginUpload).
									Where(batchCols.Status.In(), bun.List([]fuelpurchase.ImportStatus{
										fuelpurchase.ImportStatusPending,
										fuelpurchase.ImportStatusParsed,
									}))
							})
					})
			})

		rows := make([]*fuelpurchase.ImportRow, 0, defaultDeskRowLimit)
		err := dba.NewSelect().
			Model(&rows).
			Relation(fuelpurchasecol.ImportRowRelations.Batch).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return fuelpurchasecol.ImportRowScopeTenant(sq, req.TenantInfo).
					Where(cols.ResolvedFuelCardID.Eq(), req.CardID).
					Where(cols.FuelPurchaseID.IsNull()).
					Where(cols.Status.In(), bun.List([]fuelpurchase.ImportRowStatus{
						fuelpurchase.ImportRowStatusNew,
						fuelpurchase.ImportRowStatusError,
					})).
					Where(cols.ImportBatchID.In(), openBatches)
			}).
			Order(cols.CreatedAt.OrderDesc()).
			Order(cols.RowNumber.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx)
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
		cols := fuelpurchasecol.FuelPurchaseColumns
		limit := limitOr(req.Limit, defaultDeskRowLimit, maxRecentPurchases)
		purchases := make([]*fuelpurchase.FuelPurchase, 0, limit)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&purchases).
			Relation(fuelpurchasecol.FuelPurchaseRelations.Jurisdiction).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return fuelpurchasecol.FuelPurchaseScopeTenant(sq, req.TenantInfo).
					Where(cols.FuelCardID.Eq(), req.CardID)
			}).
			Order(cols.PurchasedAt.OrderDesc()).
			Order(cols.ID.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list a card's recent purchases", zap.Error(err))
			return nil, fmt.Errorf("list a card's recent purchases: %w", err)
		}

		return purchases, nil
	})
}
