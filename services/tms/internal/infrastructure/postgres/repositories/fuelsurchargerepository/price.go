package fuelsurchargerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type PriceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type priceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPriceRepository(p PriceParams) repositories.FuelIndexPriceRepository {
	return &priceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fuel-index-price-repository"),
	}
}

func (r *priceRepository) UpsertPrices(
	ctx context.Context,
	prices []*fuelsurcharge.FuelIndexPrice,
) (int, error) {
	if len(prices) == 0 {
		return 0, nil
	}

	log := r.l.With(
		zap.String("operation", "UpsertPrices"),
		zap.Int("count", len(prices)),
	)

	inserted := 0
	err := r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		for idx := range prices {
			price := prices[idx]
			var isInsert bool
			iErr := tx.NewInsert().
				Model(price).
				On("CONFLICT (organization_id, business_unit_id, fuel_index_id, price_date) DO UPDATE").
				Set("price = EXCLUDED.price").
				Set("source_raw = EXCLUDED.source_raw").
				Set("fetched_at = EXCLUDED.fetched_at").
				Returning("(xmax = 0)").
				Scan(txCtx, &isInsert)
			if iErr != nil {
				return iErr
			}
			if isInsert {
				inserted++
			}
		}
		return nil
	})
	if err != nil {
		log.Error("failed to upsert fuel index prices", zap.Error(err))
		return 0, err
	}

	return inserted, nil
}

func (r *priceRepository) ListByIndex(
	ctx context.Context,
	req *repositories.ListFuelIndexPricesRequest,
) ([]*fuelsurcharge.FuelIndexPrice, error) {
	log := r.l.With(
		zap.String("operation", "ListByIndex"),
		zap.String("fuelIndexId", req.FuelIndexID.String()),
	)

	cols := buncolgen.FuelIndexPriceColumns
	entities := make([]*fuelsurcharge.FuelIndexPrice, 0)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FuelIndexPriceScopeTenant(sq, req.TenantInfo).
				Where(cols.FuelIndexID.Eq(), req.FuelIndexID)
			if req.From != "" {
				sq = sq.Where(cols.PriceDate.Gte(), req.From)
			}
			if req.To != "" {
				sq = sq.Where(cols.PriceDate.Lte(), req.To)
			}
			return sq
		}).
		Order(cols.PriceDate.OrderDesc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to list fuel index prices", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *priceRepository) GetLatestOnOrBefore(
	ctx context.Context,
	req *repositories.GetLatestFuelPricesRequest,
) ([]*fuelsurcharge.FuelIndexPrice, error) {
	log := r.l.With(
		zap.String("operation", "GetLatestOnOrBefore"),
		zap.String("fuelIndexId", req.FuelIndexID.String()),
		zap.String("date", req.Date),
	)

	limit := req.Limit
	if limit <= 0 {
		limit = 3
	}

	cols := buncolgen.FuelIndexPriceColumns
	entities := make([]*fuelsurcharge.FuelIndexPrice, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelIndexPriceScopeTenant(sq, req.TenantInfo).
				Where(cols.FuelIndexID.Eq(), req.FuelIndexID).
				Where(cols.PriceDate.Lte(), req.Date)
		}).
		Order(cols.PriceDate.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get latest fuel index prices", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *priceRepository) LatestPerIndex(
	ctx context.Context,
	req *repositories.LatestPricesPerIndexRequest,
) (map[pulid.ID][]*fuelsurcharge.FuelIndexPrice, error) {
	log := r.l.With(zap.String("operation", "LatestPerIndex"))

	perIndex := req.PerIndex
	if perIndex <= 0 {
		perIndex = 2
	}

	entities := make([]*fuelsurcharge.FuelIndexPrice, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ColumnExpr("fip.*").
		Where("fip.organization_id = ?", req.TenantInfo.OrgID).
		Where("fip.business_unit_id = ?", req.TenantInfo.BuID).
		Where(`(
			SELECT COUNT(*)
			FROM fuel_index_prices newer
			WHERE newer.fuel_index_id = fip.fuel_index_id
			  AND newer.organization_id = fip.organization_id
			  AND newer.business_unit_id = fip.business_unit_id
			  AND newer.price_date > fip.price_date
		) < ?`, perIndex).
		Order("fip.fuel_index_id ASC", "fip.price_date DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get latest prices per index", zap.Error(err))
		return nil, err
	}

	result := make(map[pulid.ID][]*fuelsurcharge.FuelIndexPrice, len(entities)/perIndex+1)
	for _, entity := range entities {
		result[entity.FuelIndexID] = append(result[entity.FuelIndexID], entity)
	}

	return result, nil
}

func (r *priceRepository) HasPriceForDate(
	ctx context.Context,
	req *repositories.HasFuelPriceForDateRequest,
) (bool, error) {
	cols := buncolgen.FuelIndexPriceColumns
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*fuelsurcharge.FuelIndexPrice)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelIndexPriceScopeTenant(sq, req.TenantInfo).
				Where(cols.FuelIndexID.Eq(), req.FuelIndexID).
				Where(cols.PriceDate.Eq(), req.Date)
		}).
		Exists(ctx)
	if err != nil {
		r.l.Error("failed to check fuel price existence", zap.Error(err))
		return false, err
	}

	return exists, nil
}

func (r *priceRepository) Create(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
) (*fuelsurcharge.FuelIndexPrice, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create fuel index price", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *priceRepository) Update(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
) (*fuelsurcharge.FuelIndexPrice, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update fuel index price", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "FuelIndexPrice", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *priceRepository) Delete(
	ctx context.Context,
	req *repositories.GetFuelIndexPriceByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.PriceID.String()),
	)

	cols := buncolgen.FuelIndexPriceColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*fuelsurcharge.FuelIndexPrice)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.FuelIndexPriceScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.PriceID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete fuel index price", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "FuelIndexPrice", req.PriceID.String())
}

func (r *priceRepository) GetByID(
	ctx context.Context,
	req *repositories.GetFuelIndexPriceByIDRequest,
) (*fuelsurcharge.FuelIndexPrice, error) {
	cols := buncolgen.FuelIndexPriceColumns
	entity := new(fuelsurcharge.FuelIndexPrice)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelIndexPriceScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.PriceID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "FuelIndexPrice")
	}

	return entity, nil
}
