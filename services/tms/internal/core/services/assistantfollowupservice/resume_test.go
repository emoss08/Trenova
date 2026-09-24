package assistantfollowupservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type notedConversations struct {
	*fakeConversations
	notes []conversation.Message
	asked []repositories.ListMessagesRequest
}

func (f *notedConversations) ListMessages(
	_ context.Context,
	req repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	f.asked = append(f.asked, req)

	return f.notes, nil
}

type fakeProposals struct {
	proposals []*agent.AgentProposal
}

func (f *fakeProposals) ListByThread(
	_ context.Context,
	_ repositories.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	return f.proposals, nil
}

type fakePlans struct {
	plans []*agent.AgentPlan
}

func (f *fakePlans) ListByThread(
	_ context.Context,
	_ repositories.ListAgentPlansByThreadRequest,
) ([]*agent.AgentPlan, error) {
	return f.plans, nil
}

type fakeDecisions struct {
	decisions []*agent.AgentDecision
}

func (f *fakeDecisions) ListByProposals(
	_ context.Context,
	_ repositories.ListAgentDecisionsByProposalsRequest,
) ([]*agent.AgentDecision, error) {
	return f.decisions, nil
}

type resumeFixture struct {
	*fixture
	notes     *notedConversations
	proposals *fakeProposals
	plans     *fakePlans
	decisions *fakeDecisions
}

func newResumeFixture() *resumeFixture {
	f := &resumeFixture{
		fixture:   newFixture(agent.SubjectAssistantThread),
		proposals: &fakeProposals{},
		plans:     &fakePlans{},
		decisions: &fakeDecisions{},
	}
	f.notes = &notedConversations{fakeConversations: f.conversations}
	f.service.conversations = f.notes
	f.service.decided = decidedReader{
		proposals: f.proposals,
		plans:     f.plans,
		decisions: f.decisions,
	}

	return f
}

func (f *resumeFixture) decide(status agent.ProposalStatus, at int64) *agent.AgentProposal {
	proposal := &agent.AgentProposal{
		ID:       pulid.MustNew("aprop_"),
		RunID:    f.runs.run.ID,
		ToolName: "raise_exception",
		Status:   status,
	}
	f.proposals.proposals = append(f.proposals.proposals, proposal)
	f.decisions.decisions = append(f.decisions.decisions, &agent.AgentDecision{
		ID:         pulid.MustNew("adec_"),
		ProposalID: &proposal.ID,
		CreatedAt:  at,
	})

	return proposal
}

func (f *resumeFixture) resume(t *testing.T) {
	t.Helper()

	f.service.ResumeFollowUps(t.Context(), serviceports.ResumeFollowUpsRequest{
		TenantInfo: f.tenant,
		ThreadID:   f.thread.ID,
	})
}

/*
The transcript behind this: two cards approved in a row were each reported,
and a third, approved while the second's report was still being written,
produced no Decision entry at all. Its follow-up found the conversation busy
and was dropped, and the reply that was busy had read the proposal as still
waiting. The turn in the way now resumes it when it ends.
*/
func TestResumeFollowUps_ReportsADecisionMadeWhileTheConversationWasBusy(t *testing.T) {
	t.Parallel()

	f := newResumeFixture()
	proposal := f.decide(agent.ProposalStatusExecuted, timeutils.NowUnix())

	f.resume(t)

	require.Len(t, f.workflows.payloads, 1)
	payload := f.workflows.payloads[0]
	assert.Equal(t, proposal.ID, payload.Request.FollowUpProposalID)
	assert.Equal(t, f.thread.UserID, payload.Actor.UserID, "it is the owner's conversation")
	require.Len(t, f.notes.asked, 1)
	assert.Equal(
		t,
		[]conversation.MessageKind{conversation.MessageKindDecisionNote},
		f.notes.asked[0].Kinds,
		"only the notes are read, not the whole conversation",
	)
}

