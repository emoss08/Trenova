package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/internal/core/services/teamscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeScheduleReader struct {
	assignments []*worker.WorkerShiftAssignment
	preferences []*worker.WorkerAvailabilityPreference
	swaps       []*worker.ShiftSwapRequest
	templates   []*worker.ShiftTemplate
	swapQuery   *repositories.ListShiftSwapsRequest
}

func (f *fakeScheduleReader) ListAssignments(
	context.Context,
	*repositories.ListShiftAssignmentsRequest,
) ([]*worker.WorkerShiftAssignment, error) {
	return f.assignments, nil
}

func (f *fakeScheduleReader) ListPreferences(
	context.Context,
	*repositories.ListAvailabilityPreferencesRequest,
) ([]*worker.WorkerAvailabilityPreference, error) {
	return f.preferences, nil
}

func (f *fakeScheduleReader) ListSwaps(
	_ context.Context,
	req *repositories.ListShiftSwapsRequest,
) ([]*worker.ShiftSwapRequest, error) {
	f.swapQuery = req

	return f.swaps, nil
}

func (f *fakeScheduleReader) ListTemplates(
	context.Context,
	*repositories.ListShiftTemplatesRequest,
) ([]*worker.ShiftTemplate, error) {
	return f.templates, nil
}

type teamPermissions struct {
	fakePermissions
}

func (p *teamPermissions) Check(
	context.Context,
	*serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	return &serviceports.PermissionCheckResult{
		Allowed:   true,
		DataScope: permission.DataScopeTeam,
	}, nil
}

type teamOf map[pulid.ID]bool

func (t teamOf) CanActFor(
	_ context.Context,
	req *orgstructureservice.ScopeRequest,
) (*orgstructureservice.ScopeResult, error) {
	return &orgstructureservice.ScopeResult{Allowed: t[req.WorkerID]}, nil
}

func TestGetWorkerSchedule_GivesTheIdsTheSchedulingToolsTake(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	schedules := &fakeScheduleReader{
		assignments: []*worker.WorkerShiftAssignment{{
			ID:              pulid.MustNew("wsa_"),
			ShiftTemplateID: pulid.MustNew("shft_"),
			EffectiveFrom:   1_800_000_000,
			ShiftTemplate:   &worker.ShiftTemplate{Name: "Early days"},
		}},
		preferences: []*worker.WorkerAvailabilityPreference{{
			DayOfWeek:  6,
			Preference: worker.AvailabilityUnavailable,
		}},
		swaps: []*worker.ShiftSwapRequest{{
			ID:                 pulid.MustNew("sswp_"),
			RequestingWorkerID: workerID,
			Status:             worker.SwapAccepted,
			ShiftDate:          1_900_000_000,
		}},
	}
	tool := newGetWorkerScheduleTool(schedules, &fakePermissions{allowed: true}, nil)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"workerId": workerID.String()}))
	require.NoError(t, err)
	view, ok := result.(*workerScheduleView)
	require.True(t, ok)
	require.Len(t, view.Assignments, 1)
	assert.Equal(t, "Early days", view.Assignments[0].Shift)
	assert.Equal(t, schedules.assignments[0].ID.String(), view.Assignments[0].ShiftAssignmentID)
	require.Len(t, view.Preferences, 1)
	assert.Equal(t, "Saturday", view.Preferences[0].DayOfWeek)
	require.Len(t, view.OpenSwaps, 1)
	assert.True(t, schedules.swapQuery.OpenOnly)
	assert.Equal(t, permission.ResourceWorkerSchedule, tool.Policy().Resource)
}

func TestGetWorkerSchedule_ATeamGrantReadsOnlyTheTeam(t *testing.T) {
	t.Parallel()

	member, stranger := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	tool := newGetWorkerScheduleTool(
		&fakeScheduleReader{},
		&teamPermissions{},
		teamOf{member: true},
	)

	_, err := tool.Query(t.Context(), testParams(map[string]any{"workerId": member.String()}))
	require.NoError(t, err)
	_, err = tool.Query(t.Context(), testParams(map[string]any{"workerId": stranger.String()}))
	require.ErrorIs(t, err, teamscope.ErrOutsideTeam)
}

func TestListShiftTemplates_SpellsOutEachPattern(t *testing.T) {
	t.Parallel()

	tool := newListShiftTemplatesTool(&fakeScheduleReader{templates: []*worker.ShiftTemplate{{
		ID:              pulid.MustNew("shft_"),
		Code:            "EARLY",
		Name:            "Early days",
		DaysOfWeek:      "0111110",
		StartMinute:     360,
		DurationMinutes: 630,
		CycleWeeks:      1,
	}}})

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	rows := result.(map[string]any)["shiftTemplates"].([]shiftTemplateRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "Mon Tue Wed Thu Fri", rows[0].Days)
	assert.Equal(t, "06:00", rows[0].Starts)
	assert.Equal(t, "10:30", rows[0].Hours)
}
