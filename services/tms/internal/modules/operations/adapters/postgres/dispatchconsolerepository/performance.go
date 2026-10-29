package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
)

func (r *repository) ListWorkerOnTimeStats(
	ctx context.Context,
	req *dispatchboardports.ListWorkloadRequest,
) ([]*dispatchboardports.WorkerOnTimeStats, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*dispatchboardports.WorkerOnTimeStats, error) {
			if len(req.WorkerIDs) == 0 {
				return []*dispatchboardports.WorkerOnTimeStats{}, nil
			}

			asn := shipmentdb.Assignments
			sm := shipmentdb.ShipmentMoves
			st := shipmentdb.Stops
			arrivals := asn.
				Join(sm, sm.ID.EqCol(asn.ShipmentMoveID)).
				Join(st, st.ShipmentMoveID.EqCol(sm.ID)).
				Group(asn.PrimaryWorkerID).
				Pick(
					asn.PrimaryWorkerID,
					keel.Count(),
					countIf(keel.Int64Of(st.ActualArrival).LteExpr(
						st.ScheduledWindowEnd.Coalesce(st.ScheduledWindowStart),
					)),
					keel.Int64(0),
				).
				Where(
					asn.PrimaryWorkerID.In(req.WorkerIDs...),
					st.Status.Eq(shipmentvocab.StopStatusCompleted),
					st.Type.In(shipmentvocab.StopTypeDelivery, shipmentvocab.StopTypeSplitDelivery),
					st.ActualArrival.IsNotNull(),
					st.ScheduledWindowStart.Gt(0),
					keel.When(req.Since > 0, st.ActualArrival.Gte(req.Since)),
				)

			sf := servicefailuredb.ServiceFailures
			rc := servicefailuredb.ReasonCodes
			fa := shipmentdb.Assignments.As("sf_asn")
			failures := sf.
				Join(fa, fa.ShipmentMoveID.EqCol(sf.ShipmentMoveID)).
				Join(rc, rc.ID.EqCol(sf.ReasonCodeID)).
				Group(fa.PrimaryWorkerID).
				Pick(
					fa.PrimaryWorkerID,
					keel.Int64(0),
					keel.Int64(0),
					keel.Count(),
				).
				Where(
					fa.PrimaryWorkerID.In(req.WorkerIDs...),
					sf.Status.Ne(servicefailure.StatusVoided),
					rc.Category.Eq(servicefailure.ReasonCategoryDriver),
					keel.When(req.Since > 0, sf.DetectedAt.Gte(req.Since)),
				)

			var onTime, faults []dispatchboardports.WorkerOnTimeStats
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoSlice(arrivals, &onTime),
				keel.IntoSlice(failures, &faults),
			); err != nil {
				return nil, fmt.Errorf("list worker on-time stats: %w", err)
			}

			merged := make(map[string]*dispatchboardports.WorkerOnTimeStats, len(onTime))
			entities := make([]*dispatchboardports.WorkerOnTimeStats, 0, len(onTime)+len(faults))
			for i := range onTime {
				row := &onTime[i]
				merged[row.WorkerID.String()] = row
				entities = append(entities, row)
			}
			for i := range faults {
				row := &faults[i]
				if existing, ok := merged[row.WorkerID.String()]; ok {
					existing.DriverFaultFailures = row.DriverFaultFailures
					continue
				}
				merged[row.WorkerID.String()] = row
				entities = append(entities, row)
			}

			return entities, nil
		},
	)
}

func (r *repository) ListWorkerAcceptanceStats(
	ctx context.Context,
	req *dispatchboardports.ListWorkloadRequest,
) ([]*dispatchboardports.WorkerAcceptanceStats, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*dispatchboardports.WorkerAcceptanceStats, error) {
			if len(req.WorkerIDs) == 0 {
				return []*dispatchboardports.WorkerAcceptanceStats{}, nil
			}

			asn := shipmentdb.Assignments
			answered := asn.AckStatus.In(
				shipmentvocab.AssignmentAckAccepted,
				shipmentvocab.AssignmentAckDeclined,
			)
			rows, err := keel.Into[dispatchboardports.WorkerAcceptanceStats](
				ctx,
				r.db.Keel(ctx),
				asn.
					Group(asn.PrimaryWorkerID).
					Pick(
						asn.PrimaryWorkerID,
						countIf(asn.AckStatus.Eq(shipmentvocab.AssignmentAckAccepted)),
						countIf(asn.AckStatus.Eq(shipmentvocab.AssignmentAckDeclined)),
						keel.CastFloat64(keel.NumericOf(
							asn.AckAt.Sub(asn.CreatedAt).Avg().
								Filter(keel.And(
									answered,
									asn.AckAt.IsNotNull(),
									keel.Int64Of(asn.AckAt).GteExpr(asn.CreatedAt),
								)),
						).Coalesce(0)),
					).
					Where(
						asn.PrimaryWorkerID.In(req.WorkerIDs...),
						keel.When(req.Since > 0, asn.CreatedAt.Gte(req.Since)),
					),
			)
			if err != nil {
				return nil, fmt.Errorf("list worker acceptance stats: %w", err)
			}

			return pointers(rows), nil
		},
	)
}
