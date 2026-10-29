package shipmentpredicate

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold/shipmentholddb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func OptionPredicates(opts *shipmentports.ShipmentOptions) []keel.Pred {
	sp := shipmentdb.Shipments
	preds := make([]keel.Pred, 0, 8)

	if opts.Status != "" {
		preds = append(preds, sp.Status.Eq(shipmentvocab.Status(opts.Status)))
	}
	if len(opts.IDs) > 0 {
		preds = append(preds, sp.ID.In(opts.IDs...))
	}
	if len(opts.Statuses) > 0 {
		preds = append(preds, sp.Status.In(opts.Statuses...))
	}
	if opts.HasActivityWindow() {
		preds = append(preds, activityWindowPredicate(opts))
	}
	if opts.BillingTransferEligible {
		preds = append(preds, BillingTransferCandidateCondition())
	}
	if len(opts.CustomerIDs) > 0 {
		preds = append(preds, sp.CustomerID.In(opts.CustomerIDs...))
	}
	if len(opts.StopLocationIDs) > 0 {
		preds = append(preds, stopLocationPredicate(opts.StopLocationIDs))
	}
	for idx := range opts.PartyGroups {
		if pred := partyGroupPredicate(&opts.PartyGroups[idx]); pred != nil {
			preds = append(preds, pred)
		}
	}
	if opts.FiltersOnHolds() {
		preds = append(preds, activeHoldPredicate(opts))
	}
	if opts.LateAt > 0 {
		preds = append(preds, latePredicate(opts.LateAt))
	}
	if len(opts.WorkerIDs) > 0 {
		preds = append(preds, workerPredicate(opts.WorkerIDs))
	}
	if len(opts.TractorIDs) > 0 {
		a := sp.Moves().Assignment()
		preds = append(preds, liveAssignmentPredicate(a.TractorID.In(opts.TractorIDs...)))
	}
	if len(opts.TrailerIDs) > 0 {
		a := sp.Moves().Assignment()
		preds = append(preds, liveAssignmentPredicate(a.TrailerID.In(opts.TrailerIDs...)))
	}
	if len(opts.CarrierIDs) > 0 {
		preds = append(preds, assignedCarrierPredicate(opts.CarrierIDs))
	}
	if len(opts.CommodityIDs) > 0 {
		preds = append(preds, commodityPredicate(opts.CommodityIDs))
	}
	if len(opts.ShipmentTypeIDs) > 0 {
		preds = append(preds, sp.ShipmentTypeID.In(opts.ShipmentTypeIDs...))
	}

	return preds
}

func partyGroupPredicate(group *shipmentports.ShipmentPartyGroup) keel.Pred {
	sp := shipmentdb.Shipments
	readings := make([]keel.Pred, 0, 6)
	if len(group.CustomerIDs) > 0 {
		readings = append(readings, sp.CustomerID.In(group.CustomerIDs...))
	}
	if len(group.StopLocationIDs) > 0 {
		readings = append(readings, stopLocationPredicate(group.StopLocationIDs))
	}
	if len(group.WorkerIDs) > 0 {
		readings = append(readings, workerPredicate(group.WorkerIDs))
	}
	if len(group.CommodityIDs) > 0 {
		readings = append(readings, commodityPredicate(group.CommodityIDs))
	}
	if len(group.ShipmentTypeIDs) > 0 {
		readings = append(readings, sp.ShipmentTypeID.In(group.ShipmentTypeIDs...))
	}
	if len(group.OrderIDs) > 0 {
		readings = append(readings, sp.OrderID.In(group.OrderIDs...))
	}

	switch len(readings) {
	case 0:
		return nil
	case 1:
		return readings[0]
	default:
		return keel.Or(readings...)
	}
}

func workerPredicate(workerIDs []pulid.ID) keel.Pred {
	a := shipmentdb.Shipments.Moves().Assignment()

	return liveAssignmentPredicate(
		keel.Or(
			a.PrimaryWorkerID.In(workerIDs...),
			a.SecondaryWorkerID.In(workerIDs...),
		),
	)
}

func commodityPredicate(commodityIDs []pulid.ID) keel.Pred {
	sc := shipmentdb.Shipments.Commodities()

	return sc.Any(sc.CommodityID.In(commodityIDs...))
}

