package statemachine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/statemachine"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachineManager_CalculateStatuses(t *testing.T) {
	log := testutils.NewTestLogger(t)

	manager := statemachine.NewManager(statemachine.ManagerParams{
		Logger: log,
	})

	createTestShipment := func(stops []*shipment.Stop) *shipment.Shipment {
		move := &shipment.ShipmentMove{
			ID:       pulid.MustNew("move_"),
			Status:   shipment.MoveStatusNew,
			Sequence: 0,
			Stops:    stops,
		}

		return &shipment.Shipment{
			ID:             pulid.MustNew("shp_"),
			Status:         shipment.StatusNew,
			ProNumber:      "123456",
			ShipmentTypeID: pulid.MustNew("st_"),
			CustomerID:     pulid.MustNew("cust_"),
			BOL:            "1234567890",
			Moves:          []*shipment.ShipmentMove{move},
		}
	}

	// Test cases
	testCases := []struct {
		name                 string
		setupShipment        func() *shipment.Shipment
		expectedStopStatuses []shipment.StopStatus
		expectedMoveStatus   shipment.MoveStatus
		expectedShipStatus   shipment.Status
	}{
		{
			name: "No status changes when no actions taken",
			setupShipment: func() *shipment.Shipment {
				// Basic shipment with stops in New status
				stops := []*shipment.Stop{
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
				}
				return createTestShipment(stops)
			},
			expectedStopStatuses: []shipment.StopStatus{shipment.StopStatusNew, shipment.StopStatusNew},
			expectedMoveStatus:   shipment.MoveStatusNew,
			expectedShipStatus:   shipment.StatusNew,
		},
		{
			name: "First stop arrival updates only that stop status",
			setupShipment: func() *shipment.Shipment {
				stops := []*shipment.Stop{
					{
						ID:               pulid.MustNew("stp_"),
						Type:             shipment.StopTypePickup,
						Status:           shipment.StopStatusNew,
						Sequence:         0,
						PlannedArrival:   100,
						PlannedDeparture: 200,
						ActualArrival:    &[]int64{150}[0], // Driver has arrived
					},
					{
						ID:               pulid.MustNew("stp_"),
						Type:             shipment.StopTypeDelivery,
						Status:           shipment.StopStatusNew,
						Sequence:         1,
						PlannedArrival:   300,
						PlannedDeparture: 400,
					},
				}
				return createTestShipment(stops)
			},
			expectedStopStatuses: []shipment.StopStatus{shipment.StopStatusInTransit, shipment.StopStatusNew},
			expectedMoveStatus:   shipment.MoveStatusInTransit,
			expectedShipStatus:   shipment.StatusInTransit,
		},
		{
			name: "First stop departure triggers move and shipment status update",
			setupShipment: func() *shipment.Shipment {
				stops := []*shipment.Stop{
					{
						ID:               pulid.MustNew("stp_"),
						Type:             shipment.StopTypePickup,
						Status:           shipment.StopStatusNew,
						Sequence:         0,
						PlannedArrival:   100,
						PlannedDeparture: 200,
						ActualArrival:    &[]int64{150}[0],
						ActualDeparture:  &[]int64{190}[0], // Driver has departed first pickup
					},
					{
						ID:               pulid.MustNew("stp_"),
						Type:             shipment.StopTypeDelivery,
						Status:           shipment.StopStatusNew,
						Sequence:         1,
						PlannedArrival:   300,
						PlannedDeparture: 400,
					},
				}
				return createTestShipment(stops)
			},
			expectedStopStatuses: []shipment.StopStatus{shipment.StopStatusCompleted, shipment.StopStatusNew},
			expectedMoveStatus:   shipment.MoveStatusInTransit,
			expectedShipStatus:   shipment.StatusInTransit,
		},

		{
			name: "All stops completed completes move and shipment",
			setupShipment: func() *shipment.Shipment {
				stops := []*shipment.Stop{
					{
						ID:               pulid.MustNew("stp_"),
						Type:             shipment.StopTypePickup,
						Status:           shipment.StopStatusNew,
						Sequence:         0,
						PlannedArrival:   100,
						PlannedDeparture: 200,
						ActualArrival:    &[]int64{150}[0],
						ActualDeparture:  &[]int64{190}[0],
					},
					{
						ID:               pulid.MustNew("stp_"),
						Type:             shipment.StopTypeDelivery,
						Status:           shipment.StopStatusNew,
						Sequence:         1,
						PlannedArrival:   300,
						PlannedDeparture: 400,
						ActualArrival:    &[]int64{350}[0],
						ActualDeparture:  &[]int64{390}[0], // All stops completed
					},
				}
				return createTestShipment(stops)
			},
			expectedStopStatuses: []shipment.StopStatus{shipment.StopStatusCompleted, shipment.StopStatusCompleted},
			expectedMoveStatus:   shipment.MoveStatusCompleted,
			expectedShipStatus:   shipment.StatusCompleted,
		},

		{
			name: "Terminal status (canceled) prevents any transitions",
			setupShipment: func() *shipment.Shipment {
				// Create a shipment that's already canceled
				shp := createTestShipment([]*shipment.Stop{
					{
						ID:               pulid.MustNew("stp_"),
						Type:             shipment.StopTypePickup,
						Status:           shipment.StopStatusCanceled,
						Sequence:         0,
						PlannedArrival:   100,
						PlannedDeparture: 200,
					},
					{
						ID:               pulid.MustNew("stp_"),
						Type:             shipment.StopTypeDelivery,
						Status:           shipment.StopStatusCanceled,
						Sequence:         1,
						PlannedArrival:   300,
						PlannedDeparture: 400,
					},
				})
				shp.Status = shipment.StatusCanceled
				shp.Moves[0].Status = shipment.MoveStatusCanceled

				// Add actual arrival/departure times which would normally trigger transitions
				shp.Moves[0].Stops[0].ActualArrival = &[]int64{150}[0]
				shp.Moves[0].Stops[0].ActualDeparture = &[]int64{190}[0]
				shp.Moves[0].Stops[1].ActualArrival = &[]int64{350}[0]
				shp.Moves[0].Stops[1].ActualDeparture = &[]int64{390}[0]

				return shp
			},
			expectedStopStatuses: []shipment.StopStatus{shipment.StopStatusCanceled, shipment.StopStatusCanceled},
			expectedMoveStatus:   shipment.MoveStatusCanceled,
			expectedShipStatus:   shipment.StatusCanceled,
		},

		{
			name: "Multiple moves with some completed, partial completion",
			setupShipment: func() *shipment.Shipment {
				// First move is completed
				move1 := &shipment.ShipmentMove{
					ID:       pulid.MustNew("smv_"),
					Status:   shipment.MoveStatusNew,
					Sequence: 0,
					Stops: []*shipment.Stop{
						{
							ID:               pulid.MustNew("stp_"),
							Type:             shipment.StopTypePickup,
							Status:           shipment.StopStatusNew,
							Sequence:         0,
							PlannedArrival:   100,
							PlannedDeparture: 200,
							ActualArrival:    &[]int64{150}[0],
							ActualDeparture:  &[]int64{190}[0],
						},
						{
							ID:               pulid.MustNew("stp_"),
							Type:             shipment.StopTypeDelivery,
							Status:           shipment.StopStatusNew,
							Sequence:         1,
							PlannedArrival:   300,
							PlannedDeparture: 400,
							ActualArrival:    &[]int64{350}[0],
							ActualDeparture:  &[]int64{390}[0],
						},
					},
				}

				// Second move is still in progress
				move2 := &shipment.ShipmentMove{
					ID:       pulid.MustNew("smv_"),
					Status:   shipment.MoveStatusNew,
					Sequence: 1,
					Stops: []*shipment.Stop{
						{
							ID:               pulid.MustNew("stp_"),
							Type:             shipment.StopTypePickup,
							Status:           shipment.StopStatusNew,
							Sequence:         0,
							PlannedArrival:   500,
							PlannedDeparture: 600,
							ActualArrival:    &[]int64{550}[0],
							ActualDeparture:  &[]int64{600}[0],
						},
						{
							ID:               pulid.MustNew("stp_"),
							Type:             shipment.StopTypeDelivery,
							Status:           shipment.StopStatusNew,
							Sequence:         1,
							PlannedArrival:   700,
							PlannedDeparture: 800,
						},
					},
				}

				shp := &shipment.Shipment{
					ID:        pulid.MustNew("shp_"),
					Status:    shipment.StatusNew,
					ProNumber: "TEST123",
					Moves:     []*shipment.ShipmentMove{move1, move2},
				}

				return shp
			},
			expectedStopStatuses: []shipment.StopStatus{
				shipment.StopStatusCompleted, shipment.StopStatusCompleted, // First move stops
				shipment.StopStatusCompleted, shipment.StopStatusNew, // Second move stops
			},
			expectedMoveStatus: shipment.MoveStatusCompleted,
			expectedShipStatus: shipment.StatusPartiallyCompleted,
		},

		{
			name: "Move with assignment updates move status",
			setupShipment: func() *shipment.Shipment {
				// Move with assignment but no activity yet
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

				return &shipment.Shipment{
					ID:        pulid.MustNew("shp_"),
					Status:    shipment.StatusNew,
					ProNumber: "TEST123",
					Moves:     []*shipment.ShipmentMove{move},
				}
			},
			expectedStopStatuses: []shipment.StopStatus{shipment.StopStatusNew, shipment.StopStatusNew},
			expectedMoveStatus:   shipment.MoveStatusAssigned,
			expectedShipStatus:   shipment.StatusAssigned,
		},
	}

	for _, tc := range testCases {
		shp := tc.setupShipment()

		// Call the method under tests
		err := manager.CalculateStatuses(shp)
		require.NoError(t, err)

		// Verify the stop statuses
		for i, expectedStatus := range tc.expectedStopStatuses {
			var moveIdx, stopIdx int
			if i >= len(shp.Moves[0].Stops) {
				moveIdx = 1
				stopIdx = i - len(shp.Moves[0].Stops)
			} else {
				moveIdx = 0
				stopIdx = i
			}

			actualStatus := shp.Moves[moveIdx].Stops[stopIdx].Status
			assert.Equal(t, expectedStatus, actualStatus, "Stop status incorrect. Expected %s, got %s", expectedStatus, actualStatus)
		}

		assert.Equal(t, tc.expectedMoveStatus, shp.Moves[0].Status, "Move Status incorrect. Expected %s, got %s", tc.expectedMoveStatus, shp.Moves[0].Status)
		assert.Equal(t, tc.expectedShipStatus, shp.Status, "Shipment status incorrect. Expected %s, got %s", tc.expectedShipStatus, shp.Status)
	}
}

