package fuelsurchargerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge/fuelsurchargedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultLatestPrices   = 3
	defaultPricesPerIndex = 2
	latestPriceCeiling    = "9999-12-31"
)

var fuelPriceErrors = dberror.Mapping{
	Entity: "FuelIndexPrice",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_fuel_index_prices_week",
			Field:   "priceDate",
			Message: "This fuel index already has a price for that date",
		},
	},
}

type PriceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type priceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPriceRepository(p PriceParams) fuelsurchargeports.FuelIndexPriceRepository {
	return &priceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fuel-index-price-repository"),
	}
}

type priceKey struct {
	orgID       pulid.ID
	buID        pulid.ID
	fuelIndexID pulid.ID
	priceDate   string
}

func lastPricePerKey(prices []*fuelsurcharge.FuelIndexPrice) []*fuelsurcharge.FuelIndexPrice {
	position := make(map[priceKey]int, len(prices))
	unique := make([]*fuelsurcharge.FuelIndexPrice, 0, len(prices))
	for _, price := range prices {
		if price == nil {
			continue
		}
		key := priceKey{
			price.OrganizationID,
			price.BusinessUnitID,
			price.FuelIndexID,
			price.PriceDate,
		}
		if at, ok := position[key]; ok {
			unique[at] = price
			continue
		}
		position[key] = len(unique)
		unique = append(unique, price)
	}

	return unique
}

func (r *priceRepository) UpsertPrices(
	ctx context.Context,
	prices []*fuelsurcharge.FuelIndexPrice,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		rows := lastPricePerKey(prices)
		if len(rows) == 0 {
			return 0, nil
		}

		p := fuelsurchargedb.FuelIndexPrices
		inserted, err := p.Upsert(ctx, r.db.Keel(ctx), rows,
			keel.OnConflict(p.OrganizationID, p.BusinessUnitID, p.FuelIndexID, p.PriceDate).
				Update(p.Price, p.SourceRaw, p.FetchedAt))
		if err != nil {
			r.l.Error("failed to upsert fuel index prices",
				zap.Int("count", len(rows)),
				zap.Error(err),
			)
			return 0, err
		}

		count := 0
		for _, isInsert := range inserted {
			if isInsert {
				count++
			}
		}

		return count, nil
	})
}

func pricesByIndexQuery(
	req *fuelsurchargeports.ListFuelIndexPricesRequest,
) *keel.Query[fuelsurcharge.FuelIndexPrice] {
	p := fuelsurchargedb.FuelIndexPrices

	return p.Where(
		p.FuelIndexID.Eq(req.FuelIndexID),
		keel.When(req.From != "", p.PriceDate.Gte(req.From)),
		keel.When(req.To != "", p.PriceDate.Lte(req.To)),
	).Order(p.PriceDate.Desc())
}

func (r *priceRepository) ListByIndex(
	ctx context.Context,
	req *fuelsurchargeports.ListFuelIndexPricesRequest,
) ([]*fuelsurcharge.FuelIndexPrice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelsurcharge.FuelIndexPrice, error) {
		q := pricesByIndexQuery(req)

		var (
			entities []*fuelsurcharge.FuelIndexPrice
			err      error
		)
		if req.Limit > 0 {
			entities, err = q.Limit(req.Limit).List(ctx, r.db.Keel(ctx))
		} else {
			entities, err = q.All(ctx, r.db.Keel(ctx))
		}
		if err != nil {
			r.l.Error("failed to list fuel index prices",
				zap.String("fuelIndexId", req.FuelIndexID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entities, nil
	})
}

func (r *priceRepository) GetLatestOnOrBefore(
	ctx context.Context,
	req *fuelsurchargeports.GetLatestFuelPricesRequest,
) ([]*fuelsurcharge.FuelIndexPrice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelsurcharge.FuelIndexPrice, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultLatestPrices
		}

		p := fuelsurchargedb.FuelIndexPrices
		entities, err := p.Where(p.FuelIndexID.Eq(req.FuelIndexID), p.PriceDate.Lte(req.Date)).
			Order(p.PriceDate.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get latest fuel index prices",
				zap.String("fuelIndexId", req.FuelIndexID.String()),
				zap.String("date", req.Date),
				zap.Error(err),
			)
			return nil, err
		}

		return entities, nil
	})
}

