package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLocationCreator struct {
	created *location.Location
	actor   *serviceports.RequestActor
}

func (f *fakeLocationCreator) Create(
	_ context.Context,
	entity *location.Location,
	actor *serviceports.RequestActor,
) (*location.Location, error) {
	entity.ID = pulid.MustNew("loc_")
	f.created = entity
	f.actor = actor

	return entity, nil
}

type fakeStates struct {
	state *usstate.UsState
}

func (f *fakeStates) GetByAbbreviation(
	_ context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	if f.state == nil || f.state.Abbreviation != abbreviation {
		return nil, errors.New("state not found")
	}

	return f.state, nil
}

type fakeLocationCategories struct {
	category  *locationcategory.LocationCategory
	requested repositories.GetLocationCategoryByIDRequest
}

func (f *fakeLocationCategories) GetByID(
	_ context.Context,
	req repositories.GetLocationCategoryByIDRequest,
) (*locationcategory.LocationCategory, error) {
	f.requested = req
	if f.category == nil || f.category.ID != req.ID {
		return nil, errors.New("category not found")
	}

	return f.category, nil
}

type locationFixture struct {
	creator    *fakeLocationCreator
	categories *fakeLocationCategories
	state      *usstate.UsState
	category   *locationcategory.LocationCategory
	tool       serviceports.AgentTool
}

func newLocationFixture() *locationFixture {
	state := &usstate.UsState{ID: pulid.MustNew("us_"), Abbreviation: "NV", Name: "Nevada"}
	category := &locationcategory.LocationCategory{ID: pulid.MustNew("lc_"), Name: "Warehouse"}
	fixture := &locationFixture{
		creator:    &fakeLocationCreator{},
		categories: &fakeLocationCategories{category: category},
		state:      state,
		category:   category,
	}
	fixture.tool = newCreateLocationTool(
		fixture.creator, &fakeStates{state: state}, fixture.categories,
	)

	return fixture
}

func (f *locationFixture) params(overrides map[string]any) serviceports.ToolExecuteParams {
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

	params := executeParams(values)
	params.IdempotencyKey = "turn-1:create_location"

	return params
}

func TestCreateLocation_IsProposedForApprovalAndStaysInternal(t *testing.T) {
	t.Parallel()

	policy := newLocationFixture().tool.Policy()
	assert.Equal(t, agent.ToolKindAction, policy.Kind)
	assert.Equal(t, permission.ResourceLocation, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.DefaultTier)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	assert.Equal(t, agent.ToolEffectChange, policy.Effect)
	assert.True(t, policy.Idempotent)
	assert.True(t, permission.IsAgentAllowed(permission.ResourceLocation, permission.OpCreate))
}

func TestCreateLocation_CreatesTheLocationInTheCallersTenant(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	params := fixture.params(nil)

	reporter := fixture.tool.(serviceports.ToolResultReporter)
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

	assert.Equal(t, "created", result.Action)
	assert.Equal(t, created.ID.String(), result.IDs["locationId"])
	require.NotNil(t, result.Record)
	assert.Equal(t, "location", result.Record.EntityType)
}

func TestCreateLocation_ValidatesBeforeTheProposalIsShown(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	validator := fixture.tool.(serviceports.ToolValidator)

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

func TestCreateLocation_SimulatesWithoutCreating(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	simulator := fixture.tool.(serviceports.ToolSimulator)

	simulation, err := simulator.Simulate(t.Context(), fixture.params(nil))
	require.NoError(t, err)
	assert.True(t, simulation.Previewed)
	assert.Contains(t, simulation.Summary, "Reno Cold Storage")
	assert.Contains(t, simulation.Summary, "NV 89502")
	assert.Nil(t, fixture.creator.created)
}

func TestCreateLocation_RefusesWithoutAnIdempotencyKeyOrActor(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()

	params := fixture.params(nil)
	params.IdempotencyKey = ""
	require.ErrorIs(t, fixture.tool.Execute(t.Context(), params), ErrMissingIdempotencyKey)

	params = fixture.params(nil)
	params.Actor = nil
	require.ErrorIs(t, fixture.tool.Execute(t.Context(), params), ErrMissingActor)
	assert.Nil(t, fixture.creator.created)
}
