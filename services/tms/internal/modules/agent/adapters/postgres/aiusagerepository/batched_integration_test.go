//go:build integration

package aiusagerepository

import (
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateOrSkip_CountsABatchReadTwiceOnce(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	d := agentdefinitiondb.Definitions
	row, err := keel.IntoOne[definitionRow](dbscope.WithSystem(ctx, "test"), db,
		d.AllTenants().Pick(d.ID, d.OrganizationID, d.BusinessUnitID).Limit(1))
	require.NoError(t, err)
	ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: row.OrganizationID,
		BusinessUnitID: row.BusinessUnitID,
	})

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	since := timeutils.NowUnix() - 60
	submittedAt := time.Unix(since, 0)
	batchID := pulid.MustNew("msgbatch_").String()

	records := func() []*aiusage.AIUsageRecord {
		rows := make([]*aiusage.AIUsageRecord, 0, 2)
		for _, custom := range []string{"aeval_1", "aeval_2"} {
			amount := decimal.RequireFromString("0.50")
			rows = append(rows, &aiusage.AIUsageRecord{
				ID:                pulid.Derived("aiu_", submittedAt, batchID+"/"+custom),
				OrganizationID:    row.OrganizationID,
				BusinessUnitID:    row.BusinessUnitID,
				ProviderKind:      aiprovider.KindAnthropicMessages,
				Model:             "claude-sonnet-5-5",
				Task:              aiprovider.TaskEvaluationJudge,
				Surface:           aiusage.SurfaceEvaluation,
				AgentDefinitionID: row.ID,
				RunID:             pulid.MustNew("aeval_"),
				Succeeded:         true,
				CostUSD:           &amount,
				Batched:           true,
			})
		}

		return rows
	}

	inserted, err := repo.CreateOrSkip(ctx, records())
	require.NoError(t, err)
	assert.Equal(t, int64(2), inserted)

	inserted, err = repo.CreateOrSkip(ctx, records())
	require.NoError(t, err)
	assert.Zero(t, inserted, "the same batch read again records nothing more")

	cost, err := repo.EvaluationCost(ctx, aiusageports.AIUsageEvaluationCostRequest{
		TenantInfo: pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID},
		Since:      since,
	})
	require.NoError(t, err)
	assert.True(t, cost.CostUSD.GreaterThanOrEqual(decimal.RequireFromString("1.00")),
		cost.CostUSD.String())
}
