package agentquerytoolservice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/reflectutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captured struct {
	id     pulid.ID
	tenant pagination.TenantInfo
	result any
	err    error
}

func probeGetSpec(capture *captured) *getSpec {
	return &getSpec{
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
	tool := newGetTool(probeGetSpec(capture), &fakePermissions{})

	_, err := tool.Query(t.Context(), testParams(map[string]any{"probeId": "PRO-4471"}))
	require.Error(t, err)
	assert.True(t, capture.id.IsNil(), "nothing is fetched for an unparseable id")
}

func TestGetTool_ScopesTheLookupToTheActorTenant(t *testing.T) {
	t.Parallel()

	capture := &captured{result: map[string]any{"id": "x"}}
	tool := newGetTool(probeGetSpec(capture), &fakePermissions{})

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
	tool := newGetTool(probeGetSpec(capture), &fakePermissions{})

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
	tool := newGetTool(probeGetSpec(capture), &fakePermissions{})

	_, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"probeId": pulid.MustNew("wrk_").String()}),
	)
	require.ErrorIs(t, err, failure)
}

func TestGetTool_NamesItsParameterInTheSchema(t *testing.T) {
	t.Parallel()

	schema := newGetTool(probeGetSpec(&captured{}), &fakePermissions{}).ParamSchema()

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

		assert.Regexp(t, `^[a-z]+_get(_[a-z_]+)?$`, spec.name)
		assert.NotEmpty(t, spec.resource)
		assert.Contains(t, spec.summary, "_list",
			"%s must name the tool that yields its id", spec.name)
	}
}

func TestGetTools_EveryRegisteredGetToolWithholdsNestedPeople(t *testing.T) {
	t.Parallel()

	engine := &fakePermissions{allowed: true}
	supplied := map[reflect.Type]reflect.Value{
		reflect.TypeFor[*filtercatalog.Catalog](): reflect.ValueOf(FilterCatalog()),
		reflect.TypeFor[permissionports.PermissionEngine](): reflect.ValueOf(
			permissionports.PermissionEngine(engine),
		),
	}

	gets, graphQLGets := 0, 0
	for _, provider := range ToolProviders() {
		built, err := reflectutils.Construct(provider, supplied)
		require.NoError(t, err)

		if _, ok := built.(*graphQLGetTool); ok {
			graphQLGets++
			continue
		}
		tool, ok := built.(*getTool)
		if !ok {
			continue
		}
		gets++

		assert.Same(t, engine, tool.access.permissions,
			"%s must judge fields with the permission engine", tool.Name())

		userID := pulid.MustNew(tenant.UserIDPrefix).String()
		probe := *tool
		probe.spec.fetch = func(context.Context, pulid.ID, pagination.TenantInfo) (any, error) {
			return map[string]any{
				"id": pulid.MustNew("rec_").String(),
				"createdBy": map[string]any{
					"id":           userID,
					"name":         "Avery Chen",
					"emailAddress": "avery.chen@example.com",
				},
			}, nil
		}

		result, err := probe.Query(t.Context(), testParams(map[string]any{
			tool.spec.paramName: pulid.MustNew("rec_").String(),
		}))
		require.NoError(t, err, tool.Name())
		document := encodedDocument(t, result)
		assert.Equal(t, map[string]any{"id": userID, "name": "Avery Chen"},
			document["createdBy"], "%s returned a nested user whole", tool.Name())
	}

	repositoryGets := 0
	for _, spec := range getCatalogSpecs() {
		if _, ok := graphQLReads[spec.name]; !ok {
			repositoryGets++
		}
	}
	assert.GreaterOrEqual(t, gets, repositoryGets,
		"every catalogued get tool that reads a repository is registered through newGetTool")
	assert.Equal(t, len(graphQLGetSpecs()), graphQLGets,
		"every get tool that reads through GraphQL is registered, its nested people "+
			"withheld by the field guard")
}
