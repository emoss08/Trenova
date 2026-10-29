package locationtools

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLocationCreator struct {
	created *location.Location
	actor   *permissionports.RequestActor
	guard   agenttoolstest.WriteGuard
}

func (f *fakeLocationCreator) Create(
	_ context.Context,
	entity *location.Location,
	actor *permissionports.RequestActor,
) (*location.Location, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	entity.ID = pulid.MustNew("loc_")
	f.created = entity
	f.actor = actor

	return entity, nil
}

type fakeLocationCategories struct {
	category  *locationcategory.LocationCategory
	requested locationcategoryports.GetLocationCategoryByIDRequest
}

func (f *fakeLocationCategories) GetByID(
	_ context.Context,
	req locationcategoryports.GetLocationCategoryByIDRequest,
) (*locationcategory.LocationCategory, error) {
	f.requested = req
	if f.category == nil || f.category.ID != req.ID {
		return nil, errors.New("category not found")
	}

	return f.category, nil
}

type fakePlaces struct {
	found *kernelports.AutocompleteLocationResult
	err   error
	asked string
}

func (f *fakePlaces) GetPlaceDetails(
	_ context.Context,
	req *kernelports.AutoCompleteRequest,
) (*kernelports.AutocompleteLocationResult, error) {
	f.asked = req.Input

	return f.found, f.err
}

type locationFixture struct {
	creator    *fakeLocationCreator
	categories *fakeLocationCategories
	places     *fakePlaces
	state      *usstate.UsState
	category   *locationcategory.LocationCategory
	tool       agenttoolports.AgentTool
}

func newLocationFixture() *locationFixture {
	state := &usstate.UsState{ID: pulid.MustNew("us_"), Abbreviation: "NV", Name: "Nevada"}
	category := &locationcategory.LocationCategory{ID: pulid.MustNew("lc_"), Name: "Warehouse"}
	fixture := &locationFixture{
		creator:    &fakeLocationCreator{},
		categories: &fakeLocationCategories{category: category},
		places: &fakePlaces{found: &kernelports.AutocompleteLocationResult{
			Details: []*kernelports.LocationDetails{
				{
					City:       "Sparks",
					State:      "NV",
					PostalCode: "89431",
					Latitude:   39.53,
					Longitude:  -119.75,
				},
				{
					City: "Reno", State: "NV", PostalCode: "89502-1234",
					Latitude: 39.50, Longitude: -119.78, PlaceID: "place-reno",
				},
			},
			Count: 2,
		}},
		state:    state,
		category: category,
	}
	fixture.tool = NewCreateLocationTool(CreateLocationToolParams{
		Locations:  fixture.creator,
		States:     &agenttoolstest.FakeStates{State: state},
		Categories: fixture.categories,
		Places:     fixture.places,
	})

	return fixture
}

func (f *locationFixture) params(overrides map[string]any) agenttoolports.ToolExecuteParams {
	values := map[string]any{
		"name":               "Reno Cold Storage",
		"addressLine1":       "1200 Commerce Way",
		"addressLine2":       "Dock 4",
		"city":               "Reno",
		"state":              "nv",
		"postalCode":         "89502",
		"locationCategoryId": f.category.ID.String(),
	}
	for key, value := range overrides {
		if value == nil {
			delete(values, key)
			continue
		}
		values[key] = value
	}

	params := agenttoolstest.ExecuteParams(values)
	params.IdempotencyKey = "turn-1:location_create"

	return params
}

func TestCreateLocation_IsProposedForApprovalAndStaysInternal(t *testing.T) {
	t.Parallel()

	policy := newLocationFixture().tool.Policy()
	assert.Equal(t, agenttooltrust.ToolKindAction, policy.Kind)
	assert.Equal(t, permission.ResourceLocation, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.DefaultTier)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	assert.Equal(t, agenttooltrust.ToolEffectChange, policy.Effect)
	assert.True(t, policy.Idempotent)
	assert.True(t, permission.IsAgentAllowed(permission.ResourceLocation, permission.OpCreate))
}

