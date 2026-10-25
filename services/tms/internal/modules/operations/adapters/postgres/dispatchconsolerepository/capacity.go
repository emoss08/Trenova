package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *repository) ListBoardDrivers(
	ctx context.Context,
	filter *shipmentports.DispatchBoardFilter,
) ([]*shipmentports.BoardDriver, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipmentports.BoardDriver, error) {
		w := workerdb.Workers
		fc := fleetcodedb.FleetCodes
		us := usstatedb.UsStates
		wt := tractordb.Tractors.As("wtrac")
		available := string(domaintypes.EquipmentStatusAvailable)

		tractorOf := wt.
			Pick(wt.ID, wt.Code, wt.EquipmentTypeID, wt.FleetCodeID, wt.Status).
			Where(wt.PrimaryWorkerID.EqCol(w.ID)).
			Order(keel.Desc(keel.Raw("(? = ?)", wt.Status, available)), wt.Code.Asc()).
			Limit(1)

		preds := driverFilters(filter)
		if filter.Query != "" {
			preds = append(preds, keel.Or(
				w.FirstName.ContainsFold(filter.Query),
				w.LastName.ContainsFold(filter.Query),
				wt.Code.ContainsFold(filter.Query),
			))
		}

		q := w.
			LeftJoin(fc, fc.ID.EqCol(w.FleetCodeID)).
			LeftJoin(us, us.ID.EqCol(w.StateID)).
			LeftJoinLateral(tractorOf).
			Pick(
				w.ID,
				w.FirstName,
				w.LastName,
				w.Type,
				w.DriverType,
				text(w.FleetCodeID),
				text(fc.Code),
				text(w.City),
				text(us.Abbreviation),
				enumText(w.PostalCode),
				text(w.ProfilePicURL),
				text(w.AssignmentBlocked),
				w.AvailableForDispatch,
				text(wt.ID),
				text(wt.Code),
				text(wt.EquipmentTypeID),
				text(wt.FleetCodeID),
				keel.Raw("coalesce(? = ?, FALSE)", wt.Status, available),
				zero(openAssignments(w)),
			).
			Where(preds...).
			Order(w.LastName.Asc(), w.FirstName.Asc(), w.ID.Asc()).
			Limit(boardLimit(filter.Limit))

		rows, err := keel.Into[shipmentports.BoardDriver](ctx, r.db.Keel(ctx), q)
		if err != nil {
			return nil, fmt.Errorf("list dispatch board drivers: %w", err)
		}

		return pointers(rows), nil
	})
}

func liveStops(moveID keel.Expr) (*shipmentdb.StopsTable, []keel.Pred) {
	st := shipmentdb.Stops.As("win")
	return st, []keel.Pred{
		st.ShipmentMoveID.EqCol(moveID),
		st.Status.Ne(shipment.StopStatusCanceled),
	}
}

