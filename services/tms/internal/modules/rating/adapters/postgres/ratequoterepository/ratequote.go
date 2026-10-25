package ratequoterepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote/ratequotedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// defaultHistoryLimit caps how much of a shipment's rating history one request
// returns. Shipments are re-rated on every move edit, assignment and fuel price
// job, so the history can run long and a caller almost always wants the recent
// end of it.
const defaultHistoryLimit = 50

const busyMessage = "Rating is busy. Retry the request."

var rateQuoteErrors = dberror.Mapping{Entity: "RateQuote"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ratequoteports.RateQuoteRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratequote-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *ratequoteports.ListRateQuotesRequest,
) (*pagination.ListResult[*ratequote.RateQuote], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratequote.RateQuote], error) {
			rq := ratequotedb.RateQuotes
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				rq.Query(),
				&listquery.Request[ratequote.RateQuote]{
					Filter: req.Filter,
					Then:   []keel.Order{rq.RatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list rate quotes", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *ratequoteports.ListRateQuoteConnectionRequest,
) (*pagination.CursorListResult[*ratequote.RateQuote], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ratequote.RateQuote], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				ratequotedb.RateQuotes.Query(),
				&listquery.Request[ratequote.RateQuote]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.RateQuoteColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list rate quotes", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *ratequoteports.GetRateQuoteByIDRequest,
) (*ratequote.RateQuote, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratequote.RateQuote, error) {
		rq := ratequotedb.RateQuotes
		entity, err := rq.Where(rq.ID.Eq(req.RateQuoteID)).
			Load(rq.Agreement()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get rate quote",
				zap.String("id", req.RateQuoteID.String()),
				zap.Error(err),
			)
			return nil, rateQuoteErrors.Map(err, req.RateQuoteID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetAppliedForShipment(
	ctx context.Context,
	req *ratequoteports.GetShipmentRateQuoteRequest,
) (*ratequote.RateQuote, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratequote.RateQuote, error) {
		rq := ratequotedb.RateQuotes
		entity, err := rq.Where(
			rq.ShipmentID.Eq(req.ShipmentID),
			rq.PartyType.Eq(req.PartyType),
			rq.Status.Eq(ratequote.StatusApplied),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // an unrated shipment simply has no applied quote
			}
			r.l.Error("failed to get applied rate quote",
				zap.String("shipmentId", req.ShipmentID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ListForShipment(
	ctx context.Context,
	req *ratequoteports.ListShipmentRateQuotesRequest,
) ([]*ratequote.RateQuote, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratequote.RateQuote, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultHistoryLimit
		}

		rq := ratequotedb.RateQuotes
		entities, err := rq.Where(rq.ShipmentID.Eq(req.ShipmentID)).
			Order(rq.RatedAt.Desc(), rq.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list shipment rate quotes",
				zap.String("shipmentId", req.ShipmentID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entities, nil
	})
}

// Record writes a quote and, where it governs a shipment, retires the one it
// replaces.
//
// The two happen in one transaction because a unique index allows only one
// applied quote per shipment and party. Doing them separately would either
// leave a shipment with two applied quotes or, for the instant between, none —
// and re-rating is frequent enough that the instant would be noticed.
//
// Quotes that never governed a shipment — simulations, shopping comparisons,
// standalone what-ifs — skip the retirement entirely.
func (r *repository) Record(
	ctx context.Context,
	entity *ratequote.RateQuote,
) (*ratequote.RateQuote, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratequote.RateQuote, error) {
		db := r.db.Keel(ctx)
		rq := ratequotedb.RateQuotes

		var err error
		if entity.Status == ratequote.StatusApplied &&
			entity.ShipmentID != nil && !entity.ShipmentID.IsNil() {
			_, err = rq.Where(
				rq.ShipmentID.Eq(*entity.ShipmentID),
				rq.PartyType.Eq(entity.PartyType),
				rq.Status.Eq(ratequote.StatusApplied),
			).Update(ctx, db, rq.Status.To(ratequote.StatusSuperseded))
		}
		if err == nil {
			err = rq.Create(ctx, db, entity)
		}
		if err != nil {
			r.l.Error("failed to record rate quote",
				zap.String("outcome", string(entity.Outcome)),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(
				rateQuoteErrors.Map(err, entity.ID.String()),
				busyMessage,
			)
		}

		return entity, nil
	})
}
