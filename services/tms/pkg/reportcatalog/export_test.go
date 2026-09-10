package reportcatalog

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
)

// The report compiler refuses to compile a report for export unless the user
// holds OpExport on every resource the report touches. A resource whose
// definition never declares OpExport can never grant it, so a report built on
// that entity previews correctly and fails only at export.
func TestEveryCatalogResourceCanBeExported(t *testing.T) {
	registry := permission.NewRegistry()

	seen := make(map[string]struct{}, len(Default.Entities))
	for i := range Default.Entities {
		entity := &Default.Entities[i]
		resource := entity.Resource.String()
		if _, done := seen[resource]; done {
			continue
		}
		seen[resource] = struct{}{}

		definition, ok := registry.Get(resource)
		if !ok {
			t.Errorf("entity %q maps onto unregistered resource %q", entity.Key, resource)
			continue
		}
		if !slices.Contains(registry.GetOperationsForResource(resource), permission.OpExport) {
			t.Errorf(
				"resource %q (%s) backs catalog entity %q but does not declare OpExport, so no role can grant it and every export of a report on %s will fail",
				resource, definition.DisplayName, entity.Key, entity.PluralLabel,
			)
		}
	}
}
