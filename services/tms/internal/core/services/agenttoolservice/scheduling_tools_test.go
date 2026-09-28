package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
	"github.com/emoss08/trenova/internal/core/services/teamscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scopedPermissions struct {
	serviceports.PermissionEngine

	allowed  bool
	scope    permission.DataScope
	requests []*serviceports.PermissionCheckRequest
}

func (p *scopedPermissions) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	p.requests = append(p.requests, req)

	return &serviceports.PermissionCheckResult{Allowed: p.allowed, DataScope: p.scope}, nil
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
	guard      *writeGuard
	assignment *worker.WorkerShiftAssignment
	swap       *worker.ShiftSwapRequest

	assigned    *schedulingservice.AssignShiftRequest
	ended       *schedulingservice.EndAssignmentRequest
	preferred   *schedulingservice.SetPreferenceRequest
	proposed    *schedulingservice.ProposeSwapRequest
	transitions []*schedulingservice.TransitionSwapRequest
}

func newFakeSchedules() *fakeSchedules {
	return &fakeSchedules{
		guard: &writeGuard{},
		assignment: &worker.WorkerShiftAssignment{
			ID:              pulid.MustNew("wsa_"),
			WorkerID:        pulid.MustNew("wrk_"),
			ShiftTemplateID: pulid.MustNew("shft_"),
			EffectiveFrom:   1_800_000_000,
			Version:         2,
		},
		swap: &worker.ShiftSwapRequest{
			ID:                 pulid.MustNew("sswp_"),
			RequestingWorkerID: pulid.MustNew("wrk_"),
			Status:             worker.SwapAccepted,
			ShiftDate:          1_900_000_000,
			Version:            4,
		},
	}
}

func (f *fakeSchedules) GetAssignment(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.WorkerShiftAssignment, error) {
	return f.assignment, nil
}

func (f *fakeSchedules) AssignShift(
	_ context.Context,
	req *schedulingservice.AssignShiftRequest,
) (*worker.WorkerShiftAssignment, error) {
	if err := f.guard.write(); err != nil {
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
		Template: &worker.ShiftTemplate{Name: "Early days"},
		Ended:    []*worker.WorkerShiftAssignment{f.assignment},
		EndedAt:  req.Entity.EffectiveFrom - 86_400,
	}, nil
}

func (f *fakeSchedules) EndAssignment(
	_ context.Context,
	req *schedulingservice.EndAssignmentRequest,
) (*worker.WorkerShiftAssignment, error) {
	if err := f.guard.write(); err != nil {
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
) (*worker.WorkerAvailabilityPreference, error) {
	if err := f.guard.write(); err != nil {
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
) (*worker.ShiftSwapRequest, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.proposed = req
	created := *req.Entity
	created.ID = pulid.MustNew("sswp_")

	return &created, nil
}

func (f *fakeSchedules) PreviewProposeSwap(
	req *schedulingservice.ProposeSwapRequest,
) (*worker.ShiftSwapRequest, error) {
	req.Entity.Status = worker.SwapProposed

	return req.Entity, nil
}

func (f *fakeSchedules) TransitionSwap(
	_ context.Context,
	req *schedulingservice.TransitionSwapRequest,
) (*worker.ShiftSwapRequest, error) {
	if err := f.guard.write(); err != nil {
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
	tool := newAssignWorkerShiftTool(schedules, newTeamScope(permissions, nil))
	workerID, templateID := pulid.MustNew("wrk_"), pulid.MustNew("shft_")
	params := executeParams(map[string]any{
		paramWorkerID:         workerID.String(),
		paramShiftTemplateID:  templateID.String(),
		fieldEffectiveFrom:    "2026-10-12",
		paramCycleOffsetWeeks: float64(1),
	})

	preview := previewWithoutWrites(t, schedules.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)
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
	scope := newTeamScope(
		&scopedPermissions{allowed: true, scope: permission.DataScopeTeam},
		teams,
	)
	tool := newAssignWorkerShiftTool(schedules, scope)
	paramsFor := func(workerID pulid.ID) serviceports.ToolExecuteParams {
		return executeParams(map[string]any{
			paramWorkerID:        workerID.String(),
			paramShiftTemplateID: pulid.MustNew("shft_").String(),
			fieldEffectiveFrom:   "2026-10-12",
		})
	}

	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), paramsFor(member)))
	require.ErrorIs(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		paramsFor(stranger)), teamscope.ErrOutsideTeam)
	require.ErrorIs(t, tool.Execute(t.Context(), paramsFor(stranger)), teamscope.ErrOutsideTeam)
	assert.Nil(t, schedules.assigned)

	refusing := newAssignWorkerShiftTool(schedules,
		newTeamScope(&scopedPermissions{allowed: false}, nil))
	require.Error(t, refusing.(serviceports.ToolValidator).Validate(t.Context(),
		paramsFor(member)))

	unresolved := newAssignWorkerShiftTool(schedules, newTeamScope(
		&scopedPermissions{allowed: true, scope: permission.DataScopeTeam}, nil,
	))
	require.ErrorIs(t, unresolved.(serviceports.ToolValidator).Validate(t.Context(),
		paramsFor(member)), teamscope.ErrOutsideTeam, "a team grant nobody can resolve is refused")
}

