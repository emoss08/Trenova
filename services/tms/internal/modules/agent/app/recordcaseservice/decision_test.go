package recordcaseservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanDecision_ReadsWhereThePlanStands(t *testing.T) {
	commits := int64(1_800_000_010)
	decider := pulid.ID("usr_1")
	failed := 1
	cases := map[agentplan.PlanStatus]recordcase.DecisionPhase{
		agentplan.PlanStatusPending:   recordcase.PhaseAwaiting,
		agentplan.PlanStatusApproving: recordcase.PhaseUndoable,
		agentplan.PlanStatusApproved:  recordcase.PhaseRunning,
		agentplan.PlanStatusCompleted: recordcase.PhaseDone,
		agentplan.PlanStatusFailed:    recordcase.PhaseFailed,
		agentplan.PlanStatusRejected:  recordcase.PhaseDeclined,
	}

	for status, phase := range cases {
		t.Run(string(status), func(t *testing.T) {
			decision := planDecision(&agentplan.AgentPlan{
				ID:              "apl_1",
				Title:           "Recover SHP-1",
				Status:          status,
				StepCount:       3,
				CompletedSteps:  1,
				FailedStep:      &failed,
				FailureError:    "No hours left",
				CommitsAt:       &commits,
				DecidedByUserID: &decider,
				ApprovalNote:    "Not today",
				Run:             &agent.AgentRun{AgentDefinitionID: "agd_1"},
			})

			require.NotNil(t, decision)
			assert.Equal(t, recordcase.DecisionPlan, decision.Kind)
			assert.Equal(t, phase, decision.Phase)
			assert.Equal(t, pulid.ID("agd_1"), decision.AgentDefinitionID)
			assert.Equal(t, decider, decision.DecidedByUserID)
			assert.Equal(t, "Not today", decision.Note)
			assert.Equal(t, 3, decision.StepCount)
			assert.Equal(t, phase == recordcase.PhaseUndoable, decision.CommitsAt != nil,
				"only an approval in its undo window says when it starts")
		})
	}
}

func TestPlanDecision_LeavesAnExpiredPlanOut(t *testing.T) {
	assert.Nil(t, planDecision(&agentplan.AgentPlan{Status: agentplan.PlanStatusExpired}))
}

func TestProposalDecision_TakesItsNoteAndUndoWindowFromTheLatestDecision(t *testing.T) {
	commits := int64(1_800_000_010)
	found := &recordcaseports.Decisions{
		Decided: map[pulid.ID]*agentdecision.AgentDecision{
			"ap_1": {
				DecidedByUserID: "usr_1",
				Note:            "Approved from the case",
				CommitsAt:       &commits,
				CreatedAt:       1_800_000_000,
			},
		},
	}

	decision := proposalDecision(&agent.AgentProposal{
		ID:       "ap_1",
		ToolName: "shipment_reschedule_stop",
		Status:   agent.ProposalStatusApproving,
	}, found)

	require.NotNil(t, decision)
	assert.Equal(t, recordcase.DecisionProposal, decision.Kind)
	assert.Equal(t, recordcase.PhaseUndoable, decision.Phase)
	assert.Equal(t, 1, decision.StepCount)
	require.NotNil(t, decision.CommitsAt)
	assert.Equal(t, commits, *decision.CommitsAt)
	assert.Equal(t, "Approved from the case", decision.Note)
	assert.Equal(t, pulid.ID("usr_1"), decision.DecidedByUserID)
}

func TestProposalDecision_CountsItsOneStepAsRunOrFailed(t *testing.T) {
	none := &recordcaseports.Decisions{Decided: map[pulid.ID]*agentdecision.AgentDecision{}}

	ran := proposalDecision(&agent.AgentProposal{Status: agent.ProposalStatusExecuted}, none)
	failed := proposalDecision(&agent.AgentProposal{
		Status:         agent.ProposalStatusExecutionFailed,
		ExecutionError: "The stop is already complete",
	}, none)

	require.NotNil(t, ran)
	assert.Equal(t, 1, ran.CompletedSteps)
	assert.Nil(t, ran.FailedStep)
	require.NotNil(t, failed)
	require.NotNil(t, failed.FailedStep)
	assert.Equal(t, 1, *failed.FailedStep)
	assert.Equal(t, "The stop is already complete", failed.FailureError)
}