func latestPerIndexQuery(perIndex int, ceiling string) *keel.Query[fuelsurcharge.FuelIndexPrice] {
	p := fuelsurchargedb.FuelIndexPrices
	newer := p.As("newer")
	newerCount := keel.Scalar(newer.Pick(keel.Count()).Where(
		newer.FuelIndexID.EqCol(p.FuelIndexID),
		keel.Raw("? > ?", newer.PriceDate, p.PriceDate),
		newer.PriceDate.Lte(ceiling),
	))

	return p.Where(p.PriceDate.Lte(ceiling), keel.Raw("? < ?", newerCount, perIndex)).
		Order(p.FuelIndexID.Asc(), p.PriceDate.Desc())
}

func (r *priceRepository) LatestPerIndex(
	ctx context.Context,
	req *fuelsurchargeports.LatestPricesPerIndexRequest,
) (map[pulid.ID][]*fuelsurcharge.FuelIndexPrice, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*fuelsurcharge.FuelIndexPrice, error) {
			perIndex := req.PerIndex
			if perIndex <= 0 {
				perIndex = defaultPricesPerIndex
			}

			ceiling := latestPriceCeiling
			if req.OnOrBefore != "" {
				ceiling = req.OnOrBefore
			}

			entities, err := latestPerIndexQuery(perIndex, ceiling).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get latest prices per index", zap.Error(err))
				return nil, err
			}

			result := make(map[pulid.ID][]*fuelsurcharge.FuelIndexPrice, len(entities)/perIndex+1)
			for _, entity := range entities {
				result[entity.FuelIndexID] = append(result[entity.FuelIndexID], entity)
			}

			return result, nil
		},
	)
}

func (r *priceRepository) HasPriceForDate(
	ctx context.Context,
	req *fuelsurchargeports.HasFuelPriceForDateRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		p := fuelsurchargedb.FuelIndexPrices
		exists, err := p.Where(p.FuelIndexID.Eq(req.FuelIndexID), p.PriceDate.Eq(req.Date)).
			Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check fuel price existence", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}

func (r *priceRepository) Create(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
) (*fuelsurcharge.FuelIndexPrice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelsurcharge.FuelIndexPrice, error) {
		if err := fuelsurchargedb.FuelIndexPrices.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create fuel index price", zap.Error(err))
			return nil, fuelPriceErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *priceRepository) Update(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
) (*fuelsurcharge.FuelIndexPrice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelsurcharge.FuelIndexPrice, error) {
		if err := fuelsurchargedb.FuelIndexPrices.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update fuel index price",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, fuelPriceErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *priceRepository) Delete(
	ctx context.Context,
	req *fuelsurchargeports.GetFuelIndexPriceByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := fuelsurchargedb.FuelIndexPrices.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&fuelsurcharge.FuelIndexPrice{ID: req.PriceID},
		)
		if err != nil {
			r.l.Error("failed to delete fuel index price",
				zap.String("id", req.PriceID.String()),
				zap.Error(err),
			)
			return fuelPriceErrors.Map(err, req.PriceID.String())
		}

		return nil
	})
}

func (r *priceRepository) GetByID(
	ctx context.Context,
	req *fuelsurchargeports.GetFuelIndexPriceByIDRequest,
) (*fuelsurcharge.FuelIndexPrice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelsurcharge.FuelIndexPrice, error) {
		entity, err := fuelsurchargedb.FuelIndexPrices.Get(ctx, r.db.Keel(ctx), req.PriceID)
		if err != nil {
			return nil, fuelPriceErrors.Map(err, req.PriceID.String())
		}

		return entity, nil
	})
}
