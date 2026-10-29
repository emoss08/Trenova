package workforcetools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/teamscope"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLeave struct {
	LeaveKeeper

	guard *agenttoolstest.WriteGuard
	lcase *workerleave.WorkerLeaveCase
	entry *workerleave.WorkerLeaveEntry

	opened   *workerleave.WorkerLeaveCase
	updated  *workerleaveservice.UpdateCaseRequest
	closed   []pulid.ID
	recorded *workerleaveservice.RecordDayRequest
	deleted  []pulid.ID
}

func newFakeLeave() *fakeLeave {
	workerID := pulid.MustNew("wrk_")
	caseID := pulid.MustNew("wlc_")
	return &fakeLeave{
		guard: &agenttoolstest.WriteGuard{},
		lcase: &workerleave.WorkerLeaveCase{
			ID:        caseID,
			WorkerID:  workerID,
			LeaveType: worker.LeaveTypeFMLA,
			Status:    workerleave.LeaveCaseApproved,
			StartsAt:  1_788_000_000,
			Version:   3,
		},
		entry: &workerleave.WorkerLeaveEntry{
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
	entity *workerleave.WorkerLeaveCase,
	_ pulid.ID,
) (*workerleave.WorkerLeaveCase, error) {
	if entity.LeaveType == "" {
		return nil, errortypes.NewValidationError("leaveType", errortypes.ErrRequired,
			"Leave type is required")
	}
	planned := *entity
	planned.Status = workerleave.LeaveCasePending
	return &planned, nil
}

func (f *fakeLeave) OpenCase(
	ctx context.Context,
	entity *workerleave.WorkerLeaveCase,
	userID pulid.ID,
) (*workerleave.WorkerLeaveCase, error) {
	if err := f.guard.Write(); err != nil {
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
) (*workerleave.WorkerLeaveCase, error) {
	if err := f.guard.Write(); err != nil {
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
	if f.lcase.Status == workerleave.LeaveCasePending {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"A pending case is decided before it is closed")
	}
	after := *f.lcase
	after.Status = workerleave.LeaveCaseClosed
	return &workerleaveservice.CaseChange{Before: f.lcase, After: &after}, nil
}

func (f *fakeLeave) CloseCase(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) (*workerleave.WorkerLeaveCase, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.closed = append(f.closed, id)
	return f.lcase, nil
}

func (f *fakeLeave) PlanRecordDay(
	_ context.Context,
	req *workerleaveservice.RecordDayRequest,
) (*workerleave.WorkerLeaveEntry, error) {
	return &workerleave.WorkerLeaveEntry{
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
) (*workerleave.WorkerLeaveEntry, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.recorded = req
	return f.entry, nil
}

func (f *fakeLeave) PlanDeleteDay(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workerleave.WorkerLeaveEntry, error) {
	return f.entry, nil
}

func (f *fakeLeave) DeleteDay(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func TestOpenLeaveCase_WaitsForADecision(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newOpenLeaveCaseTool(cases, orgLeaveScope(cases))
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID: cases.lcase.WorkerID.String(),
		paramLeaveType:           "Parental",
		"startsAt":               "2026-10-05",
		"endsAt":                 "2026-11-13",
		"requestedAt":            "2026-09-30",
		paramEligibilityHours:    float64(1800),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		cases.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "waiting for a decision")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, cases.opened)
	assert.Equal(t, worker.LeaveTypeParental, cases.opened.LeaveType)
	require.NotNil(t, cases.opened.EndsAt)
	assert.Equal(t, time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC).Unix(),
		cases.opened.RequestedAt)
	require.NotNil(t, cases.opened.EligibilityHoursWorked)
	assert.Equal(t, int32(1800), *cases.opened.EligibilityHoursWorked)
	assert.Equal(t, permission.ResourceWorkerLeave, tool.Policy().Resource)

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			toolparams.ParamWorkerID: cases.lcase.WorkerID.String(),
			"startsAt":               "2026-10-05",
		})), "validation runs the plan, which needs a leave type")
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			toolparams.ParamWorkerID: cases.lcase.WorkerID.String(),
			paramLeaveType:           "FMLA",
			"startsAt":               "10/05/2026",
		})))
}

