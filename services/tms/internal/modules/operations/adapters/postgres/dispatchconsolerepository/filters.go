package dispatchconsolerepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

// openBoardStatuses are the moves that still need a dispatcher's attention. They are the
// board's default status filter, and they are also exempt from the window's lower bound:
// an overdue move is the most urgent thing on the board, not history to be hidden.
var openBoardStatuses = []shipmentvocab.MoveStatus{
	shipmentvocab.MoveStatusNew,
	shipmentvocab.MoveStatusAssigned,
	shipmentvocab.MoveStatusInTransit,
}

var openMoveStatuses = []shipmentvocab.MoveStatus{
	shipmentvocab.MoveStatusAssigned,
	shipmentvocab.MoveStatusInTransit,
}

type moveQuery = keel.Query[*shipment.ShipmentMove]

// stopEdge is a move's first or last live stop, read through a lateral subquery, with the
// stop's location and state joined to its output.
type stopEdge struct {
	stop  *shipmentdb.StopsTable
	loc   *locationdb.LocationsTable
	state *usstatedb.UsStatesTable
	last  bool
}

var (
	origin      = newStopEdge("orig", false)
	destination = newStopEdge("dest", true)

	assignments        = shipmentdb.Assignments
	carrierAssignments = shipmentdb.CarrierAssignments
	assignedCarriers   = carrierdb.Carriers.As("asn_car")
	assignedWorkers    = workerdb.Workers.As("asn_wrk")
	assignedTractors   = tractordb.Tractors.As("asn_trac")
	assignedTrailers   = trailerdb.Trailers.As("asn_tr")
)

func newStopEdge(alias string, last bool) stopEdge {
	return stopEdge{
		stop:  shipmentdb.Stops.As(alias),
		loc:   locationdb.Locations.As(alias + "_loc"),
		state: usstatedb.UsStates.As(alias + "_st"),
		last:  last,
	}
}

func joinEdge[T any](q *keel.Query[*T], e stopEdge, moveID keel.Expr) *keel.Query[*T] {
	st := e.stop
	order := st.Sequence.Asc()
	if e.last {
		order = st.Sequence.Desc()
	}

	return q.
		LeftJoinLateral(st.
			Pick(
				st.ID,
				st.LocationID,
				st.ScheduledWindowStart,
				st.ScheduledWindowEnd,
				st.ActualArrival,
			).
			Where(
				st.ShipmentMoveID.EqCol(moveID),
				st.Status.Ne(shipmentvocab.StopStatusCanceled),
			).
			Order(order).
			Limit(1)).
		LeftJoin(e.loc, e.loc.ID.EqCol(st.LocationID)).
		LeftJoin(e.state, e.state.ID.EqCol(e.loc.StateID))
}

func text(e keel.Expr) keel.StringExpr {
	return keel.StringOf(e).Coalesce("")
}

func enumText(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func zero(e keel.Expr) keel.Int64Expr {
	return keel.Int64Of(e).Coalesce(0)
}

// boardMoves is the move board's FROM clause: the move, its shipment and customer, its
// live assignment and carrier assignment, and its first and last stops.
func boardMoves() *moveQuery {
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	cus := customerdb.Customers
	asn := assignments
	ca := carrierAssignments

	q := sm.
		Join(sp, sp.ID.EqCol(sm.ShipmentID)).
		LeftJoin(cus, cus.ID.EqCol(sp.CustomerID)).
		LeftJoin(asn, asn.ShipmentMoveID.EqCol(sm.ID), asn.ArchivedAt.IsNull()).
		LeftJoin(ca,
			ca.ShipmentMoveID.EqCol(sm.ID),
			ca.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled),
		)
	q = joinEdge(q, origin, sm.ID)

	return joinEdge(q, destination, sm.ID)
}

func moveFilters(filter *dispatchboardports.DispatchBoardFilter) []keel.Pred {
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments

	statuses := openBoardStatuses
	if len(filter.MoveStatuses) > 0 {
		statuses = filter.MoveStatuses
	}

	preds := []keel.Pred{
		sm.Status.In(statuses...),
		keel.When(len(filter.MoveIDs) > 0, sm.ID.In(filter.MoveIDs...)),
		keel.When(len(filter.CustomerIDs) > 0, sp.CustomerID.In(filter.CustomerIDs...)),
		keel.When(len(filter.ServiceTypeIDs) > 0, sp.ServiceTypeID.In(filter.ServiceTypeIDs...)),
	}
	// The lower bound trims work the board has already finished with. Open moves are
	// exempt: a load whose delivery appointment lapsed two days ago is still uncovered,
	// still the dispatcher's problem, and would otherwise vanish from the board entirely.
	if filter.WindowStart > 0 {
		preds = append(preds, keel.Or(
			destination.stop.ScheduledWindowEnd.
				Coalesce(destination.stop.ScheduledWindowStart, origin.stop.ScheduledWindowStart).
				Gte(filter.WindowStart),
			sm.Status.In(openBoardStatuses...),
		))
	}
	if filter.WindowEnd > 0 {
		preds = append(preds,
			origin.stop.ScheduledWindowStart.Coalesce(0).Lte(filter.WindowEnd))
	}
	if filter.Query != "" {
		preds = append(preds, keel.Or(
			sp.ProNumber.ContainsFold(filter.Query),
			sp.BOL.ContainsFold(filter.Query),
			customerdb.Customers.Name.ContainsFold(filter.Query),
			origin.loc.City.ContainsFold(filter.Query),
			destination.loc.City.ContainsFold(filter.Query),
		))
	}

	return preds
}

func driverFilters(filter *dispatchboardports.DispatchBoardFilter) []keel.Pred {
	w := workerdb.Workers
	return []keel.Pred{
		w.Status.Eq(domaintypes.StatusActive),
		w.CanBeAssigned.IsTrue(),
		keel.When(len(filter.WorkerIDs) > 0, w.ID.In(filter.WorkerIDs...)),
		keel.When(len(filter.FleetCodeIDs) > 0, w.FleetCodeID.In(filter.FleetCodeIDs...)),
	}
}

// openAssignments counts the worker's live assignments on moves still under way.
func openAssignments(w *workerdb.WorkersTable) keel.Expr {
	a := shipmentdb.Assignments.As("oa")
	m := shipmentdb.ShipmentMoves.As("oa_sm")

	return keel.Scalar(a.
		Join(m, m.ID.EqCol(a.ShipmentMoveID)).
		Pick(keel.Count()).
		Where(
			a.PrimaryWorkerID.EqCol(w.ID),
			a.ArchivedAt.IsNull(),
			m.Status.In(openMoveStatuses...),
		))
}

var serviceTypes = servicetypedb.ServiceTypes
