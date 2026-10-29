package shipmentprojection

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	deliveryStopAlias = "prj_stp"
	deliveryMoveAlias = "prj_dsm"
	assignedMoveAlias = "prj_asm"
	assignmentAlias   = "prj_a"
	touchedMoveAlias  = "prj_tsm"
)

type tenancy bool

const (
	scopedTenant tenancy = false
	everyTenant  tenancy = true
)

func scoped[T any](q *keel.Query[*T], t tenancy) *keel.Query[*T] {
	if t == everyTenant {
		return q.AllTenants()
	}

	return q
}

func Refresh(ctx context.Context, db *keel.DB, shipmentIDs ...pulid.ID) error {
	if len(shipmentIDs) == 0 {
		return nil
	}

	return refresh(ctx, db, shipmentdb.Shipments.ID.In(shipmentIDs...), scopedTenant)
}

func RefreshAcrossTenants(ctx context.Context, db *keel.DB, shipmentIDs ...pulid.ID) error {
	if len(shipmentIDs) == 0 {
		return nil
	}

	return refresh(ctx, db, shipmentdb.Shipments.ID.In(shipmentIDs...), everyTenant)
}

func RefreshMoves(ctx context.Context, db *keel.DB, moveIDs ...pulid.ID) error {
	if len(moveIDs) == 0 {
		return nil
	}

	return refresh(
		ctx,
		db,
		shipmentdb.Shipments.ID.InQuery(shipmentsOfMoves(moveIDs)),
		scopedTenant,
	)
}

func RefreshTenant(ctx context.Context, db *keel.DB) error {
	return refresh(ctx, db, keel.True(), scopedTenant)
}

func shipmentsOfMoves(moveIDs []pulid.ID) *keel.Query[*shipment.ShipmentMove] {
	sm := shipmentdb.ShipmentMoves.As(touchedMoveAlias)

	return sm.Pick(sm.ShipmentID).Where(sm.ID.In(moveIDs...))
}

func refresh(ctx context.Context, db *keel.DB, target keel.Pred, t tenancy) error {
	if _, err := refreshQuery(target, t).Update(ctx, db, refreshAssignments(t)...); err != nil {
		return fmt.Errorf("refresh shipment projections: %w", err)
	}

	return nil
}

func refreshQuery(target keel.Pred, t tenancy) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments

	return scoped(sp.Where(
		target,
		keel.Or(
			sp.LastDeliveryArrivalAt.IsDistinctFromExpr(lastDeliveryArrival(sp, t)),
			sp.HasActiveAssignment.IsDistinctFromExpr(keel.BoolOf(hasActiveAssignment(sp, t))),
		),
	), t)
}

func refreshAssignments(t tenancy) []keel.Assignment {
	sp := shipmentdb.Shipments

	return []keel.Assignment{
		sp.LastDeliveryArrivalAt.ToExpr(lastDeliveryArrival(sp, t)),
		sp.HasActiveAssignment.ToExpr(hasActiveAssignment(sp, t)),
		sp.Version.ToExpr(sp.Version),
		sp.UpdatedAt.ToExpr(sp.UpdatedAt),
	}
}

func lastDeliveryArrival(sp *shipmentdb.ShipmentsTable, t tenancy) keel.Int64Expr {
	stp := shipmentdb.Stops.As(deliveryStopAlias)
	sm := shipmentdb.ShipmentMoves.As(deliveryMoveAlias)

	q := stp.Pick(stp.ActualArrival).
		Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
		Where(
			sm.ShipmentID.EqCol(sp.ID),
			keel.When(t == everyTenant, keel.And(
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

	return keel.Int64Of(keel.Scalar(scoped(q, t)))
}

func hasActiveAssignment(sp *shipmentdb.ShipmentsTable, t tenancy) keel.Pred {
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
			keel.When(t == everyTenant, keel.And(
				sm.OrganizationID.EqCol(sp.OrganizationID),
				sm.BusinessUnitID.EqCol(sp.BusinessUnitID),
				a.OrganizationID.EqCol(sp.OrganizationID),
				a.BusinessUnitID.EqCol(sp.BusinessUnitID),
			)),
			sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
		)

	return keel.Exists(scoped(q, t))
}
