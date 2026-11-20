package edioutbound

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
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
