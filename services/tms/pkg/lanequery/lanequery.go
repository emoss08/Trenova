package lanequery

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/pkg/rankquery"
)

const (
	endpointsCTE          = "lane_endpoints"
	lanesCTE              = "lanes"
	originLocationID      = "origin_location_id"
	destinationLocationID = "destination_location_id"

	originLocationAlias      = "loc_orig"
	destinationLocationAlias = "loc_dest"
	originStateAlias         = "ust_orig"
	destinationStateAlias    = "ust_dest"
)

type Options struct {
	Shipments []keel.Pred
	Limit     int
}

func New(opts Options) keel.Projection {
	return lanes(opts)
}

func lanes(opts Options) *keel.Query[*keel.Row] {
	endpoints := keel.CTE(endpointsCTE, []string{originLocationID, destinationLocationID},
		shipmentEndpoints(opts.Shipments))
	origin := endpoints.Col(originLocationID)
	destination := endpoints.Col(destinationLocationID)

	originLocation := locationdb.Locations.As(originLocationAlias)
	destinationLocation := locationdb.Locations.As(destinationLocationAlias)
	originState := usstatedb.UsStates.As(originStateAlias)
	destinationState := usstatedb.UsStates.As(destinationStateAlias)
	count := keel.Count()

	grouped := endpoints.Query().
		Join(originLocation, originLocation.ID.EqCol(origin)).
		Join(originState, originState.ID.EqCol(originLocation.StateID)).
		Join(destinationLocation, destinationLocation.ID.EqCol(destination)).
		Join(destinationState, destinationState.ID.EqCol(destinationLocation.StateID)).
		Where(origin.IsNotNull(), destination.IsNotNull()).
		Group(originState.Abbreviation, destinationState.Abbreviation).
		Pick(originState.Abbreviation, destinationState.Abbreviation, count)
	rank := []keel.Order{
		keel.Desc(count),
		originState.Abbreviation.Asc(),
		destinationState.Abbreviation.Asc(),
	}
	if opts.Limit <= 0 {
		return grouped.With(endpoints).Order(rank...)
	}

	return rankquery.Top(grouped, rankquery.Params{
		Name:    lanesCTE,
		Columns: []string{"origin_state", "destination_state", "shipments"},
		With:    []*keel.CTETable{endpoints},
		Rank:    rank,
		Limit:   opts.Limit,
	})
}

func shipmentEndpoints(filter []keel.Pred) *keel.Query[*shipment.Shipment] {
	s := shipmentdb.Shipments
	m := shipmentdb.ShipmentMoves
	st := shipmentdb.Stops

	return s.Join(m, m.ShipmentID.EqCol(s.ID)).
		Join(st, st.ShipmentMoveID.EqCol(m.ID)).
		Where(filter...).
		Group(s.ID).
		Pick(
			keel.StringArrayOf(keel.ArrayAgg(st.LocationID).
				OrderBy(m.Sequence.Asc(), st.Sequence.Asc()).
				Filter(st.Type.In(shipmentvocab.PickupStopTypes()...))).At(1),
			keel.StringArrayOf(keel.ArrayAgg(st.LocationID).
				OrderBy(m.Sequence.Desc(), st.Sequence.Desc()).
				Filter(st.Type.In(shipmentvocab.DeliveryStopTypes()...))).At(1),
		)
}
