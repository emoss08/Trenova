package decisionbatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchDecisionRequest_CarriesTheNoteToTheService(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ap_")
	note := "These customers are on credit hold."
	reason := "rejected_in_conversation"

	req, err := BatchDecisionRequest([]string{id.String()}, &gqlmodel.DecideAgentProposalsInput{
		Decision:   agentdecision.DecisionRejected,
		ReasonCode: &reason,
		Note:       &note,
	}, pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, note, req.Note)
	assert.Equal(t, reason, req.ReasonCode)
	assert.Equal(t, []pulid.ID{id}, req.ProposalIDs)
}

func TestBatchDecisionRequest_LeavesTheNoteEmptyWhenAbsent(t *testing.T) {
	t.Parallel()

	req, err := BatchDecisionRequest(
		[]string{pulid.MustNew("ap_").String()},
		&gqlmodel.DecideAgentProposalsInput{Decision: agentdecision.DecisionAccepted},
		pagination.TenantInfo{},
	)
	require.NoError(t, err)

	assert.Empty(t, req.Note)
}

func TestDecisionResult_SaysAnApprovalWasRefusedAndWhy(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ap_")
	decision := &agentdecision.AgentDecision{Decision: agentdecision.DecisionAccepted}
	why := "the record changed since this was proposed: the shipment S-1 was at version 3 " +
		"when this was proposed and is at version 5 now, so nothing was changed"

	got := DecisionResult(&agentdecisionports.AgentProposalDecisionResult{
		ProposalID: id,
		Decision:   decision,
		Error:      why,
		Refused:    true,
	})

	assert.Equal(t, &gqlmodel.AgentProposalDecisionResult{
		ProposalID: id.String(),
		Decision:   decision,
		Error:      &why,
		Executed:   false,
		Refused:    true,
	}, got)
}

func TestDecisionResult_LeavesTheErrorAbsentWhenTheWriteRan(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ap_")
	got := DecisionResult(&agentdecisionports.AgentProposalDecisionResult{
		ProposalID: id,
		Decision:   &agentdecision.AgentDecision{Decision: agentdecision.DecisionAccepted},
		Executed:   true,
	})

	assert.Nil(t, got.Error)
	assert.True(t, got.Executed)
	assert.False(t, got.Refused)
}

func TestPlanDecisionResult_RefusedOnlyForAPlanWhoseRecordMovedOn(t *testing.T) {
	t.Parallel()

	refused := &agentplan.AgentPlan{
		Status:       agentplan.PlanStatusFailed,
		FailureError: agentplan.ErrPlanTargetChanged.Error() + "; step 1: the shipment S-1 moved on",
	}
	failed := &agentplan.AgentPlan{
		Status:       agentplan.PlanStatusFailed,
		FailureError: "1 of 2 steps did not run; step 2: the customer is on credit hold",
	}
	waiting := &agentplan.AgentPlan{Status: agentplan.PlanStatusApproving}

	assert.Equal(t, &gqlmodel.AgentPlanDecisionResult{Plan: refused, Refused: true},
		PlanDecisionResult(refused))
	assert.Equal(t, &gqlmodel.AgentPlanDecisionResult{Plan: failed, Refused: false},
		PlanDecisionResult(failed))
	assert.Equal(t, &gqlmodel.AgentPlanDecisionResult{Plan: waiting, Refused: false},
		PlanDecisionResult(waiting))
}
