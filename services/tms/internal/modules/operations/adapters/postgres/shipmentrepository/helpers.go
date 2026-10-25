package shipmentrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func shipmentLoads(opts *shipmentports.ShipmentOptions) []keel.Loader {
	sp := shipmentdb.Shipments

	switch {
	case opts.ExpandShipmentDetails:
		return append(
			[]keel.Loader{
				sp.Customer(),
				sp.BillToCustomer(),
				commoditiesWithHazmat(),
				sp.ServiceType(),
				sp.ShipmentType(),
				sp.FormulaTemplate(),
				sp.TractorType(),
				sp.TrailerType(),
				sp.CanceledBy(),
				sp.Owner(),
				movesWithDetails(),
			},
			chargeLoads()...,
		)
	case opts.IncludeRoute && opts.IncludeCustomer:
		return []keel.Loader{sp.Customer(), sp.BillToCustomer(), movesWithRoute()}
	case opts.IncludeRoute:
		return []keel.Loader{movesWithRoute()}
	case opts.IncludeCustomer:
		return []keel.Loader{sp.Customer(), sp.BillToCustomer()}
	default:
		return nil
	}
}

func commoditiesWithHazmat() keel.Loader {
	sc := shipmentdb.Shipments.Commodities()
	commodity := sc.Commodity()

	return sc.Load(commodity.Load(commodity.HazardousMaterial()))
}

func chargeLoads() []keel.Loader {
	sp := shipmentdb.Shipments
	ac := sp.AdditionalCharges()
	ca := sp.ChargeAllocations()

	return []keel.Loader{
		ac.Load(ac.AccessorialCharge()),
		ca.Order(ca.Sequence.Asc(), ca.ID.Asc()).Load(ca.BillToCustomer()),
	}
}

func orderedStopsWithLocation(st *shipmentdb.StopsTable) keel.Loader {
	loc := st.Location()

	return st.Order(st.Sequence.Asc()).Load(loc.Load(loc.State()))
}

func movesWithRoute() keel.Loader {
	m := shipmentdb.Shipments.Moves()
	a := m.Assignment()
	ca := m.CarrierAssignment()

	return m.Order(m.Sequence.Asc()).Load(
		orderedStopsWithLocation(m.Stops()),
		a.Where(a.ArchivedAt.IsNull()).Load(a.PrimaryWorker(), a.Tractor()),
		ca.Where(ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled)),
	)
}

func movesWithDetails() keel.Loader {
	m := shipmentdb.Shipments.Moves()
	a := m.Assignment()
	ca := m.CarrierAssignment()

	return m.Order(m.Sequence.Asc()).Load(
		orderedStopsWithLocation(m.Stops()),
		a.Where(a.ArchivedAt.IsNull()).
			Load(a.Tractor(), a.Trailer(), a.PrimaryWorker(), a.SecondaryWorker()),
		ca.Where(ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled)).
			Load(ca.Carrier(), ca.Accessorials()),
	)
}

func shipmentOptionPredicates(opts *shipmentports.ShipmentOptions) []keel.Pred {
	sp := shipmentdb.Shipments
	preds := make([]keel.Pred, 0, 8)

	if opts.Status != "" {
		preds = append(preds, sp.Status.Eq(shipment.Status(opts.Status)))
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
	for _, group := range opts.StopEachOf {
		if len(group) > 0 {
			preds = append(preds, stopLocationPredicate(group))
		}
	}
	if len(opts.WorkerIDs) > 0 {
		a := sp.Moves().Assignment()
		preds = append(preds, liveAssignmentPredicate(
			keel.Or(
				a.PrimaryWorkerID.In(opts.WorkerIDs...),
				a.SecondaryWorkerID.In(opts.WorkerIDs...),
			),
		))
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
		sc := sp.Commodities()
		preds = append(preds, sc.Any(sc.CommodityID.In(opts.CommodityIDs...)))
	}
	if len(opts.ShipmentTypeIDs) > 0 {
		preds = append(preds, sp.ShipmentTypeID.In(opts.ShipmentTypeIDs...))
	}

	return preds
}

func liveAssignmentPredicate(match keel.Pred) keel.Pred {
	m := shipmentdb.Shipments.Moves()
	a := m.Assignment()

	return m.Any(
		m.Status.Ne(shipment.MoveStatusCanceled),
		a.ArchivedAt.IsNull(),
		a.Status.Ne(shipment.AssignmentStatusCanceled),
		match,
	)
}

func assignedCarrierPredicate(carrierIDs []pulid.ID) keel.Pred {
	m := shipmentdb.Shipments.Moves()
	ca := m.CarrierAssignment()

	return m.Any(
		m.Status.Ne(shipment.MoveStatusCanceled),
		ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
		ca.CarrierID.In(carrierIDs...),
	)
}

func stopLocationPredicate(locationIDs []pulid.ID) keel.Pred {
	m := shipmentdb.Shipments.Moves()
	st := m.Stops()

	return m.Any(
		m.Status.Ne(shipment.MoveStatusCanceled),
		st.Any(st.Status.Ne(shipment.StopStatusCanceled), st.LocationID.In(locationIDs...)),
	)
}

func activityWindowPredicate(opts *shipmentports.ShipmentOptions) keel.Pred {
	m := shipmentdb.Shipments.Moves()
	st := m.Stops()

	return m.Any(
		m.Status.Ne(shipment.MoveStatusCanceled),
		st.Any(
			st.ScheduledWindowStart.Gt(0),
			st.ScheduledWindowStart.Lte(opts.ActivityWindowEnd),
			keel.DynOf(st.ScheduledWindowEnd).
				Coalesce(st.ScheduledWindowStart).
				Gte(opts.ActivityWindowStart),
		),
	)
}

func unassignedShipmentPredicate() keel.Pred {
	m := shipmentdb.Shipments.Moves()
	a := m.Assignment()

	return m.None(
		m.Status.Ne(shipment.MoveStatusCanceled),
		a.ArchivedAt.IsNull(),
		a.Status.Ne(shipment.AssignmentStatusCanceled),
	)
}

func shipmentScope(
	filter *pagination.QueryOptions,
	opts *shipmentports.ShipmentOptions,
	quick []keel.Pred,
) *keel.Query[*shipment.Shipment] {
	q := shipmentdb.Shipments.Where(shipmentOptionPredicates(opts)...).Where(quick...)
	if filter == nil {
		return q
	}

	return listquery.Filter(q, filter)
}
