package assistantfollowupservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type fakeRuns struct {
	repositories.AgentRunRepository
	run *agent.AgentRun
}

func (f *fakeRuns) GetByID(
	_ context.Context,
	_ repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return f.run, nil
}

type fakeConversations struct {
	repositories.ConversationRepository
	thread *conversation.Thread
	asked  []repositories.GetThreadOwnedRequest
}

func (f *fakeConversations) GetThreadOwned(
	_ context.Context,
	req repositories.GetThreadOwnedRequest,
) (*conversation.Thread, error) {
	f.asked = append(f.asked, req)

	return f.thread, nil
}

// fakeWorkflows records the turn handed to a worker.
type fakeWorkflows struct {
	serviceports.WorkflowStarter
	payloads []*assistantjobs.AssistantTurnPayload
}

func (f *fakeWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	f.payloads = append(f.payloads, args[0].(*assistantjobs.AssistantTurnPayload))

	return startedRun{id: options.ID}, nil
}

type startedRun struct {
	client.WorkflowRun
	id string
}

func (r startedRun) GetID() string { return r.id }

type fakeTurns struct {
	startErr error
	started  []assistantturnservice.StartRequest
	workflow []string
}

func (f *fakeTurns) StartTurn(
	_ context.Context,
	req assistantturnservice.StartRequest,
	start func(turn *conversation.AssistantTurn) (string, error),
) (*conversation.AssistantTurn, error) {
	f.started = append(f.started, req)
	if f.startErr != nil {
		return nil, f.startErr
	}

	turn := &conversation.AssistantTurn{ID: pulid.MustNew("atrn_"), ThreadID: req.ThreadID}
	workflowID, err := start(turn)
	if err != nil {
		return nil, err
	}
	f.workflow = append(f.workflow, workflowID)

	return turn, nil
}

type fixture struct {
	service       *Service
	runs          *fakeRuns
	conversations *fakeConversations
	workflows     *fakeWorkflows
	turns         *fakeTurns
	permissions   *agentruntimetest.StubPermissions
	agent         *agentdefinition.Definition
	tenant        pagination.TenantInfo
	thread        *conversation.Thread
}

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository
	definition *agentdefinition.Definition
}

func (f *fakeDefinitions) GetByID(
	_ context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if f.definition == nil || f.definition.ID != req.ID {
		return nil, errortypes.NewNotFoundError("AgentDefinition not found")
	}

	return f.definition, nil
}

func newFixture(subject agent.SubjectType) *fixture {
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	definition := &agentdefinition.Definition{
		ID:          pulid.MustNew("agdef_"),
		Name:        "Report Builder",
		Enabled:     true,
		TriggerMode: agentdefinition.TriggerChat,
		AccessMode:  agentdefinition.AccessEveryone,
	}
	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		UserID:            pulid.MustNew("usr_"),
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: definition.ID,
	}
	f := &fixture{
		runs: &fakeRuns{run: &agent.AgentRun{
			ID:          pulid.MustNew("arun_"),
			SubjectType: subject,
			SubjectID:   thread.ID,
		}},
		conversations: &fakeConversations{thread: thread},
		workflows:     &fakeWorkflows{},
		turns:         &fakeTurns{},
		permissions:   &agentruntimetest.StubPermissions{},
		agent:         definition,
		tenant:        tenant,
		thread:        thread,
	}
	f.service = &Service{
		l:             zap.NewNop(),
		runs:          f.runs,
		conversations: f.conversations,
		definitions:   &fakeDefinitions{definition: definition},
		permissions:   f.permissions,
		turns:         f.turns,
		workflows:     f.workflows,
	}

	return f
}

/*
A decision on a proposal a conversation raised is reported in that conversation.

It used to be reported only when the card in the thread was clicked, and before
the change had run. The turn now starts from the decision service, whoever
decided and wherever, as the conversation's owner: the report is theirs.
*/
func TestFollowUp_ReportsTheDecisionInTheConversationAsItsOwner(t *testing.T) {
	t.Parallel()

	f := newFixture(agent.SubjectAssistantThread)
	proposalID := pulid.MustNew("aprop_")

	f.service.FollowUp(t.Context(), serviceports.DecisionFollowUpRequest{
		TenantInfo: f.tenant,
		RunID:      f.runs.run.ID,
		ProposalID: proposalID,
	})

	require.Len(t, f.turns.started, 1)
	started := f.turns.started[0]
	assert.Equal(t, conversation.AssistantTurnOriginDecisionFollowUp, started.Origin)
	assert.Equal(t, f.thread.UserID, started.UserID)
	assert.Equal(t, f.thread.ID, started.ThreadID)

	require.Len(t, f.workflows.payloads, 1, "a worker answers it, like any turn")
	payload := f.workflows.payloads[0]
	assert.Equal(t, proposalID, payload.Request.FollowUpProposalID)
	assert.True(t, payload.Request.FollowUpPlanID.IsNil())
	assert.Empty(t, payload.Content)
	assert.Equal(t, f.thread.UserID, payload.Actor.UserID)
	assert.Equal(t, serviceports.PrincipalTypeUser, payload.Actor.PrincipalType)
	require.Len(t, f.turns.workflow, 1)
	assert.Equal(t, assistantjobs.WorkflowIDFor(payload.TurnID), f.turns.workflow[0],
		"the turn's record names the execution carrying it")
}

