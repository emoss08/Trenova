package proposalexecutor

import (
	"context"
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	refusal agentbudgetports.BudgetRefusal
	asked   string
}

func (f *fakeBudgets) CheckRun(
	context.Context,
	*agentdefinition.Definition,
) (agentbudgetports.BudgetRefusal, error) {
	return agentbudgetports.BudgetRefusal{}, nil
}

func (f *fakeBudgets) CheckTool(
	_ context.Context,
	req agentbudgetports.CheckToolBudgetRequest,
) (agentbudgetports.BudgetRefusal, error) {
	f.asked = req.ToolName

	return f.refusal, nil
}

func (f *fakeBudgets) MoneyAllowance(
	context.Context,
	agentbudgetports.MoneyAllowanceRequest,
) (agenttooltrust.MoneyAllowance, error) {
	return agenttooltrust.MoneyAllowance{}, nil
}

func (f *fakeBudgets) Status(
	context.Context,
	*agentdefinition.Definition,
) (*agentbudgetports.AgentBudgetStatus, error) {
	return nil, nil
}

func (f *fakeBudgets) WithinBusinessHours(context.Context, *agentdefinition.Definition) bool {
	return true
}

func (f *fakeBudgets) Timezone(context.Context, *agentdefinition.Definition) string {
	return ""
}

func approver(proposal *agent.AgentProposal) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
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

	tool := &recordingTool{
		name:      "shipment_cancel",
		resource:  permission.ResourceShipment,
		operation: permission.OpCancel,
	}
	repo := &fakeProposalRepo{}
	svc := newExecutor(tool, repo, &fakePermissions{allowed: true})
	svc.definitions = fixedDefinition{
		definition: &agentdefinition.Definition{Name: "Night desk", SimulationMode: true},
	}

	proposal := testProposal(
		"shipment_cancel",
		map[string]any{"shipmentId": "shp_1", "cancelReason": "Dead load"},
	)
	require.NoError(t, svc.Execute(t.Context(), proposal, nil, approver(proposal)))

	assert.Zero(t, tool.calls, "the write must not happen")
	assert.Empty(t, repo.recorded, "no execution is recorded, only a simulation")
	require.Len(t, repo.simulated, 1)
	assert.Equal(t, proposal.ID, repo.simulated[0].ID)
	assert.Positive(t, repo.simulated[0].SimulatedAt)
	assert.Contains(t, repo.simulated[0].Simulation.Summary, "Would run shipment_cancel")
}

func TestExecute_RefusesAWritePastTheToolsDailyCap(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "dispatch_assign_move",
		resource:  permission.ResourceShipmentMove,
		operation: permission.OpUpdate,
	}
	repo := &fakeProposalRepo{}
	svc := newExecutor(tool, repo, &fakePermissions{allowed: true})
	svc.definitions = fixedDefinition{definition: &agentdefinition.Definition{Name: "Night desk"}}
	budgets := &fakeBudgets{refusal: agentbudgetports.BudgetRefusal{
		Cap: agentbudgetports.BudgetCapTool, Tool: "dispatch_assign_move", Spent: "5", Limit: "5",
	}}
	svc.budgets = budgets

	proposal := testProposal("dispatch_assign_move", map[string]any{})
	err := svc.Execute(t.Context(), proposal, nil, approver(proposal))

	require.ErrorIs(t, err, ErrBudgetSpent)
	assert.Contains(t, err.Error(), "daily limit of 5")
	assert.Equal(t, "dispatch_assign_move", budgets.asked)
	assert.Zero(t, tool.calls)
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
}

func TestExecute_RunsWhenThereIsNoCapAndNoSimulation(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "dispatch_assign_move",
		resource:  permission.ResourceShipmentMove,
		operation: permission.OpUpdate,
	}
	repo := &fakeProposalRepo{}
	svc := newExecutor(tool, repo, &fakePermissions{allowed: true})
	svc.definitions = fixedDefinition{definition: &agentdefinition.Definition{Name: "Night desk"}}
	svc.budgets = &fakeBudgets{}

	proposal := testProposal("dispatch_assign_move", map[string]any{})
	require.NoError(t, svc.Execute(t.Context(), proposal, nil, approver(proposal)))

	assert.Equal(t, 1, tool.calls)
	assert.Empty(t, repo.simulated)
}

type recordingTxConnection struct {
	opts []kernelports.TxOptions
}

func (c *recordingTxConnection) Transact(
	ctx context.Context,
	opts kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	c.opts = append(c.opts, opts)

	return fn(kernelports.WithReadOnly(ctx))
}

type snapshotPreviewTool struct {
	*recordingTool
	readOnly bool
}

func (t *snapshotPreviewTool) Preview(
	ctx context.Context,
	_ agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	t.readOnly = kernelports.IsReadOnly(ctx)

	return &agentpreview.ToolPreview{Summary: "Would cancel PRO-100."}, nil
}

// An approved write for an agent in simulation is previewed inside a
// read-only snapshot, so a preview that tried to write fails there instead
// of changing the record the approver was told stays untouched.
func TestExecute_SimulatesInsideAReadOnlySnapshot(t *testing.T) {
	t.Parallel()

	tool := &snapshotPreviewTool{recordingTool: &recordingTool{
		name:      "shipment_cancel",
		resource:  permission.ResourceShipment,
		operation: permission.OpCancel,
	}}
	repo := &fakeProposalRepo{}
	svc := newExecutor(tool, repo, &fakePermissions{allowed: true})
	svc.definitions = fixedDefinition{
		definition: &agentdefinition.Definition{Name: "Night desk", SimulationMode: true},
	}
	conn := &recordingTxConnection{}
	svc.db = conn

	proposal := testProposal("shipment_cancel", map[string]any{"shipmentId": "shp_1"})
	require.NoError(t, svc.Execute(t.Context(), proposal, nil, approver(proposal)))

	require.Len(t, conn.opts, 1)
	assert.True(t, conn.opts[0].ReadOnly)
	assert.Equal(t, sql.LevelRepeatableRead, conn.opts[0].Isolation)
	assert.True(t, tool.readOnly)
	assert.Zero(t, tool.calls)
	require.Len(t, repo.simulated, 1)
	assert.Equal(t, "Would cancel PRO-100.", repo.simulated[0].Simulation.Summary)
}
