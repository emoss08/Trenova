//go:build integration

package sequenceconfigrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSequenceConfigReadsWriteOnlyOnceAndAreCachedUntilAnUpdate(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	cached := cachetest.Open(t, ctx, dsn)
	repo := New(Params{DB: postgres.NewTestConnection(cached), Logger: zap.NewNop()})
	tctx := data.Ctx(ctx)
	req := tenantports.GetSequenceConfigRequest{TenantInfo: data.Info()}

	var doc *tenant.SequenceConfigDocument
	read := func() error {
		var readErr error
		doc, readErr = repo.GetByTenant(tctx, req)
		return readErr
	}

	require.NoError(t, read())
	require.Len(t, doc.Configs, len(tenant.RequiredSequenceTypes()),
		"the first read writes any missing default")
	require.NoError(t, read())

	cachetest.RequireHit(t, cached, read)
	require.Len(t, doc.Configs, len(tenant.RequiredSequenceTypes()))

	doc.Configs[0].Prefix = "CACHED"
	updated, err := repo.UpdateByTenant(tctx, doc)
	require.NoError(t, err)
	assert.Equal(t, "CACHED", updated.Configs[0].Prefix)

	cachetest.RequireMiss(t, cached, read)
	cachetest.RequireHit(t, cached, read)
	assert.Equal(t, "CACHED", doc.Configs[0].Prefix)
}
