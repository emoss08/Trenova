package detentionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AnalyticsParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type analyticsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAnalyticsRepository(p AnalyticsParams) detentionports.DetentionAnalyticsRepository {
	return &analyticsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-analytics-repository"),
	}
}

func pointers[T any](rows []T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out
}

func sumOrZero(e keel.Expr) keel.NumericExpr {
	return keel.NumericOf(keel.Sum(e)).Coalesce(0)
}

// ListStopFacts pulls the raw dwell history a backtest replays against. Only
// completed stops with both timestamps are returned, because a stop the driver
// has not left has no settled duration to re-price.
func (r *analyticsRepository) ListStopFacts(
	ctx context.Context,
	req *detentionports.ListStopFactsRequest,
) ([]*detentionports.StopFact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*detentionports.StopFact, error) {
		st := shipmentdb.Stops
		sm := shipmentdb.ShipmentMoves
		sp := shipmentdb.Shipments

		q := st.
			Join(sm, sm.ID.EqCol(st.ShipmentMoveID)).
			Join(sp, sp.ID.EqCol(sm.ShipmentID)).
			Pick(
				st.ID,
				sm.ShipmentID,
				sp.CustomerID,
				st.LocationID,
				sp.ShipmentTypeID,
				sp.ServiceTypeID,
				st.Type,
				st.ScheduleType,
				st.ScheduledWindowStart,
				st.ScheduledWindowEnd,
				st.ActualArrival,
				st.ActualDeparture,
				st.CountDetentionOverride,
			).
			Where(
				st.ActualArrival.IsNotNull(),
				st.ActualDeparture.IsNotNull(),
				st.Status.Ne(shipmentvocab.StopStatusCanceled),
				keel.When(req.From > 0, st.ActualArrival.Gte(req.From)),
				keel.When(req.To > 0, st.ActualArrival.Lte(req.To)),
				optionalIDEq(sp.CustomerID, req.CustomerID),
				optionalIDEq(st.LocationID, req.LocationID),
			).
			Order(st.ActualArrival.Desc())
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}

		facts, err := keel.Into[detentionports.StopFact](ctx, r.db.Keel(ctx), q)
		if err != nil {
			r.l.Error("failed to list stop facts", zap.Error(err))
			return nil, err
		}

		return pointers(facts), nil
	})
}

// BaselineByStop returns what each stop actually billed in one pass, so a
// backtest can diff against reality without a query per stop.
func (r *analyticsRepository) BaselineByStop(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.StopBaseline, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*detentionports.StopBaseline, error) {
		o := detentiondb.DetentionOccurrences
		baselines, err := keel.Into[detentionports.StopBaseline](ctx, r.db.Keel(ctx), o.
			Pick(o.StopID, o.BillableAmount).
			Where(clockWindow(req)...))
		if err != nil {
			r.l.Error("failed to load detention baselines", zap.Error(err))
			return nil, err
		}

		return pointers(baselines), nil
	})
}

func clockWindow(req *detentionports.DetentionStatsRequest) []keel.Pred {
	o := detentiondb.DetentionOccurrences
	return []keel.Pred{
		keel.When(req.From > 0, o.ClockStartAt.Gte(req.From)),
		keel.When(req.To > 0, o.ClockStartAt.Lte(req.To)),
	}
}

func atLocation(req *detentionports.DetentionStatsRequest) keel.Pred {
	if req.LocationID.IsNil() {
		return nil
	}

	return detentiondb.DetentionOccurrences.LocationID.Eq(req.LocationID)
}

func settledStats(req *detentionports.DetentionStatsRequest) []keel.Pred {
	o := detentiondb.DetentionOccurrences
	return append(clockWindow(req), o.IsOpen.IsFalse(), atLocation(req))
}

func outcomeColumns() []keel.Expr {
	o := detentiondb.DetentionOccurrences
	return []keel.Expr{
		sumOrZero(o.BillableAmount),
		sumOrZero(o.DriverPayAmount),
		sumOrZero(o.NetMargin),
		sumOrZero(o.WaivedAmount),
		keel.Count().Filter(o.Status.Eq(detention.OccurrenceStatusDisputed)),
		keel.Count().Filter(o.SuppressedByGate.IsTrue()),
	}
}

