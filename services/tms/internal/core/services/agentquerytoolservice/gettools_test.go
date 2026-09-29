package agentquerytoolservice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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

		assert.Regexp(t, `^get_[a-z_]+$`, spec.name)
		assert.NotEmpty(t, spec.resource)
		assert.Contains(t, spec.summary, "list_",
			"%s must name the tool that yields its id", spec.name)
	}
}

func TestGetTools_EveryRegisteredGetToolWithholdsNestedPeople(t *testing.T) {
	t.Parallel()

	engine := &fakePermissions{allowed: true}
	supplied := map[reflect.Type]reflect.Value{
		reflect.TypeFor[*filtercatalog.Catalog](): reflect.ValueOf(FilterCatalog()),
		reflect.TypeFor[serviceports.PermissionEngine](): reflect.ValueOf(
			serviceports.PermissionEngine(engine),
		),
	}

	gets := 0
	for _, provider := range ToolProviders() {
		built, err := reflectutils.Construct(provider, supplied)
		require.NoError(t, err)

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

	assert.GreaterOrEqual(t, gets, len(getCatalogSpecs()),
		"every catalogued get tool is registered through newGetTool")
}

type stubTractors struct {
	repositories.TractorRepository

	entity *tractor.Tractor
	last   repositories.GetTractorByIDRequest
}

func (s *stubTractors) GetByID(
	_ context.Context,
	req repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	s.last = req

	return s.entity, nil
}

type tractorFixture struct {
	entity  *tractor.Tractor
	manager *tenant.User
}

func tractorWithDrivers() tractorFixture {
	orgID := pulid.MustNew(tenant.OrganizationIDPrefix)
	buID := pulid.MustNew(tenant.BusinessUnitIDPrefix)
	manager := &tenant.User{
		ID:           pulid.MustNew(tenant.UserIDPrefix),
		Name:         "Morgan Reyes",
		Username:     "mreyes",
		EmailAddress: "morgan.reyes@example.com",
		Timezone:     "America/Chicago",
	}
	primary := &worker.Worker{
		ID:           pulid.MustNew("wrk_"),
		FirstName:    "Dana",
		LastName:     "Whitfield",
		AddressLine1: "18 Maple Row",
		City:         "Joliet",
		Email:        "dana.whitfield@example.com",
		PhoneNumber:  "815-555-0142",
		ManagerID:    manager.ID,
		Manager:      manager,
		Profile: &worker.WorkerProfile{
			ID:            pulid.MustNew("wpr_"),
			LicenseNumber: "W123-4567-8901",
		},
	}

	return tractorFixture{
		manager: manager,
		entity: &tractor.Tractor{
			ID:              pulid.MustNew("tr_"),
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Code:            "T-104",
			PrimaryWorkerID: primary.ID,
			PrimaryWorker:   primary,
			Organization: &tenant.Organization{
				ID:           orgID,
				Name:         "Lakeshore Logistics",
				AddressLine1: "900 Harbor Rd",
			},
			BusinessUnit: &tenant.BusinessUnit{ID: buID, Name: "Midwest", Code: "MW"},
		},
	}
}

func queryTractor(
	t *testing.T,
	fixture tractorFixture,
	permissions *fakePermissions,
	params *serviceports.QueryToolParams,
) map[string]any {
	t.Helper()

	repo := &stubTractors{entity: fixture.entity}
	params.Params = map[string]any{"tractorId": fixture.entity.ID.String()}
	result, err := newGetTractorTool(repo, permissions).Query(t.Context(), params)
	require.NoError(t, err)
	assert.True(t, repo.last.IncludePrimaryWorker)

	return encodedDocument(t, result)
}

func TestGetTractor_WithholdsDriverContactBelowTheCeiling(t *testing.T) {
	t.Parallel()

	fixture := tractorWithDrivers()
	document := queryTractor(t, fixture, &fakePermissions{allowed: true},
		agentParams(nil, permission.SensitivityInternal))

	encoded, err := sonic.Marshal(document)
	require.NoError(t, err)
	for _, contact := range []string{
		"dana.whitfield@example.com", "815-555-0142", "18 Maple Row", "Joliet",
		"morgan.reyes@example.com", "W123-4567-8901", "900 Harbor Rd",
	} {
		assert.NotContains(t, string(encoded), contact)
	}

	primary := objectAt(t, document, "primaryWorker")
	assert.Equal(t, "Dana", primary["firstName"])
	assert.Equal(t, "Whitfield", primary["lastName"])
	assert.Subset(t, document["withheldByAccess"], []any{
		"worker.email", "worker.phoneNumber", "worker.addressLine1", "worker.city",
	})
	assert.NotContains(t, document["withheldByAccess"], "worker.licenseNumber",
		"a confidential field is never named")
}

func TestGetTractor_ShowsDriverContactAtRestricted(t *testing.T) {
	t.Parallel()

	fixture := tractorWithDrivers()
	document := queryTractor(t, fixture,
		personReachingEach(permission.SensitivityRestricted, permission.ResourceWorker),
		chatParams(nil, ""))

	primary := objectAt(t, document, "primaryWorker")
	assert.Equal(t, "dana.whitfield@example.com", primary["email"])
	assert.Equal(t, "815-555-0142", primary["phoneNumber"])
	assert.Equal(t, "18 Maple Row", primary["addressLine1"])
	assert.Equal(t, map[string]any{
		"id": fixture.manager.ID.String(), "name": "Morgan Reyes",
	}, primary["manager"], "the manager is a user, reduced to a name")
	assert.NotContains(t, objectAt(t, primary, "profile"), "licenseNumber")
}

func TestGetTractor_ReducesTheOrganizationAndBusinessUnit(t *testing.T) {
	t.Parallel()

	fixture := tractorWithDrivers()
	document := queryTractor(t, fixture, &fakePermissions{allowed: true},
		agentParams(nil, permission.SensitivityRestricted))

	assert.Equal(t, map[string]any{
		"id": fixture.entity.OrganizationID.String(), "name": "Lakeshore Logistics",
	}, document["organization"])
	assert.Equal(t, map[string]any{
		"id": fixture.entity.BusinessUnitID.String(), "name": "Midwest",
	}, document["businessUnit"])
	assert.Equal(t, "T-104", document["code"])
}