func TestCreateLocation_CreatesTheLocationInTheCallersTenant(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	params := fixture.params(nil)

	reporter := fixture.tool.(agenttoolports.ToolResultReporter)
	result, err := reporter.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	created := fixture.creator.created
	require.NotNil(t, created)
	assert.Equal(t, params.OrganizationID, created.OrganizationID)
	assert.Equal(t, params.BusinessUnitID, created.BusinessUnitID)
	assert.Equal(t, fixture.state.ID, created.StateID)
	assert.Equal(t, fixture.category.ID, created.LocationCategoryID)
	assert.Equal(t, domaintypes.StatusActive, created.Status)
	assert.Equal(t, "Reno Cold Storage", created.Name)
	assert.Equal(t, "Dock 4", created.AddressLine2)
	assert.Same(t, params.Actor, fixture.creator.actor)
	assert.Equal(t, params.OrganizationID, fixture.categories.requested.TenantInfo.OrgID)

	assert.True(t, created.IsGeocoded, "a location is placed on the map before it is saved")
	require.NotNil(t, created.Latitude)
	require.NotNil(t, created.Longitude)
	assert.InDelta(t, 39.50, *created.Latitude, 0.0001)
	assert.InDelta(t, -119.78, *created.Longitude, 0.0001)
	assert.Equal(t, "place-reno", created.PlaceID)
	assert.Equal(t, "1200 Commerce Way, Reno, NV 89502", fixture.places.asked)

	assert.Equal(t, "created", result.Action)
	assert.Equal(t, created.ID.String(), result.IDs["locationId"])
	require.NotNil(t, result.Record)
	assert.Equal(t, "location", result.Record.EntityType)
}

func TestCreateLocation_ValidatesBeforeTheProposalIsShown(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	validator := fixture.tool.(agenttoolports.ToolValidator)

	require.NoError(t, validator.Validate(t.Context(), fixture.params(nil)))

	cases := map[string]map[string]any{
		"a missing name":       {"name": nil},
		"a blank city":         {"city": "   "},
		"a state name":         {"state": "Nevada"},
		"an unknown state":     {"state": "ZZ"},
		"an unknown category":  {"locationCategoryId": pulid.MustNew("lc_").String()},
		"a malformed category": {"locationCategoryId": "warehouse"},
	}
	for name, overrides := range cases {
		err := validator.Validate(t.Context(), fixture.params(overrides))
		assert.Error(t, err, name)
	}
	assert.Nil(t, fixture.creator.created, "validation never creates")
}

func TestCreateLocation_SavesAnAddressItCannotPlaceAndSaysWhy(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		places *fakePlaces
		says   string
	}{
		"no lookup connected": {
			places: &fakePlaces{
				err: fmt.Errorf("%w: no key", kernelports.ErrPlaceLookupUnavailable),
			},
			says: placeLookupMissing,
		},
		"the lookup failing": {
			places: &fakePlaces{err: errors.New("timeout")},
			says:   placeLookupFailed,
		},
		"no place in that state and ZIP": {
			places: &fakePlaces{found: &kernelports.AutocompleteLocationResult{
				Details: []*kernelports.LocationDetails{
					{City: "Reno", State: "TX", PostalCode: "75462"},
				},
			}},
			says: placeNotFound,
		},
	}
	for name, tc := range cases {
		fixture := newLocationFixture()
		fixture.tool.(*CreateLocationTool).places = tc.places

		preview, err := fixture.tool.(agenttoolports.ToolPreviewer).Preview(
			t.Context(), fixture.params(nil),
		)
		require.NoError(t, err, name)
		assert.Contains(t, preview.Summary, "saved without map coordinates, because "+tc.says, name)

		_, err = fixture.tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(
			t.Context(), fixture.params(nil),
		)
		require.NoError(t, err, name)
		created := fixture.creator.created
		require.NotNil(t, created, name)
		assert.False(t, created.IsGeocoded, name)
		assert.Nil(t, created.Latitude, name)
		assert.Nil(t, created.Longitude, name)
	}
}

func TestCreateLocation_RefusesWithoutAnIdempotencyKeyOrActor(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()

	params := fixture.params(nil)
	params.IdempotencyKey = ""
	require.ErrorIs(
		t,
		fixture.tool.Execute(t.Context(), params),
		toolguard.ErrMissingIdempotencyKey,
	)

	params = fixture.params(nil)
	params.Actor = nil
	require.ErrorIs(t, fixture.tool.Execute(t.Context(), params), toolguard.ErrMissingActor)
	assert.Nil(t, fixture.creator.created)
}
