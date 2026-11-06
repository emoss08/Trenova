package demandforecastrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/shared/pulid"
)

const secondsPerHour = int64(3600)

type loadQuery struct {
	q     keel.Query[*shipment.ShipmentMove]
	sm    *shipmentdb.ShipmentMovesTable
	sp    *shipmentdb.ShipmentsTable
	st    *shipmentdb.StopsTable
	start keel.Int64Expr
}

func loadsBetween(since, until int64) loadQuery {
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments.As("dh_sp")
	st := shipmentdb.Stops.As("dh_st")
	first := st.
		Pick(st.LocationID, st.ScheduledWindowStart).
		Where(
			st.ShipmentMoveID.EqCol(sm.ID),
			st.Status.Ne(shipmentvocab.StopStatusCanceled),
		).
		Order(st.Sequence.Asc()).
		Limit(1)

	return loadQuery{
		q: sm.
			Join(sp, sp.ID.EqCol(sm.ShipmentID)).
			JoinLateral(first).
			Where(
				sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
				sp.Status.Ne(shipmentvocab.StatusCanceled),
				st.ScheduledWindowStart.Gte(since),
				st.ScheduledWindowStart.Lt(until),
			),
		sm:    sm,
		sp:    sp,
		st:    st,
		start: keel.Int64Of(st.ScheduledWindowStart),
	}
}

func (l loadQuery) sinceMonday() keel.Int64Expr {
	return l.start.Sub(keel.Int64(dispatchmarket.MondayOffset))
}

type countRow struct {
	CustomerID pulid.ID
	LocationID pulid.ID
	Week       int64
	Slot       int64
	Loads      int64
}

type leadRow struct {
	CustomerID pulid.ID
	Bucket     int64
	Loads      int64
}

type locationRow struct {
	LocationID         pulid.ID
	Latitude           float64
	Longitude          float64
	City               string
	State              string
	Loads              int64
	MedianRevenueCents float64
}

func (r *repository) History(
	ctx context.Context,
	req *demandforecastports.HistoryRequest,
) (*demandforecastports.HistoryRows, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*demandforecastports.HistoryRows, error) {
			db := r.db.Keel(ctx)
			counts, err := keel.Into[countRow](ctx, db, countsQuery(req))
			if err != nil {
				return nil, fmt.Errorf("read load history: %w", err)
			}
			leads, err := keel.Into[leadRow](ctx, db, leadsQuery(req))
			if err != nil {
				return nil, fmt.Errorf("read booking leads: %w", err)
			}
			locations, err := keel.Into[locationRow](ctx, db, locationsQuery(req))
			if err != nil {
				return nil, fmt.Errorf("read origin locations: %w", err)
			}

			return historyRows(counts, leads, locations), nil
		},
	)
}

func countsQuery(req *demandforecastports.HistoryRequest) keel.Query[*shipment.ShipmentMove] {
	l := loadsBetween(req.Since, req.Until)
	week := l.sinceMonday().Div(keel.Int64(dispatchmarket.WeekSeconds))
	slot := l.sinceMonday().
		Mod(keel.Int64(dispatchmarket.WeekSeconds)).
		Div(keel.Int64(dispatchmarket.SlotSeconds))

	return l.q.
		Pick(text(l.sp.CustomerID), l.st.LocationID, week, slot, keel.Count()).
		Group(l.sp.CustomerID, l.st.LocationID, week, slot)
}

func leadsQuery(req *demandforecastports.HistoryRequest) keel.Query[*shipment.ShipmentMove] {
	l := loadsBetween(req.Since, req.Until)
	hours := l.start.Sub(l.sp.CreatedAt).Div(keel.Int64(secondsPerHour))
	bucket := keel.CaseInt64()
	for i := len(demandforecast.LeadHours) - 1; i > 0; i-- {
		bucket = bucket.When(hours.Gte(demandforecast.LeadHours[i]), keel.Int64(int64(i)))
	}
	bucketExpr := bucket.Else(keel.Int64(0))

	return l.q.
		Pick(text(l.sp.CustomerID), bucketExpr, keel.Count()).
		Group(l.sp.CustomerID, bucketExpr)
}

func locationsQuery(req *demandforecastports.HistoryRequest) keel.Query[*shipment.ShipmentMove] {
	l := loadsBetween(req.Since, req.Until)
	loc := locationdb.Locations.As("dh_loc")
	state := usstatedb.UsStates.As("dh_state")
	median := keel.Float64Of(
		keel.PercentileCont(0.5, keel.Asc(l.sp.TotalChargeAmount)),
	).Coalesce(0)

	return l.q.
		Join(loc, loc.ID.EqCol(l.st.LocationID)).
		LeftJoin(state, state.ID.EqCol(loc.StateID)).
		Pick(
			l.st.LocationID,
			loc.Latitude.Coalesce(0),
			loc.Longitude.Coalesce(0),
			text(loc.City),
			text(state.Abbreviation),
			keel.Count(),
			median.Mul(keel.Float64(100)),
		).
		Where(loc.Latitude.IsNotNull(), loc.Longitude.IsNotNull()).
		Group(l.st.LocationID, loc.Latitude, loc.Longitude, loc.City, state.Abbreviation)
}

func historyRows(
	counts []countRow,
	leads []leadRow,
	locations []locationRow,
) *demandforecastports.HistoryRows {
	rows := &demandforecastports.HistoryRows{
		Counts:    make([]demandforecastports.LocationCount, 0, len(counts)),
		Leads:     make([]demandforecast.LeadCount, 0, len(leads)),
		Locations: make([]demandforecastports.LocationSummary, 0, len(locations)),
	}
	for _, c := range counts {
		rows.Counts = append(rows.Counts, demandforecastports.LocationCount{
			CustomerID: c.CustomerID,
			LocationID: c.LocationID,
			Week:       c.Week,
			Slot:       int16(c.Slot),  //nolint:gosec // (t mod a week) / 4 h: 0 to 41
			Loads:      int32(c.Loads), //nolint:gosec // one area, week and slot: far below 2^31

		})
	}
	for _, l := range leads {
		rows.Leads = append(rows.Leads, demandforecast.LeadCount{
			CustomerID: l.CustomerID,
			Bucket:     int(l.Bucket),
			Loads: int32(
				l.Loads,
			), //nolint:gosec // one customer and lead bucket: far below 2^31
		})
	}
	for _, l := range locations {
		rows.Locations = append(rows.Locations, demandforecastports.LocationSummary{
			LocationID:         l.LocationID,
			Latitude:           l.Latitude,
			Longitude:          l.Longitude,
			City:               l.City,
			State:              l.State,
			Loads:              int32(l.Loads), //nolint:gosec // one location: far below 2^31
			MedianRevenueCents: int64(l.MedianRevenueCents),
		})
	}

	return rows
}

func text(e keel.Expr) keel.StringExpr {
	return keel.StringOf(e).Coalesce("")
}
