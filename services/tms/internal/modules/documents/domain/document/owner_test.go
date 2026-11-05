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
		{resourceType: "shipment_import", want: permission.ResourceShipment},
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

type knownKinds map[string]bool

func (k knownKinds) HasResource(resource string) bool { return k[resource] }

func TestVisibleToPerson_ConversationFileBelongsToWhoAttachedIt(t *testing.T) {
	t.Parallel()

	attacher := pulid.MustNew("usr_")
	doc := &Document{ResourceType: ConversationResourceType, UploadedByID: attacher}

	assert.True(t, doc.VisibleToPerson(attacher))
	assert.False(t, doc.VisibleToPerson(pulid.MustNew("usr_")))
	assert.False(t, doc.VisibleToPerson(pulid.Nil))
	assert.False(t, (&Document{ResourceType: ConversationResourceType}).VisibleToPerson(pulid.Nil))
	assert.True(t, (&Document{ResourceType: "shipment"}).VisibleToPerson(pulid.Nil))
}

func TestReadableOwner_FailsClosedOnAKindTheCatalogDoesNotKnow(t *testing.T) {
	t.Parallel()

	catalog := knownKinds{"shipment": true, "assistant": true}
	person := pulid.MustNew("usr_")

	owner, ok := (&Document{ResourceType: "Shipment"}).ReadableOwner(person, catalog)
	assert.True(t, ok)
	assert.Equal(t, permission.ResourceShipment, owner)

	_, ok = (&Document{ResourceType: "not_a_kind"}).ReadableOwner(person, catalog)
	assert.False(t, ok)

	_, ok = (&Document{
		ResourceType: ConversationResourceType,
		UploadedByID: pulid.MustNew("usr_"),
	}).ReadableOwner(person, catalog)
	assert.False(t, ok)
}

func TestVisibleDocuments_DropsTheFilesOfAnotherPersonsConversation(t *testing.T) {
	t.Parallel()

	person := pulid.MustNew("usr_")
	mine := &Document{ResourceType: ConversationResourceType, UploadedByID: person}
	theirs := &Document{ResourceType: ConversationResourceType, UploadedByID: pulid.MustNew("usr_")}
	shipment := &Document{ResourceType: "shipment"}

	assert.Equal(
		t,
		[]*Document{mine, shipment},
		VisibleDocuments([]*Document{mine, theirs, nil, shipment}, person),
	)
}
