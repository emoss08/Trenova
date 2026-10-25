//go:build integration

package usstaterepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUsStatesAreCachedUntilAnotherProcessChangesThem(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	cached := cachetest.Open(t, ctx, dsn)
	repo := New(Params{DB: postgres.NewTestConnection(cached), Logger: zap.NewNop()})
	tctx := data.Ctx(ctx)

	var state *usstate.UsState
	read := func() error {
		var readErr error
		state, readErr = repo.GetByAbbreviation(tctx, data.State.Abbreviation)
		return readErr
	}

	cachetest.RequireMiss(t, cached, read)
	cachetest.RequireHit(t, cached, read)
	assert.Equal(t, data.State.Name, state.Name)

	seedtest.NewState().
		WithName("Renamed State").
		WithAbbreviation(data.State.Abbreviation).
		Build(t, ctx, db)
	require.NoError(t, db.FlushDataVersions(ctx))

	require.Eventually(t, func() bool {
		return read() == nil && state.Name == "Renamed State"
	}, 5*time.Second, 20*time.Millisecond, "the other process's write reaches this cache")
	cachetest.RequireHit(t, cached, read)
	assert.Equal(t, "Renamed State", state.Name)
}
