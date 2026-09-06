package timesheetservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/timesheetservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(86400)

type fakeTimesheetRepo struct {
	repositories.TimesheetRepository

	openEntry  *worker.TimeClockEntry
	entries    []*worker.TimeClockEntry
	sheet      *worker.Timesheet
	sheets     []*worker.Timesheet
	saved      *worker.Timesheet
	export     *worker.PayrollExport
	stamped    []pulid.ID
	stampCount int
	cleared    int
	deleted    []pulid.ID
	leave      []repositories.RotaRangeRow
}

func (r *fakeTimesheetRepo) GetOpenEntry(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*worker.TimeClockEntry, error) {
	return r.openEntry, nil
}

func (r *fakeTimesheetRepo) CreateEntry(
	_ context.Context,
	entity *worker.TimeClockEntry,
) (*worker.TimeClockEntry, error) {
	entity.ID = pulid.MustNew("tce_")
	r.entries = append(r.entries, entity)
	return entity, nil
}

func (r *fakeTimesheetRepo) UpdateEntry(
	_ context.Context,
	entity *worker.TimeClockEntry,
) (*worker.TimeClockEntry, error) {
	return entity, nil
}

func (r *fakeTimesheetRepo) GetEntryByID(
	_ context.Context,
	req *repositories.GetTimeClockEntryByIDRequest,
) (*worker.TimeClockEntry, error) {
	for _, entry := range r.entries {
		if entry.ID == req.ID {
			return entry, nil
		}
	}
	return r.openEntry, nil
}

func (r *fakeTimesheetRepo) DeleteEntry(
	_ context.Context,
	req *repositories.GetTimeClockEntryByIDRequest,
) error {
	r.deleted = append(r.deleted, req.ID)
	return nil
}

func (r *fakeTimesheetRepo) ListEntries(
	_ context.Context,
	_ *repositories.ListTimeClockEntriesRequest,
) ([]*worker.TimeClockEntry, error) {
	return r.entries, nil
}

func (r *fakeTimesheetRepo) GetOrCreateTimesheet(
	_ context.Context,
	entity *worker.Timesheet,
) (*worker.Timesheet, error) {
	if r.sheet != nil {
		return r.sheet, nil
	}
	entity.ID = pulid.MustNew("tsh_")
	r.sheet = entity
	return entity, nil
}

func (r *fakeTimesheetRepo) GetTimesheetByID(
	_ context.Context,
	_ *repositories.GetTimesheetByIDRequest,
) (*worker.Timesheet, error) {
	return r.sheet, nil
}

func (r *fakeTimesheetRepo) UpdateTimesheet(
	_ context.Context,
	entity *worker.Timesheet,
) (*worker.Timesheet, error) {
	r.saved = entity
	r.sheet = entity
	return entity, nil
}

func (r *fakeTimesheetRepo) AttachEntries(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_ pulid.ID,
	_, _ int64,
) (int, error) {
	return len(r.entries), nil
}

func (r *fakeTimesheetRepo) ApprovedLeaveRanges(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_, _ int64,
) ([]repositories.RotaRangeRow, error) {
	return r.leave, nil
}

func (r *fakeTimesheetRepo) ListTimesheets(
	_ context.Context,
	_ *repositories.ListTimesheetsRequest,
) ([]*worker.Timesheet, error) {
	return r.sheets, nil
}

func (r *fakeTimesheetRepo) CreateExport(
	_ context.Context,
	entity *worker.PayrollExport,
) (*worker.PayrollExport, error) {
	entity.ID = pulid.MustNew("pxb_")
	r.export = entity
	return entity, nil
}

func (r *fakeTimesheetRepo) UpdateExport(
	_ context.Context,
	entity *worker.PayrollExport,
) (*worker.PayrollExport, error) {
	r.export = entity
	return entity, nil
}

func (r *fakeTimesheetRepo) GetExportByID(
	_ context.Context,
	_ *repositories.GetPayrollExportByIDRequest,
) (*worker.PayrollExport, error) {
	return r.export, nil
}

