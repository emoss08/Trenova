package stopvisit_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopvisit"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func visitStop(sequence int64, locationID pulid.ID, arrived, departed bool) *shipment.Stop {
	stop := &shipment.Stop{
		ID:         pulid.MustNew("stp_"),
		Sequence:   sequence,
		LocationID: locationID,
		Status:     shipmentvocab.StopStatusNew,
	}
	at := int64(100 + sequence)
	if arrived {
		stop.ActualArrival = &at
	}
	if departed {
		stop.ActualDeparture = &at
	}
	return stop
}

func TestMatchObservedVisit(t *testing.T) {
	t.Parallel()

	yard := pulid.MustNew("loc_")
	shipper := pulid.MustNew("loc_")
	consignee := pulid.MustNew("loc_")
	elsewhere := pulid.MustNew("loc_")

	t.Run("arrival matches the first unarrived stop at the location", func(t *testing.T) {
		t.Parallel()
		first := visitStop(1, yard, true, true)
		pickup := visitStop(2, shipper, false, false)
		ret := visitStop(3, yard, false, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{ret, pickup, first}}

		stop, match := stopvisit.MatchObserved(move, yard, shipmentvocab.VisitArrival)
		assert.Equal(t, stopvisit.MatchStop, match)
		assert.Equal(t, ret.ID, stop.ID)
	})

	t.Run("re-entering a stop the truck is still at is a duplicate", func(t *testing.T) {
		t.Parallel()
		onSite := visitStop(1, yard, true, false)
		later := visitStop(3, yard, false, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{onSite, later}}

		_, match := stopvisit.MatchObserved(move, yard, shipmentvocab.VisitArrival)
		assert.Equal(t, stopvisit.MatchDuplicate, match)
	})

	t.Run("departure matches the stop the truck is at", func(t *testing.T) {
		t.Parallel()
		onSite := visitStop(2, consignee, true, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{onSite}}

		stop, match := stopvisit.MatchObserved(move, consignee, shipmentvocab.VisitDeparture)
		assert.Equal(t, stopvisit.MatchStop, match)
		assert.Equal(t, onSite.ID, stop.ID)
	})

	t.Run("departure without a recorded arrival still names the stop", func(t *testing.T) {
		t.Parallel()
		unarrived := visitStop(1, yard, false, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{unarrived}}

		stop, match := stopvisit.MatchObserved(move, yard, shipmentvocab.VisitDeparture)
		assert.Equal(t, stopvisit.MatchStop, match)
		assert.Equal(t, unarrived.ID, stop.ID)
	})

	t.Run("finished stops make a repeat signal a duplicate", func(t *testing.T) {
		t.Parallel()
		done := visitStop(1, shipper, true, true)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{done}}

		_, arrive := stopvisit.MatchObserved(move, shipper, shipmentvocab.VisitArrival)
		_, depart := stopvisit.MatchObserved(move, shipper, shipmentvocab.VisitDeparture)
		assert.Equal(t, stopvisit.MatchDuplicate, arrive)
		assert.Equal(t, stopvisit.MatchDuplicate, depart)
	})

	t.Run("a location not on the move or no location never picks a stop", func(t *testing.T) {
		t.Parallel()
		canceled := visitStop(2, elsewhere, false, false)
		canceled.Status = shipmentvocab.StopStatusCanceled
		move := &shipment.ShipmentMove{
			Stops: []*shipment.Stop{visitStop(1, shipper, false, false), canceled},
		}

		for _, locationID := range []pulid.ID{elsewhere, pulid.Nil} {
			stop, match := stopvisit.MatchObserved(move, locationID, shipmentvocab.VisitArrival)
			assert.Nil(t, stop)
			assert.Equal(t, stopvisit.MatchNoStop, match)
		}
	})
}

func TestStopForObservedVisit(t *testing.T) {
	t.Parallel()

	yard := pulid.MustNew("loc_")
	shipper := pulid.MustNew("loc_")
	elsewhere := pulid.MustNew("loc_")

	t.Run("a repeat arrival still names the stop the truck is at", func(t *testing.T) {
		t.Parallel()
		onSite := visitStop(1, yard, true, false)
		later := visitStop(3, yard, false, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{onSite, later}}

		assert.Equal(
			t,
			onSite.ID,
			stopvisit.StopForObserved(move, yard, shipmentvocab.VisitArrival).ID,
		)
	})

	t.Run("a late arrival for a finished stop names that stop", func(t *testing.T) {
		t.Parallel()
		done := visitStop(1, shipper, true, true)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{done}}

		assert.Equal(
			t,
			done.ID,
			stopvisit.StopForObserved(move, shipper, shipmentvocab.VisitArrival).ID,
		)
		assert.Equal(
			t,
			done.ID,
			stopvisit.StopForObserved(move, shipper, shipmentvocab.VisitDeparture).ID,
		)
	})

	t.Run("an unarrived stop is named before a finished one at the same place", func(t *testing.T) {
		t.Parallel()
		first := visitStop(1, yard, true, true)
		ret := visitStop(3, yard, false, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{first, ret}}

		assert.Equal(
			t,
			ret.ID,
			stopvisit.StopForObserved(move, yard, shipmentvocab.VisitArrival).ID,
		)
	})

	t.Run("a place that is not a stop names nothing", func(t *testing.T) {
		t.Parallel()
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{visitStop(1, yard, false, false)}}

		assert.Nil(t, stopvisit.StopForObserved(move, elsewhere, shipmentvocab.VisitArrival))
		assert.Nil(t, stopvisit.StopForObserved(move, pulid.Nil, shipmentvocab.VisitArrival))
	})
}

func TestStopForReportedVisit(t *testing.T) {
	t.Parallel()

	pickup := func(stop *shipment.Stop) *shipment.Stop {
		stop.Type = shipmentvocab.StopTypePickup
		return stop
	}
	delivery := func(stop *shipment.Stop) *shipment.Stop {
		stop.Type = shipmentvocab.StopTypeDelivery
		return stop
	}
	isDelivery := func(stop *shipment.Stop) bool { return stop.IsDestinationStop() }

	t.Run("an arrival names the first delivery not yet reached", func(t *testing.T) {
		t.Parallel()
		first := delivery(visitStop(2, pulid.MustNew("loc_"), true, true))
		middle := delivery(visitStop(3, pulid.MustNew("loc_"), false, false))
		last := delivery(visitStop(4, pulid.MustNew("loc_"), false, false))
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{
			pickup(visitStop(1, pulid.MustNew("loc_"), true, true)), last, middle, first,
		}}

		assert.Equal(
			t,
			middle.ID,
			stopvisit.StopForReported(move, isDelivery, shipmentvocab.VisitArrival).ID,
		)
	})

	t.Run("a departure names the delivery the truck is at", func(t *testing.T) {
		t.Parallel()
		onSite := delivery(visitStop(2, pulid.MustNew("loc_"), true, false))
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{
			onSite, delivery(visitStop(3, pulid.MustNew("loc_"), false, false)),
		}}

		assert.Equal(
			t,
			onSite.ID,
			stopvisit.StopForReported(move, isDelivery, shipmentvocab.VisitDeparture).ID,
		)
	})

	t.Run("no stop of the role names nothing", func(t *testing.T) {
		t.Parallel()
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{
			pickup(visitStop(1, pulid.MustNew("loc_"), false, false)),
		}}

		assert.Nil(t, stopvisit.StopForReported(move, isDelivery, shipmentvocab.VisitArrival))
	})
}