func TestEndWorkerShiftAssignment_ChecksTheTeamOfTheAssignmentsWorker(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	teams := &fakeTeams{members: map[pulid.ID]bool{schedules.assignment.WorkerID: true}}
	tool := newEndWorkerShiftAssignmentTool(schedules, newTeamScope(
		&scopedPermissions{allowed: true, scope: permission.DataScopeTeam},
		teams,
	))
	params := executeParams(map[string]any{
		paramShiftAssignmentID: schedules.assignment.ID.String(),
		fieldEffectiveTo:       "2026-11-01",
	})

	preview := previewWithoutWrites(t, schedules.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "2026-11-01")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, schedules.ended)
	assert.Contains(t, teams.asked, schedules.assignment.WorkerID)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceWorkerSchedule, target.Resource)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramShiftAssignmentID: schedules.assignment.ID.String(),
			fieldEffectiveTo:       float64(1_900_000_000),
		})))
}

func TestSetWorkerAvailabilityPreference_ReadsTheWeekdayByName(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	tool := newSetWorkerAvailabilityPreferenceTool(schedules, newTeamScope(
		&scopedPermissions{allowed: true}, nil))
	workerID := pulid.MustNew("wrk_")
	params := executeParams(map[string]any{
		paramWorkerID:   workerID.String(),
		paramDayOfWeek:  "Saturday",
		paramPreference: "Unavailable",
		paramNote:       "Coaches his son's team",
	})

	preview := previewWithoutWrites(t, schedules.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "unavailable on Saturday")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, schedules.preferred)
	assert.Equal(t, int16(time.Saturday), schedules.preferred.Entity.DayOfWeek)
	assert.Equal(t, worker.AvailabilityUnavailable, schedules.preferred.Entity.Preference)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)

	for name, raw := range map[string]map[string]any{
		"a numbered day":     {paramWorkerID: workerID.String(), paramDayOfWeek: float64(6), paramPreference: "Available"},
		"an unknown day":     {paramWorkerID: workerID.String(), paramDayOfWeek: "Funday", paramPreference: "Available"},
		"a wrong preference": {paramWorkerID: workerID.String(), paramDayOfWeek: "Monday", paramPreference: "Maybe"},
	} {
		require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
			executeParams(raw)), name)
	}
}

func TestProposeShiftSwap_RaisesATradeWithBothDays(t *testing.T) {
	t.Parallel()

	schedules := newFakeSchedules()
	tool := newProposeShiftSwapTool(schedules)
	requesting, counterparty := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	params := executeParams(map[string]any{
		paramRequestingWorkerID:   requesting.String(),
		paramCounterpartyWorkerID: counterparty.String(),
		paramShiftDate:            "2030-03-18",
		paramCounterpartyDate:     "2030-03-20",
		fieldReason:               "Wedding",
	})

	preview := previewWithoutWrites(t, schedules.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)

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
		tool      serviceports.AgentTool
		operation permission.Operation
		status    worker.ShiftSwapStatus
		maxTier   agent.AutonomyTier
	}{
		{newApproveShiftSwapTool(schedules), permission.OpApprove, worker.SwapApproved, agent.TierPropose},
		{newRejectShiftSwapTool(schedules), permission.OpReject, worker.SwapRejected, agent.TierActWithApproval},
		{newWithdrawShiftSwapTool(schedules), permission.OpCancel, worker.SwapWithdrawn, agent.TierActWithApproval},
	} {
		policy := tc.tool.Policy()
		assert.Equal(t, tc.operation, policy.Operation, policy.Name)
		assert.Equal(t, tc.maxTier, policy.MaxTier, policy.Name)
		assert.Equal(t, []agent.EgressClass{agent.EgressDriverVisible}, policy.Egress, policy.Name)

		params := approvedParams(map[string]any{
			paramSwapID: schedules.swap.ID.String(),
			paramNote:   "Covered",
		})
		preview := previewWithoutWrites(t, schedules.guard, func() (*agent.ToolPreview, error) {
			return tc.tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
		})
		assert.Equal(t, string(tc.status),
			fieldByPath(t, previewChange(t, preview, 0), fieldStatus).After, policy.Name)
		require.NoError(t, tc.tool.Execute(t.Context(), params), policy.Name)
		assert.Equal(t, tc.status, schedules.transitions[len(schedules.transitions)-1].Status)
	}

	approve := newApproveShiftSwapTool(schedules)
	require.ErrorIs(t, approve.Execute(t.Context(), executeParams(map[string]any{
		paramSwapID: schedules.swap.ID.String(),
	})), ErrNeedsAPersonsApproval)
}
