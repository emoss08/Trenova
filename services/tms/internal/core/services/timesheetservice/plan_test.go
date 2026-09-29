package timesheetservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/timesheetservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanGenerateExport_TotalsWithoutLocking(t *testing.T) {
	t.Parallel()

	sheet := openSheet()
	sheet.Status = worker.TimesheetApproved
	sheet.RegularMinutes = 2400
	repo := &fakeTimesheetRepo{sheets: []*worker.Timesheet{sheet}}

	plan, err := newService(repo, employee()).PlanGenerateExport(
		t.Context(),
		&timesheetservice.GenerateExportRequest{
			PeriodStart: weekStart(),
			PeriodEnd:   weekStart() + 7*day,
			TenantInfo:  tenant(),
			UserID:      pulid.MustNew("usr_"),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, int32(1), plan.Export.TimesheetCount)
	assert.Equal(t, int32(2400), plan.Export.RegularMinutes)
	assert.Equal(t, []pulid.ID{sheet.ID}, plan.TimesheetIDs)
	assert.Empty(t, repo.stamped, "a plan locks nothing")
}

func TestPlanVoidExport_ReopensNothingYet(t *testing.T) {
	t.Parallel()

	repo := &fakeTimesheetRepo{export: &worker.PayrollExport{
		ID:          pulid.MustNew("pxb_"),
		Status:      worker.PayrollExportGenerated,
		PeriodStart: weekStart(),
		PeriodEnd:   weekStart() + 7*day,
	}}

	change, err := newService(repo, employee()).PlanVoidExport(
		t.Context(),
		&timesheetservice.VoidExportRequest{ID: "pxb_1", Reason: "Wrong period",
			TenantInfo: tenant()},
	)
	require.NoError(t, err)
	assert.Equal(t, worker.PayrollExportVoided, change.After.Status)
	assert.Equal(t, worker.PayrollExportGenerated, change.Before.Status)
	assert.Equal(t, 0, repo.cleared)

	_, err = newService(repo, employee()).PlanVoidExport(
		t.Context(),
		&timesheetservice.VoidExportRequest{ID: "pxb_1", TenantInfo: tenant()},
	)
	require.Error(t, err, "a void needs a reason")
}
