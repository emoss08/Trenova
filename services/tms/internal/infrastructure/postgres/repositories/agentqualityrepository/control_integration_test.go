//go:build integration

package agentqualityrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestControlUpsertCreatesTheFirstControl(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := NewControls(ControlParams{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	_, err := repo.Get(ctx, tenant)
	require.True(t, errortypes.IsNotFoundError(err))

	created, err := repo.Upsert(ctx, &agentquality.Control{
		OrganizationID:      tenant.OrgID,
		BusinessUnitID:      tenant.BuID,
		Enabled:             true,
		RunHourLocal:        3,
		MaxCasesPerAgent:    40,
		NightlyBudgetUSD:    decimal.NewFromInt(5),
		MonthlyBudgetUSD:    decimal.NewFromInt(50),
		JudgeSampleRate:     0.2,
		RegressionThreshold: 0.1,
		MinCases:            10,
		ForceRerunDays:      7,
	})
	require.NoError(t, err)
	require.False(t, created.ID.IsNil())

	created.RunHourLocal = 4
	updated, err := repo.Upsert(ctx, created)
	require.NoError(t, err)
	require.Equal(t, 4, updated.RunHourLocal)
	require.Equal(t, created.ID, updated.ID)
}
