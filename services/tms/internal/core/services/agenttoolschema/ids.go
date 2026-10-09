package agenttoolschema

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// neverGuess closes every id parameter's sentence. A model short of an id
// invents one shaped like the examples it has seen, and the service answers
// "not found", which the model reads as the record not existing.
const neverGuess = "Never guess one."

// IDDescription is the one sentence an id parameter carries: what it is, and
// where the model gets one. The runtime reads the source back out of it to
// name it in a refusal.
func IDDescription(what, source string) string {
	return what + ", from " + source + ". " + neverGuess
}

// RecordID is a parameter that takes the id of one record of the resource,
// marked with the resource so the runtime can refuse an id of another kind
// by its prefix before the tool reads it. The mark is stripped from what the
// model is shown.
func RecordID(resource permission.Resource, what, source string) map[string]any {
	return RecordIDText(resource, IDDescription(what, source))
}

// RecordIDText is RecordID for a parameter whose description says more than
// the one sentence: when the id is needed, or what leaving it out means. The
// description still names where the id comes from.
func RecordIDText(resource permission.Resource, description string) map[string]any {
	property := IDText(description)
	property[toolschema.KeyRecordOf] = resource.String()

	return property
}

// KindID is a parameter that takes the id of a record of one of the kinds
// named: a qualification record that is either an employment verification or
// a clearinghouse query, or a record kept inside another resource's, such as
// a line of an invoice. The runtime refuses an id whose prefix is none of the
// kinds' before the tool reads it, as RecordID does for a resource's. The
// description names where the id comes from, as RecordIDText's does.
func KindID(description string, kinds ...permission.RecordKind) map[string]any {
	property := IDText(description)
	property[toolschema.KeyRecordKinds] = kindNames(kinds)

	return property
}

// KindIDs is a list of ids each of which is of one of the kinds named, as
// KindID marks one.
func KindIDs(description string, maxItems int, kinds ...permission.RecordKind) map[string]any {
	property := IDList(description, maxItems)
	items, _ := property[toolschema.KeyItems].(map[string]any)
	items[toolschema.KeyRecordKinds] = kindNames(kinds)

	return property
}

// OfResource marks a property built some other way, an id or a list of ids,
// with the resource its ids belong to, as RecordID and RecordIDs mark theirs.
// Nothing else about the property changes, so a parameter with bounds or
// wording of its own is typed without being rebuilt.
func OfResource(property map[string]any, resource permission.Resource) map[string]any {
	markedTarget(property)[toolschema.KeyRecordOf] = resource.String()

	return property
}

// OfKinds is OfResource for a property whose ids are of the kinds named, as
// KindID and KindIDs mark theirs.
func OfKinds(property map[string]any, kinds ...permission.RecordKind) map[string]any {
	markedTarget(property)[toolschema.KeyRecordKinds] = kindNames(kinds)

	return property
}

// markedTarget is where an id property's mark goes: on the property for one
// id, on its items for a list of them.
func markedTarget(property map[string]any) map[string]any {
	if property[toolschema.KeyType] != toolschema.TypeArray {
		return property
	}
	items, ok := property[toolschema.KeyItems].(map[string]any)
	if !ok {
		items = map[string]any{toolschema.KeyType: toolschema.TypeString}
		property[toolschema.KeyItems] = items
	}

	return items
}

func kindNames(kinds []permission.RecordKind) []string {
	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, string(kind))
	}

	return names
}

// ID is a parameter that takes a record id whose kind has no prefix of its
// own in permission's table: a line of an invoice, a day of a leave case.
// The runtime still refuses a value that is not shaped like an id at all.
func ID(what, source string) map[string]any {
	return IDText(IDDescription(what, source))
}

// IDText is ID with a description of the tool's own wording.
func IDText(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}

// RecordIDs is a list of ids of one resource's records, each marked as
// RecordID marks one. A list an approver may narrow is also wrapped in
// toolschema.RecordSubset.
func RecordIDs(resource permission.Resource, description string, maxItems int) map[string]any {
	property := IDList(description, maxItems)
	items, _ := property[toolschema.KeyItems].(map[string]any)
	items[toolschema.KeyRecordOf] = resource.String()

	return property
}

// IDList is a list of at least one record id and at most maxItems, of a kind
// with no prefix of its own.
func IDList(description string, maxItems int) map[string]any {
	property := map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyItems:       map[string]any{toolschema.KeyType: toolschema.TypeString},
		toolschema.KeyMinItems:    1,
	}
	if maxItems > 0 {
		property[toolschema.KeyMaxItems] = maxItems
	}

	return property
}

// Noted adds a sentence to a property's description: what an id parameter
// must also be, or what leaving it out means.
func Noted(property map[string]any, note string) map[string]any {
	description, _ := property[toolschema.KeyDescription].(string)
	if description == "" {
		property[toolschema.KeyDescription] = note
	} else {
		property[toolschema.KeyDescription] = description + " " + note
	}

	return property
}
