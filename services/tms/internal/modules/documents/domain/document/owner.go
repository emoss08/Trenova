package document

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/textcase"
)

const ConversationResourceType = "assistant_thread"

var ownerResourceAliases = map[string]permission.Resource{
	ConversationResourceType: permission.ResourceAssistant,
	"invoice_adjustment":     permission.ResourceInvoice,
	"shipment_import":        permission.ResourceShipment,
}

func normalizedOwner(resourceType string) string {
	return strings.ToLower(textcase.ConvertCamelToSnake(strings.TrimSpace(resourceType)))
}

func OwnerResource(resourceType string) permission.Resource {
	normalized := normalizedOwner(resourceType)
	if alias, ok := ownerResourceAliases[normalized]; ok {
		return alias
	}

	return permission.Resource(normalized)
}

func (d *Document) OwnerResource() permission.Resource {
	return OwnerResource(d.ResourceType)
}

func (d *Document) OwnedByConversation() bool {
	return normalizedOwner(d.ResourceType) == ConversationResourceType
}

func (d *Document) ConversationID() (pulid.ID, bool) {
	if !d.OwnedByConversation() {
		return pulid.Nil, false
	}
	id, err := pulid.Parse(d.ResourceID)
	if err != nil || id.IsNil() {
		return pulid.Nil, false
	}

	return id, true
}

type ResourceCatalog interface {
	HasResource(resource string) bool
}

func (d *Document) VisibleToPerson(userID pulid.ID) bool {
	if !d.OwnedByConversation() {
		return true
	}

	return userID.IsNotNil() && d.UploadedByID == userID
}

func (d *Document) ReadableOwner(
	userID pulid.ID,
	catalog ResourceCatalog,
) (permission.Resource, bool) {
	owner := d.OwnerResource()
	if !catalog.HasResource(owner.String()) || !d.VisibleToPerson(userID) {
		return "", false
	}

	return owner, true
}

func VisibleDocuments(docs []*Document, userID pulid.ID) []*Document {
	visible := make([]*Document, 0, len(docs))
	for _, doc := range docs {
		if doc != nil && doc.VisibleToPerson(userID) {
			visible = append(visible, doc)
		}
	}

	return visible
}

func (d *Document) AttachedTo(conversationID pulid.ID) bool {
	if conversationID.IsNil() {
		return false
	}
	id, ok := d.ConversationID()

	return ok && id == conversationID
}

func (d *Document) AttachedBy(conversationID, userID pulid.ID) bool {
	if conversationID.IsNil() || userID.IsNil() || d.UploadedByID != userID {
		return false
	}
	id, ok := d.ConversationID()

	return ok && id == conversationID
}
