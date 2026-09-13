package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestValidateConsolidatedQueueItems(t *testing.T) {
	t.Parallel()

	legA := &shipment.Shipment{ID: pulid.MustNew("shp_")}
	legB := &shipment.Shipment{ID: pulid.MustNew("shp_")}
	legs := []*shipment.Shipment{legA, legB}

	approved := func(leg *shipment.Shipment) *billingqueue.BillingQueueItem {
		return &billingqueue.BillingQueueItem{
			ID:         pulid.MustNew("bqi_"),
			ShipmentID: leg.ID,
			Status:     billingqueue.StatusApproved,
			BillType:   billingqueue.BillTypeInvoice,
		}
	}

	tests := []struct {
		name    string
		items   func() []*billingqueue.BillingQueueItem
		wantErr bool
	}{
		{
			name: "one approved unbilled item per leg",
			items: func() []*billingqueue.BillingQueueItem {
				return []*billingqueue.BillingQueueItem{approved(legB), approved(legA)}
			},
		},
		{
			name: "missing items",
			items: func() []*billingqueue.BillingQueueItem {
				return nil
			},
			wantErr: true,
		},
		{
			name: "nil item",
			items: func() []*billingqueue.BillingQueueItem {
				return []*billingqueue.BillingQueueItem{approved(legA), nil}
			},
			wantErr: true,
		},
		{
			name: "two items for the same leg",
			items: func() []*billingqueue.BillingQueueItem {
				return []*billingqueue.BillingQueueItem{approved(legA), approved(legA)}
			},
			wantErr: true,
		},
		{
			name: "item for a shipment not on the invoice",
			items: func() []*billingqueue.BillingQueueItem {
				return []*billingqueue.BillingQueueItem{
					approved(legA),
					approved(&shipment.Shipment{ID: pulid.MustNew("shp_")}),
				}
			},
			wantErr: true,
		},
		{
			name: "item already on an invoice",
			items: func() []*billingqueue.BillingQueueItem {
				billed := approved(legB)
				billed.InvoiceID = pulid.MustNew("inv_")
				return []*billingqueue.BillingQueueItem{approved(legA), billed}
			},
			wantErr: true,
		},
		{
			name: "item no longer approved",
			items: func() []*billingqueue.BillingQueueItem {
				posted := approved(legB)
				posted.Status = billingqueue.StatusPosted
				return []*billingqueue.BillingQueueItem{approved(legA), posted}
			},
			wantErr: true,
		},
		{
			name: "adjustment origin item",
			items: func() []*billingqueue.BillingQueueItem {
				origin := approved(legB)
				origin.IsAdjustmentOrigin = true
				return []*billingqueue.BillingQueueItem{approved(legA), origin}
			},
			wantErr: true,
		},
		{
			name: "credit memo item",
			items: func() []*billingqueue.BillingQueueItem {
				memo := approved(legB)
				memo.BillType = billingqueue.BillTypeCreditMemo
				return []*billingqueue.BillingQueueItem{approved(legA), memo}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateConsolidatedQueueItems(legs, tt.items())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
