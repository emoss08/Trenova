package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type decisionFixture struct {
	recorder  *artifactrecorder.Recorder
	artifacts *assistanttest.StubArtifactRepo
	proposals *assistanttest.StubProposalRepo
	thread    *conversation.Thread
	tenant    pagination.TenantInfo
}

func newDecisionFixture(t *testing.T, stored ...*agent.AgentProposal) decisionFixture {
	t.Helper()

	artifacts := &assistanttest.StubArtifactRepo{}
	proposals := &assistanttest.StubProposalRepo{ByThread: stored}
	svc := &Service{logger: zap.NewNop(), artifacts: artifacts, proposals: proposals}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return decisionFixture{
		recorder:  svc.newArtifactRecorder(t.Context(), thread, tenant, assistanttest.Actor(), nil),
		artifacts: artifacts,
		proposals: proposals,
		thread:    thread,
		tenant:    tenant,
	}
}

func requested(id pulid.ID) agentruntimeports.ToolObservation {
	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: "call_decide", Name: "request_decision"},
		Data: agentruntimeports.DecisionRequest{ProposalID: id},
	}
}

func waitingProposal(status agent.ProposalStatus) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:        pulid.MustNew("aprop_"),
		RunID:     pulid.MustNew("arun_"),
		ToolName:  "shipment_create",
		Rationale: "Copy PRO-100 for Tuesday.",
		Status:    status,
	}
}

func TestRequestDecision_PutsTheWaitingCardBackAsAnArtifact(t *testing.T) {
	t.Parallel()

	proposal := waitingProposal(agent.ProposalStatusPending)
	fixture := newDecisionFixture(t, proposal)

	shown, err := fixture.recorder.Observe(requested(proposal.ID))
	require.NoError(t, err)
	require.NotNil(t, shown)

	assert.Equal(t, agentports.ListAgentProposalsByThreadRequest{
		ThreadID:   fixture.thread.ID,
		TenantInfo: fixture.tenant,
	}, fixture.proposals.LastList, "only this conversation's proposals, in this tenant")

	require.Len(t, fixture.artifacts.Upserts, 1)
	kept := fixture.artifacts.Upserts[0]
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

	_, err := fixture.recorder.Observe(requested(proposal.ID))
	require.NoError(t, err)

	require.Len(t, fixture.artifacts.Upserts, 1)
	assert.Equal(t, planID, fixture.artifacts.Upserts[0].PlanID)
}

func TestRequestDecision_RefusesAProposalFromAnotherConversation(t *testing.T) {
	t.Parallel()

	fixture := newDecisionFixture(t, waitingProposal(agent.ProposalStatusPending))

	shown, err := fixture.recorder.Observe(requested(pulid.MustNew("aprop_")))
	require.ErrorIs(t, err, toolartifacts.ErrUnknownProposal)
	assert.Nil(t, shown)
	assert.Empty(t, fixture.artifacts.Upserts)
}

func TestRequestDecision_RefusesAProposalAlreadyDecided(t *testing.T) {
	t.Parallel()

	proposal := waitingProposal(agent.ProposalStatusRejected)
	fixture := newDecisionFixture(t, proposal)

	_, err := fixture.recorder.Observe(requested(proposal.ID))
	require.ErrorIs(t, err, toolartifacts.ErrProposalDecided)
	assert.Contains(t, err.Error(), "rejected")
	assert.Empty(t, fixture.artifacts.Upserts)
}

func TestRequestDecision_SaysWhenTheProposalsCannotBeRead(t *testing.T) {
	t.Parallel()

	proposal := waitingProposal(agent.ProposalStatusPending)
	fixture := newDecisionFixture(t, proposal)
	fixture.proposals.ListErr = assert.AnError

	_, err := fixture.recorder.Observe(requested(proposal.ID))
	require.ErrorIs(t, err, toolartifacts.ErrDecisionUnavailable)
	assert.Empty(t, fixture.artifacts.Upserts)
}

func requestedMany(request agentruntimeports.DecisionRequest) agentruntimeports.ToolObservation {
	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: "call_decide", Name: "request_decision"},
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

	shown, err := fixture.recorder.Observe(requestedMany(agentruntimeports.DecisionRequest{
		PlanID: planID,
	}))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, fixture.artifacts.Upserts, 1)
	kept := fixture.artifacts.Upserts[0]
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

	_, err := fixture.recorder.Observe(requestedMany(agentruntimeports.DecisionRequest{
		ProposalID:  first.ID,
		ProposalIDs: []pulid.ID{first.ID, second.ID},
	}))
	require.NoError(t, err)

	require.Len(t, fixture.artifacts.Upserts, 1)
	kept := fixture.artifacts.Upserts[0]
	assert.Equal(t, first.ID, kept.ProposalID)
	assert.Equal(t, []string{first.ID.String(), second.ID.String()}, kept.Payload["proposalIds"])
	assert.Equal(t, "Create shipment (2)", kept.Title)
}

func TestRequestDecision_RefusesABunchThatCannotShareACard(t *testing.T) {
	t.Parallel()

	first, other := waitingProposal(agent.ProposalStatusPending),
		waitingProposal(agent.ProposalStatusPending)
	other.ToolName = "invoice_post"
	decided := waitingProposal(agent.ProposalStatusExecuted)
	fixture := newDecisionFixture(t, first, other, decided)

	_, err := fixture.recorder.Observe(requestedMany(agentruntimeports.DecisionRequest{
		ProposalID: first.ID, ProposalIDs: []pulid.ID{first.ID, other.ID},
	}))
	require.ErrorIs(t, err, toolartifacts.ErrMixedTools)

	_, err = fixture.recorder.Observe(requestedMany(agentruntimeports.DecisionRequest{
		ProposalID: first.ID, ProposalIDs: []pulid.ID{first.ID, decided.ID},
	}))
	require.ErrorIs(t, err, toolartifacts.ErrProposalDecided)

	_, err = fixture.recorder.Observe(requestedMany(agentruntimeports.DecisionRequest{
		PlanID: pulid.MustNew("apl_"),
	}))
	require.ErrorIs(t, err, toolartifacts.ErrUnknownPlan)
	assert.Empty(t, fixture.artifacts.Upserts)
}
