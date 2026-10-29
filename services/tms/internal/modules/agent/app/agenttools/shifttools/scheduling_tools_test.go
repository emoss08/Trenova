package shifttools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/teamscope"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scopedPermissions struct {
	permissionports.PermissionEngine

	allowed  bool
	scope    permission.DataScope
	requests []*permissionports.PermissionCheckRequest
}

func (p *scopedPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	p.requests = append(p.requests, req)

	return &permissionports.PermissionCheckResult{Allowed: p.allowed, DataScope: p.scope}, nil
}

type fakeTeams struct {
	members map[pulid.ID]bool
	asked   []pulid.ID
}

func (f *fakeTeams) CanActFor(
	_ context.Context,
	req *orgstructureservice.ScopeRequest,
) (*orgstructureservice.ScopeResult, error) {
	f.asked = append(f.asked, req.WorkerID)

	return &orgstructureservice.ScopeResult{Allowed: f.members[req.WorkerID]}, nil
}

type fakeSchedules struct {
	guard      *agenttoolstest.WriteGuard
	assignment *workerschedule.WorkerShiftAssignment
	swap       *workerschedule.ShiftSwapRequest

	assigned    *schedulingservice.AssignShiftRequest
	ended       *schedulingservice.EndAssignmentRequest
	preferred   *schedulingservice.SetPreferenceRequest
	proposed    *schedulingservice.ProposeSwapRequest
	transitions []*schedulingservice.TransitionSwapRequest
}

func newFakeSchedules() *fakeSchedules {
	return &fakeSchedules{
		guard: &agenttoolstest.WriteGuard{},
		assignment: &workerschedule.WorkerShiftAssignment{
			ID:              pulid.MustNew("wsa_"),
			WorkerID:        pulid.MustNew("wrk_"),
			ShiftTemplateID: pulid.MustNew("shft_"),
			EffectiveFrom:   1_800_000_000,
			Version:         2,
		},
		swap: &workerschedule.ShiftSwapRequest{
			ID:                 pulid.MustNew("sswp_"),
			RequestingWorkerID: pulid.MustNew("wrk_"),
			Status:             workerschedule.SwapAccepted,
			ShiftDate:          1_900_000_000,
			Version:            4,
		},
	}
}

func (f *fakeSchedules) GetAssignment(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workerschedule.WorkerShiftAssignment, error) {
	return f.assignment, nil
}

func (f *fakeSchedules) AssignShift(
	_ context.Context,
	req *schedulingservice.AssignShiftRequest,
) (*workerschedule.WorkerShiftAssignment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.assigned = req
	created := *req.Entity
	created.ID = pulid.MustNew("wsa_")

	return &created, nil
}

func (f *fakeSchedules) PreviewAssignShift(
	_ context.Context,
	req *schedulingservice.AssignShiftRequest,
) (*schedulingservice.ShiftAssignmentPlan, error) {
	return &schedulingservice.ShiftAssignmentPlan{
		Created:  req.Entity,
		Template: &workerschedule.ShiftTemplate{Name: "Early days"},
		Ended:    []*workerschedule.WorkerShiftAssignment{f.assignment},
		EndedAt:  req.Entity.EffectiveFrom - 86_400,
	}, nil
}

func (f *fakeSchedules) EndAssignment(
	_ context.Context,
	req *schedulingservice.EndAssignmentRequest,
) (*workerschedule.WorkerShiftAssignment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.ended = req

	return f.assignment, nil
}

func (f *fakeSchedules) PreviewEndAssignment(
	_ context.Context,
	req *schedulingservice.EndAssignmentRequest,
) (*schedulingservice.AssignmentChange, error) {
	after := *f.assignment
	end := req.EffectiveTo
	after.EffectiveTo = &end

	return &schedulingservice.AssignmentChange{Before: f.assignment, After: &after}, nil
}

func (f *fakeSchedules) SetPreference(
	_ context.Context,
	req *schedulingservice.SetPreferenceRequest,
) (*workerschedule.WorkerAvailabilityPreference, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.preferred = req

	return req.Entity, nil
}

func (f *fakeSchedules) PreviewSetPreference(
	_ context.Context,
	req *schedulingservice.SetPreferenceRequest,
) (*schedulingservice.PreferenceChange, error) {
	return &schedulingservice.PreferenceChange{After: req.Entity}, nil
}

func (f *fakeSchedules) ProposeSwap(
	_ context.Context,
	req *schedulingservice.ProposeSwapRequest,
) (*workerschedule.ShiftSwapRequest, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.proposed = req
	created := *req.Entity
	created.ID = pulid.MustNew("sswp_")

	return &created, nil
}

