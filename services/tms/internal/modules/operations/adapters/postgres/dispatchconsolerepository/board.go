package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultBoardLimit = 500

const maxBoardLimit = 2000

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) dispatchboardports.DispatchConsoleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dispatch-console-repository"),
	}
}

func boardLimit(requested int) int {
	switch {
	case requested <= 0:
		return defaultBoardLimit
	case requested > maxBoardLimit:
		return maxBoardLimit
	default:
		return requested
	}
}

func (r *repository) ListBoardMoves(
	ctx context.Context,
	filter *dispatchboardports.DispatchBoardFilter,
) ([]*dispatchboardports.BoardMove, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*dispatchboardports.BoardMove, error) {
		sm := shipmentdb.ShipmentMoves
		sp := shipmentdb.Shipments
		asn := assignments
		ca := carrierAssignments
		orig := origin
		dest := destination

		q := boardMoves().
			LeftJoin(serviceTypes, serviceTypes.ID.EqCol(sp.ServiceTypeID)).
			LeftJoin(assignedWorkers, assignedWorkers.ID.EqCol(asn.PrimaryWorkerID)).
			LeftJoin(assignedTractors, assignedTractors.ID.EqCol(asn.TractorID)).
			LeftJoin(assignedTrailers, assignedTrailers.ID.EqCol(asn.TrailerID)).
			LeftJoin(assignedCarriers, assignedCarriers.ID.EqCol(ca.CarrierID)).
			Pick(
				sm.ID,
				sm.ShipmentID,
				sp.ProNumber,
				text(sp.BOL),
				sm.Status,
				sp.Status,
				sm.Sequence,
				moveCount(sm),
				sm.Loaded,
				sm.Distance,
				sp.CustomerID,
				text(customerdb.Customers.Name),
				text(sp.ServiceTypeID),
				text(serviceTypes.Code),
				text(sp.TractorTypeID),
				text(sp.TrailerTypeID),
				sp.TemperatureMin,
				sp.TemperatureMax,
				hasHazmat(sm),
				hasActiveHold(sm),
				keel.CastFloat64(sp.TotalChargeAmount),
				text(orig.stop.ID),
				text(orig.stop.LocationID),
				text(orig.loc.Name),
				text(orig.loc.City),
				text(orig.state.Abbreviation),
				orig.loc.Latitude,
				orig.loc.Longitude,
				zero(orig.stop.ScheduledWindowStart),
				orig.stop.ScheduledWindowEnd,
				orig.stop.ActualArrival,
				text(dest.stop.ID),
				text(dest.stop.LocationID),
				text(dest.loc.Name),
				text(dest.loc.City),
				text(dest.state.Abbreviation),
				dest.loc.Latitude,
				dest.loc.Longitude,
				zero(dest.stop.ScheduledWindowStart),
				dest.stop.ScheduledWindowEnd,
				dest.stop.ActualArrival,
				text(asn.ID),
				text(asn.PrimaryWorkerID),
				keel.CaseString().
					When(assignedWorkers.ID.IsNull(), keel.Text("")).
					Else(keel.Concat(assignedWorkers.FirstName, " ", assignedWorkers.LastName)),
				text(asn.TractorID),
				text(assignedTractors.Code),
				text(asn.TrailerID),
				text(assignedTrailers.Code),
				enumText(asn.AckStatus),
				text(previousTrailer(sm)),
				sm.CoverageType,
				text(ca.ID),
				text(ca.CarrierID),
				text(assignedCarriers.Name),
				keel.CastFloat64(ca.TotalCost),
			).
			Where(moveFilters(filter)...).
			Where(
				keel.When(!filter.IncludeCovered, asn.ID.IsNull()),
				keel.When(!filter.IncludeCovered, ca.ID.IsNull()),
				keel.When(filter.LateAt > 0, lateAt(filter.LateAt)),
			).
			Order(boardOrder(filter)...).
			Limit(boardLimit(filter.Limit))
		if filter.Offset > 0 {
			q = q.Offset(filter.Offset)
		}

		rows, err := keel.Into[dispatchboardports.BoardMove](ctx, r.db.Keel(ctx), q)
		if err != nil {
			return nil, fmt.Errorf("list dispatch board moves: %w", err)
		}

		return pointers(rows), nil
	})
}

