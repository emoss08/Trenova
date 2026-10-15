package document

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
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

func TestSearchable(t *testing.T) {
	t.Parallel()

	assert.True(t, (&Document{IsCurrentVersion: true, Status: StatusActive}).Searchable())
	assert.True(t, (&Document{IsCurrentVersion: true, Status: StatusArchived}).Searchable())
	assert.False(t, (&Document{IsCurrentVersion: false, Status: StatusActive}).Searchable())
	assert.False(t, (&Document{IsCurrentVersion: true, Status: StatusRejected}).Searchable())
}

func TestAttachedTo_IsTheConversationWhoeverAttachedIt(t *testing.T) {
	t.Parallel()

	conversation := pulid.MustNew("ath_")
	doc := &Document{
		ResourceType: ConversationResourceType,
		ResourceID:   conversation.String(),
		UploadedByID: pulid.MustNew("usr_"),
	}
	assert.True(t, doc.AttachedTo(conversation))
	assert.False(t, doc.AttachedTo(pulid.MustNew("ath_")))
	assert.False(t, doc.AttachedTo(pulid.Nil))

	onShipment := &Document{ResourceType: "shipment", ResourceID: conversation.String()}
	assert.False(t, onShipment.AttachedTo(conversation), "only a conversation's own file")
}