// FacilityStats aggregates settled detention by facility so the worst docks are
// identifiable at planning and negotiation time rather than only at billing.
func (r *analyticsRepository) FacilityStats(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.FacilityDetentionStat, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detentionports.FacilityDetentionStat, error) {
			o := detentiondb.DetentionOccurrences
			loc := locationdb.Locations

			picks := append([]keel.Expr{
				o.LocationID,
				loc.Name.Coalesce(""),
				keel.Count(),
				keel.Count().Filter(o.RoundedMinutes.Gt(0)),
				keel.CastInt64(keel.NumericOf(o.RawDwellMinutes.Avg()).Round().Coalesce(0)),
				keel.Float64Of(keel.PercentileCont(0.5, o.RawDwellMinutes.Asc())).Coalesce(0),
				keel.Float64Of(keel.PercentileCont(0.9, o.RawDwellMinutes.Asc())).Coalesce(0),
			}, outcomeColumns()...)

			q := o.LeftJoin(loc, loc.ID.EqCol(o.LocationID)).
				Group(o.LocationID, loc.Name).
				Pick(picks...).
				Where(settledStats(req)...).
				Order(keel.Desc(sumOrZero(o.BillableAmount)))
			if req.Limit > 0 {
				q = q.Limit(req.Limit)
			}

			stats, err := keel.Into[detentionports.FacilityDetentionStat](ctx, r.db.Keel(ctx), q)
			if err != nil {
				r.l.Error("failed to aggregate facility detention stats", zap.Error(err))
				return nil, err
			}

			return pointers(stats), nil
		},
	)
}

// CustomerStats aggregates settled detention by customer for margin review and
// rate negotiation.
func (r *analyticsRepository) CustomerStats(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.CustomerDetentionStat, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detentionports.CustomerDetentionStat, error) {
			o := detentiondb.DetentionOccurrences
			cus := customerdb.Customers

			picks := append([]keel.Expr{
				o.CustomerID,
				cus.Name.Coalesce(""),
				keel.Count(),
				keel.Count().Filter(o.RoundedMinutes.Gt(0)),
			}, outcomeColumns()...)

			q := o.LeftJoin(cus, cus.ID.EqCol(o.CustomerID)).
				Group(o.CustomerID, cus.Name).
				Pick(picks...).
				Where(settledStats(req)...).
				Order(keel.Desc(sumOrZero(o.BillableAmount)))
			if req.Limit > 0 {
				q = q.Limit(req.Limit)
			}

			stats, err := keel.Into[detentionports.CustomerDetentionStat](ctx, r.db.Keel(ctx), q)
			if err != nil {
				r.l.Error("failed to aggregate customer detention stats", zap.Error(err))
				return nil, err
			}

			return pointers(stats), nil
		},
	)
}

// WaiverStats surfaces where discretionary revenue is going: which coded reason,
// how much, and who approved it.
func (r *analyticsRepository) WaiverStats(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.WaiverLeakageStat, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detentionports.WaiverLeakageStat, error) {
			o := detentiondb.DetentionOccurrences
			stats, err := keel.Into[detentionports.WaiverLeakageStat](ctx, r.db.Keel(ctx), o.
				Group(o.WaiverReason).
				Pick(
					keel.CastText(o.WaiverReason).Coalesce(""),
					keel.Count(),
					o.WaivedByID.CountDistinct(),
					sumOrZero(o.WaivedAmount),
				).
				Where(
					o.Status.Eq(detention.OccurrenceStatusWaived),
					keel.When(req.From > 0, o.WaivedAt.Gte(req.From)),
					keel.When(req.To > 0, o.WaivedAt.Lte(req.To)),
					atLocation(req),
				).
				Order(keel.Desc(sumOrZero(o.WaivedAmount))))
			if err != nil {
				r.l.Error("failed to aggregate detention waiver stats", zap.Error(err))
				return nil, err
			}

			return pointers(stats), nil
		},
	)
}

type entityNameRow struct {
	ID   pulid.ID
	Name string
}

// EntityNames resolves display names for the location and customer IDs a
// backtest groups by, so its buckets can be labeled without a query per row.
func (r *analyticsRepository) EntityNames(
	ctx context.Context,
	req *detentionports.DetentionEntityNamesRequest,
) (*detentionports.DetentionEntityNames, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*detentionports.DetentionEntityNames, error) {
			names := &detentionports.DetentionEntityNames{
				Locations: make(map[pulid.ID]string, len(req.LocationIDs)),
				Customers: make(map[pulid.ID]string, len(req.CustomerIDs)),
			}

			loc := locationdb.Locations
			cus := customerdb.Customers
			var locations, customers []entityNameRow
			var items []keel.BatchItem
			if len(req.LocationIDs) > 0 {
				items = append(items, keel.IntoSlice(
					loc.Pick(loc.ID, loc.Name).Where(loc.ID.In(req.LocationIDs...)),
					&locations,
				))
			}
			if len(req.CustomerIDs) > 0 {
				items = append(items, keel.IntoSlice(
					cus.Pick(cus.ID, cus.Name).Where(cus.ID.In(req.CustomerIDs...)),
					&customers,
				))
			}
			if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
				r.l.Error("failed to resolve detention entity names", zap.Error(err))
				return nil, err
			}

			for _, row := range locations {
				names.Locations[row.ID] = row.Name
			}
			for _, row := range customers {
				names.Customers[row.ID] = row.Name
			}

			return names, nil
		},
	)
}
