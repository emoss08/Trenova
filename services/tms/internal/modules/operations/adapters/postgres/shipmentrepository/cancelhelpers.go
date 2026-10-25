package shipmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

type tenancy bool

const (
	scopedTenant tenancy = false
	everyTenant  tenancy = true
)

func scoped[T any](q *keel.Query[T], t tenancy) *keel.Query[T] {
	if t == everyTenant {
		return q.AllTenants()
	}

	return q
}

func restoredShipmentStatus(req *shipmentports.UncancelShipmentRequest) shipment.Status {
	if req.RestoredStatus == "" || req.RestoredStatus == shipment.StatusCanceled {
		return shipment.StatusNew
	}

	return req.RestoredStatus
}

func movesOf(shipmentIDs []pulid.ID, t tenancy) *keel.Query[shipment.ShipmentMove] {
	sm := shipmentdb.ShipmentMoves.As("cancel_sm")
	return scoped(sm.Pick(sm.ID).Where(sm.ShipmentID.In(shipmentIDs...)), t)
}

func (r *repository) cancelShipmentComponents(
	ctx context.Context,
	shipmentIDs []pulid.ID,
	t tenancy,
) error {
	if len(shipmentIDs) == 0 {
		return nil
	}

	db := r.db.Keel(ctx)

	sm := shipmentdb.ShipmentMoves
	if _, err := scoped(sm.Where(
		sm.ShipmentID.In(shipmentIDs...),
		sm.Status.NotIn(shipmentstate.CancelPreservedMoveStatuses()...),
	), t).Update(ctx, db, sm.Status.To(shipment.MoveStatusCanceled)); err != nil {
		return err
	}

	a := shipmentdb.Assignments
	if _, err := scoped(a.Where(
		a.ShipmentMoveID.InQuery(movesOf(shipmentIDs, t)),
		a.ArchivedAt.IsNull(),
		a.Status.NotIn(shipmentstate.CancelPreservedAssignmentStatuses()...),
	), t).Update(ctx, db, a.Status.To(shipment.AssignmentStatusCanceled)); err != nil {
		return err
	}

	st := shipmentdb.Stops
	if _, err := scoped(st.Where(
		st.ShipmentMoveID.InQuery(movesOf(shipmentIDs, t)),
		st.Status.NotIn(shipmentstate.CancelPreservedStopStatuses()...),
	), t).Update(ctx, db, st.Status.To(shipment.StopStatusCanceled)); err != nil {
		return err
	}

	return nil
}

func (r *repository) uncancelShipmentComponents(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
) error {
	db := r.db.Keel(ctx)
	moves := movesOf([]pulid.ID{req.ShipmentID}, scopedTenant)

	if err := restoreMoveStatuses(ctx, db, req.ShipmentID, req.MoveStatuses); err != nil {
		return err
	}

	a := shipmentdb.Assignments
	if _, err := a.Where(
		a.ShipmentMoveID.InQuery(moves),
		a.ArchivedAt.IsNull(),
		a.Status.Eq(shipment.AssignmentStatusCanceled),
	).Update(ctx, db, a.Status.To(shipment.AssignmentStatusNew)); err != nil {
		return err
	}

	return restoreStopStatuses(ctx, db, req.ShipmentID, req.StopStatuses)
}

func restoreMoveStatuses(
	ctx context.Context,
	db *keel.DB,
	shipmentID pulid.ID,
	restores []shipmentports.MoveStatusRestore,
) error {
	sm := shipmentdb.ShipmentMoves
	groups := sliceutils.GroupBy(
		restores,
		func(item shipmentports.MoveStatusRestore) shipment.MoveStatus {
			return item.Status
		},
	)

	for status, items := range groups {
		if status == shipment.MoveStatusCanceled {
			continue
		}

		ids := make([]pulid.ID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.MoveID)
		}

		if _, err := sm.Where(
			sm.ID.In(ids...),
			sm.ShipmentID.Eq(shipmentID),
			sm.Status.Eq(shipment.MoveStatusCanceled),
		).Update(ctx, db, sm.Status.To(status)); err != nil {
			return err
		}
	}

	return nil
}

func restoreStopStatuses(
	ctx context.Context,
	db *keel.DB,
	shipmentID pulid.ID,
	restores []shipmentports.StopStatusRestore,
) error {
	st := shipmentdb.Stops
	groups := sliceutils.GroupBy(
		restores,
		func(item shipmentports.StopStatusRestore) shipment.StopStatus {
			return item.Status
		},
	)

	for status, items := range groups {
		if status == shipment.StopStatusCanceled {
			continue
		}

		ids := make([]pulid.ID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.StopID)
		}

		if _, err := st.Where(
			st.ID.In(ids...),
			st.ShipmentMoveID.InQuery(movesOf([]pulid.ID{shipmentID}, scopedTenant)),
			st.Status.Eq(shipment.StopStatusCanceled),
		).Update(ctx, db, st.Status.To(status)); err != nil {
			return err
		}
	}

	return nil
}