func (r *fakeTimesheetRepo) StampExport(
	_ context.Context,
	req *repositories.StampExportRequest,
) (int, error) {
	r.stamped = req.TimesheetIDs
	if r.stampCount > 0 {
		return r.stampCount, nil
	}
	return len(req.TimesheetIDs), nil
}

func (r *fakeTimesheetRepo) ClearExport(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (int, error) {
	r.cleared++
	return 1, nil
}

type fakeWorkerReader struct {
	workerType worker.WorkerType
}

func (r *fakeWorkerReader) GetByID(
	_ context.Context,
	_ repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	return &worker.Worker{ID: "wrk_1", Type: r.workerType}, nil
}

func newService(repo *fakeTimesheetRepo, reader *fakeWorkerReader) *timesheetservice.Service {
	return timesheetservice.NewWithDeps(timesheetservice.Deps{
		Repo:       repo,
		WorkerRepo: reader,
	})
}

func employee() *fakeWorkerReader {
	return &fakeWorkerReader{workerType: worker.WorkerTypeEmployee}
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

// A Monday in the past: a punch dated forward is refused, so the fixture week
// has to be one that has already happened.
func mondayAt(hour int) int64 {
	return time.Date(2026, time.August, 31, hour, 0, 0, 0, time.UTC).Unix()
}

func weekStart() int64 {
	return worker.StartOfWeekUTC(mondayAt(0), time.UTC)
}

func openSheet() *worker.Timesheet {
	return &worker.Timesheet{
		ID:                       pulid.MustNew("tsh_"),
		WorkerID:                 "wrk_1",
		Status:                   worker.TimesheetOpen,
		PeriodStart:              weekStart(),
		PeriodEnd:                weekStart() + 7*day,
		OvertimeThresholdMinutes: 2400,
	}
}

func entry(startHour, endHour int) *worker.TimeClockEntry {
	end := mondayAt(endHour)
	return &worker.TimeClockEntry{
		ID:           pulid.MustNew("tce_"),
		WorkerID:     "wrk_1",
		Source:       worker.TimeEntrySourceClock,
		ClockedInAt:  mondayAt(startHour),
		ClockedOutAt: &end,
	}
}

// A contractor invoices. Recording their hours here would produce a wage
// record for somebody who is not owed wages.
func TestClockIn_RefusesAContractor(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{}
	service := newService(repo, &fakeWorkerReader{workerType: worker.WorkerTypeContractor})

	_, err := service.ClockIn(t.Context(), &timesheetservice.ClockRequest{
		WorkerID:   "wrk_1",
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "a contractor invoices instead")
}

// Two open punches is somebody being paid twice for the same hour.
func TestClockIn_RefusesASecondPunch(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{openEntry: &worker.TimeClockEntry{
		ID:          pulid.MustNew("tce_"),
		WorkerID:    "wrk_1",
		ClockedInAt: mondayAt(8),
	}}

	_, err := newService(repo, employee()).ClockIn(t.Context(), &timesheetservice.ClockRequest{
		WorkerID:   "wrk_1",
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Already on the clock")
}

// A punch dated forward would sit on a week nobody can approve yet and would
// grow on its own until somebody closed it.
func TestClockIn_RefusesAFuturePunch(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{}

	_, err := newService(repo, employee()).ClockIn(t.Context(), &timesheetservice.ClockRequest{
		WorkerID:   "wrk_1",
		At:         time.Now().Add(2 * time.Hour).Unix(),
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dated in the future")
}

func TestClockIn_OpensTheWeekAndAttachesThePunch(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{}

	created, err := newService(repo, employee()).ClockIn(
		t.Context(),
		&timesheetservice.ClockRequest{
			WorkerID:   "wrk_1",
			At:         mondayAt(8),
			TenantInfo: tenant(),
		},
	)

	require.NoError(t, err)
	require.NotNil(t, repo.sheet)
	// The week starts on the Sunday the rota starts on, so a timesheet and a
	// rota row line up without anybody converting between them.
	assert.Equal(t, weekStart(), repo.sheet.PeriodStart)
	assert.Equal(t, repo.sheet.ID, created.TimesheetID)
	assert.True(t, created.IsOpen())
}

func TestClockOut_RefusesWhenNobodyIsOnTheClock(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{}

	_, err := newService(repo, employee()).ClockOut(t.Context(), &timesheetservice.ClockRequest{
		WorkerID:   "wrk_1",
		TenantInfo: tenant(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Not on the clock")
}

func TestClockOut_RollsTheWeekUp(t *testing.T) {
	t.Parallel()

	sheet := openSheet()
	repo := &fakeTimesheetRepo{
		sheet: sheet,
		openEntry: &worker.TimeClockEntry{
			ID:          pulid.MustNew("tce_"),
			WorkerID:    "wrk_1",
			Source:      worker.TimeEntrySourceClock,
			TimesheetID: sheet.ID,
			ClockedInAt: mondayAt(8),
		},
	}
	repo.entries = []*worker.TimeClockEntry{entry(8, 17)}

	updated, err := newService(repo, employee()).ClockOut(
		t.Context(),
		&timesheetservice.ClockRequest{
			WorkerID:     "wrk_1",
			At:           mondayAt(17),
			BreakMinutes: 30,
			TenantInfo:   tenant(),
		},
	)

	require.NoError(t, err)
	assert.False(t, updated.IsOpen())
	require.NotNil(t, repo.saved)
	assert.Equal(t, int32(540), repo.saved.RegularMinutes)
	assert.Equal(t, int32(1), repo.saved.EntryCount)
}

// A wage record altered by somebody with no reason recorded is not a record
// anybody can defend.
func TestRecordEntry_NeedsAReason(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{sheet: openSheet()}

	_, err := newService(repo, employee()).RecordEntry(
		t.Context(),
		&timesheetservice.RecordEntryRequest{
			WorkerID:     "wrk_1",
			ClockedInAt:  mondayAt(8),
			ClockedOutAt: mondayAt(17),
			TenantInfo:   tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs a reason")
}

// The totals a manager is looking at have to be the ones they are being asked
// to sign.
func TestRecordEntry_RefusedOnceTheWeekIsSubmitted(t *testing.T) {
	t.Parallel()

	sheet := openSheet()
	sheet.Status = worker.TimesheetSubmitted
	repo := &fakeTimesheetRepo{sheet: sheet}

	_, err := newService(repo, employee()).RecordEntry(
		t.Context(),
		&timesheetservice.RecordEntryRequest{
			WorkerID:     "wrk_1",
			ClockedInAt:  mondayAt(8),
			ClockedOutAt: mondayAt(17),
			Reason:       "Missed punch",
			TenantInfo:   tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been Submitted")
}

func TestDeleteEntry_NeedsAReason(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{sheet: openSheet()}

	err := newService(repo, employee()).DeleteEntry(
		t.Context(),
		&timesheetservice.DeleteEntryRequest{ID: "tce_1", TenantInfo: tenant()},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs a reason")
	assert.Empty(t, repo.deleted)
}

func TestTransition_EnforcesTheStateMachine(t *testing.T) {
	t.Parallel()

	sheet := openSheet()
	sheet.RegularMinutes = 2400
	repo := &fakeTimesheetRepo{sheet: sheet}

	_, err := newService(repo, employee()).Transition(
		t.Context(),
		&timesheetservice.TransitionTimesheetRequest{
			ID:         sheet.ID,
			Status:     worker.TimesheetApproved,
			TenantInfo: tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be Approved")
}

// A zero-hour line in the payroll file for somebody who was simply away is a
// line somebody has to explain.
func TestTransition_RefusesAnEmptyWeek(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{sheet: openSheet()}

	_, err := newService(repo, employee()).Transition(
		t.Context(),
		&timesheetservice.TransitionTimesheetRequest{
			ID:         repo.sheet.ID,
			Status:     worker.TimesheetSubmitted,
			TenantInfo: tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no hours on this week")
}

// Submitting is the last roll-up. After it the totals are the record rather
// than a summary of one.
func TestTransition_FreezesTheTotalsAtSubmit(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{
		sheet:   openSheet(),
		entries: []*worker.TimeClockEntry{entry(8, 17), entry(18, 20)},
	}
	service := newService(repo, employee())

	submitted, err := service.Transition(t.Context(), &timesheetservice.TransitionTimesheetRequest{
		ID:         repo.sheet.ID,
		Status:     worker.TimesheetSubmitted,
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
	})

	require.NoError(t, err)
	assert.Equal(t, worker.TimesheetSubmitted, submitted.Status)
	require.NotNil(t, submitted.SubmittedAt)
	assert.Equal(t, int32(660), submitted.RegularMinutes)

	// A further punch cannot move them: the sheet is no longer editable, so a
	// later roll-up leaves the frozen numbers alone.
	repo.entries = append(repo.entries, entry(20, 22))
	rejected, err := service.Transition(t.Context(), &timesheetservice.TransitionTimesheetRequest{
		ID:         repo.sheet.ID,
		Status:     worker.TimesheetRejected,
		Note:       "Check Monday",
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
	})

	require.NoError(t, err)
	assert.Equal(t, int32(660), rejected.RegularMinutes)
}

// A worker hands their own week over and nothing else. Letting them approve it
// would make the sign-off meaningless.
func TestTransition_AWorkerCannotApproveTheirOwnWeek(t *testing.T) {
	t.Parallel()

	sheet := openSheet()
	sheet.Status = worker.TimesheetSubmitted
	repo := &fakeTimesheetRepo{sheet: sheet}

	_, err := newService(repo, employee()).Transition(
		t.Context(),
		&timesheetservice.TransitionTimesheetRequest{
			ID:            sheet.ID,
			Status:        worker.TimesheetApproved,
			ActorWorkerID: "wrk_1",
			TenantInfo:    tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "approved by a manager")
}

func TestTransition_AWorkerCannotSubmitSomebodyElsesWeek(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{sheet: openSheet()}

	_, err := newService(repo, employee()).Transition(
		t.Context(),
		&timesheetservice.TransitionTimesheetRequest{
			ID:            repo.sheet.ID,
			Status:        worker.TimesheetSubmitted,
			ActorWorkerID: "wrk_2",
			TenantInfo:    tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not your timesheet")
}

// A resubmission is a fresh decision, so the old approval is cleared rather
// than left to read as a sign-off on numbers that have since changed.
func TestTransition_ResubmissionClearsTheOldApproval(t *testing.T) {
	t.Parallel()

	approvedAt := mondayAt(9)
	sheet := openSheet()
	sheet.Status = worker.TimesheetRejected
	sheet.ApprovedAt = &approvedAt
	sheet.ApprovedByID = pulid.MustNew("usr_")
	repo := &fakeTimesheetRepo{sheet: sheet, entries: []*worker.TimeClockEntry{entry(8, 17)}}

	updated, err := newService(repo, employee()).Transition(
		t.Context(),
		&timesheetservice.TransitionTimesheetRequest{
			ID:         sheet.ID,
			Status:     worker.TimesheetSubmitted,
			TenantInfo: tenant(),
		},
	)

	require.NoError(t, err)
	assert.Nil(t, updated.ApprovedAt)
	assert.True(t, updated.ApprovedByID.IsNil())
}

func TestGenerateExport_RefusesAPeriodWithNothingApproved(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{}

	_, err := newService(repo, employee()).GenerateExport(
		t.Context(),
		&timesheetservice.GenerateExportRequest{
			PeriodStart: weekStart(),
			PeriodEnd:   weekStart() + 7*day,
			TenantInfo:  tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no approved timesheets")
}

// A period that could be exported twice is a period somebody gets paid twice
// for, so the sheets are locked to the run rather than merely listed.
func TestGenerateExport_LocksTheSheetsItCarried(t *testing.T) {
	t.Parallel()

	first := openSheet()
	first.Status = worker.TimesheetApproved
	first.RegularMinutes = 2400
	first.OvertimeMinutes = 120

	second := openSheet()
	second.ID = pulid.MustNew("tsh_")
	second.Status = worker.TimesheetApproved
	second.RegularMinutes = 1800

	repo := &fakeTimesheetRepo{sheets: []*worker.Timesheet{first, second}}

	export, err := newService(repo, employee()).GenerateExport(
		t.Context(),
		&timesheetservice.GenerateExportRequest{
			PeriodStart: weekStart(),
			PeriodEnd:   weekStart() + 7*day,
			TenantInfo:  tenant(),
			UserID:      pulid.MustNew("usr_"),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(2), export.TimesheetCount)
	assert.Equal(t, int32(4200), export.RegularMinutes)
	assert.Equal(t, int32(120), export.OvertimeMinutes)
	assert.Len(t, repo.stamped, 2)
	assert.Equal(t, worker.PayrollExportGenerated, export.Status)
}

// Voiding a run reopens every week in it, so it is refused without a reason.
func TestVoidExport_NeedsAReason(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{export: &worker.PayrollExport{
		ID:          pulid.MustNew("pxb_"),
		Status:      worker.PayrollExportGenerated,
		PeriodStart: weekStart(),
		PeriodEnd:   weekStart() + 7*day,
	}}

	_, err := newService(repo, employee()).VoidExport(
		t.Context(),
		&timesheetservice.VoidExportRequest{ID: "pxb_1", TenantInfo: tenant()},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs a reason")
	assert.Equal(t, 0, repo.cleared)
}

func TestVoidExport_ReopensTheWeeksItCarried(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{export: &worker.PayrollExport{
		ID:          pulid.MustNew("pxb_"),
		Status:      worker.PayrollExportGenerated,
		PeriodStart: weekStart(),
		PeriodEnd:   weekStart() + 7*day,
	}}

	voided, err := newService(repo, employee()).VoidExport(
		t.Context(),
		&timesheetservice.VoidExportRequest{
			ID:         "pxb_1",
			Reason:     "Wrong period",
			TenantInfo: tenant(),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, worker.PayrollExportVoided, voided.Status)
	require.NotNil(t, voided.VoidedAt)
	assert.Equal(t, 1, repo.cleared)
}

// A day of approved time off inside the week is one standard day of paid
// leave, and it sits outside the overtime split. A span that only touches the
// week counts the days it actually covers, not the whole span.
func TestClockOut_CountsApprovedLeaveInsideTheWeek(t *testing.T) {
	t.Parallel()

	sheet := openSheet()
	repo := &fakeTimesheetRepo{
		sheet: sheet,
		openEntry: &worker.TimeClockEntry{
			ID:          pulid.MustNew("tce_"),
			WorkerID:    "wrk_1",
			Source:      worker.TimeEntrySourceClock,
			TimesheetID: sheet.ID,
			ClockedInAt: mondayAt(8),
		},
		entries: []*worker.TimeClockEntry{entry(8, 17)},
		// Friday before the week through the Monday of it: one day inside.
		leave: []repositories.RotaRangeRow{{
			WorkerID: "wrk_1",
			StartsAt: weekStart() - 2*day,
			EndsAt:   weekStart() + day,
		}},
	}

	_, err := newService(repo, employee()).ClockOut(t.Context(), &timesheetservice.ClockRequest{
		WorkerID:   "wrk_1",
		At:         mondayAt(17),
		TenantInfo: tenant(),
	})

	require.NoError(t, err)
	require.NotNil(t, repo.saved)
	// Two days inside the window (Sunday and Monday) at 480 minutes each.
	assert.Equal(t, int32(960), repo.saved.PaidLeaveMinutes)
	assert.Equal(t, int32(540), repo.saved.RegularMinutes)
}
