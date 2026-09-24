package document

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/assert"
)

func TestOwnerResource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		resourceType string
		want         permission.Resource
	}{
		{resourceType: "shipment", want: permission.ResourceShipment},
		{resourceType: "Shipment", want: permission.ResourceShipment},
		{resourceType: " Worker ", want: permission.ResourceWorker},
		{resourceType: "InboundMessage", want: permission.ResourceInboundMessage},
		{resourceType: "inbound_message", want: permission.ResourceInboundMessage},
		{resourceType: "assistant_thread", want: permission.ResourceAssistant},
		{resourceType: "invoice_adjustment", want: permission.ResourceInvoice},
		{resourceType: "fuel_purchase_import", want: permission.ResourceFuelPurchaseImport},
		{resourceType: "", want: permission.Resource("")},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, OwnerResource(tt.resourceType))
		})
	}
}
