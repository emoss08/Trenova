//go:build integration

package orgstructurerepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/orgstructureports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedJobPosition(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *worker.JobPosition {
	t.Helper()

	entity := &worker.JobPosition{Code: code, Title: code + " position"}
	require.NoError(t, workerdb.JobPositions.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetPositionsByIDsReturnsOnlyTheTenantsPositions(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "JP")
	first := seedJobPosition(t, ctx, db, tenantA, "DRIVER")
	second := seedJobPosition(t, ctx, db, tenantA, "DISPATCH")
	foreign := seedJobPosition(t, ctx, db, tenantB, "MECHANIC")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetPositionsByIDs(
		tenantA.Ctx(ctx),
		&orgstructureports.GetJobPositionsByIDsRequest{
			TenantInfo: tenantA.Info(),
			PositionIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("jpos_")},
		},
	)
	require.NoError(t, err)

	keys := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		keys[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "DRIVER", second.ID: "DISPATCH"}, keys)

	none, err := repo.GetPositionsByIDs(
		tenantA.Ctx(ctx),
		&orgstructureports.GetJobPositionsByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
