package proposalexecutor

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedDefinition struct{ definition *agentdefinition.Definition }

func (f fixedDefinition) ForRun(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*agentdefinition.Definition, error) {
	return f.definition, nil
}

type fakeBudgets struct {
	refusal services.BudgetRefusal
	asked   string
}

func (f *fakeBudgets) CheckRun(
	context.Context,
	*agentdefinition.Definition,
) (services.BudgetRefusal, error) {
	return services.BudgetRefusal{}, nil
}

func (f *fakeBudgets) CheckTool(
	_ context.Context,
	_ *agentdefinition.Definition,
	tool string,
) (services.BudgetRefusal, error) {
	f.asked = tool

	return f.refusal, nil
}

func (f *fakeBudgets) Status(
	context.Context,
	*agentdefinition.Definition,
) (*services.AgentBudgetStatus, error) {
	return nil, nil
}

func approver(proposal *agent.AgentProposal) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
	}
}

// An agent in simulation gets a preview in place of the write. The approval
// is real and recorded; the tool never runs.
func TestExecute_SimulatesInsteadOfRunningForAnAgentInSimulation(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{name: "cancel_shipment", resource: permission.ResourceShipment, operation: permission.OpCancel}
	repo := &fakeProposalRepo{}
	svc := newExecutor(tool, repo, &fakePermissions{allowed: true})
	svc.definitions = fixedDefinition{definition: &agentdefinition.Definition{Name: "Night desk", SimulationMode: true}}

	proposal := testProposal("cancel_shipment", map[string]any{"shipmentId": "shp_1", "cancelReason": "Dead load"})
	require.NoError(t, svc.Execute(t.Context(), proposal, nil, approver(proposal)))

	assert.Zero(t, tool.calls, "the write must not happen")
	assert.Empty(t, repo.recorded, "no execution is recorded, only a simulation")
	require.Len(t, repo.simulated, 1)
	assert.Equal(t, proposal.ID, repo.simulated[0].ID)
	assert.Positive(t, repo.simulated[0].SimulatedAt)
	assert.Contains(t, repo.simulated[0].Simulation.Summary, "Would run cancel_shipment")
}

func TestExecute_RefusesAWritePastTheToolsDailyCap(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{name: "assign_move", resource: permission.ResourceShipmentMove, operation: permission.OpUpdate}
	repo := &fakeProposalRepo{}
	svc := newExecutor(tool, repo, &fakePermissions{allowed: true})
	svc.definitions = fixedDefinition{definition: &agentdefinition.Definition{Name: "Night desk"}}
	budgets := &fakeBudgets{refusal: services.BudgetRefusal{
		Cap: services.BudgetCapTool, Tool: "assign_move", Spent: "5", Limit: "5",
	}}
	svc.budgets = budgets

	proposal := testProposal("assign_move", map[string]any{})
	err := svc.Execute(t.Context(), proposal, nil, approver(proposal))

	require.ErrorIs(t, err, ErrBudgetSpent)
	assert.Contains(t, err.Error(), "daily limit of 5")
	assert.Equal(t, "assign_move", budgets.asked)
	assert.Zero(t, tool.calls)
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
}

func TestExecute_RunsWhenThereIsNoCapAndNoSimulation(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{name: "assign_move", resource: permission.ResourceShipmentMove, operation: permission.OpUpdate}
	repo := &fakeProposalRepo{}
	svc := newExecutor(tool, repo, &fakePermissions{allowed: true})
	svc.definitions = fixedDefinition{definition: &agentdefinition.Definition{Name: "Night desk"}}
	svc.budgets = &fakeBudgets{}

	proposal := testProposal("assign_move", map[string]any{})
	require.NoError(t, svc.Execute(t.Context(), proposal, nil, approver(proposal)))

	assert.Equal(t, 1, tool.calls)
	assert.Empty(t, repo.simulated)
}
