package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingPTO struct {
	fakePTODecider

	pto     *worker.WorkerPTO
	balance decimal.Decimal
	guard   agenttoolstest.WriteGuard
}

func (f *savingPTO) approve(req *workerptoports.UpdatePTOStatusRequest) (*worker.WorkerPTO, error) {
	if !f.pto.Status.CanTransitionTo(worker.PTOStatusApproved) {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation, "PTO cannot be approved",
		)
	}
	approved := *f.pto
	approved.Status = worker.PTOStatusApproved
	approved.ApproverID = req.UserID
	approved.BalanceAfterDays = decimal.NewNullDecimal(f.balance.Sub(f.pto.Days))

	return &approved, nil
}

func (f *savingPTO) PreviewApprove(
	_ context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*workerptoports.WorkerPTOTransitionPreview, error) {
	approved, err := f.approve(req)
	if err != nil {
		return nil, err
	}
	current := *f.pto

	return &workerptoports.WorkerPTOTransitionPreview{
		Before: &current,
		After:  approved,
		Driver: &notificationports.DriverNotificationPreview{
			WorkerID:   current.WorkerID,
			WorkerName: "Ana Reyes",
			Reachable:  true,
			Title:      "Time off approved",
			Message:    "Your time off was approved.",
		},
	}, nil
}

func (f *savingPTO) Approve(
	_ context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	approved, err := f.approve(req)
	if err != nil {
		return nil, err
	}
	*f.pto = *approved

	return f.pto, nil
}

func requestedPTO() *worker.WorkerPTO {
	return &worker.WorkerPTO{
		ID:      pulid.MustNew("wpto_"),
		Status:  worker.PTOStatusRequested,
		Type:    worker.PTOType("Vacation"),
		Days:    decimal.NewFromInt(3),
		Version: 1,
	}
}

func TestApproveWorkerPTO_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	pto := requestedPTO()
	before := *pto
	fake := &savingPTO{pto: pto, balance: decimal.NewFromInt(10)}
	tool := operationstools.NewApproveWorkerPTOTool(fake).(*operationstools.ApproveWorkerPTOTool)
	params := agenttoolstest.ExecuteParams(map[string]any{"ptoId": pto.ID.String()})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&fake.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceWorkerPTO, change.Resource)
	assert.Equal(t, "Approved", agenttoolstest.FieldByPath(t, change, "status").After)
	balance := agenttoolstest.FieldByPath(t, change, "balanceAfterDays")
	assert.Equal(t, "7", balance.After)
	assert.Contains(t, preview.Summary, "booking 3 days")
	assert.Empty(t, preview.Warnings)
	require.Len(t, preview.Changes, 2)
	told := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agentpreview.PreviewOperationSend, told.Operation)
	require.NotNil(t, told.Message)
	assert.Equal(t, agentpreview.MessageChannelDash, told.Message.Channel)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(
		t,
		change,
		&before,
		pto,
		operationstools.PtoDecisionOptions()...)
}

func TestApproveWorkerPTO_PreviewWarnsForARequestAlreadyDecided(t *testing.T) {
	t.Parallel()

	pto := requestedPTO()
	pto.Status = worker.PTOStatusRejected
	fake := &savingPTO{pto: pto}
	tool := operationstools.NewApproveWorkerPTOTool(fake).(*operationstools.ApproveWorkerPTOTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&fake.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(map[string]any{"ptoId": pto.ID.String()}),
			)
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

func TestApproveWorkerPTO_PreviewWarnsWhenAnAgentWouldApprove(t *testing.T) {
	t.Parallel()

	fake := &savingPTO{pto: requestedPTO()}
	tool := operationstools.NewApproveWorkerPTOTool(fake).(*operationstools.ApproveWorkerPTOTool)
	params := agenttoolstest.ExecuteParams(map[string]any{"ptoId": fake.pto.ID.String()})
	params.Actor = agenttoolstest.AgentActorFor(params.OrganizationID, params.BusinessUnitID)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&fake.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}