// A decision the conversation already carries a note for has been reported,
// and a second report would answer the same click twice.
func TestResumeFollowUps_LeavesAReportedDecisionAlone(t *testing.T) {
	t.Parallel()

	f := newResumeFixture()
	proposal := f.decide(agent.ProposalStatusRejected, timeutils.NowUnix())
	f.notes.notes = []conversation.Message{{
		Role:    conversation.RoleUser,
		Kind:    conversation.MessageKindDecisionNote,
		Content: "Rejected raise_exception.\nDecision on proposal " + proposal.ID.String() + ".",
	}}

	f.resume(t)

	assert.Empty(t, f.turns.started)
}

// Every ending a person decided is reported, a failure included; an approval
// still being carried out is reported by the decision carrying it out.
func TestResumeFollowUps_ReportsEveryEndingAPersonDecided(t *testing.T) {
	t.Parallel()

	for _, status := range []agent.ProposalStatus{
		agent.ProposalStatusExecuted,
		agent.ProposalStatusExecutionFailed,
		agent.ProposalStatusRejected,
		agent.ProposalStatusSimulated,
	} {
		f := newResumeFixture()
		f.decide(status, timeutils.NowUnix())

		f.resume(t)

		assert.Lenf(t, f.workflows.payloads, 1, "%s is reported", status)
	}

	for _, status := range []agent.ProposalStatus{
		agent.ProposalStatusPending,
		agent.ProposalStatusAccepted,
		agent.ProposalStatusModified,
		agent.ProposalStatusExpired,
	} {
		f := newResumeFixture()
		f.decide(status, timeutils.NowUnix())

		f.resume(t)

		assert.Emptyf(t, f.turns.started, "%s is not reported here", status)
	}
}

// A write that ran on its own was never decided, a plan's step is reported by
// its plan, and a decision from before the window is history, not news.
func TestResumeFollowUps_ReportsOnlyRecentDecisionsOnSingleProposals(t *testing.T) {
	t.Parallel()

	f := newResumeFixture()
	now := timeutils.NowUnix()

	f.proposals.proposals = append(f.proposals.proposals, &agent.AgentProposal{
		ID:           pulid.MustNew("aprop_"),
		RunID:        f.runs.run.ID,
		Status:       agent.ProposalStatusExecuted,
		AutonomyTier: agent.TierAutoExecute,
	})
	step := f.decide(agent.ProposalStatusExecuted, now)
	planID := pulid.MustNew("apl_")
	step.PlanID = &planID
	f.decide(agent.ProposalStatusExecuted, now-int64((resumeWindow+time.Hour)/time.Second))

	f.resume(t)

	assert.Empty(t, f.turns.started)
}

// Decisions made in a burst are reported in the order they were made, one
// turn at a time: each report resumes the next when it ends.
func TestResumeFollowUps_ReportsTheOldestDecisionFirst(t *testing.T) {
	t.Parallel()

	f := newResumeFixture()
	now := timeutils.NowUnix()
	f.decide(agent.ProposalStatusExecuted, now)
	older := now - 60
	plan := &agent.AgentPlan{
		ID:        pulid.MustNew("apl_"),
		RunID:     f.runs.run.ID,
		Status:    agent.PlanStatusFailed,
		DecidedAt: &older,
	}
	f.plans.plans = []*agent.AgentPlan{plan}

	f.resume(t)

	require.Len(t, f.workflows.payloads, 1, "one follow-up at a time")
	assert.Equal(t, plan.ID, f.workflows.payloads[0].Request.FollowUpPlanID)
	assert.True(t, f.workflows.payloads[0].Request.FollowUpProposalID.IsNil())
}

// Nothing to report reads nothing more than it has to.
func TestResumeFollowUps_DoesNothingWhenEverythingIsReported(t *testing.T) {
	t.Parallel()

	f := newResumeFixture()

	f.resume(t)

	assert.Empty(t, f.turns.started)
	assert.Empty(t, f.notes.asked, "no candidates, so the notes are not read")
}
