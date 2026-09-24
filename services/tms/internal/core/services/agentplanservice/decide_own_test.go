package agentplanservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRuns struct {
	run *agent.AgentRun
}

func (f *fakeRuns) GetByID(
	_ context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	if f.run == nil || f.run.ID != req.ID {
		return nil, errortypes.NewNotFoundError("Agent run not found")
	}

	return f.run, nil
}

type fakeThreads struct {
	owner  pulid.ID
	thread pulid.ID
	asked  []repositories.GetThreadRequest
}

func (f *fakeThreads) GetThread(
	_ context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	f.asked = append(f.asked, req)
	if req.ID != f.thread || req.UserID != f.owner {
		return nil, errortypes.NewNotFoundError("Thread not found")
	}

	return &conversation.Thread{ID: f.thread, UserID: f.owner}, nil
}

type fakeAgents struct {
	byID map[pulid.ID]*agentdefinition.Definition
}

func (f *fakeAgents) GetByID(
	_ context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	definition, ok := f.byID[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Agent definition not found")
	}

	return definition, nil
}

type fakeAccess struct {
	granted map[pulid.ID]bool
	checked []pulid.ID
}

func (f *fakeAccess) MayUseAgent(
	_ context.Context,
	_ *services.RequestActor,
	definition *agentdefinition.Definition,
) (bool, error) {
	f.checked = append(f.checked, definition.ID)

	return definition.OpenToEveryone() || f.granted[definition.ID], nil
}

type ownFixture struct {
	*harness
	runs    *fakeRuns
	threads *fakeThreads
	agents  *fakeAgents
	access  *fakeAccess
	agent   *agentdefinition.Definition
}

func newOwnFixture(t *testing.T) *ownFixture {
	t.Helper()

	h := newHarness(t, 2, false)
	threadID := pulid.MustNew("athr_")
	definition := &agentdefinition.Definition{
		ID:             pulid.MustNew("agdef_"),
		Name:           "Dispatch",
		OrganizationID: h.req.TenantInfo.OrgID,
		BusinessUnitID: h.req.TenantInfo.BuID,
		AccessMode:     agentdefinition.AccessRoles,
		Enabled:        true,
	}
	f := &ownFixture{
		harness: h,
		runs: &fakeRuns{run: &agent.AgentRun{
			ID:                h.plans.plan.RunID,
			AgentDefinitionID: definition.ID,
			SubjectType:       agent.SubjectAssistantThread,
			SubjectID:         threadID,
		}},
		threads: &fakeThreads{owner: h.actor.UserID, thread: threadID},
		agents: &fakeAgents{byID: map[pulid.ID]*agentdefinition.Definition{
			definition.ID: definition,
		}},
		access: &fakeAccess{granted: map[pulid.ID]bool{definition.ID: true}},
		agent:  definition,
	}
	h.svc.runs = f.runs
	h.svc.threads = f.threads
	h.svc.agents = f.agents
	h.svc.access = f.access

	return f
}

/*
A person approves a plan raised in their own conversation without the
approver's permission the decisions queue needs. The plan still runs through
the same decision path: every step is decided in order, as the person, so each
write is permission-checked against them when it runs.
*/
func TestDecideOwn_ApprovesAPlanFromTheCallersOwnConversation(t *testing.T) {
	t.Parallel()

	f := newOwnFixture(t)

	plan, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)

	require.NoError(t, err)
	assert.Equal(t, agent.PlanStatusCompleted, plan.Status)
	require.Len(t, f.decider.decided, 2, "every step goes through the decision service")
	for _, decided := range f.decider.decided {
		assert.True(t, decided.WithinPlan)
	}
	require.Len(t, f.threads.asked, 1)
	assert.Equal(t, f.actor.UserID, f.threads.asked[0].UserID,
		"the conversation is read under the caller's own id")
	assert.Equal(t, []pulid.ID{f.agent.ID}, f.access.checked)
}

// Someone else's plan is not found, the way their conversation is: nothing
// is decided, and the plan is not moved out of Pending.
func TestDecideOwn_SomeoneElsesPlanIsNotFound(t *testing.T) {
	t.Parallel()

	f := newOwnFixture(t)
	f.threads.owner = pulid.MustNew("usr_")

	_, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, f.decider.decided)
	assert.Empty(t, f.plans.statuses)
}

// A plan from a background run was never raised in a conversation, so it is
// no one's own; it is decided from the queue.
func TestDecideOwn_APlanFromABackgroundRunIsNotFound(t *testing.T) {
	t.Parallel()

	f := newOwnFixture(t)
	f.runs.run.SubjectType = agent.SubjectShipmentMove

	_, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)

	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, f.threads.asked)
	assert.Empty(t, f.decider.decided)
}

// Owning the conversation is not enough: the person must still be allowed
// the agent that raised the plan. Granting it lets the same request through.
func TestDecideOwn_RefusesAnAgentTheCallerMayNoLongerUse(t *testing.T) {
	t.Parallel()

	f := newOwnFixture(t)
	f.access.granted = map[pulid.ID]bool{}

	_, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Contains(t, err.Error(), "Dispatch")
	assert.Empty(t, f.decider.decided)
	assert.Empty(t, f.plans.statuses)

	f.access.granted[f.agent.ID] = true
	_, err = f.svc.DecideOwn(t.Context(), f.req, f.actor)
	require.NoError(t, err)
}

func TestDecideOwn_RefusesAPlanWhoseAgentWasRemoved(t *testing.T) {
	t.Parallel()

	f := newOwnFixture(t)
	delete(f.agents.byID, f.agent.ID)

	_, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer exists")
	assert.Empty(t, f.decider.decided)
}

// A service without the conversations wired cannot tell whose a plan is,
// so it treats every plan as someone else's rather than as everyone's.
func TestDecideOwn_WithoutConversationsNothingIsYours(t *testing.T) {
	t.Parallel()

	f := newOwnFixture(t)
	f.svc.threads = nil

	_, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)

	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, f.decider.decided)
}
