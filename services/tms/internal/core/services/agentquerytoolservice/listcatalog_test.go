package agentquerytoolservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The catalog's one unforgiving requirement: every curated field has to still map
to a column on its entity.

QueryBuilder.ApplyFilters skips a field that is not in FilterableFields and runs
the query anyway (pkg/querybuilder/querybuilder.go:89), so a column renamed in a
migration turns "shipments delivered yesterday" into "the 25 most recent
shipments" with no error anywhere — and the model presents that as the answer.
Pinning the spec against the computed configuration makes the rename a failing
test instead.
*/
func TestListCatalog_EveryFieldStillMapsToItsEntity(t *testing.T) {
	t.Parallel()

	for _, spec := range listCatalogSpecs() {
		require.NotNil(t, spec.config, "%s must carry its entity configuration", spec.name)

		for _, field := range spec.fields {
			assert.True(t, spec.config.FilterableFields[field.Name],
				"%s declares %q, which no longer maps to a column on %s",
				spec.name, field.Name, spec.entityPlural)
		}
	}
}

func TestListCatalog_EverySortableFieldIsSortable(t *testing.T) {
	t.Parallel()

	for _, spec := range listCatalogSpecs() {
		for _, field := range spec.fields {
			if !field.Sortable {
				continue
			}

			assert.True(t, spec.config.SortableFields[field.Name],
				"%s offers %q as a sort, which the entity cannot sort by",
				spec.name, field.Name)
		}
	}
}

// Every enum field has to publish its values, or the model guesses the spelling
// and gets an empty page it will report as "there are none".
func TestListCatalog_EveryEnumFieldPublishesItsValues(t *testing.T) {
	t.Parallel()

	for _, spec := range listCatalogSpecs() {
		for _, field := range spec.fields {
			if field.Kind != filterEnum {
				continue
			}

			assert.NotEmpty(t, field.Values,
				"%s exposes %q as an enum without saying what it can be",
				spec.name, field.Name)
		}
	}
}

func TestListCatalog_NamesAreUniqueAndPrefixed(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for _, spec := range listCatalogSpecs() {
		assert.False(t, seen[spec.name], "%s is registered twice", spec.name)
		seen[spec.name] = true

		assert.Regexp(t, `^list_[a-z_]+$`, spec.name)
		assert.NotEmpty(t, spec.resource, "%s must authorize against a resource", spec.name)
		assert.NotEmpty(t, spec.fields, "%s must offer something to filter on", spec.name)
	}
}

/*
Each entry authorizes against its own permission resource, which is why the
catalog is one tool per entity rather than list_records(entity, …): the agent
builder groups the picker by resource, so an organization can grant an agent
list_workers without also granting list_customers.
*/
func TestListCatalog_EachEntityCarriesItsOwnResource(t *testing.T) {
	t.Parallel()

	seen := make(map[string]string)
	for _, spec := range listCatalogSpecs() {
		previous, taken := seen[string(spec.resource)]
		assert.False(t, taken,
			"%s and %s share the resource %q, so they cannot be granted apart",
			spec.name, previous, spec.resource)
		seen[string(spec.resource)] = spec.name
	}
}

/*
The question that started this: "which drivers hold a current hazmat
endorsement?" Endorsements live on the worker profile, not the worker, so
without a traversal filter the catalog could not express it at all — and the
model's only near-miss was list_expiring_credentials, which answers "whose is
about to lapse", a different question with a plausible-looking wrong answer.
*/
func TestListWorkers_CanFilterOnTheProfile(t *testing.T) {
	t.Parallel()

	spec := specOf(newListWorkersTool(nil))

	byName := make(map[string]listField, len(spec.fields))
	for _, field := range spec.fields {
		byName[field.Name] = field
	}

	endorsement, ok := byName["profile.endorsement"]
	require.True(t, ok, "endorsements are on the profile, and the roster has to reach them")
	assert.Equal(t, filterEnum, endorsement.Kind)

	for _, dated := range []string{"profile.hazmatExpiry", "profile.medicalCardExpiry"} {
		field, found := byName[dated]
		require.True(t, found, "%s is what makes 'current' answerable", dated)
		assert.Equal(t, filterDate, field.Kind)
	}
}

/*
An endorsement is stored as a single letter: H is hazmat, X is tanker *and*
hazmat, N is tanker alone. No model guesses that, and one that guesses "Hazmat"
gets an empty page it will report as "nobody holds one". The values have to be
published with what they mean.
*/
func TestListWorkers_ExplainsTheEndorsementCodes(t *testing.T) {
	t.Parallel()

	spec := specOf(newListWorkersTool(nil))

	var endorsement listField
	for _, field := range spec.fields {
		if field.Name == "profile.endorsement" {
			endorsement = field
		}
	}

	assert.ElementsMatch(t, []string{"O", "N", "H", "X", "P", "T"}, endorsement.Values)
	assert.Contains(t, endorsement.Note, "hazmat")
	assert.Contains(t, endorsement.Note, "X",
		"the code that means both has to be spelled out or hazmat holders get undercounted")
}
