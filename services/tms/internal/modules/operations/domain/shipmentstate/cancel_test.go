package shipmentstate

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
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
		{name: "new", entity: &shipment.Shipment{Status: shipmentvocab.StatusNew}},
		{name: "in transit", entity: &shipment.Shipment{Status: shipmentvocab.StatusInTransit}},
		{
			name: "sent back to operations",
			entity: &shipment.Shipment{
				Status:                shipmentvocab.StatusReadyToInvoice,
				BillingTransferStatus: shipmentvocab.BillingTransferSentBackToOps,
			},
		},
		{
			name:     "already canceled",
			entity:   &shipment.Shipment{Status: shipmentvocab.StatusCanceled},
			business: true,
		},
		{
			name:     "invoiced",
			entity:   &shipment.Shipment{Status: shipmentvocab.StatusInvoiced},
			conflict: true,
		},
		{
			name: "in billing queue",
			entity: &shipment.Shipment{
				Status:                shipmentvocab.StatusReadyToInvoice,
				BillingTransferStatus: shipmentvocab.BillingTransferApproved,
			},
			conflict: true,
		},
		{
			name:     "completed",
			entity:   &shipment.Shipment{Status: shipmentvocab.StatusCompleted},
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
		Status:       shipmentvocab.StatusCanceled,
		CanceledAt:   &canceledAt,
		CancelReason: "customer request",
		Moves: []*shipment.ShipmentMove{
			{
				Status: shipmentvocab.MoveStatusCompleted,
				Stops: []*shipment.Stop{
					{
						Type:            shipmentvocab.StopTypePickup,
						Status:          shipmentvocab.StopStatusCompleted,
						ActualArrival:   &arrived,
						ActualDeparture: &departed,
					},
					{
						Type:            shipmentvocab.StopTypeDelivery,
						Status:          shipmentvocab.StopStatusCompleted,
						ActualArrival:   &arrived,
						ActualDeparture: &departed,
					},
				},
			},
			{
				Status:     shipmentvocab.MoveStatusCanceled,
				Assignment: &shipment.Assignment{Status: shipmentvocab.AssignmentStatusCanceled},
				Stops: []*shipment.Stop{
					{Type: shipmentvocab.StopTypePickup, Status: shipmentvocab.StopStatusCanceled},
					{
						Type:   shipmentvocab.StopTypeDelivery,
						Status: shipmentvocab.StopStatusCanceled,
					},
				},
			},
		},
	}

	NewCoordinatorWithClock(func() int64 { return arrived }).
		PrepareForUncancel(entity, DisabledDelayThresholdMinutes)

	assert.Nil(t, entity.CanceledAt)
	assert.Empty(t, entity.CancelReason)
	assert.Equal(t, shipmentvocab.MoveStatusCompleted, entity.Moves[0].Status)
	assert.Equal(t, shipmentvocab.StopStatusCompleted, entity.Moves[0].Stops[1].Status)
	assert.Equal(t, shipmentvocab.AssignmentStatusNew, entity.Moves[1].Assignment.Status)
	assert.Equal(t, shipmentvocab.MoveStatusAssigned, entity.Moves[1].Status)
	assert.Equal(t, shipmentvocab.StopStatusNew, entity.Moves[1].Stops[0].Status)
	assert.Equal(t, shipmentvocab.StatusPartiallyCompleted, entity.Status)
}
