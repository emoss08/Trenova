package shipmentprojection

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentpredicate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	deliveryStopAlias = "prj_stp"
	deliveryMoveAlias = "prj_dsm"
	deliveryAtStop    = "prj_dts"
	deliveryAtMove    = "prj_dtm"
	pickupAtStop      = "prj_pts"
	pickupAtMove      = "prj_ptm"
	assignedMoveAlias = "prj_asm"
	assignmentAlias   = "prj_a"
	touchedMoveAlias  = "prj_tsm"

	tenantRefreshBatch = 1000
)

func Refresh(ctx context.Context, db *keel.DB, shipmentIDs ...pulid.ID) error {
	if len(shipmentIDs) == 0 {
		return nil
	}

	return refresh(ctx, db, shipmentdb.Shipments.ID.In(shipmentIDs...))
}

func RefreshMoves(ctx context.Context, db *keel.DB, moveIDs ...pulid.ID) error {
	if len(moveIDs) == 0 {
		return nil
	}

	return refresh(
		ctx,
		db,
		shipmentdb.Shipments.ID.InQuery(shipmentsOfMoves(ctx, moveIDs)),
	)
}

func RefreshTenant(ctx context.Context, db *keel.DB) error {
	var after pulid.ID
	for {
		ids, err := keel.Into[pulid.ID](ctx, db, tenantPage(after))
		if err != nil {
			return fmt.Errorf("page shipments to refresh: %w", err)
		}
		if err = Refresh(ctx, db, ids...); err != nil {
			return err
		}
		if len(ids) < tenantRefreshBatch {
			return nil
		}
		after = ids[len(ids)-1]
	}
}

func tenantPage(after pulid.ID) keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments

	return sp.Pick(sp.ID).
		Where(keel.When(!after.IsNil(), sp.ID.Gt(after))).
		Order(sp.ID.Asc()).
		Limit(tenantRefreshBatch)
}

func shipmentsOfMoves(ctx context.Context, moveIDs []pulid.ID) keel.Query[*shipment.ShipmentMove] {
	sm := shipmentdb.ShipmentMoves.As(touchedMoveAlias)

	return tenantscope.Scoped(ctx, sm.Pick(sm.ShipmentID).Where(sm.ID.In(moveIDs...)))
}

func refresh(ctx context.Context, db *keel.DB, target keel.Pred) error {
	if _, err := refreshQuery(ctx, target).Update(ctx, db, refreshAssignments(ctx)...); err != nil {
		return fmt.Errorf("refresh shipment projections: %w", err)
	}

	return nil
}

func refreshQuery(ctx context.Context, target keel.Pred) keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments

	return tenantscope.Scoped(ctx, sp.Where(
		target,
		keel.Or(
			sp.LastDeliveryArrivalAt.IsDistinctFromExpr(lastDeliveryArrival(ctx, sp)),
			sp.PickupAt.IsDistinctFromExpr(pickupAt(ctx, sp)),
			sp.DeliveryAt.IsDistinctFromExpr(deliveryAt(ctx, sp)),
			sp.HasActiveAssignment.IsDistinctFromExpr(keel.BoolOf(hasActiveAssignment(ctx, sp))),
		),
	))
}

func refreshAssignments(ctx context.Context) []keel.Assignment {
	sp := shipmentdb.Shipments

	return []keel.Assignment{
		sp.LastDeliveryArrivalAt.ToExpr(lastDeliveryArrival(ctx, sp)),
		sp.PickupAt.ToExpr(pickupAt(ctx, sp)),
		sp.DeliveryAt.ToExpr(deliveryAt(ctx, sp)),
		sp.HasActiveAssignment.ToExpr(hasActiveAssignment(ctx, sp)),
		sp.Version.ToExpr(sp.Version),
		sp.UpdatedAt.ToExpr(sp.UpdatedAt),
	}
}

