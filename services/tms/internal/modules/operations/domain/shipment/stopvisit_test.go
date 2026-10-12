package shipment_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func visitStop(sequence int64, locationID pulid.ID, arrived, departed bool) *shipment.Stop {
	stop := &shipment.Stop{
		ID:         pulid.MustNew("stp_"),
		Sequence:   sequence,
		LocationID: locationID,
		Status:     shipment.StopStatusNew,
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

		stop, match := move.MatchObservedVisit(yard, shipment.VisitArrival)
		assert.Equal(t, shipment.VisitMatchStop, match)
		assert.Equal(t, ret.ID, stop.ID)
	})

	t.Run("re-entering a stop the truck is still at is a duplicate", func(t *testing.T) {
		t.Parallel()
		onSite := visitStop(1, yard, true, false)
		later := visitStop(3, yard, false, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{onSite, later}}

		_, match := move.MatchObservedVisit(yard, shipment.VisitArrival)
		assert.Equal(t, shipment.VisitMatchDuplicate, match)
	})

	t.Run("departure matches the stop the truck is at", func(t *testing.T) {
		t.Parallel()
		onSite := visitStop(2, consignee, true, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{onSite}}

		stop, match := move.MatchObservedVisit(consignee, shipment.VisitDeparture)
		assert.Equal(t, shipment.VisitMatchStop, match)
		assert.Equal(t, onSite.ID, stop.ID)
	})

	t.Run("departure without a recorded arrival still names the stop", func(t *testing.T) {
		t.Parallel()
		unarrived := visitStop(1, yard, false, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{unarrived}}

		stop, match := move.MatchObservedVisit(yard, shipment.VisitDeparture)
		assert.Equal(t, shipment.VisitMatchStop, match)
		assert.Equal(t, unarrived.ID, stop.ID)
	})

	t.Run("finished stops make a repeat signal a duplicate", func(t *testing.T) {
		t.Parallel()
		done := visitStop(1, shipper, true, true)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{done}}

		_, arrive := move.MatchObservedVisit(shipper, shipment.VisitArrival)
		_, depart := move.MatchObservedVisit(shipper, shipment.VisitDeparture)
		assert.Equal(t, shipment.VisitMatchDuplicate, arrive)
		assert.Equal(t, shipment.VisitMatchDuplicate, depart)
	})

	t.Run("a location not on the move or no location never picks a stop", func(t *testing.T) {
		t.Parallel()
		canceled := visitStop(2, elsewhere, false, false)
		canceled.Status = shipment.StopStatusCanceled
		move := &shipment.ShipmentMove{
			Stops: []*shipment.Stop{visitStop(1, shipper, false, false), canceled},
		}

		for _, locationID := range []pulid.ID{elsewhere, pulid.Nil} {
			stop, match := move.MatchObservedVisit(locationID, shipment.VisitArrival)
			assert.Nil(t, stop)
			assert.Equal(t, shipment.VisitMatchNoStop, match)
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

		assert.Equal(t, onSite.ID, move.StopForObservedVisit(yard, shipment.VisitArrival).ID)
	})

	t.Run("a late arrival for a finished stop names that stop", func(t *testing.T) {
		t.Parallel()
		done := visitStop(1, shipper, true, true)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{done}}

		assert.Equal(t, done.ID, move.StopForObservedVisit(shipper, shipment.VisitArrival).ID)
		assert.Equal(t, done.ID, move.StopForObservedVisit(shipper, shipment.VisitDeparture).ID)
	})

	t.Run("an unarrived stop is named before a finished one at the same place", func(t *testing.T) {
		t.Parallel()
		first := visitStop(1, yard, true, true)
		ret := visitStop(3, yard, false, false)
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{first, ret}}

		assert.Equal(t, ret.ID, move.StopForObservedVisit(yard, shipment.VisitArrival).ID)
	})

	t.Run("a place that is not a stop names nothing", func(t *testing.T) {
		t.Parallel()
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{visitStop(1, yard, false, false)}}

		assert.Nil(t, move.StopForObservedVisit(elsewhere, shipment.VisitArrival))
		assert.Nil(t, move.StopForObservedVisit(pulid.Nil, shipment.VisitArrival))
	})
}

func TestStopForReportedVisit(t *testing.T) {
	t.Parallel()

	pickup := func(stop *shipment.Stop) *shipment.Stop {
		stop.Type = shipment.StopTypePickup
		return stop
	}
	delivery := func(stop *shipment.Stop) *shipment.Stop {
		stop.Type = shipment.StopTypeDelivery
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

		assert.Equal(t, middle.ID, move.StopForReportedVisit(isDelivery, shipment.VisitArrival).ID)
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
			move.StopForReportedVisit(isDelivery, shipment.VisitDeparture).ID,
		)
	})

	t.Run("no stop of the role names nothing", func(t *testing.T) {
		t.Parallel()
		move := &shipment.ShipmentMove{Stops: []*shipment.Stop{
			pickup(visitStop(1, pulid.MustNew("loc_"), false, false)),
		}}

		assert.Nil(t, move.StopForReportedVisit(isDelivery, shipment.VisitArrival))
	})
}
