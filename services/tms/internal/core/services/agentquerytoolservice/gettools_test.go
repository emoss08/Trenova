package agentquerytoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captured struct {
	id     pulid.ID
	tenant pagination.TenantInfo
	result any
	err    error
}

func probeGetSpec(capture *captured) getSpec {
	return getSpec{
		name:      "get_probe",
		entity:    "probe",
		summary:   "Retrieve one probe.",
		resource:  permission.ResourceWorker,
		paramName: "probeId",
		fetch: func(
			_ context.Context,
			id pulid.ID,
			tenant pagination.TenantInfo,
		) (any, error) {
			capture.id = id
			capture.tenant = tenant

			return capture.result, capture.err
		},
	}
}

/*
A get tool is the expand step after a list or a search, so the id it is handed
came from us. Parsing it rather than passing the string through means a model
that paraphrases an id — or pastes a pro number where an id belongs — is told
so, instead of the repository returning nothing and the model reporting the
record as missing.
*/
func TestGetTool_RefusesSomethingThatIsNotAnID(t *testing.T) {
	t.Parallel()

	capture := &captured{}
	tool := newGetTool(probeGetSpec(capture))

	_, err := tool.Query(t.Context(), testParams(map[string]any{"probeId": "PRO-4471"}))
	require.Error(t, err)
	assert.True(t, capture.id.IsNil(), "nothing is fetched for an unparseable id")
}

func TestGetTool_ScopesTheLookupToTheActorTenant(t *testing.T) {
	t.Parallel()

	capture := &captured{result: map[string]any{"id": "x"}}
	tool := newGetTool(probeGetSpec(capture))

	id := pulid.MustNew("wrk_")
	params := testParams(map[string]any{"probeId": id.String()})

	_, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, id, capture.id)
	assert.Equal(t, params.OrganizationID, capture.tenant.OrgID)
	assert.Equal(t, params.BusinessUnitID, capture.tenant.BuID)
}

func TestGetTool_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	capture := &captured{}
	tool := newGetTool(probeGetSpec(capture))

	params := testParams(map[string]any{"probeId": pulid.MustNew("wrk_").String()})
	params.Actor.BusinessUnitID = pulid.MustNew("bu_")

	_, err := tool.Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.True(t, capture.id.IsNil())
}

func TestGetTool_SurfacesTheRepositoryFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("probe not found")
	capture := &captured{err: failure}
	tool := newGetTool(probeGetSpec(capture))

	_, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"probeId": pulid.MustNew("wrk_").String()}),
	)
	require.ErrorIs(t, err, failure)
}

func TestGetTool_NamesItsParameterInTheSchema(t *testing.T) {
	t.Parallel()

	schema := newGetTool(probeGetSpec(&captured{})).ParamSchema()

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "probeId")
	assert.Equal(t, []string{"probeId"}, schema["required"])
}

// Every get tool points at the list tool that produces the id it needs, so a
// model holding only a name has somewhere to go.
func TestGetCatalog_EveryToolSaysWhereItsIDComesFrom(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for _, spec := range getCatalogSpecs() {
		assert.False(t, seen[spec.name], "%s is registered twice", spec.name)
		seen[spec.name] = true

		assert.Regexp(t, `^get_[a-z_]+$`, spec.name)
		assert.NotEmpty(t, spec.resource)
		assert.Contains(t, spec.summary, "list_",
			"%s must name the tool that yields its id", spec.name)
	}
}
