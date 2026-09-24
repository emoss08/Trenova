package document

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const ConversationResourceType = "assistant_thread"

var ownerResourceAliases = map[string]permission.Resource{
	ConversationResourceType: permission.ResourceAssistant,
	"invoice_adjustment":     permission.ResourceInvoice,
}

func normalizedOwner(resourceType string) string {
	return strings.ToLower(stringutils.ConvertCamelToSnake(strings.TrimSpace(resourceType)))
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

func (d *Document) conversationID() (pulid.ID, bool) {
	if !d.OwnedByConversation() {
		return pulid.Nil, false
	}
	id, err := pulid.Parse(d.ResourceID)
	if err != nil || id.IsNil() {
		return pulid.Nil, false
	}

	return id, true
}

func ConversationIDs(docs []*Document) []pulid.ID {
	ids := make([]pulid.ID, 0, len(docs))
	seen := make(map[pulid.ID]struct{}, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		id, ok := doc.conversationID()
		if !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids
}

func (d *Document) VisibleToPerson(conversationOwners map[pulid.ID]pulid.ID, userID pulid.ID) bool {
	if !d.OwnedByConversation() {
		return true
	}
	if userID.IsNil() {
		return false
	}
	id, ok := d.conversationID()
	if !ok {
		return false
	}
	owner, ok := conversationOwners[id]

	return ok && owner == userID
}
