package shipmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentprojection"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func restoredShipmentStatus(req *shipmentports.UncancelShipmentRequest) shipmentvocab.Status {
	if req.RestoredStatus == "" || req.RestoredStatus == shipmentvocab.StatusCanceled {
		return shipmentvocab.StatusNew
	}

	return req.RestoredStatus
}

func movesOf(ctx context.Context, shipmentIDs []pulid.ID) keel.Query[*shipment.ShipmentMove] {
	sm := shipmentdb.ShipmentMoves.As("cancel_sm")
	return tenantscope.Scoped(ctx, sm.Pick(sm.ID).Where(sm.ShipmentID.In(shipmentIDs...)))
}

func (r *repository) cancelShipmentComponents(
	ctx context.Context,
	shipmentIDs []pulid.ID,
) error {
	if len(shipmentIDs) == 0 {
		return nil
	}

	db := r.db.Keel(ctx)

	sm := shipmentdb.ShipmentMoves
	if _, err := tenantscope.Scoped(ctx, sm.Where(
		sm.ShipmentID.In(shipmentIDs...),
		sm.Status.NotIn(shipmentstate.CancelPreservedMoveStatuses()...),
	)).Update(ctx, db, sm.Status.To(shipmentvocab.MoveStatusCanceled)); err != nil {
		return err
	}

	a := shipmentdb.Assignments
	if _, err := tenantscope.Scoped(ctx, a.Where(
		a.ShipmentMoveID.InQuery(movesOf(ctx, shipmentIDs)),
		a.ArchivedAt.IsNull(),
		a.Status.NotIn(shipmentstate.CancelPreservedAssignmentStatuses()...),
	)).Update(ctx, db, a.Status.To(shipmentvocab.AssignmentStatusCanceled)); err != nil {
		return err
	}

	st := shipmentdb.Stops
	if _, err := tenantscope.Scoped(ctx, st.Where(
		st.ShipmentMoveID.InQuery(movesOf(ctx, shipmentIDs)),
		st.Status.NotIn(shipmentstate.CancelPreservedStopStatuses()...),
	)).Update(ctx, db, st.Status.To(shipmentvocab.StopStatusCanceled)); err != nil {
		return err
	}

	return shipmentprojection.Refresh(ctx, db, shipmentIDs...)
}

func (r *repository) uncancelShipmentComponents(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
) error {
	db := r.db.Keel(ctx)
	moves := movesOf(ctx, []pulid.ID{req.ShipmentID})

	if err := restoreMoveStatuses(ctx, db, req.ShipmentID, req.MoveStatuses); err != nil {
		return err
	}

	a := shipmentdb.Assignments
	if _, err := a.Where(
		a.ShipmentMoveID.InQuery(moves),
		a.ArchivedAt.IsNull(),
		a.Status.Eq(shipmentvocab.AssignmentStatusCanceled),
	).Update(ctx, db, a.Status.To(shipmentvocab.AssignmentStatusNew)); err != nil {
		return err
	}

	if err := restoreStopStatuses(ctx, db, req.ShipmentID, req.StopStatuses); err != nil {
		return err
	}

	return shipmentprojection.Refresh(ctx, db, req.ShipmentID)
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
		func(item shipmentports.MoveStatusRestore) shipmentvocab.MoveStatus {
			return item.Status
		},
	)

	for status, items := range groups {
		if status == shipmentvocab.MoveStatusCanceled {
			continue
		}

		ids := make([]pulid.ID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.MoveID)
		}

		if _, err := sm.Where(
			sm.ID.In(ids...),
			sm.ShipmentID.Eq(shipmentID),
			sm.Status.Eq(shipmentvocab.MoveStatusCanceled),
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
		func(item shipmentports.StopStatusRestore) shipmentvocab.StopStatus {
			return item.Status
		},
	)

	for status, items := range groups {
		if status == shipmentvocab.StopStatusCanceled {
			continue
		}

		ids := make([]pulid.ID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.StopID)
		}

		if _, err := st.Where(
			st.ID.In(ids...),
			st.ShipmentMoveID.InQuery(movesOf(ctx, []pulid.ID{shipmentID})),
			st.Status.Eq(shipmentvocab.StopStatusCanceled),
		).Update(ctx, db, st.Status.To(status)); err != nil {
			return err
		}
	}

	return nil
}
