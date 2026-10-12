package shipment_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func changeMove(stops ...*shipment.Stop) *shipment.ShipmentMove {
	return &shipment.ShipmentMove{Status: shipment.MoveStatusAssigned, Stops: stops}
}

func at(v int64) *int64 { return &v }

func TestApplyStopActualChangesSetsTimesAndStatuses(t *testing.T) {
	t.Parallel()

	first := visitStop(1, pulid.MustNew("loc_"), false, false)
	second := visitStop(2, pulid.MustNew("loc_"), false, false)
	move := changeMove(first, second)

	touched, err := move.ApplyStopActualChanges([]shipment.StopActualChange{
		{StopID: first.ID, Arrival: at(100), Departure: at(200)},
		{StopID: second.ID, Arrival: at(300)},
	})

	require.NoError(t, err)
	assert.Len(t, touched, 2)
	assert.Equal(t, shipment.StopStatusCompleted, first.Status)
	assert.Equal(t, shipment.StopStatusInTransit, second.Status)
	assert.Equal(t, at(300), second.ActualArrival)
}

func TestApplyStopActualChangesRefusesABrokenTimeline(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		changes func(first, second *shipment.Stop) []shipment.StopActualChange
		message string
	}{
		{
			name: "a later stop before the earlier one finishes",
			changes: func(_, second *shipment.Stop) []shipment.StopActualChange {
				return []shipment.StopActualChange{{StopID: second.ID, Arrival: at(300)}}
			},
			message: "Complete the earlier stops on this load first",
		},
		{
			name: "a departure before the arrival",
			changes: func(first, _ *shipment.Stop) []shipment.StopActualChange {
				return []shipment.StopActualChange{
					{StopID: first.ID, Arrival: at(200), Departure: at(100)},
				}
			},
			message: "Event time predates the arrival at this stop",
		},
		{
			name: "an arrival before the previous departure",
			changes: func(first, second *shipment.Stop) []shipment.StopActualChange {
				return []shipment.StopActualChange{
					{StopID: first.ID, Arrival: at(100), Departure: at(500)},
					{StopID: second.ID, Arrival: at(400)},
				}
			},
			message: "Event time predates the previous stop's departure",
		},
		{
			name: "a departure without an arrival",
			changes: func(first, _ *shipment.Stop) []shipment.StopActualChange {
				return []shipment.StopActualChange{{StopID: first.ID, Departure: at(100)}}
			},
			message: "Arrive at this stop before departing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			first := visitStop(1, pulid.MustNew("loc_"), false, false)
			second := visitStop(2, pulid.MustNew("loc_"), false, false)
			move := changeMove(first, second)

			_, err := move.ApplyStopActualChanges(tc.changes(first, second))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.message)
		})
	}
}

func TestApplyStopActualChangesRefusesACanceledLoadOrAMissingStop(t *testing.T) {
	t.Parallel()

	stop := visitStop(1, pulid.MustNew("loc_"), false, false)
	canceled := changeMove(stop)
	canceled.Status = shipment.MoveStatusCanceled
	_, err := canceled.ApplyStopActualChanges(
		[]shipment.StopActualChange{{StopID: stop.ID, Arrival: at(1)}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "This load has been canceled")

	_, err = changeMove(stop).ApplyStopActualChanges(
		[]shipment.StopActualChange{{StopID: pulid.MustNew("stp_"), Arrival: at(1)}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Stop not found on this load")
}

func TestApplyStopActualChangesCorrectsACompletedLoad(t *testing.T) {
	t.Parallel()

	stop := visitStop(1, pulid.MustNew("loc_"), true, true)
	move := changeMove(stop)
	move.Status = shipment.MoveStatusCompleted

	_, err := move.ApplyStopActualChanges(
		[]shipment.StopActualChange{{StopID: stop.ID, Arrival: at(50)}},
	)

	require.NoError(t, err)
	assert.Equal(t, at(50), stop.ActualArrival)
	assert.Equal(t, shipment.StopStatusCompleted, stop.Status)
}