// boardOrder is the board's order: by pickup window, or, for a reader that
// wants the most urgent first, by when each move's next stop is due.
func boardOrder(filter *dispatchboardports.DispatchBoardFilter) []keel.Order {
	sm := shipmentdb.ShipmentMoves
	if filter.DueFirst {
		return []keel.Order{keel.Asc(dueAt()), sm.Sequence.Asc(), sm.ID.Asc()}
	}

	return []keel.Order{
		keel.Asc(zero(origin.stop.ScheduledWindowStart)),
		sm.Sequence.Asc(),
		sm.ID.Asc(),
	}
}

// windowClose is when a stop's window closes: its end, or its start when it
// has none.
func windowClose(edge *stopEdge) keel.Int64Expr {
	return keel.CaseInt64().
		When(zero(edge.stop.ScheduledWindowEnd).Gt(0), zero(edge.stop.ScheduledWindowEnd)).
		Else(zero(edge.stop.ScheduledWindowStart))
}

func pickedUp() keel.Pred {
	return zero(origin.stop.ActualArrival).Gt(0)
}

// dueAt is when a move's next stop is due: the pickup window's close until
// it is picked up, then the delivery window's close.
func dueAt() keel.Int64Expr {
	return keel.CaseInt64().
		When(pickedUp(), windowClose(&destination)).
		Else(windowClose(&origin))
}

// lateAt keeps the moves late at an instant, as the board's urgency reads
// them: a stop not yet reached whose window has closed, the pickup until the
// load is picked up and then the delivery, which is what the shipment list's
// late reads from the stops too.
func lateAt(at int64) keel.Pred {
	return keel.Or(
		keel.And(
			zero(origin.stop.ActualArrival).Lte(0),
			zero(origin.stop.ScheduledWindowStart).Gt(0),
			windowClose(&origin).Lt(at),
		),
		keel.And(
			pickedUp(),
			zero(destination.stop.ActualArrival).Lte(0),
			windowClose(&destination).Gt(0),
			windowClose(&destination).Lt(at),
		),
	)
}

func pointers[T any](rows []T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out
}

func moveCount(sm *shipmentdb.ShipmentMovesTable) keel.Expr {
	sib := shipmentdb.ShipmentMoves.As("sib")
	return keel.Scalar(sib.
		Pick(keel.Count()).
		Where(sib.ShipmentID.EqCol(sm.ShipmentID)))
}

func hasHazmat(sm *shipmentdb.ShipmentMovesTable) keel.Expr {
	sc := shipmentdb.ShipmentCommodities
	com := commoditydb.Commodities
	return keel.Exists(sc.
		Join(com, com.ID.EqCol(sc.CommodityID)).
		Where(sc.ShipmentID.EqCol(sm.ShipmentID), com.HazardousMaterialID.IsNotNull()))
}

func hasActiveHold(sm *shipmentdb.ShipmentMovesTable) keel.Expr {
	h := shipmentdb.ShipmentHolds
	return keel.Exists(h.
		Where(
			h.ShipmentID.EqCol(sm.ShipmentID),
			h.BlocksDispatch.IsTrue(),
			h.ReleasedAt.IsNull(),
		))
}

func previousTrailer(sm *shipmentdb.ShipmentMovesTable) keel.Expr {
	pm := shipmentdb.ShipmentMoves.As("pm")
	pa := shipmentdb.Assignments.As("pa")
	return keel.Scalar(pm.
		Join(pa, pa.ShipmentMoveID.EqCol(pm.ID), pa.ArchivedAt.IsNull()).
		Pick(pa.TrailerID).
		Where(
			pm.ShipmentID.EqCol(sm.ShipmentID),
			keel.Int64Of(pm.Sequence).LtExpr(sm.Sequence),
			pa.TrailerID.IsNotNull(),
		).
		Order(pm.Sequence.Desc()).
		Limit(1))
}
