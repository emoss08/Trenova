//go:build integration

package aiusagerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type definitionRow struct {
	ID             pulid.ID `bun:"id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func TestCostByDefinition_LeavesEvaluationSpendOut(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	cols := buncolgen.DefinitionColumns
	var row definitionRow
	require.NoError(t, db.NewSelect().
		TableExpr(buncolgen.DefinitionTable.As(buncolgen.DefinitionTable.Alias)).
		Column(cols.ID.Bare(), cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()).
		Limit(1).
		Scan(ctx, &row))

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	since := timeutils.NowUnix() - 60

	record := func(surface aiusage.Surface, cost string, runPrefix string) {
		amount := decimal.RequireFromString(cost)
		require.NoError(t, repo.Create(ctx, &aiusage.AIUsageRecord{
			OrganizationID:    row.OrganizationID,
			BusinessUnitID:    row.BusinessUnitID,
			ProviderKind:      aiprovider.KindOpenAIChat,
			Model:             "test-model",
			Task:              aiprovider.TaskAssistantChat,
			Surface:           surface,
			AgentDefinitionID: row.ID,
			RunID:             pulid.MustNew(runPrefix),
			Succeeded:         true,
			CostUSD:           &amount,
		}))
	}
	record(aiusage.SurfaceChat, "1.50", "ar_")
	record(aiusage.SurfaceBackground, "0.25", "ar_")
	record(aiusage.SurfaceEvaluation, "7.00", "aeval_")

	cost, err := repo.CostByDefinition(ctx, repositories.AIUsageCostRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID},
		DefinitionID: row.ID,
		Since:        since,
	})
	require.NoError(t, err)

	assert.True(t, cost.CostUSD.Equal(decimal.RequireFromString("1.75")), cost.CostUSD.String())
	assert.Equal(t, 2, cost.Calls)
}
