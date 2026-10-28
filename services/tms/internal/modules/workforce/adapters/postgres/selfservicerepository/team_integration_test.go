//go:build integration

package selfservicerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange/profilechangedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/selfserviceports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/internal/testutil/workerteamtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestChangeRequestsNarrowToTheTeamAManagerAnswersFor(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	team := workerteamtest.Seed(t, ctx, db, data)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	ctx = data.Ctx(ctx)

	for _, wrk := range []*worker.Worker{team.Direct, team.ViaFleet, team.Outsider} {
		err := profilechangedb.WorkerProfileChangeRequests.Create(
			ctx,
			db,
			&profilechange.WorkerProfileChangeRequest{
				ID:             pulid.MustNew("wpcr_"),
				OrganizationID: data.Organization.ID,
				BusinessUnitID: data.BusinessUnit.ID,
				WorkerID:       wrk.ID,
				Status:         profilechange.ProfileChangePending,
				Changes:        []profilechange.FieldChange{},
				SubmittedAt:    1_700_000_000,
			},
		)
		require.NoError(t, err)
	}

	workers := func(managerIDs []pulid.ID) []pulid.ID {
		requests, err := repo.ListChangeRequests(
			ctx,
			&selfserviceports.ListProfileChangeRequestsRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID:  data.Organization.ID,
					BuID:   data.BusinessUnit.ID,
					UserID: team.Manager.ID,
				},
				ManagerIDs: managerIDs,
			},
		)
		require.NoError(t, err)
		out := make([]pulid.ID, 0, len(requests))
		for _, request := range requests {
			out = append(out, request.WorkerID)
		}
		return out
	}

	assert.ElementsMatch(
		t,
		[]pulid.ID{team.Direct.ID, team.ViaFleet.ID},
		workers(team.ManagerIDs()),
	)
	assert.Len(t, workers(nil), 3)
}