func (r *repository) ListWorkerCommitments(
	ctx context.Context,
	req *shipmentports.ListWorkerWindowsRequest,
) ([]*shipmentports.WorkerCommitment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.WorkerCommitment, error) {
			if len(req.WorkerIDs) == 0 {
				return []*shipmentports.WorkerCommitment{}, nil
			}

			asn := shipmentdb.Assignments
			sm := shipmentdb.ShipmentMoves
			sp := shipmentdb.Shipments
			dest := destination

			st, live := liveStops(sm.ID)
			windowStart := keel.Scalar(st.Pick(st.ScheduledWindowStart.Min()).Where(live...))
			windowEnd := keel.Scalar(st.
				Pick(
					keel.Max(
						keel.DynOf(st.ScheduledWindowEnd).Coalesce(st.ScheduledWindowStart),
					),
				).
				Where(live...))

			q := joinEdge(asn.
				Join(sm, sm.ID.EqCol(asn.ShipmentMoveID)).
				LeftJoin(sp, sp.ID.EqCol(sm.ShipmentID)), dest, sm.ID).
				Pick(
					asn.PrimaryWorkerID,
					asn.ShipmentMoveID,
					sm.ShipmentID,
					text(sp.ProNumber),
					sm.Status,
					zero(windowStart),
					zero(windowEnd),
					text(dest.stop.LocationID),
					text(dest.loc.City),
					text(dest.state.Abbreviation),
					dest.loc.Latitude,
					dest.loc.Longitude,
					text(asn.TrailerID),
				).
				Where(
					asn.ArchivedAt.IsNull(),
					asn.PrimaryWorkerID.In(req.WorkerIDs...),
					sm.Status.In(openMoveStatuses...),
					keel.When(req.WindowEnd > 0,
						keel.DynOf(windowStart).Coalesce(0).Lte(req.WindowEnd)),
					keel.When(
						req.WindowStart > 0,
						keel.Raw(
							"coalesce(?, ?, 0) >= ?",
							windowEnd,
							windowStart,
							req.WindowStart,
						),
					),
				).
				Order(keel.Asc(windowStart))

			rows, err := keel.Into[shipmentports.WorkerCommitment](ctx, r.db.Keel(ctx), q)
			if err != nil {
				return nil, fmt.Errorf("list worker commitments: %w", err)
			}

			return pointers(rows), nil
		},
	)
}

func (r *repository) ListWorkerTimeOff(
	ctx context.Context,
	req *shipmentports.ListWorkerWindowsRequest,
) ([]*shipmentports.WorkerTimeOff, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipmentports.WorkerTimeOff, error) {
		if len(req.WorkerIDs) == 0 {
			return []*shipmentports.WorkerTimeOff{}, nil
		}

		pto := workerdb.WorkerPTOs
		rows, err := keel.Into[shipmentports.WorkerTimeOff](ctx, r.db.Keel(ctx), pto.
			Pick(pto.WorkerID, pto.StartDate, pto.EndDate, pto.Type).
			Where(
				pto.Status.Eq(worker.PTOStatusApproved),
				pto.WorkerID.In(req.WorkerIDs...),
				keel.When(req.WindowEnd > 0, pto.StartDate.Lte(req.WindowEnd)),
				keel.When(req.WindowStart > 0, pto.EndDate.Gte(req.WindowStart)),
			).
			Order(pto.StartDate.Asc()))
		if err != nil {
			return nil, fmt.Errorf("list worker time off: %w", err)
		}

		return pointers(rows), nil
	})
}

func (r *repository) ListWorkerWorkload(
	ctx context.Context,
	req *shipmentports.ListWorkloadRequest,
) ([]*shipmentports.WorkerWorkload, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipmentports.WorkerWorkload, error) {
		if len(req.WorkerIDs) == 0 {
			return []*shipmentports.WorkerWorkload{}, nil
		}

		asn := shipmentdb.Assignments
		sm := shipmentdb.ShipmentMoves
		sp := shipmentdb.Shipments
		st, live := liveStops(sm.ID)
		departure := keel.Scalar(st.
			Pick(st.ActualDeparture).
			Where(live...).
			Order(st.Sequence.Desc()).
			Limit(1))

		perShipment := keel.CTE(
			"wl",
			[]string{"worker_id", "move_count", "total_miles", "shipment_revenue", "last_ended_at"},
			asn.
				Join(sm, sm.ID.EqCol(asn.ShipmentMoveID)).
				LeftJoin(sp, sp.ID.EqCol(sm.ShipmentID)).
				Group(asn.PrimaryWorkerID, sm.ShipmentID).
				Pick(
					asn.PrimaryWorkerID,
					keel.Raw("count(*)::int"),
					keel.DynOf(sm.Distance).Sum().Coalesce(0).Cast(keel.TypeFloat64),
					keel.DynOf(sp.TotalChargeAmount).Max().Coalesce(0).Cast(keel.TypeFloat64),
					keel.Raw("max(coalesce(?, ?))", departure, asn.CreatedAt),
				).
				Where(
					asn.PrimaryWorkerID.In(req.WorkerIDs...),
					sm.Status.Ne(shipment.MoveStatusCanceled),
					keel.When(req.Since > 0, asn.CreatedAt.Gte(req.Since)),
				),
		)

		workerID := perShipment.Col("worker_id")
		rows, err := keel.Into[shipmentports.WorkerWorkload](ctx, r.db.Keel(ctx), perShipment.
			Query().
			With(perShipment).
			Group(workerID).
			Pick(
				workerID,
				keel.Raw("sum(?)::int", perShipment.Col("move_count")),
				keel.Raw("sum(?)::float8", perShipment.Col("total_miles")),
				keel.Raw("sum(?)::float8", perShipment.Col("shipment_revenue")),
				keel.DynOf(perShipment.Col("last_ended_at")).Max(),
			))
		if err != nil {
			return nil, fmt.Errorf("list worker workload: %w", err)
		}

		return pointers(rows), nil
	})
}

