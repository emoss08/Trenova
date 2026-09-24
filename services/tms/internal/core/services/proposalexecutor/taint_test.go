package proposalexecutor

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type egressTool struct {
	recordingTool

	egress       agent.EgressClass
	carriesTaint bool
}

func (t *egressTool) Policy() services.ToolPolicy {
	policy := t.recordingTool.Policy()
	policy.Egress = []agent.EgressClass{t.egress}
	policy.CarriesTaint = t.carriesTaint

	return policy
}

func (t *egressTool) Execute(ctx context.Context, params services.ToolExecuteParams) error {
	return t.recordingTool.Execute(ctx, params)
}

func newEgressTool(name string, egress agent.EgressClass) *egressTool {
	return &egressTool{
		recordingTool: recordingTool{
			name:      name,
			resource:  permission.ResourceCustomerPayment,
			operation: permission.OpCreate,
		},
		egress: egress,
	}
}

func taintedProposal(tool string, orgID, buID pulid.ID) *agent.AgentProposal {
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{
		Source:   agent.TaintSourceBankReceipt,
		ToolName: "get_bank_receipt",
		CallID:   "call_1",
	})

	return &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		RunID:          pulid.MustNew("ar_"),
		ToolName:       tool,
		ToolParams:     map[string]any{"amount": "900.00"},
		Tainted:        true,
		Taint:          taint,
		EgressClass:    agent.EgressMoney,
		HeldBy:         []string{"tainted"},
	}
}

func principal(kind services.PrincipalType, orgID, buID pulid.ID) *services.RequestActor {
	actor := &services.RequestActor{
		PrincipalType:  kind,
		PrincipalID:    pulid.MustNew("prn_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
	if kind == services.PrincipalTypeUser {
		actor.UserID = actor.PrincipalID
	}

	return actor
}

func TestExecute_ATaintedWriteThatLeavesRunsOnlyOnAPersonsDecision(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	for name, tc := range map[string]struct {
		actor *services.RequestActor
		runs  bool
	}{
		"a person": {actor: principal(services.PrincipalTypeUser, orgID, buID), runs: true},
		"an agent": {actor: principal(services.PrincipalTypeAgent, orgID, buID)},
		"an API key": {
			actor: principal(services.PrincipalTypeAPIKey, orgID, buID),
		},
	} {
		tool := newEgressTool("post_customer_payment", agent.EgressMoney)
		repo := &fakeProposalRepo{}
		executor := newExecutor(tool, repo, &fakePermissions{allowed: true})

		err := executor.Execute(t.Context(), taintedProposal(tool.name, orgID, buID), nil, tc.actor)

		require.Len(t, repo.recorded, 1, name)
		assert.Equal(t, agent.EgressMoney, repo.recorded[0].egress,
			"%s: the class is recorded", name)
		if tc.runs {
			require.NoError(t, err, name)
			assert.Equal(t, 1, tool.calls, name)
			assert.Equal(t, agent.ProposalStatusExecuted, repo.recorded[0].status, name)
			continue
		}
		require.ErrorIs(t, err, ErrTaintedNeedsPerson, name)
		assert.Zero(t, tool.calls, "%s: nothing ran", name)
		assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status, name)
	}
}

func TestExecute_ATaintedInternalWriteIsNotHeld(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	tool := newEgressTool("assign_move", agent.EgressInternal)
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: true})
	proposal := taintedProposal(tool.name, orgID, buID)
	proposal.EgressClass = agent.EgressInternal

	err := executor.Execute(t.Context(), proposal, nil,
		principal(services.PrincipalTypeAgent, orgID, buID))

	require.NoError(t, err)
	assert.Equal(t, 1, tool.calls)
	assert.Equal(t, agent.EgressInternal, repo.recorded[0].egress)
}

func TestExecute_AnApproversChangeCannotTalkTheGuardOutOfTheProposedClass(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	tool := newEgressTool("email_customer", agent.EgressInternal)
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: true})
	proposal := taintedProposal(tool.name, orgID, buID)
	proposal.EgressClass = agent.EgressCustomerVisible

	err := executor.Execute(t.Context(), proposal, nil,
		principal(services.PrincipalTypeAPIKey, orgID, buID))

	require.ErrorIs(t, err, ErrTaintedNeedsPerson)
	assert.Zero(t, tool.calls)
}

func TestExecute_ACleanWriteThatLeavesMayRunUnattended(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	tool := newEgressTool("post_customer_payment", agent.EgressMoney)
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: true})
	proposal := taintedProposal(tool.name, orgID, buID)
	proposal.Tainted = false
	proposal.Taint = nil

	err := executor.Execute(t.Context(), proposal, nil,
		principal(services.PrincipalTypeAgent, orgID, buID))

	require.NoError(t, err)
	assert.Equal(t, 1, tool.calls)
}

func TestExecute_AToolThatCarriesTaintIsHandedTheProposalsTaint(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	remember := newEgressTool("remember", agent.EgressInternal)
	remember.carriesTaint = true
	executor := newExecutor(remember, &fakeProposalRepo{}, &fakePermissions{allowed: true})

	proposal := taintedProposal(remember.name, orgID, buID)
	proposal.EgressClass = agent.EgressInternal
	proposal.Taint = nil

	err := executor.Execute(t.Context(), proposal, nil,
		principal(services.PrincipalTypeUser, orgID, buID))

	require.NoError(t, err)
	require.True(t, remember.lastParams.Taint.Tainted(),
		"a tainted proposal whose marks were not kept still carries its run")
	mark := remember.lastParams.Taint.Marks[0]
	assert.Equal(t, agent.TaintSourceRunRecord, mark.Source)
	assert.Equal(t, proposal.RunID.String(), mark.Ref.ID)
}

func TestExecute_ToolsThatDoNotCarryTaintAreNotHandedIt(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	tool := newEgressTool("assign_move", agent.EgressInternal)
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	proposal := taintedProposal(tool.name, orgID, buID)
	proposal.EgressClass = agent.EgressInternal

	require.NoError(t, executor.Execute(t.Context(), proposal, nil,
		principal(services.PrincipalTypeUser, orgID, buID)))
	assert.Nil(t, tool.lastParams.Taint)
}
