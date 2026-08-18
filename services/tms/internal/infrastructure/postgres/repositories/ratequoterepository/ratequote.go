package ratequoterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// defaultHistoryLimit caps how much of a shipment's rating history one request
// returns. Shipments are re-rated on every move edit, assignment and fuel price
// job, so the history can run long and a caller almost always wants the recent
// end of it.
const defaultHistoryLimit = 50

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.RateQuoteRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratequote-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListRateQuotesRequest,
) *bun.SelectQuery {
	cols := buncolgen.RateQuoteColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.RateQuoteTable.Alias,
		req.Filter,
		(*ratequote.RateQuote)(nil),
	)

	return q.Apply(buncolgen.RateQuoteApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.RatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRateQuotesRequest,
) (*pagination.ListResult[*ratequote.RateQuote], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*ratequote.RateQuote, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count rate quotes", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*ratequote.RateQuote]{Items: entities, Total: total}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateQuoteConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.RateQuoteTable.Alias,
		req.Filter,
		req.Cursor,
		(*ratequote.RateQuote)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateQuoteConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.RateQuoteTable.Alias,
		req.Filter,
		(*ratequote.RateQuote)(nil),
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListRateQuoteConnectionRequest,
) (*pagination.CursorListResult[*ratequote.RateQuote], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*ratequote.RateQuote)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count rate quotes", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*ratequote.RateQuote]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*ratequote.RateQuote) *bun.SelectQuery {
			return dba.
				NewSelect().
				Model(entities).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return applyRateQuoteColumns(sq, req.RateQuoteColumns)
				})
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return r.applyCursorPageFilters(sq, req)
		},
	})
	if err != nil {
		log.Error("failed to scan rate quotes", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRateQuoteByIDRequest,
) (*ratequote.RateQuote, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.RateQuoteID.String()),
	)

	entity := new(ratequote.RateQuote)
	cols := buncolgen.RateQuoteColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.Rel(buncolgen.RateQuoteRelations.Agreement)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateQuoteScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateQuoteID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get rate quote", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateQuote")
	}

	return entity, nil
}

func (r *repository) GetAppliedForShipment(
	ctx context.Context,
	req *repositories.GetShipmentRateQuoteRequest,
) (*ratequote.RateQuote, error) {
	log := r.l.With(
		zap.String("operation", "GetAppliedForShipment"),
		zap.String("shipmentId", req.ShipmentID.String()),
	)

	entity := new(ratequote.RateQuote)
	cols := buncolgen.RateQuoteColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateQuoteScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentID.Eq(), req.ShipmentID).
				Where(cols.PartyType.Eq(), req.PartyType).
				Where(cols.Status.Eq(), ratequote.StatusApplied)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // an unrated shipment simply has no applied quote
		}
		log.Error("failed to get applied rate quote", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListForShipment(
	ctx context.Context,
	req *repositories.ListShipmentRateQuotesRequest,
) ([]*ratequote.RateQuote, error) {
	log := r.l.With(
		zap.String("operation", "ListForShipment"),
		zap.String("shipmentId", req.ShipmentID.String()),
	)

	limit := req.Limit
	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	cols := buncolgen.RateQuoteColumns
	entities := make([]*ratequote.RateQuote, 0, limit)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateQuoteScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentID.Eq(), req.ShipmentID)
		}).
		Order(cols.RatedAt.OrderDesc()).
		Order(cols.ID.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		log.Error("failed to list shipment rate quotes", zap.Error(err))
		return nil, err
	}

	return entities, nil
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
	log := r.l.With(
		zap.String("operation", "Record"),
		zap.String("outcome", string(entity.Outcome)),
	)

	supersedes := entity.Status == ratequote.StatusApplied &&
		entity.ShipmentID != nil && !entity.ShipmentID.IsNil()

	if !supersedes {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			log.Error("failed to record rate quote", zap.Error(err))
			return nil, err
		}

		return entity, nil
	}

	cols := buncolgen.RateQuoteColumns
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model((*ratequote.RateQuote)(nil)).
			Set(cols.Status.Set(), ratequote.StatusSuperseded).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.RateQuoteScopeTenantUpdate(uq, tenantInfo).
					Where(cols.ShipmentID.Eq(), *entity.ShipmentID).
					Where(cols.PartyType.Eq(), entity.PartyType).
					Where(cols.Status.Eq(), ratequote.StatusApplied)
			}).
			Exec(c); uErr != nil {
			return uErr
		}

		_, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c)

		return iErr
	})
	if err != nil {
		log.Error("failed to record rate quote", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rating is busy. Retry the request.",
		)
	}

	return entity, nil
}
