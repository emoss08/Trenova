package shipmentpredicate

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
)

const (
	originStopAlias     = "origin_stp"
	originMoveAlias     = "origin_sm"
	originLocationAlias = "origin_loc"
	originStateAlias    = "origin_ust"
	destStopAlias       = "dest_stp"
	destMoveAlias       = "dest_sm"
	destLocationAlias   = "dest_loc"
	destStateAlias      = "dest_ust"
)

type StopPlace struct {
	stop  *shipmentdb.StopsTable
	loc   *locationdb.LocationsTable
	state *usstatedb.UsStatesTable
	query *keel.Query[*shipment.Stop]
}

func FirstPickupPlace() StopPlace {
	st := shipmentdb.Stops.As(originStopAlias)
	sm := shipmentdb.ShipmentMoves.As(originMoveAlias)

	return StopPlace{
		stop:  st,
		loc:   locationdb.Locations.As(originLocationAlias),
		state: usstatedb.UsStates.As(originStateAlias),
		query: st.Join(sm, sm.ID.EqCol(st.ShipmentMoveID)).
			Where(
				sm.ShipmentID.EqCol(shipmentdb.Shipments.ID),
				st.Type.In(shipmentvocab.PickupStopTypes()...),
			).
			Order(sm.Sequence.Asc(), st.Sequence.Asc()).
			Limit(1).
			Pick(st.LocationID),
	}
}

func LastDeliveryPlace() StopPlace {
	st := shipmentdb.Stops.As(destStopAlias)
	sm := shipmentdb.ShipmentMoves.As(destMoveAlias)

	return StopPlace{
		stop:  st,
		loc:   locationdb.Locations.As(destLocationAlias),
		state: usstatedb.UsStates.As(destStateAlias),
		query: st.Join(sm, sm.ID.EqCol(st.ShipmentMoveID)).
			Where(
				sm.ShipmentID.EqCol(shipmentdb.Shipments.ID),
				st.Type.In(shipmentvocab.DeliveryStopTypes()...),
			).
			Order(sm.Sequence.Desc(), st.Sequence.Desc()).
			Limit(1).
			Pick(st.LocationID),
	}
}

func (p StopPlace) Join(q *keel.Query[*shipment.Shipment]) *keel.Query[*shipment.Shipment] {
	return q.LeftJoinLateral(p.query).
		LeftJoin(p.loc, p.loc.ID.EqCol(p.stop.LocationID)).
		LeftJoin(p.state, p.state.ID.EqCol(p.loc.StateID))
}

func (p StopPlace) City() keel.StringExpr {
	return p.loc.City.Coalesce("")
}

func (p StopPlace) StateCode() keel.StringExpr {
	return p.state.Abbreviation.Coalesce("")
}