func (f *fakeSchedules) PreviewProposeSwap(
	req *schedulingservice.ProposeSwapRequest,
) (*workerschedule.ShiftSwapRequest, error) {
	req.Entity.Status = workerschedule.SwapProposed

	return req.Entity, nil
}

func (f *fakeSchedules) TransitionSwap(
	_ context.Context,
	req *schedulingservice.TransitionSwapRequest,
) (*workerschedule.ShiftSwapRequest, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.transitions = append(f.transitions, req)
	after := *f.swap
	after.Status = req.Status

	return &after, nil
}

func (f *fakeSchedules) PreviewTransitionSwap(
	_ context.Context,
	req *schedulingservice.TransitionSwapRequest,
) (*schedulingservice.SwapChange, error) {
	after := *f.swap
	after.Status = req.Status
	after.ResponseNote = req.Note

	return &schedulingservice.SwapChange{Before: f.swap, After: &after}, nil
}

func TestAssignWorkerShift_PreviewsTheNewPatternAndTheOneItEnds(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	permissions := &scopedPermissions{allowed: true, scope: permission.DataScopeOrganization}
	tool := NewAssignWorkerShiftTool(schedules, NewTeamScope(permissions, nil))
	workerID, templateID := pulid.MustNew("wrk_"), pulid.MustNew("shft_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID:         workerID.String(),
		paramShiftTemplateID:             templateID.String(),
		changepreview.FieldEffectiveFrom: "2026-10-12",
		paramCycleOffsetWeeks:            float64(1),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		schedules.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 2)
	assert.Equal(
		t,
		agentpreview.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	assert.Contains(t, preview.Summary, "Early days")
	assert.Contains(t, preview.Summary, "2026-10-12")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, schedules.assigned)
	assert.Equal(t, time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC).Unix(),
		schedules.assigned.Entity.EffectiveFrom)
	assert.Equal(t, int16(1), schedules.assigned.Entity.CycleOffsetWeeks)
	require.NotEmpty(t, permissions.requests)
	assert.Equal(t, permission.ResourceWorkerSchedule.String(), permissions.requests[0].Resource)
	assert.Equal(t, permission.OpAssign, permissions.requests[0].Operation)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceWorkerSchedule, policy.Resource)
	assert.Equal(t, permission.OpAssign, policy.Operation)
}

func TestAssignWorkerShift_ATeamGrantReachesOnlyTheTeam(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	member, stranger := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	teams := &fakeTeams{members: map[pulid.ID]bool{member: true}}
	scope := NewTeamScope(
		&scopedPermissions{allowed: true, scope: permission.DataScopeTeam},
		teams,
	)
	tool := NewAssignWorkerShiftTool(schedules, scope)
	paramsFor := func(workerID pulid.ID) agenttoolports.ToolExecuteParams {
		return agenttoolstest.ExecuteParams(map[string]any{
			toolparams.ParamWorkerID:         workerID.String(),
			paramShiftTemplateID:             pulid.MustNew("shft_").String(),
			changepreview.FieldEffectiveFrom: "2026-10-12",
		})
	}

	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), paramsFor(member)))
	require.ErrorIs(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		paramsFor(stranger)), teamscope.ErrOutsideTeam)
	require.ErrorIs(t, tool.Execute(t.Context(), paramsFor(stranger)), teamscope.ErrOutsideTeam)
	assert.Nil(t, schedules.assigned)

	refusing := NewAssignWorkerShiftTool(schedules,
		NewTeamScope(&scopedPermissions{allowed: false}, nil))
	require.Error(t, refusing.(agenttoolports.ToolValidator).Validate(t.Context(),
		paramsFor(member)))

	unresolved := NewAssignWorkerShiftTool(schedules, NewTeamScope(
		&scopedPermissions{allowed: true, scope: permission.DataScopeTeam}, nil,
	))
	require.ErrorIs(t, unresolved.(agenttoolports.ToolValidator).Validate(t.Context(),
		paramsFor(member)), teamscope.ErrOutsideTeam, "a team grant nobody can resolve is refused")
}

func TestEndWorkerShiftAssignment_ChecksTheTeamOfTheAssignmentsWorker(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	teams := &fakeTeams{members: map[pulid.ID]bool{schedules.assignment.WorkerID: true}}
	tool := NewEndWorkerShiftAssignmentTool(schedules, NewTeamScope(
		&scopedPermissions{allowed: true, scope: permission.DataScopeTeam},
		teams,
	))
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramShiftAssignmentID:         schedules.assignment.ID.String(),
		changepreview.FieldEffectiveTo: "2026-11-01",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		schedules.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "2026-11-01")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, schedules.ended)
	assert.Contains(t, teams.asked, schedules.assignment.WorkerID)
	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceWorkerSchedule, target.Resource)

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			paramShiftAssignmentID:         schedules.assignment.ID.String(),
			changepreview.FieldEffectiveTo: float64(1_900_000_000),
		})))
}

