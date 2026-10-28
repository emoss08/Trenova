//go:build integration

package timesheetrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet/timesheetdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/internal/testutil/workerteamtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestListsNarrowToTheTeamAManagerAnswersFor(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	team := workerteamtest.Seed(t, ctx, db, data)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	ctx = data.Ctx(ctx)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: team.Manager.ID,
	}

	for _, wrk := range []*worker.Worker{team.Direct, team.ViaFleet, team.Outsider} {
		err := timesheetdb.Timesheets.Create(ctx, db, &timesheet.Timesheet{
			ID:             pulid.MustNew("tsh_"),
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			WorkerID:       wrk.ID,
			Status:         timesheet.TimesheetOpen,
			PeriodStart:    1_700_000_000,
			PeriodEnd:      1_700_604_800,
		})
		require.NoError(t, err)
		err = timesheetdb.TimeClockEntries.Create(ctx, db, &timesheet.TimeClockEntry{
			ID:             pulid.MustNew("tce_"),
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			WorkerID:       wrk.ID,
			ClockedInAt:    1_700_000_100,
		})
		require.NoError(t, err)
	}

	sheetWorkers := func(managerIDs []pulid.ID) []pulid.ID {
		sheets, err := repo.ListTimesheets(ctx, &workerports.ListTimesheetsRequest{
			TenantInfo: tenant,
			ManagerIDs: managerIDs,
		})
		require.NoError(t, err)
		out := make([]pulid.ID, 0, len(sheets))
		for _, sheet := range sheets {
			out = append(out, sheet.WorkerID)
		}
		return out
	}
	entryWorkers := func(managerIDs []pulid.ID) []pulid.ID {
		entries, err := repo.ListEntries(ctx, &workerports.ListTimeClockEntriesRequest{
			TenantInfo: tenant,
			OpenOnly:   true,
			ManagerIDs: managerIDs,
		})
		require.NoError(t, err)
		out := make([]pulid.ID, 0, len(entries))
		for _, entry := range entries {
			out = append(out, entry.WorkerID)
		}
		return out
	}

	assert.ElementsMatch(
		t,
		[]pulid.ID{team.Direct.ID, team.ViaFleet.ID},
		sheetWorkers(team.ManagerIDs()),
		"the team is the workers the manager manages directly or through the fleet code they run",
	)
	assert.ElementsMatch(
		t,
		[]pulid.ID{team.Direct.ID, team.ViaFleet.ID},
		entryWorkers(team.ManagerIDs()),
	)
	assert.Len(t, sheetWorkers(nil), 3, "no managers is the organization's list")
	assert.Len(t, entryWorkers(nil), 3)
}
