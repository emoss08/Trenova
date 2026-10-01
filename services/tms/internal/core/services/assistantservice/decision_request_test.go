package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type decisionFixture struct {
	recorder  *artifactRecorder
	artifacts *stubArtifactRepo
	proposals *stubProposalRepo
	thread    *conversation.Thread
	tenant    pagination.TenantInfo
}

func newDecisionFixture(t *testing.T, stored ...*agent.AgentProposal) decisionFixture {
	t.Helper()

	artifacts := &stubArtifactRepo{}
	proposals := &stubProposalRepo{byThread: stored}
	svc := &Service{logger: zap.NewNop(), artifacts: artifacts, proposals: proposals}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return decisionFixture{
		recorder:  svc.newArtifactRecorder(t.Context(), thread, tenant, testActor(), nil),
		artifacts: artifacts,
		proposals: proposals,
		thread:    thread,
		tenant:    tenant,
	}
}

func requested(id pulid.ID) serviceports.ToolObservation {
	return serviceports.ToolObservation{
		Call: serviceports.ToolCall{ID: "call_decide", Name: "request_decision"},
		Data: serviceports.DecisionRequest{ProposalID: id},
	}
}

func waitingProposal(status agent.ProposalStatus) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:        pulid.MustNew("aprop_"),
		RunID:     pulid.MustNew("arun_"),
		ToolName:  "create_shipment",
		Rationale: "Copy PRO-100 for Tuesday.",
		Status:    status,
	}
}

func TestRequestDecision_PutsTheWaitingCardBackAsAnArtifact(t *testing.T) {
	t.Parallel()

	proposal := waitingProposal(agent.ProposalStatusPending)
	fixture := newDecisionFixture(t, proposal)

	shown, err := fixture.recorder.observe(requested(proposal.ID))
	require.NoError(t, err)
	require.NotNil(t, shown)

	assert.Equal(t, repositories.ListAgentProposalsByThreadRequest{
		ThreadID:   fixture.thread.ID,
		TenantInfo: fixture.tenant,
	}, fixture.proposals.lastList, "only this conversation's proposals, in this tenant")

	require.Len(t, fixture.artifacts.upserts, 1)
	kept := fixture.artifacts.upserts[0]
	assert.Equal(t, assistantartifact.KindDecisionRequest, kept.Kind)
	assert.Equal(t, assistantartifact.StatusPending, kept.Status)
	assert.Equal(t, proposal.ID, kept.ProposalID)
	assert.Equal(t, proposal.RunID, kept.RunID)
	assert.Equal(t, "call_decide", kept.SourceToolCallID)
	assert.Equal(t, "Create shipment", kept.Title)
	assert.Equal(t, proposal.ID.String(), kept.Payload["proposalId"])
	assert.Equal(t, "decision_request", shown.Kind)
}

func TestRequestDecision_CarriesThePlanAStepBelongsTo(t *testing.T) {
	t.Parallel()

	proposal := waitingProposal(agent.ProposalStatusPending)
	planID := pulid.MustNew("aplan_")
	proposal.PlanID = &planID
	fixture := newDecisionFixture(t, proposal)

	_, err := fixture.recorder.observe(requested(proposal.ID))
	require.NoError(t, err)

	require.Len(t, fixture.artifacts.upserts, 1)
	assert.Equal(t, planID, fixture.artifacts.upserts[0].PlanID)
}

func TestRequestDecision_RefusesAProposalFromAnotherConversation(t *testing.T) {
	t.Parallel()

	fixture := newDecisionFixture(t, waitingProposal(agent.ProposalStatusPending))

	shown, err := fixture.recorder.observe(requested(pulid.MustNew("aprop_")))
	require.ErrorIs(t, err, errUnknownProposal)
	assert.Nil(t, shown)
	assert.Empty(t, fixture.artifacts.upserts)
}

func TestRequestDecision_RefusesAProposalAlreadyDecided(t *testing.T) {
	t.Parallel()

	proposal := waitingProposal(agent.ProposalStatusRejected)
	fixture := newDecisionFixture(t, proposal)

	_, err := fixture.recorder.observe(requested(proposal.ID))
	require.ErrorIs(t, err, errProposalDecided)
	assert.Contains(t, err.Error(), "rejected")
	assert.Empty(t, fixture.artifacts.upserts)
}

