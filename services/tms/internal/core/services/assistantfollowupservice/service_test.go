package assistantfollowupservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type fakeAssistant struct {
	serviceports.AssistantService
	requests []*serviceports.SendMessageRequest
	actors   []*serviceports.RequestActor
}

func (f *fakeAssistant) SendMessageStream(
	_ context.Context,
	req *serviceports.SendMessageRequest,
	actor *serviceports.RequestActor,
	_ serviceports.AssistantStreamEmitter,
) (*serviceports.SendMessageResult, error) {
	f.requests = append(f.requests, req)
	f.actors = append(f.actors, actor)

	return &serviceports.SendMessageResult{}, nil
}

type fakeTurns struct {
	startErr  error
	started   []assistantturnservice.StartRequest
	completed []conversation.AssistantTurnStatus
}

func (f *fakeTurns) Start(
	_ context.Context,
	req assistantturnservice.StartRequest,
) (*conversation.AssistantTurn, error) {
	f.started = append(f.started, req)
	if f.startErr != nil {
		return nil, f.startErr
	}

	return &conversation.AssistantTurn{ID: pulid.MustNew("atrn_"), ThreadID: req.ThreadID}, nil
}

func (f *fakeTurns) StartDurable(
	context.Context,
	assistantturnservice.StartRequest,
	func(turn *conversation.AssistantTurn) (string, error),
) (*conversation.AssistantTurn, error) {
	return nil, errors.New("not durable in these tests")
}

func (f *fakeTurns) Observe(
	_ context.Context,
	_ *conversation.AssistantTurn,
	_ serviceports.AssistantStreamEmitter,
) (serviceports.AssistantStreamEmitter, func(serviceports.StreamEvent)) {
	return func(serviceports.StreamEvent) {}, func(serviceports.StreamEvent) {}
}

func (f *fakeTurns) Stoppable(
	ctx context.Context,
	_ *conversation.AssistantTurn,
) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

func (f *fakeTurns) Complete(
	_ context.Context,
	_ *conversation.AssistantTurn,
	status conversation.AssistantTurnStatus,
	_ error,
) {
	f.completed = append(f.completed, status)
}

type fixture struct {
	service       *Service
	runs          *fakeRuns
	conversations *fakeConversations
	assistant     *fakeAssistant
	turns         *fakeTurns
	tenant        pagination.TenantInfo
	thread        *conversation.Thread
}

func newFixture(subject agent.SubjectType) *fixture {
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	thread := &conversation.Thread{
		ID:             pulid.MustNew("athr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
	f := &fixture{
		runs: &fakeRuns{run: &agent.AgentRun{
			ID:          pulid.MustNew("arun_"),
			SubjectType: subject,
			SubjectID:   thread.ID,
		}},
		conversations: &fakeConversations{thread: thread},
		assistant:     &fakeAssistant{},
		turns:         &fakeTurns{},
		tenant:        tenant,
		thread:        thread,
	}
	f.service = &Service{
		l:             zap.NewNop(),
		runs:          f.runs,
		conversations: f.conversations,
		assistant:     f.assistant,
		turns:         f.turns,
		inProcess:     func(run func()) { run() },
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

	require.Len(t, f.assistant.requests, 1)
	assert.Equal(t, proposalID, f.assistant.requests[0].FollowUpProposalID)
	assert.True(t, f.assistant.requests[0].FollowUpPlanID.IsNil())
	assert.Empty(t, f.assistant.requests[0].Content)
	assert.Equal(t, f.thread.UserID, f.assistant.actors[0].UserID)
	assert.Equal(t, serviceports.PrincipalTypeUser, f.assistant.actors[0].PrincipalType)

	assert.Equal(t, []conversation.AssistantTurnStatus{conversation.AssistantTurnStatusCompleted},
		f.turns.completed, "the turn is closed, or it holds the conversation's live slot")
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

	require.Len(t, f.assistant.requests, 1)
	assert.Equal(t, planID, f.assistant.requests[0].FollowUpPlanID)
	assert.True(t, f.assistant.requests[0].FollowUpProposalID.IsNil())
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
	assert.Empty(t, f.assistant.requests)
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
	assert.Empty(t, f.assistant.requests)
	assert.Empty(t, f.turns.completed)
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
