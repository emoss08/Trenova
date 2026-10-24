//go:build integration

package aiusagerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/pkg/buncolgen/agentdefinitioncol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
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
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	cols := agentdefinitioncol.DefinitionColumns
	var row definitionRow
	require.NoError(t, db.NewSelect().
		TableExpr(agentdefinitioncol.DefinitionTable.As(agentdefinitioncol.DefinitionTable.Alias)).
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

	cost, err := repo.CostByDefinition(ctx, aiusageports.AIUsageCostRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID},
		DefinitionID: row.ID,
		Since:        since,
	})
	require.NoError(t, err)

	assert.True(t, cost.CostUSD.Equal(decimal.RequireFromString("1.75")), cost.CostUSD.String())
	assert.Equal(t, 2, cost.Calls)
}
