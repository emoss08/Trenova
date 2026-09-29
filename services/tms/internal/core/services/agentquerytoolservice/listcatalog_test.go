package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

	spec := specOf(newListWorkersTool(nil, nil))

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

	spec := specOf(newListWorkersTool(nil, nil))

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
		resource, ok := catalog.For(spec.resource)
		require.True(t, ok, "%s is not in the filter catalog", spec.name)
		assert.Equal(t, spec.name, resource.Tool)
		assert.Equal(t, spec.entityPlural, resource.Entity)
		assert.Equal(t, len(spec.fields), len(resource.Fields), "%s lost fields", spec.name)

		// Every catalogued field has to be one the entity still maps, or a
		// filter compiles to nothing and the page comes back unfiltered.
		for _, field := range resource.Fields {
			_, found := resource.Field(field.Name)
			assert.True(t, found, "%s: %s is not indexed", spec.name, field.Name)
		}
	}
}

type stubTractorList struct {
	repositories.TractorRepository

	items []*tractor.Tractor
	last  *repositories.ListTractorsRequest
}

func (s *stubTractorList) List(
	_ context.Context,
	req *repositories.ListTractorsRequest,
) (*pagination.CursorListResult[*tractor.Tractor], error) {
	s.last = req

	return &pagination.CursorListResult[*tractor.Tractor]{Items: s.items}, nil
}

func listedRows[T any](t *testing.T, result any) ([]T, []string) {
	t.Helper()

	var outcome searchOutcome
	var withheld []string
	switch typed := result.(type) {
	case searchOutcome:
		outcome = typed
	case *gatedOutcome:
		outcome = typed.searchOutcome
		withheld = typed.Withheld
	default:
		require.Failf(t, "unexpected result", "%T", result)
	}

	rows := make([]T, 0, len(outcome.Items.([]any)))
	for _, item := range outcome.Items.([]any) {
		row, ok := item.(T)
		require.True(t, ok, "%T", item)
		rows = append(rows, row)
	}

	return rows, withheld
}

func TestListTractors_NamesTheDriverOnlyToSomeoneWhoMayReadWorkers(t *testing.T) {
	t.Parallel()

	unit := &tractor.Tractor{
		ID:            pulid.MustNew("tr_"),
		Code:          "T-104",
		PrimaryWorker: &worker.Worker{FirstName: "Dana", LastName: "Whitfield"},
	}

	for _, tc := range []struct {
		name    string
		allowed bool
		want    string
	}{
		{name: "reads workers", allowed: true, want: "Dana Whitfield"},
		{name: "does not read workers", allowed: false, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &stubTractorList{items: []*tractor.Tractor{unit}}
			params := chatParams(map[string]any{}, "")
			result, err := newListTractorsTool(repo, &fakePermissions{allowed: tc.allowed}).
				Query(t.Context(), params)
			require.NoError(t, err)

			rows, _ := listedRows[equipmentRow](t, result)
			require.Len(t, rows, 1)
			assert.Equal(t, "T-104", rows[0].Code)
			assert.Equal(t, tc.want, rows[0].AssignedTo)
			assert.Equal(t, tc.allowed, repo.last.IncludePrimaryWorker,
				"a driver nobody may be told about is not loaded")
		})
	}
}

func rosterWorker() *worker.Worker {
	return &worker.Worker{
		ID:        pulid.MustNew("wrk_"),
		FirstName: "Dana",
		LastName:  "Whitfield",
		City:      "Joliet",
		Profile:   &worker.WorkerProfile{Endorsement: worker.EndorsementType("X")},
	}
}

func TestListWorkers_WithholdsTheCityBelowTheCeiling(t *testing.T) {
	t.Parallel()

	tool := newListWorkersTool(&fakeWorkerRepo{items: []*worker.Worker{rosterWorker()}},
		&fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{}, permission.SensitivityInternal))
	require.NoError(t, err)
	rows, withheld := listedRows[workerRow](t, result)
	require.Len(t, rows, 1)
	assert.Equal(t, "Dana Whitfield", rows[0].Name)
	assert.Empty(t, rows[0].City)
	assert.Equal(t, []string{"city"}, withheld)

	_, err = tool.Query(t.Context(), agentParams(map[string]any{
		"filters": []any{map[string]any{"field": "city", "operator": "eq", "value": "Joliet"}},
	}, permission.SensitivityInternal))
	require.Error(t, err, "filtering on a withheld field would reveal it")

	_, err = tool.Query(t.Context(), agentParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "profile.endorsement", "operator": "eq", "value": "X",
		}},
	}, permission.SensitivityInternal))
	require.NoError(t, err, "a profile field is judged by the worker's own classification")

	result, err = tool.Query(t.Context(),
		agentParams(map[string]any{}, permission.SensitivityRestricted))
	require.NoError(t, err)
	rows, withheld = listedRows[workerRow](t, result)
	assert.Equal(t, "Joliet", rows[0].City)
	assert.Empty(t, withheld)
}

func TestSearchWorker_WithholdsTheCityBelowTheCeiling(t *testing.T) {
	t.Parallel()

	tool := newSearchWorkerTool(&fakeWorkerRepo{items: []*worker.Worker{rosterWorker()}},
		&fakePermissions{allowed: true})

	for _, tc := range []struct {
		ceiling  permission.FieldSensitivity
		city     string
		withheld []string
	}{
		{ceiling: permission.SensitivityInternal, city: "", withheld: []string{"city"}},
		{ceiling: permission.SensitivityRestricted, city: "Joliet"},
	} {
		result, err := tool.Query(t.Context(), agentParams(map[string]any{}, tc.ceiling))
		require.NoError(t, err)

		gated, ok := result.(*gatedOutcome)
		require.True(t, ok)
		rows, ok := gated.Items.([]workerRow)
		require.True(t, ok)
		require.Len(t, rows, 1)
		assert.Equal(t, tc.city, rows[0].City, tc.ceiling)
		assert.Equal(t, tc.withheld, gated.Withheld, tc.ceiling)
	}
}
