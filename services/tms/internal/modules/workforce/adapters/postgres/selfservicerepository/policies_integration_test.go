//go:build integration

package selfservicerepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpolicy"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpolicy/workerpolicydb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/selfserviceports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedWorkerPolicy(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *workerpolicy.WorkerPolicy {
	t.Helper()

	entity := &workerpolicy.WorkerPolicy{Code: code, Title: code + " policy", Body: "Read and sign."}
	require.NoError(t, workerpolicydb.WorkerPolicies.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetPoliciesByIDsReturnsOnlyTheTenantsPolicies(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "WP")
	first := seedWorkerPolicy(t, ctx, db, tenantA, "HANDBOOK")
	second := seedWorkerPolicy(t, ctx, db, tenantA, "SAFETY")
	foreign := seedWorkerPolicy(t, ctx, db, tenantB, "CONDUCT")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetPoliciesByIDs(
		tenantA.Ctx(ctx),
		&selfserviceports.GetWorkerPoliciesByIDsRequest{
			TenantInfo: tenantA.Info(),
			PolicyIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("wpol_")},
		},
	)
	require.NoError(t, err)

	keys := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		keys[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "HANDBOOK", second.ID: "SAFETY"}, keys)

	none, err := repo.GetPoliciesByIDs(
		tenantA.Ctx(ctx),
		&selfserviceports.GetWorkerPoliciesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
