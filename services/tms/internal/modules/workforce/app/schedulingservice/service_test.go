package schedulingservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(86400)

type fakeSchedulingRepo struct {
	workerscheduleports.SchedulingRepository

	template        *workerschedule.ShiftTemplate
	templates       []*workerschedule.ShiftTemplate
	assignments     []*workerschedule.WorkerShiftAssignment
	updated         []*workerschedule.WorkerShiftAssignment
	assignmentCount int
	swap            *workerschedule.ShiftSwapRequest
	savedSwap       *workerschedule.ShiftSwapRequest
	preferences     []*workerschedule.WorkerAvailabilityPreference
	workers         []workerscheduleports.RotaWorkerRow
	timeOff         []workerscheduleports.RotaRangeRow
	leave           []workerscheduleports.RotaRangeRow
	assigned        []workerscheduleports.RotaDayRow
}

func (r *fakeSchedulingRepo) GetTemplateByID(
	_ context.Context,
	_ *workerscheduleports.GetShiftTemplateByIDRequest,
) (*workerschedule.ShiftTemplate, error) {
	return r.template, nil
}

func (r *fakeSchedulingRepo) ListTemplates(
	_ context.Context,
	_ *workerscheduleports.ListShiftTemplatesRequest,
) ([]*workerschedule.ShiftTemplate, error) {
	return r.templates, nil
}

func (r *fakeSchedulingRepo) UpdateTemplate(
	_ context.Context,
	entity *workerschedule.ShiftTemplate,
) (*workerschedule.ShiftTemplate, error) {
	return entity, nil
}

