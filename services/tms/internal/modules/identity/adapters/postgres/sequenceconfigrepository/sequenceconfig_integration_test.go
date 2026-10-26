//go:build integration

package sequenceconfigrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetByTenantSeedsEveryRequiredSequenceOnceThenReadsThem(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})
	tctx := data.Ctx(ctx)
	req := tenantports.GetSequenceConfigRequest{TenantInfo: data.Info()}

	sc := tenantdb.SequenceConfigs
	_, err := dbtx.Write(tctx, conn, func(ctx context.Context) (int64, error) {
		return sc.Where(sc.SequenceType.In(tenant.RequiredSequenceTypes()...)).
			Delete(ctx, conn.Keel(ctx))
	})
	require.NoError(t, err)

	seeded, err := repo.GetByTenant(tctx, req)
	require.NoError(t, err)
	require.Len(t, seeded.Configs, len(tenant.RequiredSequenceTypes()))
	assert.Equal(t, data.Info().OrgID, seeded.OrganizationID)
	assert.Equal(t, data.Info().BuID, seeded.BusinessUnitID)
	for i := 1; i < len(seeded.Configs); i++ {
		assert.LessOrEqual(t,
			tenant.SequenceTypeSortOrder(seeded.Configs[i-1].SequenceType),
			tenant.SequenceTypeSortOrder(seeded.Configs[i].SequenceType),
			"configs are in the catalog's order")
	}

	again, err := repo.GetByTenant(tctx, req)
	require.NoError(t, err)
	require.Len(t, again.Configs, len(seeded.Configs))
	for i := range seeded.Configs {
		assert.Equal(t, seeded.Configs[i].ID, again.Configs[i].ID,
			"a second read returns the seeded rows instead of seeding again")
	}
}