func (r *repository) ListWorkerLaneExperience(
	ctx context.Context,
	req *shipmentports.ListLaneExperienceRequest,
) ([]*shipmentports.WorkerLaneExperience, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.WorkerLaneExperience, error) {
			if len(req.WorkerIDs) == 0 || len(req.CustomerIDs) == 0 {
				return []*shipmentports.WorkerLaneExperience{}, nil
			}

			asn := shipmentdb.Assignments
			sm := shipmentdb.ShipmentMoves
			sp := shipmentdb.Shipments
			rows, err := keel.Into[shipmentports.WorkerLaneExperience](ctx, r.db.Keel(ctx), asn.
				Join(sm, sm.ID.EqCol(asn.ShipmentMoveID)).
				Join(sp, sp.ID.EqCol(sm.ShipmentID)).
				Group(asn.PrimaryWorkerID, sp.CustomerID).
				Pick(asn.PrimaryWorkerID, sp.CustomerID, keel.Raw("count(*)::int")).
				Where(
					asn.PrimaryWorkerID.In(req.WorkerIDs...),
					sp.CustomerID.In(req.CustomerIDs...),
					keel.When(req.Since > 0, asn.CreatedAt.Gte(req.Since)),
				))
			if err != nil {
				return nil, fmt.Errorf("list worker lane experience: %w", err)
			}

			return pointers(rows), nil
		},
	)
}

func (r *repository) ListWorkersByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerIDs []pulid.ID,
) ([]*worker.Worker, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.Worker, error) {
		if len(workerIDs) == 0 {
			return []*worker.Worker{}, nil
		}

		w := workerdb.Workers
		entities, err := w.Where(w.ID.In(workerIDs...)).
			Load(w.Profile()).
			Limit(len(workerIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list workers by ids: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListEquipmentByIDs(
	ctx context.Context,
	req *shipmentports.ListEquipmentByIDsRequest,
) ([]*tractor.Tractor, []*trailer.Trailer, error) {
	return dbtx.Read2(
		ctx,
		r.db,
		func(ctx context.Context) ([]*tractor.Tractor, []*trailer.Trailer, error) {
			tractors := make([]*tractor.Tractor, 0, len(req.TractorIDs))
			trailers := make([]*trailer.Trailer, 0, len(req.TrailerIDs))

			tc := tractordb.Tractors
			tl := trailerdb.Trailers
			var items []keel.BatchItem
			if len(req.TractorIDs) > 0 {
				items = append(items, keel.ListInto(
					tc.Where(tc.ID.In(req.TractorIDs...)).Limit(len(req.TractorIDs)),
					&tractors,
				))
			}
			if len(req.TrailerIDs) > 0 {
				items = append(items, keel.ListInto(
					tl.Where(tl.ID.In(req.TrailerIDs...)).Limit(len(req.TrailerIDs)),
					&trailers,
				))
			}
			if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
				return nil, nil, fmt.Errorf("list equipment by ids: %w", err)
			}

			return tractors, trailers, nil
		},
	)
}