func TestUpdateLeaveCase_KeepsWhatIsNotNamed(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newUpdateLeaveCaseTool(cases, orgLeaveScope(cases))
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramLeaveCaseID: cases.lcase.ID.String(),
		"endsAt":         "2026-10-30",
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
	tool := newCloseLeaveCaseTool(cases, orgLeaveScope(cases))
	raw := map[string]any{paramLeaveCaseID: cases.lcase.ID.String()}

	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)),
		toolspec.ErrNeedsAPersonsApproval,
	)
	assert.Empty(t, cases.closed)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	assert.Equal(t, []pulid.ID{cases.lcase.ID}, cases.closed)

	cases.lcase.Status = workerleave.LeaveCasePending
	require.Error(
		t,
		tool.(agenttoolports.ToolValidator).Validate(
			t.Context(),
			agenttoolstest.ExecuteParams(raw),
		),
	)
}

func TestRecordLeaveDay_SaysWhetherItCounts(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newRecordLeaveDayTool(cases, orgLeaveScope(cases))
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramLeaveCaseID: cases.lcase.ID.String(),
		"usedOn":         "2026-10-06",
		paramLeaveHours:  "4.5",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		cases.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "4.5 hours")
	assert.Contains(t, preview.Summary, "counts against the entitlement")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, cases.recorded)
	assert.True(t, decimal.RequireFromString("4.5").Equal(cases.recorded.Hours))
}

func TestDeleteLeaveDay_IsOnlyProposed(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	tool := newDeleteLeaveDayTool(cases, orgLeaveScope(cases))
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramLeaveEntryID: cases.entry.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		cases.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{cases.entry.ID}, cases.deleted)
}

func (f *fakeLeave) CaseWorker(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (pulid.ID, error) {
	if id != f.lcase.ID {
		return pulid.Nil, errortypes.NewNotFoundError("Leave case not found")
	}
	return f.lcase.WorkerID, nil
}

func (f *fakeLeave) EntryWorker(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (pulid.ID, error) {
	if id != f.entry.ID {
		return pulid.Nil, errortypes.NewNotFoundError("Leave day not found")
	}
	return f.entry.WorkerID, nil
}

type leaveGrant struct{ scope permission.DataScope }

func (g leaveGrant) Check(
	_ context.Context,
	_ *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{Allowed: true, DataScope: g.scope}, nil
}

type leaveTeam map[pulid.ID]bool

func (t leaveTeam) CanActFor(
	_ context.Context,
	req *orgstructureservice.ScopeRequest,
) (*orgstructureservice.ScopeResult, error) {
	return &orgstructureservice.ScopeResult{Allowed: t[req.WorkerID]}, nil
}

func orgLeaveScope(cases *fakeLeave) LeaveScope {
	return NewLeaveScope(leaveGrant{scope: permission.DataScopeOrganization}, nil, cases)
}

func TestLeaveTools_ATeamGrantReachesOnlyTheTeamsCases(t *testing.T) {
	t.Parallel()

	cases := newFakeLeave()
	onTeam := NewLeaveScope(leaveGrant{scope: permission.DataScopeTeam},
		leaveTeam{cases.lcase.WorkerID: true}, cases)
	offTeam := NewLeaveScope(leaveGrant{scope: permission.DataScopeTeam}, leaveTeam{}, cases)
	closeCase := map[string]any{paramLeaveCaseID: cases.lcase.ID.String()}
	removeDay := map[string]any{paramLeaveEntryID: cases.entry.ID.String()}
	openCase := map[string]any{
		toolparams.ParamWorkerID: cases.lcase.WorkerID.String(),
		paramLeaveType:           "FMLA",
		"startsAt":               "2026-10-05",
	}

	require.ErrorIs(t, newCloseLeaveCaseTool(cases, offTeam).Execute(t.Context(),
		agenttoolstest.ApprovedParams(closeCase)), teamscope.ErrOutsideTeam)
	require.ErrorIs(t, newDeleteLeaveDayTool(cases, offTeam).Execute(t.Context(),
		agenttoolstest.ExecuteParams(removeDay)), teamscope.ErrOutsideTeam)
	require.ErrorIs(t, newOpenLeaveCaseTool(cases, offTeam).(agenttoolports.ToolValidator).Validate(
		t.Context(), agenttoolstest.ExecuteParams(openCase)), teamscope.ErrOutsideTeam)
	assert.Empty(t, cases.closed)
	assert.Empty(t, cases.deleted)

	require.NoError(t, newCloseLeaveCaseTool(cases, onTeam).Execute(t.Context(),
		agenttoolstest.ApprovedParams(closeCase)))
	require.NoError(t, newDeleteLeaveDayTool(cases, onTeam).Execute(t.Context(),
		agenttoolstest.ExecuteParams(removeDay)))
	assert.Equal(t, []pulid.ID{cases.lcase.ID}, cases.closed)
	assert.Equal(t, []pulid.ID{cases.entry.ID}, cases.deleted)
}
