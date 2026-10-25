package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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

func New(p Params) shipmentports.DispatchConsoleRepository {
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
	filter *shipmentports.DispatchBoardFilter,
) ([]*shipmentports.BoardMove, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipmentports.BoardMove, error) {
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
				keel.DynOf(sp.TotalChargeAmount).Cast(keel.TypeFloat64),
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
				text(asn.ID),
				text(asn.PrimaryWorkerID),
				keel.DynCase().
					When(keel.DynOf(assignedWorkers.ID).IsNull(), keel.DynLit("").Cast(keel.TypeString)).
					Else(keel.DynFunc(
						"concat",
						assignedWorkers.FirstName,
						keel.DynLit(" ").Cast(keel.TypeString),
						assignedWorkers.LastName,
					)),
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
				keel.DynOf(ca.TotalCost).Cast(keel.TypeFloat64),
			).
			Where(moveFilters(filter)...).
			Where(
				keel.When(!filter.IncludeCovered, asn.ID.IsNull()),
				keel.When(!filter.IncludeCovered, ca.ID.IsNull()),
			).
			Order(
				keel.Asc(zero(orig.stop.ScheduledWindowStart)),
				sm.Sequence.Asc(),
				sm.ID.Asc(),
			).
			Limit(boardLimit(filter.Limit))

		rows, err := keel.Into[shipmentports.BoardMove](ctx, r.db.Keel(ctx), q)
		if err != nil {
			return nil, fmt.Errorf("list dispatch board moves: %w", err)
		}

		return pointers(rows), nil
	})
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
			keel.DynOf(pm.Sequence).Lt(sm.Sequence),
			pa.TrailerID.IsNotNull(),
		).
		Order(pm.Sequence.Desc()).
		Limit(1))
}
