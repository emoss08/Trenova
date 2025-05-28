package statemachine

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugMoveAssignment(t *testing.T) {
	log := testutils.NewTestLogger(t)

	manager := NewManager(ManagerParams{
		Logger: log,
	})

	// Create test shipment with assignment - exactly like the failing test
	move := &shipment.ShipmentMove{
		ID:         pulid.MustNew("smv_"),
		Status:     shipment.MoveStatusNew,
		Sequence:   0,
		Assignment: &shipment.Assignment{ID: pulid.MustNew("a_"), Status: shipment.AssignmentStatusNew},
		Stops: []*shipment.Stop{
			{
				ID:               pulid.MustNew("stp_"),
				Type:             shipment.StopTypePickup,
				Status:           shipment.StopStatusNew,
				Sequence:         0,
				PlannedArrival:   100,
				PlannedDeparture: 200,
			},
			{
				ID:               pulid.MustNew("stp_"),
				Type:             shipment.StopTypeDelivery,
				Status:           shipment.StopStatusNew,
				Sequence:         1,
				PlannedArrival:   300,
				PlannedDeparture: 400,
			},
		},
	}

	shp := &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		Status:    shipment.StatusNew,
		ProNumber: "TEST123",
		Moves:     []*shipment.ShipmentMove{move},
	}

	t.Logf("Initial state: Shipment=%s, Move=%s", shp.Status, move.Status)
	t.Logf("Move has assignment: %v", move.Assignment != nil)

	// Test determineMoveEvent directly
	moveEvent := manager.determineMoveEvent(move)
	if moveEvent != nil {
		t.Logf("Determined move event: %s", moveEvent.EventType())
	} else {
		t.Logf("No move event determined")
	}

	// Call the method under test
	err := manager.CalculateStatuses(shp)
	require.NoError(t, err)

	// Test determineShipmentEvent directly after move is updated
	shipmentEvent := manager.determineShipmentEvent(shp)
	if shipmentEvent != nil {
		t.Logf("Determined shipment event: %s", shipmentEvent.EventType())
	} else {
		t.Logf("No shipment event determined")
	}

	// Debug the counters
	var movesAssigned, movesInTransit, movesCompleted int
	for _, move := range shp.Moves {
		switch move.Status {
		case shipment.MoveStatusAssigned:
			movesAssigned++
		case shipment.MoveStatusInTransit:
			movesInTransit++
		case shipment.MoveStatusCompleted:
			movesCompleted++
		}
	}
	t.Logf("Counters: assigned=%d, inTransit=%d, completed=%d, total=%d", movesAssigned, movesInTransit, movesCompleted, len(shp.Moves))

	t.Logf("Final state: Shipment=%s, Move=%s", shp.Status, move.Status)

	// Check move status
	assert.Equal(t, shipment.MoveStatusAssigned, move.Status, "Move should be Assigned")
	// Check shipment status
	assert.Equal(t, shipment.StatusAssigned, shp.Status, "Shipment should be Assigned")
}