func lastDeliveryArrival(ctx context.Context, sp *shipmentdb.ShipmentsTable) keel.Int64Expr {
	stp := shipmentdb.Stops.As(deliveryStopAlias)
	sm := shipmentdb.ShipmentMoves.As(deliveryMoveAlias)

	q := stp.Pick(stp.ActualArrival).
		Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
		Where(
			sm.ShipmentID.EqCol(sp.ID),
			keel.When(dbscope.IsSystem(ctx), keel.And(
				stp.OrganizationID.EqCol(sp.OrganizationID),
				stp.BusinessUnitID.EqCol(sp.BusinessUnitID),
				sm.OrganizationID.EqCol(sp.OrganizationID),
				sm.BusinessUnitID.EqCol(sp.BusinessUnitID),
			)),
			sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
			stp.Type.In(shipmentvocab.StopTypeDelivery, shipmentvocab.StopTypeSplitDelivery),
			stp.Status.Ne(shipmentvocab.StopStatusCanceled),
			stp.ActualArrival.IsNotNull(),
		).
		Order(sm.Sequence.Desc(), stp.Sequence.Desc()).
		Limit(1)

	return keel.Int64Of(keel.Scalar(tenantscope.Scoped(ctx, q)))
}

// pickupAt is when the shipment is scheduled to be picked up: the scheduled
// window start of its first pickup stop. The pickup-window quick filters and
// the next uncovered pickup read it from the shipment instead of finding the
// first pickup stop of every uncovered shipment.
func pickupAt(ctx context.Context, sp *shipmentdb.ShipmentsTable) keel.Int64Expr {
	stp := shipmentdb.Stops.As(pickupAtStop)
	sm := shipmentdb.ShipmentMoves.As(pickupAtMove)

	q := shipmentpredicate.FirstPickupStop(stp, sm).
		Pick(stp.ScheduledWindowStart).
		Where(sameTenant(ctx, sp, stp, sm))

	return keel.Int64Of(keel.Scalar(tenantscope.Scoped(ctx, q)))
}

// deliveryAt is when the shipment delivers: the actual arrival of its last
// delivery stop, or that stop's scheduled window start until it arrives. The
// delivering-today and delivery-hour quick filters read it from the shipment
// instead of finding the last delivery stop of every shipment they count.
func deliveryAt(ctx context.Context, sp *shipmentdb.ShipmentsTable) keel.Int64Expr {
	stp := shipmentdb.Stops.As(deliveryAtStop)
	sm := shipmentdb.ShipmentMoves.As(deliveryAtMove)

	q := shipmentpredicate.LastDeliveryStop(stp, sm).
		Pick(shipmentpredicate.DeliveryAt(stp)).
		Where(sameTenant(ctx, sp, stp, sm))

	return keel.Int64Of(keel.Scalar(tenantscope.Scoped(ctx, q)))
}

// sameTenant ties a stop and its move to the shipment's tenant when the refresh
// runs across tenants, where row-level security does not.
func sameTenant(
	ctx context.Context,
	sp *shipmentdb.ShipmentsTable,
	stp *shipmentdb.StopsTable,
	sm *shipmentdb.ShipmentMovesTable,
) keel.Pred {
	return keel.When(dbscope.IsSystem(ctx), keel.And(
		stp.OrganizationID.EqCol(sp.OrganizationID),
		stp.BusinessUnitID.EqCol(sp.BusinessUnitID),
		sm.OrganizationID.EqCol(sp.OrganizationID),
		sm.BusinessUnitID.EqCol(sp.BusinessUnitID),
	))
}

func hasActiveAssignment(ctx context.Context, sp *shipmentdb.ShipmentsTable) keel.Pred {
	sm := shipmentdb.ShipmentMoves.As(assignedMoveAlias)
	a := shipmentdb.Assignments.As(assignmentAlias)

	q := sm.Join(
		a,
		a.ShipmentMoveID.EqCol(sm.ID),
		a.ArchivedAt.IsNull(),
		a.Status.Ne(shipmentvocab.AssignmentStatusCanceled),
	).
		Where(
			sm.ShipmentID.EqCol(sp.ID),
			keel.When(dbscope.IsSystem(ctx), keel.And(
				sm.OrganizationID.EqCol(sp.OrganizationID),
				sm.BusinessUnitID.EqCol(sp.BusinessUnitID),
				a.OrganizationID.EqCol(sp.OrganizationID),
				a.BusinessUnitID.EqCol(sp.BusinessUnitID),
			)),
			sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
		)

	return keel.Exists(tenantscope.Scoped(ctx, q))
}