// A plan reports once, as a plan.
func TestFollowUp_ReportsAPlanByItsID(t *testing.T) {
	t.Parallel()

	f := newFixture(agent.SubjectAssistantThread)
	planID := pulid.MustNew("apl_")

	f.service.FollowUp(t.Context(), serviceports.DecisionFollowUpRequest{
		TenantInfo: f.tenant,
		RunID:      f.runs.run.ID,
		PlanID:     planID,
	})

	require.Len(t, f.workflows.payloads, 1)
	assert.Equal(t, planID, f.workflows.payloads[0].Request.FollowUpPlanID)
	assert.True(t, f.workflows.payloads[0].Request.FollowUpProposalID.IsNil())
}

// A background run's proposal belongs to no conversation, so nothing is
// started and no conversation is read.
func TestFollowUp_LeavesDecisionsOutsideAConversationAlone(t *testing.T) {
	t.Parallel()

	f := newFixture(agent.SubjectShipment)

	f.service.FollowUp(t.Context(), serviceports.DecisionFollowUpRequest{
		TenantInfo: f.tenant,
		RunID:      f.runs.run.ID,
		ProposalID: pulid.MustNew("aprop_"),
	})

	assert.Empty(t, f.conversations.asked)
	assert.Empty(t, f.turns.started)
	assert.Empty(t, f.workflows.payloads)
}

// A conversation already producing a reply is not interrupted. The outcome
// reaches the agent on that turn instead.
func TestFollowUp_DoesNotInterruptAConversationMidReply(t *testing.T) {
	t.Parallel()

	f := newFixture(agent.SubjectAssistantThread)
	f.turns.startErr = errors.New("already working on a reply")

	f.service.FollowUp(t.Context(), serviceports.DecisionFollowUpRequest{
		TenantInfo: f.tenant,
		RunID:      f.runs.run.ID,
		ProposalID: pulid.MustNew("aprop_"),
	})

	assert.Len(t, f.turns.started, 1)
	assert.Empty(t, f.workflows.payloads)
}

// A request that names both a proposal and a plan, or neither, is a bug in
// the caller and starts nothing.
func TestFollowUp_RefusesARequestThatNamesNoSingleDecision(t *testing.T) {
	t.Parallel()

	f := newFixture(agent.SubjectAssistantThread)

	f.service.FollowUp(t.Context(), serviceports.DecisionFollowUpRequest{
		TenantInfo: f.tenant,
		RunID:      f.runs.run.ID,
	})
	f.service.FollowUp(t.Context(), serviceports.DecisionFollowUpRequest{
		TenantInfo: f.tenant,
		RunID:      f.runs.run.ID,
		ProposalID: pulid.MustNew("aprop_"),
		PlanID:     pulid.MustNew("apl_"),
	})

	assert.Empty(t, f.turns.started)
}

/*
A person who lost access to the conversation's agent keeps the conversation to
read, and the decision stands, but the agent is not asked to report it: the
turn would run as someone who may no longer use it.
*/
func TestFollowUp_SkipsWhenTheOwnerMayNoLongerUseTheAgent(t *testing.T) {
	t.Parallel()

	tests := map[string]func(f *fixture){
		"the agent is restricted to roles the owner does not hold": func(f *fixture) {
			f.agent.AccessMode = agentdefinition.AccessRoles
		},
		"the owner may no longer use the assistant": func(f *fixture) {
			f.permissions.Denied = map[string]bool{"assistant:create": true}
		},
		"the agent is gone": func(f *fixture) {
			f.service.definitions = &fakeDefinitions{}
		},
	}

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(agent.SubjectAssistantThread)
			change(f)

			f.service.FollowUp(t.Context(), serviceports.DecisionFollowUpRequest{
				TenantInfo: f.tenant,
				RunID:      f.runs.run.ID,
				ProposalID: pulid.MustNew("aprop_"),
			})

			assert.Empty(t, f.turns.started)
			assert.Empty(t, f.workflows.payloads)
		})
	}
}

// A grant through one of the owner's roles keeps the report coming.
func TestFollowUp_ReportsWhenTheOwnersRoleIsGrantedTheAgent(t *testing.T) {
	t.Parallel()

	f := newFixture(agent.SubjectAssistantThread)
	f.agent.AccessMode = agentdefinition.AccessRoles
	f.permissions.GrantedAgents = []pulid.ID{f.agent.ID}

	f.service.FollowUp(t.Context(), serviceports.DecisionFollowUpRequest{
		TenantInfo: f.tenant,
		RunID:      f.runs.run.ID,
		ProposalID: pulid.MustNew("aprop_"),
	})

	require.Len(t, f.turns.started, 1)
	assert.Len(t, f.workflows.payloads, 1)
}

