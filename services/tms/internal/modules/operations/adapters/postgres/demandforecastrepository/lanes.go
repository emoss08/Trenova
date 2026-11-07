package demandforecastrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/shared/pulid"
)

type laneRow struct {
	MoveID               pulid.ID
	ProNumber            string
	OriginLatitude       float64
	OriginLongitude      float64
	PickupStart          int64
	PickupEnd            int64
	DestinationLatitude  float64
	DestinationLongitude float64
	DeliveryEnd          int64
}

type departureRow struct {
	MoveID           pulid.ID
	ProNumber        string
	WorkerID         pulid.ID
	OriginLatitude   float64
	OriginLongitude  float64
	PickupStart      int64
	PickupEnd        int64
	DestinationLabel string
}

type laneQuery struct {
	loadQuery
	origin *locationdb.LocationsTable
	dest   *locationdb.LocationsTable
	last   *shipmentdb.StopsTable
}

func lanesBetween(from, until int64) laneQuery {
	l := loadsBetween(from, until)
	origin := locationdb.Locations.As("ln_o")
	dest := locationdb.Locations.As("ln_d")
	last := shipmentdb.Stops.As("ln_last")
	final := last.
		Pick(last.LocationID, last.ScheduledWindowStart, last.ScheduledWindowEnd).
		Where(
			last.ShipmentMoveID.EqCol(l.sm.ID),
			last.Status.Ne(shipmentvocab.StopStatusCanceled),
		).
		Order(last.Sequence.Desc()).
		Limit(1)
	l.q = l.q.
		JoinLateral(final).
		Join(origin, origin.ID.EqCol(l.st.LocationID)).
		Join(dest, dest.ID.EqCol(last.LocationID)).
		Where(
			origin.Latitude.IsNotNull(),
			origin.Longitude.IsNotNull(),
			dest.Latitude.IsNotNull(),
			dest.Longitude.IsNotNull(),
		)

	return laneQuery{loadQuery: l, origin: origin, dest: dest, last: last}
}

func (l laneQuery) pickupEnd() keel.Int64Expr {
	return keel.Int64Of(l.st.ScheduledWindowEnd).Coalesce(l.st.ScheduledWindowStart)
}

func (r *repository) ListOpenLanes(
	ctx context.Context,
	req *demandforecastports.WindowRequest,
) ([]demandforecastports.OpenLane, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]demandforecastports.OpenLane, error) {
		l := lanesBetween(req.From, req.Until)
		rows, err := keel.Into[laneRow](ctx, r.db.Keel(ctx), l.q.
			Pick(
				l.sm.ID,
				text(l.sp.ProNumber),
				l.origin.Latitude.Coalesce(0),
				l.origin.Longitude.Coalesce(0),
				l.st.ScheduledWindowStart,
				l.pickupEnd(),
				l.dest.Latitude.Coalesce(0),
				l.dest.Longitude.Coalesce(0),
				keel.Int64Of(l.last.ScheduledWindowEnd).Coalesce(l.last.ScheduledWindowStart),
			).
			Where(l.sm.Status.Eq(shipmentvocab.MoveStatusNew)).
			Order(l.st.ScheduledWindowStart.Asc(), l.sm.ID.Asc()).
			Limit(demandforecastports.MaxOpenLanes))
		if err != nil {
			return nil, fmt.Errorf("read open lanes: %w", err)
		}

		out := make([]demandforecastports.OpenLane, 0, len(rows))
		for _, row := range rows {
			out = append(out, demandforecastports.OpenLane(row))
		}

		return out, nil
	})
}

func (r *repository) ListDepartures(
	ctx context.Context,
	req *demandforecastports.WindowRequest,
) ([]demandforecastports.Departure, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]demandforecastports.Departure, error) {
		l := lanesBetween(req.From, req.Until)
		a := shipmentdb.Assignments.As("ln_a")
		rows, err := keel.Into[departureRow](ctx, r.db.Keel(ctx), l.q.
			Join(
				a,
				a.ShipmentMoveID.EqCol(l.sm.ID),
				a.ArchivedAt.IsNull(),
				a.PrimaryWorkerID.IsNotNull(),
			).
			Pick(
				l.sm.ID,
				text(l.sp.ProNumber),
				text(a.PrimaryWorkerID),
				l.origin.Latitude.Coalesce(0),
				l.origin.Longitude.Coalesce(0),
				l.st.ScheduledWindowStart,
				l.pickupEnd(),
				text(l.dest.City),
			).
			Where(l.sm.Status.Eq(shipmentvocab.MoveStatusAssigned)).
			Order(l.st.ScheduledWindowStart.Asc(), l.sm.ID.Asc()).
			Limit(demandforecastports.MaxDepartures))
		if err != nil {
			return nil, fmt.Errorf("read planned departures: %w", err)
		}

		out := make([]demandforecastports.Departure, 0, len(rows))
		for _, row := range rows {
			out = append(out, demandforecastports.Departure(row))
		}

		return out, nil
	})
}
