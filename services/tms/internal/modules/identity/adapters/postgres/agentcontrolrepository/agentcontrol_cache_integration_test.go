//go:build integration

package agentcontrolrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAgentControlIsCreatedOnceThenServedFromTheCacheUntilAnUpdate(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	cached := cachetest.Open(t, ctx, dsn)
	repo := New(Params{DB: postgres.NewTestConnection(cached), Logger: zap.NewNop()})
	tctx := data.Ctx(ctx)

	created, err := repo.GetOrCreate(tctx, data.Info())
	require.NoError(t, err)
	require.True(t, created.ID.IsNotNil(), "the first read finds or writes the row")

	var control *tenant.AgentControl
	read := func() error {
		var readErr error
		control, readErr = repo.GetOrCreate(tctx, data.Info())
		return readErr
	}

	require.NoError(t, read())
	cachetest.RequireHit(t, cached, read)
	assert.Equal(t, created.ID, control.ID)

	control.ShadowMode = !created.ShadowMode
	_, err = repo.Update(tctx, control)
	require.NoError(t, err)

	cachetest.RequireMiss(t, cached, read)
	cachetest.RequireHit(t, cached, read)
	assert.Equal(t, !created.ShadowMode, control.ShadowMode)
}
