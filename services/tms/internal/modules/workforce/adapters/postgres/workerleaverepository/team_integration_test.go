//go:build integration

package workerleaverepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/internal/testutil/workerteamtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCasesNarrowToTheTeamAManagerAnswersFor(t *testing.T) {
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

	requestedAt := int64(1_700_000_000)
	for _, wrk := range []*worker.Worker{team.Direct, team.ViaFleet, team.Outsider} {
		err := workerdb.WorkerLeaveCases.Create(ctx, db, &worker.WorkerLeaveCase{
			ID:                       pulid.MustNew("wlc_"),
			OrganizationID:           data.Organization.ID,
			BusinessUnitID:           data.BusinessUnit.ID,
			WorkerID:                 wrk.ID,
			LeaveType:                worker.LeaveTypeFMLA,
			Status:                   worker.LeaveCasePending,
			Frequency:                worker.LeaveContinuous,
			RequestedAt:              1_700_000_000,
			StartsAt:                 1_700_000_000,
			CertificationStatus:      worker.CertificationRequested,
			CertificationRequestedAt: &requestedAt,
		})
		require.NoError(t, err)
	}

	outstanding := func(managerIDs []pulid.ID) *workerports.ListLeaveCasesRequest {
		return &workerports.ListLeaveCasesRequest{
			TenantInfo:                   tenant,
			CertificationOutstandingOnly: true,
			IncludeWorker:                true,
			ManagerIDs:                   managerIDs,
		}
	}

	cases, err := repo.ListCases(ctx, outstanding(team.ManagerIDs()))
	require.NoError(t, err)
	workers := make([]pulid.ID, 0, len(cases))
	for _, leave := range cases {
		workers = append(workers, leave.WorkerID)
	}
	assert.ElementsMatch(t, []pulid.ID{team.Direct.ID, team.ViaFleet.ID}, workers,
		"a driver manager's leave queue holds only the workers they manage, directly or by fleet")

	count, err := repo.CountCases(ctx, outstanding(team.ManagerIDs()))
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	count, err = repo.CountCases(ctx, outstanding(nil))
	require.NoError(t, err)
	assert.Equal(t, 3, count, "Human resources, with no managers, sees every case")
}