// Test to ensure transition validation works correctly
func TestStateMachine_Transitions(t *testing.T) {
	t.Run("Stop Transitions", func(t *testing.T) {
		stop := &shipment.Stop{
			ID:     pulid.MustNew("stp_"),
			Status: shipment.StopStatusNew,
		}

		sm := statemachine.NewStopStateMachine(stop)

		// Valid transitions
		assert.True(t, sm.CanTransition(statemachine.EventStopArrived))
		require.NoError(t, sm.Transition(statemachine.EventStopArrived))
		assert.Equal(t, shipment.StopStatusInTransit, stop.Status)

		assert.True(t, sm.CanTransition(statemachine.EventStopDeparted))
		require.NoError(t, sm.Transition(statemachine.EventStopDeparted))
		assert.Equal(t, shipment.StopStatusCompleted, stop.Status)

		// Invalid transition
		assert.False(t, sm.CanTransition(statemachine.EventStopArrived))
		assert.Error(t, sm.Transition(statemachine.EventStopArrived))
	})

	t.Run("Move Transitions", func(t *testing.T) {
		move := &shipment.ShipmentMove{
			ID:     pulid.MustNew("smv_"),
			Status: shipment.MoveStatusNew,
		}

		sm := statemachine.NewMoveStateMachine(move)

		// Valid transitions
		assert.True(t, sm.CanTransition(statemachine.EventMoveAssigned))
		require.NoError(t, sm.Transition(statemachine.EventMoveAssigned))
		assert.Equal(t, shipment.MoveStatusAssigned, move.Status)

		assert.True(t, sm.CanTransition(statemachine.EventMoveStarted))
		require.NoError(t, sm.Transition(statemachine.EventMoveStarted))
		assert.Equal(t, shipment.MoveStatusInTransit, move.Status)

		assert.True(t, sm.CanTransition(statemachine.EventMoveCompleted))
		require.NoError(t, sm.Transition(statemachine.EventMoveCompleted))
		assert.Equal(t, shipment.MoveStatusCompleted, move.Status)

		// Invalid transition
		assert.False(t, sm.CanTransition(statemachine.EventMoveAssigned))
		assert.Error(t, sm.Transition(statemachine.EventMoveAssigned))
	})

	t.Run("Shipment Transitions", func(t *testing.T) {
		shp := &shipment.Shipment{
			ID:     pulid.MustNew("shp_"),
			Status: shipment.StatusNew,
		}

		sm := statemachine.NewShipmentStateMachine(shp)

		// Valid transitions
		assert.True(t, sm.CanTransition(statemachine.EventShipmentAssigned))
		require.NoError(t, sm.Transition(statemachine.EventShipmentAssigned))
		assert.Equal(t, shipment.StatusAssigned, shp.Status)

		assert.True(t, sm.CanTransition(statemachine.EventShipmentInTransit))
		require.NoError(t, sm.Transition(statemachine.EventShipmentInTransit))
		assert.Equal(t, shipment.StatusInTransit, shp.Status)

		assert.True(t, sm.CanTransition(statemachine.EventShipmentCompleted))
		require.NoError(t, sm.Transition(statemachine.EventShipmentCompleted))
		assert.Equal(t, shipment.StatusCompleted, shp.Status)

		// Invalid transition
		assert.False(t, sm.CanTransition(statemachine.EventShipmentInTransit))
		assert.Error(t, sm.Transition(statemachine.EventShipmentInTransit))
	})

	t.Run("Terminal States", func(t *testing.T) {
		// Test terminal states for stops
		canceledStop := &shipment.Stop{
			ID:     pulid.MustNew("stp_"),
			Status: shipment.StopStatusCanceled,
		}
		stopSM := statemachine.NewStopStateMachine(canceledStop)
		assert.True(t, stopSM.IsInTerminalState())
		assert.False(t, stopSM.CanTransition(statemachine.EventStopArrived))

		// Test terminal states for moves
		canceledMove := &shipment.ShipmentMove{
			ID:     pulid.MustNew("smv_"),
			Status: shipment.MoveStatusCanceled,
		}
		moveSM := statemachine.NewMoveStateMachine(canceledMove)
		assert.True(t, moveSM.IsInTerminalState())
		assert.False(t, moveSM.CanTransition(statemachine.EventMoveStarted))

		// Test terminal states for shipments
		canceledShipment := &shipment.Shipment{
			ID:     pulid.MustNew("shp_"),
			Status: shipment.StatusCanceled,
		}
		shipmentSM := statemachine.NewShipmentStateMachine(canceledShipment)
		assert.True(t, shipmentSM.IsInTerminalState())
		assert.False(t, shipmentSM.CanTransition(statemachine.EventShipmentInTransit))
	})
}
