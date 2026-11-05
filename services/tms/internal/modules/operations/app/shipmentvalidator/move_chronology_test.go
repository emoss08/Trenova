package shipmentvalidator

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func storedShipmentWithOverlappingFinishedMove() *shipment.Shipment {
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 3
	finished := shipmentservicetest.ValidMove()
	finished.Status = shipmentvocab.MoveStatusCompleted
	finished.Stops = []*shipment.Stop{
		shipmentservicetest.MakeStop(shipmentvocab.StopTypePickup, 0, 1_000, 2_000),
		shipmentservicetest.MakeStop(shipmentvocab.StopTypeDelivery, 1, 1_500, 2_000),
	}
	open := shipmentservicetest.ValidMove()
	open.Sequence = 1
	open.Stops = []*shipment.Stop{
		shipmentservicetest.MakeStop(shipmentvocab.StopTypePickup, 0, 5_000, 6_000),
		shipmentservicetest.MakeStop(shipmentvocab.StopTypeDelivery, 1, 9_000, 10_000),
	}
	entity.Moves = []*shipment.ShipmentMove{finished, open}
	for _, move := range entity.Moves {
		move.ID = pulid.MustNew("sm_")
		move.ShipmentID = entity.ID
		for _, stop := range move.Stops {
			stop.ID = pulid.MustNew("stp_")
			stop.ShipmentMoveID = move.ID
		}
	}

	return entity
}

func TestMoveChronology_AWriteThatLeavesTheStopsAloneIsNotRefusedOverAFinishedMovesOverlap(
	t *testing.T,
) {
	t.Parallel()

	original := storedShipmentWithOverlappingFinishedMove()
	updated := shipmentservicetest.CloneShipment(original)
	updated.Moves[1].CoverageType = shipmentvocab.MoveCoverageTypeDriver

	assert.Nil(t, validateMoveChronology(original, updated))
}

func TestMoveChronology_ChecksEveryPairOfANewShipment(t *testing.T) {
	t.Parallel()

	entity := storedShipmentWithOverlappingFinishedMove()

	multiErr := validateMoveChronology(nil, entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].scheduledWindowEnd")
}

func TestMoveChronology_RefusesAnUpdateThatRetimesAStopIntoTheNextOnesWindow(t *testing.T) {
	t.Parallel()

	original := storedShipmentWithOverlappingFinishedMove()
	updated := shipmentservicetest.CloneShipment(original)
	updated.Moves[1].Stops[0].ScheduledWindowEnd = new(int64(9_500))

	multiErr := validateMoveChronology(original, updated)

	require.NotNil(t, multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "moves[1].stops[0].scheduledWindowEnd", multiErr.Errors[0].Field)
}

func TestMoveChronology_ChecksAStopAddedBetweenTwoStoredStops(t *testing.T) {
	t.Parallel()

	original := storedShipmentWithOverlappingFinishedMove()
	updated := shipmentservicetest.CloneShipment(original)
	added := shipmentservicetest.MakeStop(shipmentvocab.StopTypePickup, 1, 5_500, 9_500)
	updated.Moves[1].Stops = []*shipment.Stop{
		updated.Moves[1].Stops[0],
		added,
		updated.Moves[1].Stops[1],
	}

	multiErr := validateMoveChronology(original, updated)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[1].stops[0].scheduledWindowEnd")
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[1].stops[1].scheduledWindowEnd")
}

func TestMoveChronology_RefusesAnActualDepartureRecordedAfterTheNextArrival(t *testing.T) {
	t.Parallel()

	original := storedShipmentWithOverlappingFinishedMove()
	updated := shipmentservicetest.CloneShipment(original)
	departure, arrival := int64(7_000), int64(6_500)
	updated.Moves[1].Stops[0].ActualDeparture = &departure
	updated.Moves[1].Stops[1].ActualArrival = &arrival

	multiErr := validateMoveChronology(original, updated)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[1].stops[0].actualDeparture")
}
