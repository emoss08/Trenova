package reportcatalog

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

// Classify is the class a field or edge of an entity takes from its
// resource's reference view, matched by its API name. An entity that is only
// part of its resource's record shows none of that view.
func Classify(registry *permission.Registry, entity *Entity, key string) FieldClass {
	if !entity.WithinRecord && registry.InReferenceView(entity.Resource.String(), key) {
		return ClassReference
	}
	return ClassKindRead
}