func TestSetWorkerAvailabilityPreference_ReadsTheWeekdayByName(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	tool := NewSetWorkerAvailabilityPreferenceTool(schedules, NewTeamScope(
		&scopedPermissions{allowed: true}, nil))
	workerID := pulid.MustNew("wrk_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID: workerID.String(),
		paramDayOfWeek:           "Saturday",
		paramPreference:          "Unavailable",
		toolparams.ParamNote:     "Coaches his son's team",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		schedules.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "unavailable on Saturday")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, schedules.preferred)
	assert.Equal(t, int16(time.Saturday), schedules.preferred.Entity.DayOfWeek)
	assert.Equal(t, workerschedule.AvailabilityUnavailable, schedules.preferred.Entity.Preference)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)

	for name, raw := range map[string]map[string]any{
		"a numbered day":     {toolparams.ParamWorkerID: workerID.String(), paramDayOfWeek: float64(6), paramPreference: "Available"},
		"an unknown day":     {toolparams.ParamWorkerID: workerID.String(), paramDayOfWeek: "Funday", paramPreference: "Available"},
		"a wrong preference": {toolparams.ParamWorkerID: workerID.String(), paramDayOfWeek: "Monday", paramPreference: "Maybe"},
	} {
		require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
			agenttoolstest.ExecuteParams(raw)), name)
	}
}

func TestProposeShiftSwap_RaisesATradeWithBothDays(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	tool := NewProposeShiftSwapTool(schedules)
	requesting, counterparty := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramRequestingWorkerID:   requesting.String(),
		paramCounterpartyWorkerID: counterparty.String(),
		paramShiftDate:            "2030-03-18",
		paramCounterpartyDate:     "2030-03-20",
		changepreview.FieldReason: "Wedding",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		schedules.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, schedules.proposed)
	assert.Equal(t, counterparty, schedules.proposed.Entity.CounterpartyWorkerID)
	require.NotNil(t, schedules.proposed.Entity.CounterpartyShiftDate)
	assert.Equal(t, time.Date(2030, 3, 20, 0, 0, 0, 0, time.UTC).Unix(),
		*schedules.proposed.Entity.CounterpartyShiftDate)
	assert.Equal(t, permission.ResourceShiftSwap, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)
}

func TestShiftSwapDecisions_EachNamesItsOwnPermission(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	for _, tc := range []struct {
		tool      agenttoolports.AgentTool
		operation permission.Operation
		status    workerschedule.ShiftSwapStatus
		maxTier   agent.AutonomyTier
	}{
		{NewApproveShiftSwapTool(schedules), permission.OpApprove, workerschedule.SwapApproved, agent.TierPropose},
		{NewRejectShiftSwapTool(schedules), permission.OpReject, workerschedule.SwapRejected, agent.TierActWithApproval},
		{NewWithdrawShiftSwapTool(schedules), permission.OpCancel, workerschedule.SwapWithdrawn, agent.TierActWithApproval},
	} {
		policy := tc.tool.Policy()
		assert.Equal(t, tc.operation, policy.Operation, policy.Name)
		assert.Equal(t, tc.maxTier, policy.MaxTier, policy.Name)
		assert.Equal(t, []agent.EgressClass{agent.EgressDriverVisible}, policy.Egress, policy.Name)

		params := agenttoolstest.ApprovedParams(map[string]any{
			paramSwapID:          schedules.swap.ID.String(),
			toolparams.ParamNote: "Covered",
		})
		preview := agenttoolstest.PreviewWithoutWrites(
			t,
			schedules.guard,
			func() (*agentpreview.ToolPreview, error) {
				return tc.tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
			},
		)
		assert.Equal(
			t,
			string(tc.status),
			agenttoolstest.FieldByPath(
				t,
				agenttoolstest.PreviewChange(t, preview, 0),
				changepreview.FieldStatus,
			).After,
			policy.Name,
		)
		require.NoError(t, tc.tool.Execute(t.Context(), params), policy.Name)
		assert.Equal(t, tc.status, schedules.transitions[len(schedules.transitions)-1].Status)
	}

	approve := NewApproveShiftSwapTool(schedules)
	require.ErrorIs(t, approve.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		paramSwapID: schedules.swap.ID.String(),
	})), toolspec.ErrNeedsAPersonsApproval)
}
