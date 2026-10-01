package shipmentstate

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCancel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entity   *shipment.Shipment
		conflict bool
		business bool
	}{
		{name: "new", entity: &shipment.Shipment{Status: shipment.StatusNew}},
		{name: "in transit", entity: &shipment.Shipment{Status: shipment.StatusInTransit}},
		{
			name: "sent back to operations",
			entity: &shipment.Shipment{
				Status:                shipment.StatusReadyToInvoice,
				BillingTransferStatus: shipment.BillingTransferSentBackToOps,
			},
		},
		{
			name:     "already canceled",
			entity:   &shipment.Shipment{Status: shipment.StatusCanceled},
			business: true,
		},
		{
			name:     "invoiced",
			entity:   &shipment.Shipment{Status: shipment.StatusInvoiced},
			conflict: true,
		},
		{
			name: "in billing queue",
			entity: &shipment.Shipment{
				Status:                shipment.StatusReadyToInvoice,
				BillingTransferStatus: shipment.BillingTransferApproved,
			},
			conflict: true,
		},
		{
			name:     "completed",
			entity:   &shipment.Shipment{Status: shipment.StatusCompleted},
			conflict: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCancel(tt.entity)
			switch {
			case tt.conflict:
				var conflict *errortypes.ConflictError
				require.ErrorAs(t, err, &conflict)
				assert.Equal(t, errortypes.ErrInvalidOperation, conflict.Code)
			case tt.business:
				var business *errortypes.BusinessError
				require.ErrorAs(t, err, &business)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestPrepareForUncancel(t *testing.T) {
	t.Parallel()

	arrived := int64(1_700_000_000)
	departed := arrived + 600
	canceledAt := arrived + 7200

	entity := &shipment.Shipment{
		Status:       shipment.StatusCanceled,
		CanceledAt:   &canceledAt,
		CancelReason: "customer request",
		Moves: []*shipment.ShipmentMove{
			{
				Status: shipment.MoveStatusCompleted,
				Stops: []*shipment.Stop{
					{
						Type:            shipment.StopTypePickup,
						Status:          shipment.StopStatusCompleted,
						ActualArrival:   &arrived,
						ActualDeparture: &departed,
					},
					{
						Type:            shipment.StopTypeDelivery,
						Status:          shipment.StopStatusCompleted,
						ActualArrival:   &arrived,
						ActualDeparture: &departed,
					},
				},
			},
			{
				Status:     shipment.MoveStatusCanceled,
				Assignment: &shipment.Assignment{Status: shipment.AssignmentStatusCanceled},
				Stops: []*shipment.Stop{
					{Type: shipment.StopTypePickup, Status: shipment.StopStatusCanceled},
					{Type: shipment.StopTypeDelivery, Status: shipment.StopStatusCanceled},
				},
			},
		},
	}

	NewCoordinatorWithClock(func() int64 { return arrived }).
		PrepareForUncancel(entity, DisabledDelayThresholdMinutes)

	assert.Nil(t, entity.CanceledAt)
	assert.Empty(t, entity.CancelReason)
	assert.Equal(t, shipment.MoveStatusCompleted, entity.Moves[0].Status)
	assert.Equal(t, shipment.StopStatusCompleted, entity.Moves[0].Stops[1].Status)
	assert.Equal(t, shipment.AssignmentStatusNew, entity.Moves[1].Assignment.Status)
	assert.Equal(t, shipment.MoveStatusAssigned, entity.Moves[1].Status)
	assert.Equal(t, shipment.StopStatusNew, entity.Moves[1].Stops[0].Status)
	assert.Equal(t, shipment.StatusPartiallyCompleted, entity.Status)
}
