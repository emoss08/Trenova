//go:build integration

package quotacounterrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestQuotaCounterRepository_CountsEveryMeterInsideATransaction(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	err := conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		for _, meter := range Meters() {
			require.NoError(t, repo.Lock(ctx, tenant, meter))

			used, countErr := repo.Count(ctx, &repositories.QuotaCountRequest{
				TenantInfo: tenant,
				Meter:      meter,
			})
			require.NoError(t, countErr, meter)
			assert.GreaterOrEqual(t, used, int64(0), meter)
		}

		return nil
	})
	require.NoError(t, err)

	timezone, err := repo.OrganizationTimezone(ctx, tenant)
	require.NoError(t, err)
	assert.NotEmpty(t, timezone)

	_, err = repo.Count(ctx, &repositories.QuotaCountRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterDocumentFileBytes,
	})
	require.ErrorIs(t, err, ErrUnsupportedMeter)
}
