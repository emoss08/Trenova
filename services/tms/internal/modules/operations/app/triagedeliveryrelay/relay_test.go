package triagedeliveryrelay

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSink struct {
	delivered []pulid.ID
}

func (s *recordingSink) ShipmentDelivered(_ context.Context, _ pagination.TenantInfo, id pulid.ID) {
	s.delivered = append(s.delivered, id)
}

func (s *recordingSink) DocumentAttached(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	pulid.ID,
) {
}

func (s *recordingSink) InvoicePaid(context.Context, pagination.TenantInfo, pulid.ID) {}

type stubShipments struct {
	status shipment.Status
	err    error
}

func (s stubShipments) GetByID(
	_ context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &shipment.Shipment{ID: req.ID, Status: s.status}, nil
}

func statusEvent(previous, next shipment.Status) *shipmentevent.Event {
	return &shipmentevent.Event{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ShipmentID:     pulid.MustNew("shp_"),
		Type:           shipmentevent.TypeStatusChanged,
		Metadata: map[string]any{
			"previousStatus": string(previous),
			"newStatus":      string(next),
		},
	}
}

func TestOnShipmentEvent_WakesTriageOnlyWhenTheShipmentFirstDelivers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		previous shipment.Status
		next     shipment.Status
		wakes    bool
	}{
		{"in transit to completed", shipment.StatusInTransit, shipment.StatusCompleted, true},
		{"new to ready to invoice", shipment.StatusNew, shipment.StatusReadyToInvoice, true},
		{"completed to invoiced", shipment.StatusCompleted, shipment.StatusInvoiced, false},
		{"new to in transit", shipment.StatusNew, shipment.StatusInTransit, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sink := &recordingSink{}
			relay := &Relay{sink: sink}
			event := statusEvent(tc.previous, tc.next)

			require.NoError(t, relay.OnShipmentEvent(t.Context(), event))
			if tc.wakes {
				assert.Equal(t, []pulid.ID{event.ShipmentID}, sink.delivered)
			} else {
				assert.Empty(t, sink.delivered)
			}
		})
	}
}

func TestOnShipmentEvent_IgnoresOtherEventsAndAMissingSink(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	event := statusEvent(shipment.StatusInTransit, shipment.StatusCompleted)
	event.Type = shipmentevent.TypeMoveArrived
	require.NoError(t, (&Relay{sink: sink}).OnShipmentEvent(t.Context(), event))
	assert.Empty(t, sink.delivered)

	require.NoError(t, (&Relay{}).OnShipmentEvent(t.Context(),
		statusEvent(shipment.StatusInTransit, shipment.StatusCompleted)))
}

func TestAfterMoveStatusChange_WakesWhenTheLastMoveDeliversTheShipment(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	move := &shipment.ShipmentMove{
		ShipmentID: pulid.MustNew("shp_"),
		Status:     shipment.MoveStatusCompleted,
	}

	sink := &recordingSink{}
	relay := &Relay{sink: sink, shipments: stubShipments{status: shipment.StatusCompleted}}
	require.NoError(t, relay.AfterMoveStatusChange(
		t.Context(), tenant, move, shipment.MoveStatusInTransit,
	))
	assert.Equal(t, []pulid.ID{move.ShipmentID}, sink.delivered)

	sink = &recordingSink{}
	relay = &Relay{sink: sink, shipments: stubShipments{status: shipment.StatusInTransit}}
	require.NoError(t, relay.AfterMoveStatusChange(
		t.Context(), tenant, move, shipment.MoveStatusInTransit,
	))
	assert.Empty(t, sink.delivered, "one move of several finishing does not deliver the shipment")

	sink = &recordingSink{}
	relay = &Relay{sink: sink, shipments: stubShipments{status: shipment.StatusCompleted}}
	require.NoError(t, relay.AfterMoveStatusChange(
		t.Context(), tenant, move, shipment.MoveStatusCompleted,
	))
	assert.Empty(t, sink.delivered, "a move already complete is not a new delivery")

	failure := errors.New("read failed")
	relay = &Relay{sink: &recordingSink{}, shipments: stubShipments{err: failure}}
	assert.ErrorIs(t, relay.AfterMoveStatusChange(
		t.Context(), tenant, move, shipment.MoveStatusInTransit,
	), failure)
}
