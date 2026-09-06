package resolver

import (
	"sort"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentEventTypeParity_DomainMatchesSchema(t *testing.T) {
	t.Parallel()

	domain := make(map[string]struct{}, len(shipmentevent.AllTypes))
	for _, v := range shipmentevent.AllTypes {
		domain[string(v)] = struct{}{}
	}
	require.Len(t, domain, len(shipmentevent.AllTypes), "shipmentevent.AllTypes contains duplicates")

	schema := make(map[string]struct{}, len(gqlmodel.AllShipmentEventType))
	for _, v := range gqlmodel.AllShipmentEventType {
		schema[string(v)] = struct{}{}
	}
	require.Len(
		t,
		schema,
		len(gqlmodel.AllShipmentEventType),
		"gqlmodel.AllShipmentEventType contains duplicates",
	)

	assert.Empty(
		t,
		missingFrom(domain, schema),
		"shipmentevent.Type values missing from enum ShipmentEventType in shipment.graphqls",
	)
	assert.Empty(
		t,
		missingFrom(schema, domain),
		"ShipmentEventType schema values missing from shipmentevent.AllTypes",
	)
}

func TestShipmentEventTypeParity_AllTypesAreValid(t *testing.T) {
	t.Parallel()

	for _, v := range shipmentevent.AllTypes {
		assert.Truef(t, v.IsValid(), "shipmentevent.Type(%q) reported invalid", string(v))
	}
	assert.False(t, shipmentevent.Type("NotARealShipmentEventType").IsValid())
	assert.False(t, shipmentevent.Type("").IsValid())
}

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
		assert.IsTypef(t, want, model, "shipmentevent.Type(%q) mapped to the wrong model", eventType)
		assert.Equal(t, gqlmodel.ShipmentEventType(eventType), model.GetType())
		assert.NotNil(t, model.GetMetadata())
	}
}

func missingFrom(want, have map[string]struct{}) []string {
	missing := make([]string, 0, len(want))
	for v := range want {
		if _, ok := have[v]; !ok {
			missing = append(missing, v)
		}
	}
	sort.Strings(missing)
	return missing
}
