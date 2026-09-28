package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLeave struct {
	leaveKeeper

	guard *writeGuard
	lcase *worker.WorkerLeaveCase
	entry *worker.WorkerLeaveEntry

	opened   *worker.WorkerLeaveCase
	updated  *workerleaveservice.UpdateCaseRequest
	closed   []pulid.ID
	recorded *workerleaveservice.RecordDayRequest
	deleted  []pulid.ID
}

func newFakeLeave() *fakeLeave {
	workerID := pulid.MustNew("wrk_")
	caseID := pulid.MustNew("wlc_")
	return &fakeLeave{
		guard: &writeGuard{},
		lcase: &worker.WorkerLeaveCase{
			ID:        caseID,
			WorkerID:  workerID,
			LeaveType: worker.LeaveTypeFMLA,
			Status:    worker.LeaveCaseApproved,
			StartsAt:  1_788_000_000,
			Version:   3,
		},
		entry: &worker.WorkerLeaveEntry{
			ID:                       pulid.MustNew("wle_"),
			WorkerID:                 workerID,
			LeaveCaseID:              caseID,
			UsedOn:                   1_788_048_000,
			Hours:                    decimal.NewFromInt(8),
			CountsAgainstEntitlement: true,
			Version:                  1,
		},
	}
}

func (f *fakeLeave) PlanOpenCase(
	_ context.Context,
	entity *worker.WorkerLeaveCase,
	_ pulid.ID,
) (*worker.WorkerLeaveCase, error) {
	if entity.LeaveType == "" {
		return nil, errortypes.NewValidationError("leaveType", errortypes.ErrRequired,
			"Leave type is required")
	}
	planned := *entity
	planned.Status = worker.LeaveCasePending
	return &planned, nil
}

func (f *fakeLeave) OpenCase(
	ctx context.Context,
	entity *worker.WorkerLeaveCase,
	userID pulid.ID,
) (*worker.WorkerLeaveCase, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	planned, err := f.PlanOpenCase(ctx, entity, userID)
	if err != nil {
		return nil, err
	}
	planned.ID = pulid.MustNew("wlc_")
	f.opened = planned
	return planned, nil
}

func (f *fakeLeave) PlanUpdateCase(
	_ context.Context,
	req *workerleaveservice.UpdateCaseRequest,
) (*workerleaveservice.CaseChange, error) {
	after := *f.lcase
	if req.EndsAt != nil {
		after.EndsAt = req.EndsAt
	}
	return &workerleaveservice.CaseChange{Before: f.lcase, After: &after}, nil
}

func (f *fakeLeave) UpdateCase(
	_ context.Context,
	req *workerleaveservice.UpdateCaseRequest,
) (*worker.WorkerLeaveCase, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = req
	return f.lcase, nil
}

func (f *fakeLeave) PlanCloseCase(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workerleaveservice.CaseChange, error) {
	if f.lcase.Status == worker.LeaveCasePending {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"A pending case is decided before it is closed")
	}
	after := *f.lcase
	after.Status = worker.LeaveCaseClosed
	return &workerleaveservice.CaseChange{Before: f.lcase, After: &after}, nil
}

func (f *fakeLeave) CloseCase(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) (*worker.WorkerLeaveCase, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.closed = append(f.closed, id)
	return f.lcase, nil
}

func (f *fakeLeave) PlanRecordDay(
	_ context.Context,
	req *workerleaveservice.RecordDayRequest,
) (*worker.WorkerLeaveEntry, error) {
	return &worker.WorkerLeaveEntry{
		WorkerID:                 f.lcase.WorkerID,
		LeaveCaseID:              req.CaseID,
		UsedOn:                   req.UsedOn,
		Hours:                    req.Hours,
		CountsAgainstEntitlement: f.lcase.LeaveType == worker.LeaveTypeFMLA,
	}, nil
}

func (f *fakeLeave) RecordDay(
	_ context.Context,
	req *workerleaveservice.RecordDayRequest,
) (*worker.WorkerLeaveEntry, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.recorded = req
	return f.entry, nil
}

func (f *fakeLeave) PlanDeleteDay(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.WorkerLeaveEntry, error) {
	return f.entry, nil
}

func (f *fakeLeave) DeleteDay(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func TestOpenLeaveCase_WaitsForADecision(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newOpenLeaveCaseTool(cases)
	params := executeParams(map[string]any{
		paramWorkerID:         cases.lcase.WorkerID.String(),
		paramLeaveType:        "Parental",
		paramStartsOn:         "2026-10-05",
		paramEndsOn:           "2026-11-13",
		paramEligibilityHours: float64(1800),
	})

	preview := previewWithoutWrites(t, cases.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "waiting for a decision")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, cases.opened)
	assert.Equal(t, worker.LeaveTypeParental, cases.opened.LeaveType)
	require.NotNil(t, cases.opened.EndsAt)
	require.NotNil(t, cases.opened.EligibilityHoursWorked)
	assert.Equal(t, int32(1800), *cases.opened.EligibilityHoursWorked)
	assert.Equal(t, permission.ResourceWorkerLeave, tool.Policy().Resource)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramWorkerID: cases.lcase.WorkerID.String(),
			paramStartsOn: "2026-10-05",
		})), "validation runs the plan, which needs a leave type")
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramWorkerID:  cases.lcase.WorkerID.String(),
			paramLeaveType: "FMLA",
			paramStartsOn:  "10/05/2026",
		})))
}

func TestUpdateLeaveCase_KeepsWhatIsNotNamed(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newUpdateLeaveCaseTool(cases)
	params := executeParams(map[string]any{
		paramLeaveCaseID: cases.lcase.ID.String(),
		paramEndsOn:      "2026-10-30",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, cases.updated)
	require.NotNil(t, cases.updated.EndsAt)
	assert.Nil(t, cases.updated.LeaveType)
	assert.Nil(t, cases.updated.Reason)
	assert.Nil(t, cases.updated.StartsAt)
}

func TestCloseLeaveCase_IsAPersonsDecision(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newCloseLeaveCaseTool(cases)
	raw := map[string]any{paramLeaveCaseID: cases.lcase.ID.String()}

	require.ErrorIs(t, tool.Execute(t.Context(), executeParams(raw)), ErrNeedsAPersonsApproval)
	assert.Empty(t, cases.closed)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	assert.Equal(t, []pulid.ID{cases.lcase.ID}, cases.closed)

	cases.lcase.Status = worker.LeaveCasePending
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(raw)))
}

func TestRecordLeaveDay_SaysWhetherItCounts(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newRecordLeaveDayTool(cases)
	params := executeParams(map[string]any{
		paramLeaveCaseID: cases.lcase.ID.String(),
		paramUsedOn:      "2026-10-06",
		paramLeaveHours:  "4.5",
	})

	preview := previewWithoutWrites(t, cases.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "4.5 hours")
	assert.Contains(t, preview.Summary, "counts against the entitlement")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, cases.recorded)
	assert.True(t, decimal.RequireFromString("4.5").Equal(cases.recorded.Hours))
}

func TestDeleteLeaveDay_IsOnlyProposed(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newDeleteLeaveDayTool(cases)
	params := executeParams(map[string]any{paramLeaveEntryID: cases.entry.ID.String()})

	preview := previewWithoutWrites(t, cases.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationDelete, previewChange(t, preview, 0).Operation)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{cases.entry.ID}, cases.deleted)
}