func (r *fakeSchedulingRepo) CountTemplateAssignments(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (int, error) {
	return r.assignmentCount, nil
}

func (r *fakeSchedulingRepo) ListAssignments(
	_ context.Context,
	_ *workerscheduleports.ListShiftAssignmentsRequest,
) ([]*workerschedule.WorkerShiftAssignment, error) {
	return r.assignments, nil
}

func (r *fakeSchedulingRepo) CreateAssignment(
	_ context.Context,
	entity *workerschedule.WorkerShiftAssignment,
) (*workerschedule.WorkerShiftAssignment, error) {
	entity.ID = pulid.MustNew("wsa_")
	return entity, nil
}

func (r *fakeSchedulingRepo) UpdateAssignment(
	_ context.Context,
	entity *workerschedule.WorkerShiftAssignment,
) (*workerschedule.WorkerShiftAssignment, error) {
	r.updated = append(r.updated, entity)
	return entity, nil
}

func (r *fakeSchedulingRepo) GetSwapByID(
	_ context.Context,
	_ *workerscheduleports.GetShiftSwapByIDRequest,
) (*workerschedule.ShiftSwapRequest, error) {
	return r.swap, nil
}

func (r *fakeSchedulingRepo) UpdateSwap(
	_ context.Context,
	entity *workerschedule.ShiftSwapRequest,
) (*workerschedule.ShiftSwapRequest, error) {
	r.savedSwap = entity
	return entity, nil
}

func (r *fakeSchedulingRepo) ListPreferences(
	_ context.Context,
	_ *workerscheduleports.ListAvailabilityPreferencesRequest,
) ([]*workerschedule.WorkerAvailabilityPreference, error) {
	return r.preferences, nil
}

func (r *fakeSchedulingRepo) RotaWorkers(
	_ context.Context,
	_ *workerscheduleports.RotaQuery,
) ([]workerscheduleports.RotaWorkerRow, error) {
	return r.workers, nil
}

func (r *fakeSchedulingRepo) RotaTimeOffRanges(
	_ context.Context,
	_ *workerscheduleports.RotaQuery,
) ([]workerscheduleports.RotaRangeRow, error) {
	return r.timeOff, nil
}

func (r *fakeSchedulingRepo) RotaLeaveRanges(
	_ context.Context,
	_ *workerscheduleports.RotaQuery,
) ([]workerscheduleports.RotaRangeRow, error) {
	return r.leave, nil
}

func (r *fakeSchedulingRepo) RotaAssignedDays(
	_ context.Context,
	_ *workerscheduleports.RotaQuery,
) ([]workerscheduleports.RotaDayRow, error) {
	return r.assigned, nil
}

func newService(repo *fakeSchedulingRepo) *schedulingservice.Service {
	return schedulingservice.NewWithDeps(schedulingservice.Deps{Repo: repo})
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func activeTemplate(cycleWeeks int16) *workerschedule.ShiftTemplate {
	return &workerschedule.ShiftTemplate{
		ID:              pulid.MustNew("shft_"),
		Status:          domaintypes.StatusActive,
		Code:            "DAY",
		Name:            "Weekday",
		DaysOfWeek:      "0111110",
		StartMinute:     360,
		DurationMinutes: 600,
		CycleWeeks:      cycleWeeks,
	}
}

func mondayEpoch() int64 {
	return time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC).Unix()
}

// Two open assignments would put a worker on two patterns at once, and the
// rota would silently pick one.
func TestAssignShift_EndsThePriorAssignmentTheDayBefore(t *testing.T) {
	t.Parallel()

	template := activeTemplate(1)
	prior := &workerschedule.WorkerShiftAssignment{
		ID:              pulid.MustNew("wsa_"),
		WorkerID:        "wrk_1",
		ShiftTemplateID: template.ID,
		EffectiveFrom:   mondayEpoch() - 30*day,
	}
	repo := &fakeSchedulingRepo{
		template:    template,
		assignments: []*workerschedule.WorkerShiftAssignment{prior},
	}

	created, err := newService(repo).AssignShift(t.Context(), &schedulingservice.AssignShiftRequest{
		Entity: &workerschedule.WorkerShiftAssignment{
			WorkerID:        "wrk_1",
			ShiftTemplateID: template.ID,
			EffectiveFrom:   mondayEpoch(),
		},
		TenantInfo: tenant(),
	})

	require.NoError(t, err)
	assert.Equal(t, mondayEpoch(), created.EffectiveFrom)
	require.Len(t, repo.updated, 1)
	require.NotNil(t, repo.updated[0].EffectiveTo)
	assert.Equal(t, mondayEpoch()-day, *repo.updated[0].EffectiveTo)
}

// An assignment that has not started yet cannot be ended before it began, so
// the overlap is refused rather than written as a backwards period.
func TestAssignShift_RefusesAnAssignmentStartingLater(t *testing.T) {
	t.Parallel()

	template := activeTemplate(1)
	future := &workerschedule.WorkerShiftAssignment{
		ID:              pulid.MustNew("wsa_"),
		WorkerID:        "wrk_1",
		ShiftTemplateID: template.ID,
		EffectiveFrom:   mondayEpoch() + 30*day,
	}
	repo := &fakeSchedulingRepo{
		template:    template,
		assignments: []*workerschedule.WorkerShiftAssignment{future},
	}

	_, err := newService(repo).AssignShift(t.Context(), &schedulingservice.AssignShiftRequest{
		Entity: &workerschedule.WorkerShiftAssignment{
			WorkerID:        "wrk_1",
			ShiftTemplateID: template.ID,
			EffectiveFrom:   mondayEpoch(),
		},
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has a shift starting")
	assert.Empty(t, repo.updated)
}

// An offset beyond the cycle wraps to a week the pattern never reaches, which
// would roster somebody onto nothing.
func TestAssignShift_RefusesAnOffsetBeyondTheCycle(t *testing.T) {
	t.Parallel()

	template := activeTemplate(2)
	repo := &fakeSchedulingRepo{template: template}

	_, err := newService(repo).AssignShift(t.Context(), &schedulingservice.AssignShiftRequest{
		Entity: &workerschedule.WorkerShiftAssignment{
			WorkerID:         "wrk_1",
			ShiftTemplateID:  template.ID,
			EffectiveFrom:    mondayEpoch(),
			CycleOffsetWeeks: 2,
		},
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "offset is 0 to 1")
}

func TestAssignShift_RefusesARetiredShift(t *testing.T) {
	t.Parallel()

	template := activeTemplate(1)
	template.Status = domaintypes.StatusInactive
	repo := &fakeSchedulingRepo{template: template}

	_, err := newService(repo).AssignShift(t.Context(), &schedulingservice.AssignShiftRequest{
		Entity: &workerschedule.WorkerShiftAssignment{
			WorkerID:        "wrk_1",
			ShiftTemplateID: template.ID,
			EffectiveFrom:   mondayEpoch(),
		},
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "retired")
}

// Retiring a pattern people are still on would empty their rota rather than
// move them.
func TestUpdateTemplate_RefusesRetiringAShiftPeopleAreOn(t *testing.T) {
	t.Parallel()

	template := activeTemplate(1)
	repo := &fakeSchedulingRepo{template: template, assignmentCount: 3}

	retired := activeTemplate(1)
	retired.ID = template.ID
	retired.Status = domaintypes.StatusInactive

	_, err := newService(repo).UpdateTemplate(t.Context(), &schedulingservice.UpdateTemplateRequest{
		Entity:     retired,
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "3 worker(s) are still on this shift")
}

func TestTransitionSwap_EnforcesTheStateMachine(t *testing.T) {
	t.Parallel()

	// The office cannot approve a swap the colleague has not answered.
	repo := &fakeSchedulingRepo{swap: &workerschedule.ShiftSwapRequest{
		ID:                   pulid.MustNew("sswp_"),
		RequestingWorkerID:   "wrk_1",
		CounterpartyWorkerID: "wrk_2",
		Status:               workerschedule.SwapProposed,
		ShiftDate:            mondayEpoch(),
	}}

	_, err := newService(repo).TransitionSwap(t.Context(), &schedulingservice.TransitionSwapRequest{
		ID:         "sswp_1",
		Status:     workerschedule.SwapApproved,
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be Approved")
	assert.Nil(t, repo.savedSwap)
}

func TestTransitionSwap_StampsTheDecision(t *testing.T) {
	t.Parallel()

	repo := &fakeSchedulingRepo{swap: &workerschedule.ShiftSwapRequest{
		ID:                   pulid.MustNew("sswp_"),
		RequestingWorkerID:   "wrk_1",
		CounterpartyWorkerID: "wrk_2",
		Status:               workerschedule.SwapAccepted,
		ShiftDate:            mondayEpoch(),
	}}
	decider := pulid.MustNew("usr_")

	updated, err := newService(repo).TransitionSwap(
		t.Context(),
		&schedulingservice.TransitionSwapRequest{
			ID:         "sswp_1",
			Status:     workerschedule.SwapApproved,
			TenantInfo: tenant(),
			UserID:     decider,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, updated.DecidedAt)
	assert.Positive(t, *updated.DecidedAt)
	assert.Equal(t, decider, updated.DecidedByID)
}

// A driver answering on somebody else's behalf would let one person accept a
// swap that was never offered to them.
func TestTransitionSwap_OnlyTheCounterpartyCanAnswer(t *testing.T) {
	t.Parallel()

	repo := &fakeSchedulingRepo{swap: &workerschedule.ShiftSwapRequest{
		ID:                   pulid.MustNew("sswp_"),
		RequestingWorkerID:   "wrk_1",
		CounterpartyWorkerID: "wrk_2",
		Status:               workerschedule.SwapProposed,
		ShiftDate:            mondayEpoch(),
	}}

	_, err := newService(repo).TransitionSwap(t.Context(), &schedulingservice.TransitionSwapRequest{
		ID:            "sswp_1",
		Status:        workerschedule.SwapAccepted,
		ActorWorkerID: "wrk_3",
		TenantInfo:    tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "offered to")
}

func TestTransitionSwap_OnlyTheRequesterCanWithdraw(t *testing.T) {
	t.Parallel()

	repo := &fakeSchedulingRepo{swap: &workerschedule.ShiftSwapRequest{
		ID:                   pulid.MustNew("sswp_"),
		RequestingWorkerID:   "wrk_1",
		CounterpartyWorkerID: "wrk_2",
		Status:               workerschedule.SwapProposed,
		ShiftDate:            mondayEpoch(),
	}}

	_, err := newService(repo).TransitionSwap(t.Context(), &schedulingservice.TransitionSwapRequest{
		ID:            "sswp_1",
		Status:        workerschedule.SwapWithdrawn,
		ActorWorkerID: "wrk_2",
		TenantInfo:    tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "who proposed a swap")
}

func TestTransitionSwap_ADriverCannotDecideTheirOwnSwap(t *testing.T) {
	t.Parallel()

	repo := &fakeSchedulingRepo{swap: &workerschedule.ShiftSwapRequest{
		ID:                   pulid.MustNew("sswp_"),
		RequestingWorkerID:   "wrk_1",
		CounterpartyWorkerID: "wrk_2",
		Status:               workerschedule.SwapAccepted,
		ShiftDate:            mondayEpoch(),
	}}

	_, err := newService(repo).TransitionSwap(t.Context(), &schedulingservice.TransitionSwapRequest{
		ID:            "sswp_1",
		Status:        workerschedule.SwapApproved,
		ActorWorkerID: "wrk_1",
		TenantInfo:    tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decided by the office")
}

func TestProposeSwap_RefusesADayThatHasPassed(t *testing.T) {
	t.Parallel()

	repo := &fakeSchedulingRepo{}

	_, err := newService(repo).ProposeSwap(t.Context(), &schedulingservice.ProposeSwapRequest{
		Entity: &workerschedule.ShiftSwapRequest{
			RequestingWorkerID:   "wrk_1",
			CounterpartyWorkerID: "wrk_2",
			ShiftDate:            time.Now().AddDate(0, 0, -3).Unix(),
		},
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already passed")
}

// A leave case with no return date runs on. Stopping it on its start day would
// show a driver back at work the next morning.
func TestRota_OpenEndedLeaveCoversTheRestOfTheWeek(t *testing.T) {
	t.Parallel()

	template := activeTemplate(1)
	repo := &fakeSchedulingRepo{
		templates: []*workerschedule.ShiftTemplate{template},
		workers: []workerscheduleports.RotaWorkerRow{{
			WorkerID:        "wrk_1",
			FirstName:       "Ada",
			LastName:        "Byron",
			ShiftTemplateID: template.ID,
		}},
		leave: []workerscheduleports.RotaRangeRow{{
			WorkerID: "wrk_1",
			StartsAt: mondayEpoch(),
			EndsAt:   0,
		}},
	}

	rota, err := newService(repo).Rota(t.Context(), &schedulingservice.RotaRequest{
		TenantInfo: tenant(),
		At:         mondayEpoch(),
	})

	require.NoError(t, err)
	require.Len(t, rota.Rows, 1)
	row := rota.Rows[0]
	// The week starts on the Sunday before, so Monday through Saturday are the
	// days the open case covers.
	assert.Equal(t, workerschedule.RotaOff, row.Days[0].State)
	for i := 1; i < 7; i++ {
		assert.Equal(t, workerschedule.RotaLeave, row.Days[i].State, "day %d", i)
	}
	assert.Equal(t, 5, row.Conflicts)
}

// An A/B rotation drawn several weeks out has to alternate rather than flip
// midweek: the seven-day buckets counted from the epoch do not begin on a
// Sunday.
func TestRota_AlternatesAcrossWeeks(t *testing.T) {
	t.Parallel()

	template := activeTemplate(2)
	repo := &fakeSchedulingRepo{
		templates: []*workerschedule.ShiftTemplate{template},
		workers: []workerscheduleports.RotaWorkerRow{{
			WorkerID:        "wrk_1",
			FirstName:       "Ada",
			LastName:        "Byron",
			ShiftTemplateID: template.ID,
		}},
	}

	rota, err := newService(repo).Rota(t.Context(), &schedulingservice.RotaRequest{
		TenantInfo: tenant(),
		At:         mondayEpoch(),
		Weeks:      2,
	})

	require.NoError(t, err)
	require.Len(t, rota.Rows, 1)
	require.Len(t, rota.Rows[0].Days, 14)
	assert.Equal(t, 2, rota.Weeks)

	firstWeek := scheduledIn(rota.Rows[0].Days[:7])
	secondWeek := scheduledIn(rota.Rows[0].Days[7:])
	assert.NotEqual(t, firstWeek > 0, secondWeek > 0, "exactly one of the two weeks is on")
	assert.Equal(t, 5, firstWeek+secondWeek)
}

func scheduledIn(days []workerschedule.RotaDay) int {
	count := 0
	for _, day := range days {
		if day.Scheduled {
			count++
		}
	}
	return count
}

// A preference is carried through even when something stronger decided the
// state, so the rota shows where dispatch overrode it rather than hiding it.
func TestRota_CountsAnAssignmentOnADayOffAsAConflict(t *testing.T) {
	t.Parallel()

	template := activeTemplate(1)
	tuesday := mondayEpoch() + day
	repo := &fakeSchedulingRepo{
		templates: []*workerschedule.ShiftTemplate{template},
		workers: []workerscheduleports.RotaWorkerRow{{
			WorkerID:        "wrk_1",
			FirstName:       "Ada",
			LastName:        "Byron",
			ShiftTemplateID: template.ID,
		}},
		timeOff: []workerscheduleports.RotaRangeRow{{
			WorkerID: "wrk_1",
			StartsAt: tuesday,
			EndsAt:   tuesday,
		}},
		assigned: []workerscheduleports.RotaDayRow{{
			WorkerID: "wrk_1",
			DayStart: tuesday,
			Count:    2,
		}},
	}

	rota, err := newService(repo).Rota(t.Context(), &schedulingservice.RotaRequest{
		TenantInfo: tenant(),
		At:         mondayEpoch(),
	})

	require.NoError(t, err)
	tuesdayCell := rota.Rows[0].Days[2]
	assert.Equal(t, workerschedule.RotaTimeOff, tuesdayCell.State)
	assert.Equal(t, 2, tuesdayCell.AssignmentCount)
	assert.True(t, tuesdayCell.IsConflict())
	assert.Equal(t, 1, rota.Conflicts)
}