func TestProposalDecision_LeavesOutWhatNoLongerStands(t *testing.T) {
	none := &recordcaseports.Decisions{}
	for _, status := range []agent.ProposalStatus{
		agent.ProposalStatusExpired,
		agent.ProposalStatusSuperseded,
		agent.ProposalStatusSkipped,
		agent.ProposalStatusSimulated,
	} {
		assert.Nil(t, proposalDecision(&agent.AgentProposal{Status: status}, none), string(status))
	}
}

func TestPhaseRank_PutsWhatIsInMotionBeforeWhatWaitsBeforeWhatIsSettled(t *testing.T) {
	assert.Less(t, phaseRank(recordcase.PhaseUndoable), phaseRank(recordcase.PhaseAwaiting))
	assert.Less(t, phaseRank(recordcase.PhaseRunning), phaseRank(recordcase.PhaseAwaiting))
	assert.Less(t, phaseRank(recordcase.PhaseAwaiting), phaseRank(recordcase.PhaseFailed))
	assert.Less(t, phaseRank(recordcase.PhaseFailed), phaseRank(recordcase.PhaseDone))
	assert.Equal(t, phaseRank(recordcase.PhaseDone), phaseRank(recordcase.PhaseDeclined))
}

func TestOtherPending_CountsDistinctWaitingDecisionsOnce(t *testing.T) {
	rest := []*recordcase.Decision{
		{Title: "Evaluate service failures", Phase: recordcase.PhaseAwaiting},
		{Title: "Evaluate service failures", Phase: recordcase.PhaseAwaiting},
		{Title: "Email the customer", Phase: recordcase.PhaseAwaiting},
		{Title: "Place a hold", Phase: recordcase.PhaseDone},
		{Title: "Recover SHP-1", Phase: recordcase.PhaseAwaiting},
	}

	assert.Equal(t, 2, otherPending(rest, "Recover SHP-1"))
	assert.Equal(t, 0, otherPending(nil, "Recover SHP-1"))
}

func TestItemFor_FindsTheNeedsYouItemBehindADecision(t *testing.T) {
	items := []*watchtower.Item{
		{ID: "wt_1", SourceKind: watchtower.SourceEtaAtRisk, SourceID: "apl_1"},
		{ID: "wt_2", SourceKind: watchtower.SourceAgentPlan, SourceID: "apl_1"},
	}

	assert.Equal(t, pulid.ID("wt_2"), itemFor(items, "apl_1"))
	assert.True(t, itemFor(items, "apl_9").IsNil())
}

func TestRank_WhatWaitsLeadsAPlanThatJustRanAndFollowsOneThatJustStopped(t *testing.T) {
	t.Parallel()

	now := int64(10_000)
	recent, old := now-60, now-justDecidedSeconds-1
	justRan := &recordcase.Decision{Phase: recordcase.PhaseDone, DecidedAt: &recent}
	justStopped := &recordcase.Decision{Phase: recordcase.PhaseFailed, DecidedAt: &recent}
	stoppedEarlier := &recordcase.Decision{Phase: recordcase.PhaseFailed, DecidedAt: &old}
	ranEarlier := &recordcase.Decision{Phase: recordcase.PhaseDone, DecidedAt: &old}
	waiting := &recordcase.Decision{Phase: recordcase.PhaseAwaiting}
	running := &recordcase.Decision{Phase: recordcase.PhaseRunning, DecidedAt: &recent}

	assert.Less(t, rank(running, now), rank(justStopped, now), "what is in motion still leads")
	assert.Less(t, rank(justStopped, now), rank(waiting, now))
	assert.Less(t, rank(waiting, now), rank(justRan, now),
		"a record that needs approval shows the thing to approve")
	assert.Less(t, rank(justRan, now), rank(stoppedEarlier, now))
	assert.Less(t, rank(stoppedEarlier, now), rank(ranEarlier, now))
}

func TestOfferedBy_OptionsAreShownOnlyWithTheirOwnAgentsDecision(t *testing.T) {
	t.Parallel()

	billing, monitor := pulid.MustNew("agdef_"), pulid.MustNew("agdef_")
	options := []*recordcase.Option{{AgentDefinitionID: billing}, {AgentDefinitionID: billing}}

	assert.True(t, offeredBy(options, billing))
	assert.False(t, offeredBy(options, monitor))
	assert.True(t, offeredBy(nil, monitor))
	assert.True(t, offeredBy(options, pulid.Nil), "a decision with no agent on it hides nothing")
}