func TestRequestDecision_SaysWhenTheProposalsCannotBeRead(t *testing.T) {
	t.Parallel()

	proposal := waitingProposal(agent.ProposalStatusPending)
	fixture := newDecisionFixture(t, proposal)
	fixture.proposals.listErr = assert.AnError

	_, err := fixture.recorder.observe(requested(proposal.ID))
	require.ErrorIs(t, err, errDecisionUnavailable)
	assert.Empty(t, fixture.artifacts.upserts)
}

func TestDecisionStatus_FollowsTheProposal(t *testing.T) {
	t.Parallel()

	assert.Equal(t, assistantartifact.StatusPending, decisionStatus(agent.ProposalStatusPending))
	assert.Equal(t, assistantartifact.StatusReady, decisionStatus(agent.ProposalStatusExecuted))
	assert.Equal(t, assistantartifact.StatusFailed, decisionStatus(agent.ProposalStatusRejected))
}

func requestedMany(request serviceports.DecisionRequest) serviceports.ToolObservation {
	return serviceports.ToolObservation{
		Call: serviceports.ToolCall{ID: "call_decide", Name: "request_decision"},
		Data: request,
	}
}

func TestRequestDecision_KeepsOneCardForAWaitingPlan(t *testing.T) {
	t.Parallel()

	planID := pulid.MustNew("apl_")
	first, second := waitingProposal(agent.ProposalStatusPending),
		waitingProposal(agent.ProposalStatusPending)
	first.PlanID, second.PlanID = &planID, &planID
	first.PlanStep, second.PlanStep = 2, 1
	fixture := newDecisionFixture(t, first, second)

	shown, err := fixture.recorder.observe(requestedMany(serviceports.DecisionRequest{
		PlanID: planID,
	}))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, fixture.artifacts.upserts, 1)
	kept := fixture.artifacts.upserts[0]
	assert.Equal(t, assistantartifact.KindDecisionRequest, kept.Kind)
	assert.Equal(t, planID, kept.PlanID)
	assert.Equal(t, second.ID, kept.ProposalID, "the card is anchored on the plan's first step")
	assert.Equal(t, planID.String(), kept.Payload["planId"])
	assert.Equal(t, []string{second.ID.String(), first.ID.String()}, kept.Payload["proposalIds"])
	assert.Equal(t, "Plan: 2 changes", kept.Title)
}

func TestRequestDecision_KeepsOneCardForSeveralProposalsOfOneTool(t *testing.T) {
	t.Parallel()

	first, second := waitingProposal(agent.ProposalStatusPending),
		waitingProposal(agent.ProposalStatusPending)
	fixture := newDecisionFixture(t, first, second)

	_, err := fixture.recorder.observe(requestedMany(serviceports.DecisionRequest{
		ProposalID:  first.ID,
		ProposalIDs: []pulid.ID{first.ID, second.ID},
	}))
	require.NoError(t, err)

	require.Len(t, fixture.artifacts.upserts, 1)
	kept := fixture.artifacts.upserts[0]
	assert.Equal(t, first.ID, kept.ProposalID)
	assert.Equal(t, []string{first.ID.String(), second.ID.String()}, kept.Payload["proposalIds"])
	assert.Equal(t, "Create shipment (2)", kept.Title)
}

func TestRequestDecision_RefusesABunchThatCannotShareACard(t *testing.T) {
	t.Parallel()

	first, other := waitingProposal(agent.ProposalStatusPending),
		waitingProposal(agent.ProposalStatusPending)
	other.ToolName = "post_invoice"
	decided := waitingProposal(agent.ProposalStatusExecuted)
	fixture := newDecisionFixture(t, first, other, decided)

	_, err := fixture.recorder.observe(requestedMany(serviceports.DecisionRequest{
		ProposalID: first.ID, ProposalIDs: []pulid.ID{first.ID, other.ID},
	}))
	require.ErrorIs(t, err, errMixedTools)

	_, err = fixture.recorder.observe(requestedMany(serviceports.DecisionRequest{
		ProposalID: first.ID, ProposalIDs: []pulid.ID{first.ID, decided.ID},
	}))
	require.ErrorIs(t, err, errProposalDecided)

	_, err = fixture.recorder.observe(requestedMany(serviceports.DecisionRequest{
		PlanID: pulid.MustNew("apl_"),
	}))
	require.ErrorIs(t, err, errUnknownPlan)
	assert.Empty(t, fixture.artifacts.upserts)
}
