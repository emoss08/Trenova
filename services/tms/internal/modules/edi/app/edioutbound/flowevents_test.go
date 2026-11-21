package edioutbound

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/stretchr/testify/assert"
)

func TestStatusShipmentEvent_StopEventsOwnTheStopStatuses(t *testing.T) {
	t.Parallel()

	fromStop := map[string]any{shipmentevent.MetadataFromStopActual: true}
	tests := []struct {
		name  string
		event shipmentevent.Event
		sends bool
	}{
		{
			name:  "an arrival at a stop",
			event: shipmentevent.Event{Type: shipmentevent.TypeStopArrived},
			sends: true,
		},
		{
			name:  "a departure from a stop",
			event: shipmentevent.Event{Type: shipmentevent.TypeStopDeparted},
			sends: true,
		},
		{
			name:  "a corrected arrival",
			event: shipmentevent.Event{Type: shipmentevent.TypeStopArrivalCorrected},
		},
		{
			name:  "a corrected departure",
			event: shipmentevent.Event{Type: shipmentevent.TypeStopDepartureCorrected},
		},
		{
			name: "a move starting because a stop was reached",
			event: shipmentevent.Event{
				Type:     shipmentevent.TypeMoveDeparted,
				Metadata: fromStop,
			},
		},
		{
			name: "a move completing because its last stop was left",
			event: shipmentevent.Event{
				Type:     shipmentevent.TypeMoveArrived,
				Metadata: fromStop,
			},
		},
		{
			name:  "a move a dispatcher started by hand",
			event: shipmentevent.Event{Type: shipmentevent.TypeMoveDeparted},
			sends: true,
		},
		{
			name:  "a move a dispatcher completed by hand",
			event: shipmentevent.Event{Type: shipmentevent.TypeMoveArrived},
			sends: true,
		},
		{
			name:  "a status change",
			event: shipmentevent.Event{Type: shipmentevent.TypeStatusChanged},
		},
		{
			name:  "a driver committed to a move",
			event: shipmentevent.Event{Type: shipmentevent.TypeDriverAssigned},
		},
		{
			name:  "a carrier covering a move",
			event: shipmentevent.Event{Type: shipmentevent.TypeCarrierAssigned},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.sends, statusShipmentEvent(&tt.event))
		})
	}
}

func TestFlowShipmentEvent_StopProgressReachesTheFlows(t *testing.T) {
	t.Parallel()

	assert.True(t, FlowShipmentEvent(shipmentevent.TypeStopArrived))
	assert.True(t, FlowShipmentEvent(shipmentevent.TypeStopDeparted))
	assert.False(t, FlowShipmentEvent(shipmentevent.TypeStopArrivalCorrected))
	assert.False(t, FlowShipmentEvent(shipmentevent.TypeStopDepartureCorrected))
}

func TestFlowShipmentEvent_AssignmentsReachTheFlows(t *testing.T) {
	t.Parallel()

	assert.True(t, FlowShipmentEvent(shipmentevent.TypeDriverAssigned))
	assert.True(t, FlowShipmentEvent(shipmentevent.TypeDriverUnassigned))
	assert.True(t, FlowShipmentEvent(shipmentevent.TypeCarrierAssigned))
	assert.True(t, FlowShipmentEvent(shipmentevent.TypeCarrierUnassigned))
}

func TestLoadStateOf_ALoadIsDispatchedOnceAnyMoveIsAssigned(t *testing.T) {
	t.Parallel()

	tests := map[shipmentvocab.Status]string{
		shipmentvocab.StatusNew:                lading.LoadOpen,
		shipmentvocab.StatusPartiallyAssigned:  lading.LoadDispatched,
		shipmentvocab.StatusAssigned:           lading.LoadDispatched,
		shipmentvocab.StatusInTransit:          lading.LoadDispatched,
		shipmentvocab.StatusDelayed:            lading.LoadDispatched,
		shipmentvocab.StatusPartiallyCompleted: lading.LoadDispatched,
		shipmentvocab.StatusCompleted:          lading.LoadDelivered,
		shipmentvocab.StatusReadyToInvoice:     lading.LoadDelivered,
		shipmentvocab.StatusInvoiced:           lading.LoadInvoiced,
		shipmentvocab.StatusCanceled:           lading.LoadCancelled,
	}
	for status, want := range tests {
		assert.Equal(t, want, LoadStateOf(status), status)
	}
}

func TestDispatchShipmentEvent_IsTheCommitmentOfADriverOrCarrier(t *testing.T) {
	t.Parallel()

	statusChange := func(status shipmentvocab.Status) *shipmentevent.Event {
		return &shipmentevent.Event{
			Type:     shipmentevent.TypeStatusChanged,
			Metadata: map[string]any{"newStatus": string(status)},
		}
	}
	assert.True(
		t,
		dispatchShipmentEvent(&shipmentevent.Event{Type: shipmentevent.TypeDriverAssigned}),
	)
	assert.True(
		t,
		dispatchShipmentEvent(&shipmentevent.Event{Type: shipmentevent.TypeCarrierAssigned}),
	)
	assert.True(t, dispatchShipmentEvent(statusChange(shipmentvocab.StatusAssigned)))
	assert.True(t, dispatchShipmentEvent(statusChange(shipmentvocab.StatusPartiallyAssigned)))
	assert.False(t, dispatchShipmentEvent(statusChange(shipmentvocab.StatusInTransit)))
	assert.False(
		t,
		dispatchShipmentEvent(&shipmentevent.Event{Type: shipmentevent.TypeDriverReassigned}),
	)
	assert.False(
		t,
		dispatchShipmentEvent(&shipmentevent.Event{Type: shipmentevent.TypeStopDeparted}),
	)
}

func TestFirstDispatch_IsOnlyTheLoadLeavingOpen(t *testing.T) {
	t.Parallel()

	dispatched := &lading.LoadState{Exists: true, State: lading.LoadDispatched}
	assert.True(
		t,
		firstDispatch(&lading.LoadState{Exists: true, State: lading.LoadOpen}, dispatched),
	)
	assert.True(t, firstDispatch(&lading.LoadState{}, dispatched))
	assert.False(t, firstDispatch(dispatched, dispatched))
	assert.False(t, firstDispatch(
		&lading.LoadState{Exists: true, State: lading.LoadOpen},
		&lading.LoadState{Exists: true, State: lading.LoadDelivered},
	))
}
