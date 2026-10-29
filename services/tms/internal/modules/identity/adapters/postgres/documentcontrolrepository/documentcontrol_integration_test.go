//go:build integration

package documentcontrolrepository

import (
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol/documentcontroldb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetOrCreateConcurrentCreatesSingleTenantControl(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	const calls = 50
	results := make([]string, calls)
	errs := make([]error, calls)

	var wg sync.WaitGroup
	wg.Add(calls)
	for idx := range calls {
		go func() {
			defer wg.Done()

			entity, err := repo.GetOrCreate(ctx, data.Organization.ID, data.BusinessUnit.ID)
			if err != nil {
				errs[idx] = err
				return
			}
			results[idx] = entity.ID.String()
		}()
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	for _, id := range results {
		require.NotEmpty(t, id)
		assert.Equal(t, results[0], id)
	}

	dc := documentcontroldb.DocumentControls
	count, err := dc.Query().Count(data.Ctx(ctx), db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
