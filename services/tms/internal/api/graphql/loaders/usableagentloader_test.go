package loaders

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingDefinitions struct {
	mu    sync.Mutex
	byID  map[pulid.ID]*agentdefinition.Definition
	calls [][]pulid.ID
}

func (d *countingDefinitions) ListByIDs(
	_ context.Context,
	req repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, req.IDs)

	out := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if definition, ok := d.byID[id]; ok {
			out = append(out, definition)
		}
	}

	return out, nil
}

type countingUsable struct {
	mu     sync.Mutex
	usable services.UsableAgents
	actors []*services.RequestActor
	asked  []permission.Operation
}

func (u *countingUsable) AgentsUsable(
	_ context.Context,
	actor *services.RequestActor,
	operation permission.Operation,
) (*services.UsableAgents, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.actors = append(u.actors, actor)
	u.asked = append(u.asked, operation)
	usable := u.usable

	return &usable, nil
}

type usableAgentsFixture struct {
	tenant      pagination.TenantInfo
	definitions *countingDefinitions
	usable      *countingUsable
	factory     *UsableAgentByIDLoaderFactory
	open        *agentdefinition.Definition
	granted     *agentdefinition.Definition
	withheld    *agentdefinition.Definition
	disabled    *agentdefinition.Definition
	scheduled   *agentdefinition.Definition
}

func newUsableAgentsFixture() *usableAgentsFixture {
	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	agentOf := func(name string, mode agentdefinition.AccessMode) *agentdefinition.Definition {
		return &agentdefinition.Definition{
			ID:          pulid.MustNew("agdef_"),
			Name:        name,
			Enabled:     true,
			TriggerMode: agentdefinition.TriggerChat,
			AccessMode:  mode,
		}
	}
	f := &usableAgentsFixture{
		tenant:    tenant,
		open:      agentOf("Report Builder", agentdefinition.AccessEveryone),
		granted:   agentOf("Payroll", agentdefinition.AccessRoles),
		withheld:  agentOf("Treasury", agentdefinition.AccessRoles),
		disabled:  agentOf("Old Desk", agentdefinition.AccessEveryone),
		scheduled: agentOf("Nightly", agentdefinition.AccessEveryone),
	}
	f.disabled.Enabled = false
	f.scheduled.TriggerMode = agentdefinition.TriggerScheduled
	f.definitions = &countingDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{}}
	for _, definition := range []*agentdefinition.Definition{
		f.open, f.granted, f.withheld, f.disabled, f.scheduled,
	} {
		f.definitions.byID[definition.ID] = definition
	}
	f.usable = &countingUsable{usable: services.UsableAgents{
		Assistant:  true,
		GrantedIDs: []pulid.ID{f.granted.ID},
	}}
	f.factory = &UsableAgentByIDLoaderFactory{definitions: f.definitions, permissions: f.usable}

	return f
}

/*
A page of agents asks for all its delegates at once, and the loader answers in
one query for the agents and one read of what the person may use. Only an
agent the person could ask comes back: open to everyone or granted to one of
their roles, enabled, and talked to. The rest are not found.
*/
func TestUsableAgentByID_BatchesAndKeepsOnlyWhatThePersonMayUse(t *testing.T) {
	t.Parallel()

	f := newUsableAgentsFixture()
	loader := f.factory.NewForTenant(f.tenant)

	ids := []pulid.ID{
		f.open.ID, f.granted.ID, f.withheld.ID, f.disabled.ID, f.scheduled.ID, f.open.ID,
	}
	thunks := make([]func() (*agentdefinition.Definition, error), 0, len(ids))
	for _, id := range ids {
		thunks = append(thunks, loader.LoadThunk(t.Context(), id.String()))
	}

	found := make([]string, 0, len(thunks))
	for _, thunk := range thunks {
		definition, err := thunk()
		if err != nil {
			assert.True(t, errortypes.IsNotFoundError(err), "a withheld agent is not found")
			continue
		}
		found = append(found, definition.Name)
	}

	assert.Equal(t, []string{"Report Builder", "Payroll", "Report Builder"}, found)
	require.Len(t, f.definitions.calls, 1, "one query for the whole page")
	assert.Len(t, f.definitions.calls[0], 5, "each agent is asked for once")
	require.Len(t, f.usable.actors, 1, "one read of what the person may use")
	assert.Equal(t, f.tenant.UserID, f.usable.actors[0].UserID)
	assert.Equal(t, services.PrincipalTypeUser, f.usable.actors[0].PrincipalType)
	assert.Equal(t, permission.OpCreate, f.usable.asked[0],
		"the delegates a person may use are those they could ask, as the turn decides")
}

func TestUsableAgentByID_NothingWithoutTheAssistant(t *testing.T) {
	t.Parallel()

	f := newUsableAgentsFixture()
	f.usable.usable = services.UsableAgents{}

	_, err := f.factory.NewForTenant(f.tenant).Load(t.Context(), f.open.ID.String())

	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, f.definitions.calls, "the agents are not read for a person who may use none")
}

func TestUsableAgentByID_NothingWithoutAPerson(t *testing.T) {
	t.Parallel()

	f := newUsableAgentsFixture()
	f.tenant.UserID = pulid.Nil

	_, err := f.factory.NewForTenant(f.tenant).Load(t.Context(), f.open.ID.String())

	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, f.usable.actors)
}
