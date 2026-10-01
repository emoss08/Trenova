package shipmentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentEventTypeParity_EveryTypeMapsToOneConcreteModel(t *testing.T) {
	t.Parallel()

	expected := map[shipmentevent.Type]gqlmodel.ShipmentEvent{
		shipmentevent.TypeShipmentCreated:       &gqlmodel.ShipmentLifecycleEvent{},
		shipmentevent.TypeShipmentUpdated:       &gqlmodel.ShipmentLifecycleEvent{},
		shipmentevent.TypeStatusChanged:         &gqlmodel.ShipmentLifecycleEvent{},
		shipmentevent.TypeShipmentCanceled:      &gqlmodel.ShipmentLifecycleEvent{},
		shipmentevent.TypeShipmentUncanceled:    &gqlmodel.ShipmentLifecycleEvent{},
		shipmentevent.TypeOwnershipTransferred:  &gqlmodel.ShipmentOwnershipEvent{},
		shipmentevent.TypeMoveStatusChanged:     &gqlmodel.ShipmentMoveEvent{},
		shipmentevent.TypeMoveDeparted:          &gqlmodel.ShipmentMoveEvent{},
		shipmentevent.TypeMoveArrived:           &gqlmodel.ShipmentMoveEvent{},
		shipmentevent.TypeStopCompleted:         &gqlmodel.ShipmentMoveEvent{},
		shipmentevent.TypeDriverAssigned:        &gqlmodel.ShipmentAssignmentEvent{},
		shipmentevent.TypeDriverReassigned:      &gqlmodel.ShipmentAssignmentEvent{},
		shipmentevent.TypeDriverUnassigned:      &gqlmodel.ShipmentAssignmentEvent{},
		shipmentevent.TypeCarrierAssigned:       &gqlmodel.ShipmentCarrierEvent{},
		shipmentevent.TypeCarrierUnassigned:     &gqlmodel.ShipmentCarrierEvent{},
		shipmentevent.TypeTenderOffered:         &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderAccepted:        &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderDeclined:        &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderExpired:         &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderWithdrawn:       &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderNeedsReview:     &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeRoutingGuideExhausted: &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderLateResponse:    &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderDeliveryFailed:  &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderEntrySkipped:    &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeTenderEntryWarned:     &gqlmodel.ShipmentTenderEvent{},
		shipmentevent.TypeHoldPlaced:            &gqlmodel.ShipmentHoldEvent{},
		shipmentevent.TypeHoldUpdated:           &gqlmodel.ShipmentHoldEvent{},
		shipmentevent.TypeHoldReleased:          &gqlmodel.ShipmentHoldEvent{},
		shipmentevent.TypeCommentPosted:         &gqlmodel.ShipmentCommentEvent{},
	}
	require.Len(
		t,
		expected,
		len(shipmentevent.AllTypes),
		"every shipmentevent.Type must be assigned to exactly one concrete ShipmentEvent model",
	)

	for _, eventType := range shipmentevent.AllTypes {
		want, ok := expected[eventType]
		require.Truef(t, ok, "shipmentevent.Type(%q) has no expected concrete model", eventType)

		model, err := shipmentEventToModel(&shipmentevent.Event{Type: eventType})
		require.NoErrorf(t, err, "shipmentevent.Type(%q) failed to map", eventType)
		assert.IsTypef(
			t,
			want,
			model,
			"shipmentevent.Type(%q) mapped to the wrong model",
			eventType,
		)
		assert.Equal(t, gqlmodel.ShipmentEventType(eventType), model.GetType())
		assert.NotNil(t, model.GetMetadata())
	}
}
