//go:build integration

package weekhoursrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet/timesheetdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule/workerscheduledb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/internal/testutil/workerteamtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheWeekIsReadFromTheRotaTheClockAndTheTimesheet(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	team := workerteamtest.Seed(t, ctx, db, data)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.WrapDB(db)})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	now := time.Date(2026, time.November, 24, 12, 0, 0, 0, time.UTC).Unix()
	start, _ := weekhours.Bounds(now, time.UTC)
	shift := func(code, days string, minutes int16) *workerschedule.ShiftTemplate {
		template := &workerschedule.ShiftTemplate{
			OrganizationID:  data.Organization.ID,
			BusinessUnitID:  data.BusinessUnit.ID,
			Status:          domaintypes.StatusActive,
			Code:            code,
			Name:            code,
			DaysOfWeek:      days,
			StartMinute:     360,
			DurationMinutes: minutes,
			CycleWeeks:      1,
		}
		require.NoError(t, workerscheduledb.ShiftTemplates.Create(ctx, db, template))
		return template
	}
	assign := func(wrk *worker.Worker, template *workerschedule.ShiftTemplate) {
		require.NoError(t, workerscheduledb.WorkerShiftAssignments.Create(ctx, db,
			&workerschedule.WorkerShiftAssignment{
				OrganizationID:  data.Organization.ID,
				BusinessUnitID:  data.BusinessUnit.ID,
				WorkerID:        wrk.ID,
				ShiftTemplateID: template.ID,
				EffectiveFrom:   start - 30*86400,
			}))
	}
	long := shift("LONG", "1111110", 540)
	short := shift("SHORT", "0011000", 480)
	assign(team.Direct, long)
	assign(team.ViaFleet, short)
	assign(team.Outsider, long)

	sheet := &timesheet.Timesheet{
		ID:                       pulid.MustNew("tsh_"),
		OrganizationID:           data.Organization.ID,
		BusinessUnitID:           data.BusinessUnit.ID,
		WorkerID:                 team.Direct.ID,
		Status:                   timesheet.TimesheetOpen,
		PeriodStart:              weekhours.TimesheetPeriod(start, time.UTC),
		PeriodEnd:                weekhours.TimesheetPeriod(start, time.UTC) + 7*86400,
		OvertimeThresholdMinutes: 2000,
	}
	require.NoError(t, timesheetdb.Timesheets.Create(ctx, db, sheet))
	for day := int64(0); day < 2; day++ {
		in := start + day*86400 + 6*3600
		out := in + 540*60
		require.NoError(t, timesheetdb.TimeClockEntries.Create(ctx, db, &timesheet.TimeClockEntry{
			ID:             pulid.MustNew("tce_"),
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			WorkerID:       team.Direct.ID,
			TimesheetID:    sheet.ID,
			ClockedInAt:    in,
			ClockedOutAt:   &out,
		}))
	}

	weeks, err := repo.Workers(ctx, &weekhoursports.WorkersRequest{
		Tenant:    tenant,
		WorkerIDs: []pulid.ID{team.Direct.ID, team.ViaFleet.ID},
		Now:       now,
		Location:  time.UTC,
	})
	require.NoError(t, err)
	require.Contains(t, weeks, team.Direct.ID)
	direct := weeks[team.Direct.ID].Week
	assert.Equal(t, int32(1080), direct.Worked)
	assert.Equal(t, int32(2160), direct.Remaining)
	assert.Equal(t, int32(2000), direct.Threshold, "the line is the timesheet's")
	assert.True(t, direct.Over())
	assert.Equal(t, "LONG", direct.Shift)
	assert.Equal(t, int32(960), weeks[team.ViaFleet.ID].Week.Planned())
	assert.Equal(t, int32(2400), weeks[team.ViaFleet.ID].Week.Threshold)

	leading, err := repo.Leading(ctx, &weekhoursports.LeadingRequest{
		Tenant: tenant, Now: now, Location: time.UTC, Limit: 8, Room: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), leading.OnRota)
	assert.True(t, leading.Exact)
	assert.Equal(t, int64(1), leading.Over, "the outsider missed two shifts and stays under")

	narrowed, err := repo.Leading(ctx, &weekhoursports.LeadingRequest{
		Tenant: tenant, ManagerIDs: team.ManagerIDs(), Now: now, Location: time.UTC, Limit: 8,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), narrowed.OnRota, "a team-scoped read leaves the outsider out")
	for _, row := range narrowed.Rows {
		assert.NotEqual(t, team.Outsider.ID, row.WorkerID)
	}
}
