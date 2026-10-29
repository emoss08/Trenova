package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The catalog's one unforgiving requirement: every curated field has to still map
to a column on its entity.

listquery skips a filter on a field it cannot resolve and runs the query
anyway (pkg/listquery/filters.go), so a column renamed in a
migration turns "shipments delivered yesterday" into "the 25 most recent
shipments" with no error anywhere — and the model presents that as the answer.
Pinning the spec against the computed configuration makes the rename a failing
test instead.
*/
func TestListCatalog_EveryFieldStillMapsToItsEntity(t *testing.T) {
	t.Parallel()

	for _, spec := range listCatalogSpecs() {
		require.NotNil(t, spec.Config, "%s must carry its entity configuration", spec.Name)

		for _, field := range spec.Fields {
			assert.True(t, spec.Config.FilterableFields[field.Name],
				"%s declares %q, which no longer maps to a column on %s",
				spec.Name, field.Name, spec.EntityPlural)
		}
	}
}

func TestListCatalog_EverySortableFieldIsSortable(t *testing.T) {
	t.Parallel()

	for _, spec := range listCatalogSpecs() {
		for _, field := range spec.Fields {
			if !field.Sortable {
				continue
			}

			assert.True(t, spec.Config.SortableFields[field.Name],
				"%s offers %q as a sort, which the entity cannot sort by",
				spec.Name, field.Name)
		}
	}
}

// Every enum field has to publish its values, or the model guesses the spelling
// and gets an empty page it will report as "there are none".
func TestListCatalog_EveryEnumFieldPublishesItsValues(t *testing.T) {
	t.Parallel()

	for _, spec := range listCatalogSpecs() {
		for _, field := range spec.Fields {
			if field.Kind != filtercatalog.KindEnum {
				continue
			}

			assert.NotEmpty(t, field.Values,
				"%s exposes %q as an enum without saying what it can be",
				spec.Name, field.Name)
		}
	}
}

func TestListCatalog_NamesAreUniqueAndPrefixed(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for _, spec := range listCatalogSpecs() {
		assert.False(t, seen[spec.Name], "%s is registered twice", spec.Name)
		seen[spec.Name] = true

		assert.Regexp(t, `^([a-z]+_list(_[a-z]+)*|list_[a-z_]+)$`, spec.Name)
		assert.NotEmpty(t, spec.Resource, "%s must authorize against a resource", spec.Name)
		assert.NotEmpty(t, spec.Fields, "%s must offer something to filter on", spec.Name)
	}
}

/*
Each entry authorizes against its own permission resource, which is why the
catalog is one tool per entity rather than list_records(entity, …): the agent
builder groups the picker by resource, so an organization can grant an agent
worker_list without also granting customer_list.
*/
func TestListCatalog_EachEntityCarriesItsOwnResource(t *testing.T) {
	t.Parallel()

	seen := make(map[string]string)
	for _, spec := range listCatalogSpecs() {
		previous, taken := seen[string(spec.Resource)]
		assert.False(t, taken,
			"%s and %s share the resource %q, so they cannot be granted apart",
			spec.Name, previous, spec.Resource)
		seen[string(spec.Resource)] = spec.Name
	}
}

/*
The catalog behind the Ask input is the catalog behind the list tools.

If they drift, "in transit" means one thing to an agent and another to the
grid, and only one of the two is ever tested. This holds them to the same
source: every list tool's resource is in the catalog, under the permission
resource a page would look it up by.
*/
func TestFilterCatalog_CoversEveryListTool(t *testing.T) {
	t.Parallel()

	catalog := FilterCatalog()

	for _, spec := range listCatalogSpecs() {
		resource, ok := catalog.For(spec.Resource)
		require.True(t, ok, "%s is not in the filter catalog", spec.Name)
		assert.Equal(t, spec.Name, resource.Tool)
		assert.Equal(t, spec.EntityPlural, resource.Entity)
		assert.Equal(t, len(spec.Fields), len(resource.Fields), "%s lost fields", spec.Name)

		// Every catalogued field has to be one the entity still maps, or a
		// filter compiles to nothing and the page comes back unfiltered.
		for _, field := range resource.Fields {
			_, found := resource.Field(field.Name)
			assert.True(t, found, "%s: %s is not indexed", spec.Name, field.Name)
		}
	}
}
