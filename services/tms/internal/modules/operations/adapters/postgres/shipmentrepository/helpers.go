package shipmentrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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
	default:
		loads := make([]keel.Loader, 0, 4)
		if opts.IncludeCustomer {
			loads = append(loads, sp.Customer(), sp.BillToCustomer())
		}
		if opts.IncludeRoute {
			loads = append(loads, movesWithRoute())
		}
		if opts.IncludeCommodities {
			loads = append(loads, commodityNames())
		}
		return loads
	}
}

func commodityNames() keel.Loader {
	sc := shipmentdb.Shipments.Commodities()

	return sc.Order(sc.CreatedAt.Asc(), sc.ID.Asc()).Load(sc.Commodity())
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