func activeHoldPredicate(opts *shipmentports.ShipmentOptions) keel.Pred {
	sh := shipmentholddb.ShipmentHolds
	held := sh.Pick(sh.ShipmentID).Where(sh.ReleasedAt.IsNull())
	if codes := holdReasonCodes(opts.HoldReasonCodes); len(codes) > 0 {
		held = held.Where(sh.ReasonCode.Lower().In(codes...))
	}

	id := shipmentdb.Shipments.ID
	if opts.ActiveHold != nil && !*opts.ActiveHold {
		return id.NotInQuery(held)
	}

	return id.InQuery(held)
}

// latePredicate keeps the shipments with a stop not yet reached whose window
// closed before at: its end, or its start when it has none. The stop's
// window opened before at too, which is what the open-stop index walks, so
// the read passes over the work under way and never the tenant's history.
func latePredicate(at int64) keel.Pred {
	sm := shipmentdb.ShipmentMoves
	st := shipmentdb.Stops
	closes := keel.CaseInt64().
		When(st.ScheduledWindowEnd.Coalesce(0).Gt(0), st.ScheduledWindowEnd.Coalesce(0)).
		Else(st.ScheduledWindowStart.Coalesce(0))
	late := sm.Join(st, st.ShipmentMoveID.EqCol(sm.ID)).
		Pick(sm.ShipmentID).
		Where(
			sm.Status.NotIn(shipmentvocab.MoveStatusCompleted, shipmentvocab.MoveStatusCanceled),
			st.Status.In(shipmentvocab.StopStatusNew, shipmentvocab.StopStatusInTransit),
			st.ScheduledWindowStart.Gt(0),
			st.ScheduledWindowStart.Lt(at),
			st.ActualArrival.Coalesce(0).Lte(0),
			closes.Lt(at),
		)

	return shipmentdb.Shipments.ID.InQuery(late)
}

func holdReasonCodes(codes []string) []string {
	lowered := make([]string, 0, len(codes))
	for _, code := range codes {
		lowered = append(lowered, strings.ToLower(code))
	}

	return sliceutils.DedupeStrings(lowered)
}

func liveAssignmentPredicate(match keel.Pred) keel.Pred {
	m := shipmentdb.Shipments.Moves()
	a := m.Assignment()

	return m.Any(
		m.Status.Ne(shipmentvocab.MoveStatusCanceled),
		a.ArchivedAt.IsNull(),
		a.Status.Ne(shipmentvocab.AssignmentStatusCanceled),
		match,
	)
}

func assignedCarrierPredicate(carrierIDs []pulid.ID) keel.Pred {
	m := shipmentdb.Shipments.Moves()
	ca := m.CarrierAssignment()

	return m.Any(
		m.Status.Ne(shipmentvocab.MoveStatusCanceled),
		ca.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled),
		ca.CarrierID.In(carrierIDs...),
	)
}

func stopLocationPredicate(locationIDs []pulid.ID) keel.Pred {
	m := shipmentdb.Shipments.Moves()
	st := m.Stops()

	return m.Any(
		m.Status.Ne(shipmentvocab.MoveStatusCanceled),
		st.Any(st.Status.Ne(shipmentvocab.StopStatusCanceled), st.LocationID.In(locationIDs...)),
	)
}

func activityWindowPredicate(opts *shipmentports.ShipmentOptions) keel.Pred {
	m := shipmentdb.Shipments.Moves()
	st := m.Stops()

	return m.Any(
		m.Status.Ne(shipmentvocab.MoveStatusCanceled),
		st.Any(
			st.ScheduledWindowStart.Gt(0),
			st.ScheduledWindowStart.Lte(opts.ActivityWindowEnd),
			st.ScheduledWindowEnd.
				Coalesce(st.ScheduledWindowStart).
				Gte(opts.ActivityWindowStart),
		),
	)
}

func UnassignedPredicate() keel.Pred {
	m := shipmentdb.Shipments.Moves()
	a := m.Assignment()

	return m.None(
		m.Status.Ne(shipmentvocab.MoveStatusCanceled),
		a.ArchivedAt.IsNull(),
		a.Status.Ne(shipmentvocab.AssignmentStatusCanceled),
	)
}

func Scope(
	filter *pagination.QueryOptions,
	opts *shipmentports.ShipmentOptions,
	quick []keel.Pred,
) *keel.Query[*shipment.Shipment] {
	q := shipmentdb.Shipments.Where(OptionPredicates(opts)...).Where(quick...)
	if filter == nil {
		return q
	}

	return listquery.Filter(q, filter)
}
