package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
	guard   writeGuard
}

func (f *savingPTO) approve(req *repositories.UpdatePTOStatusRequest) (*worker.WorkerPTO, error) {
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
	req *repositories.UpdatePTOStatusRequest,
) (*serviceports.WorkerPTOTransitionPreview, error) {
	approved, err := f.approve(req)
	if err != nil {
		return nil, err
	}
	current := *f.pto

	return &serviceports.WorkerPTOTransitionPreview{
		Before: &current,
		After:  approved,
		Driver: &serviceports.DriverNotificationPreview{
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
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	if err := f.guard.write(); err != nil {
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
	tool := newApproveWorkerPTOTool(fake).(*approveWorkerPTOTool)
	params := executeParams(map[string]any{"ptoId": pto.ID.String()})

	preview := previewWithoutWrites(t, &fake.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceWorkerPTO, change.Resource)
	assert.Equal(t, "Approved", fieldByPath(t, change, "status").After)
	balance := fieldByPath(t, change, "balanceAfterDays")
	assert.Equal(t, "7", balance.After)
	assert.Contains(t, preview.Summary, "booking 3 days")
	assert.Empty(t, preview.Warnings)
	require.Len(t, preview.Changes, 2)
	told := previewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationSend, told.Operation)
	require.NotNil(t, told.Message)
	assert.Equal(t, agent.MessageChannelDash, told.Message.Channel)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, pto, ptoDecisionOptions()...)
}

func TestApproveWorkerPTO_PreviewWarnsForARequestAlreadyDecided(t *testing.T) {
	t.Parallel()

	pto := requestedPTO()
	pto.Status = worker.PTOStatusRejected
	fake := &savingPTO{pto: pto}
	tool := newApproveWorkerPTOTool(fake).(*approveWorkerPTOTool)

	preview := previewWithoutWrites(t, &fake.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{"ptoId": pto.ID.String()}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestApproveWorkerPTO_PreviewWarnsWhenAnAgentWouldApprove(t *testing.T) {
	t.Parallel()

	fake := &savingPTO{pto: requestedPTO()}
	tool := newApproveWorkerPTOTool(fake).(*approveWorkerPTOTool)
	params := executeParams(map[string]any{"ptoId": fake.pto.ID.String()})
	params.Actor = agentActorFor(params.OrganizationID, params.BusinessUnitID)

	preview := previewWithoutWrites(t, &fake.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
