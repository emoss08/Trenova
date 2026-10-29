//go:build integration

package workercredentialrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential/workercredentialdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workercredentialports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedCredentialType(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *workercredential.WorkerCredentialType {
	t.Helper()

	entity := &workercredential.WorkerCredentialType{Code: code, Name: code + " credential"}
	require.NoError(t, workercredentialdb.WorkerCredentialTypes.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetTypesByIDsReturnsOnlyTheTenantsTypes(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "CQ")
	first := seedCredentialType(t, ctx, db, tenantA, "HAZMAT")
	second := seedCredentialType(t, ctx, db, tenantA, "TWIC")
	foreign := seedCredentialType(t, ctx, db, tenantB, "PASSPORT")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetTypesByIDs(
		tenantA.Ctx(ctx),
		&workercredentialports.GetCredentialTypesByIDsRequest{
			TenantInfo: tenantA.Info(),
			TypeIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("wct_")},
		},
	)
	require.NoError(t, err)

	keys := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		keys[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "HAZMAT", second.ID: "TWIC"}, keys)

	none, err := repo.GetTypesByIDs(
		tenantA.Ctx(ctx),
		&workercredentialports.GetCredentialTypesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
