package agentproposalrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSumAutomaticMoney_CountsOnlyWritesThatRanInTheFunctionalCurrency(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, automaticMoneyQuery(&agentports.SumAutomaticMoneyRequest{
		Currency: "USD",
		Since:    100,
	}, "USD"))

	assert.Regexp(t, `^SELECT count\(\*\), coalesce\(sum\(abs\(ap\."money_minor"\)\), \$\d+\) FROM "agent_proposals" ap `, sql)
	assert.Contains(t, sql, `ap."autonomy_tier" = `)
	assert.Contains(t, sql, `ap."status" = `)
	assert.Contains(t, sql, `ap."money_minor" IS NOT NULL`)
	assert.Contains(t, sql, `ap."money_currency" = `)
	assert.Contains(t, sql, `ap."executed_at" IS NOT NULL`)
	assert.Contains(t, sql, `ap."executed_at" >= `)
	assert.NotContains(t, sql, `"run_id"`)
	assert.Contains(t, args, "AutoExecute")
	assert.Contains(t, args, "Executed")
	assert.Contains(t, args, "USD")
	assert.Contains(t, args, int64(100))
	assert.Contains(t, args, 0)
}

func TestSumAutomaticMoney_ASimulatedRunCountsItsSimulatedWrites(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("arun_")
	sql, args := keeltest.Render(t, automaticMoneyQuery(&agentports.SumAutomaticMoneyRequest{
		Currency:  "CAD",
		RunID:     runID,
		Simulated: true,
	}, "CAD"))

	assert.Contains(t, sql, `ap."simulated_at" IS NOT NULL`)
	assert.Contains(t, sql, `ap."run_id" = `)
	assert.NotContains(t, sql, `"executed_at"`)
	assert.Contains(t, args, "Simulated")
	assert.Contains(t, args, "CAD")
	assert.Contains(t, args, runID.String())
}

func TestSumAutomaticMoney_RefusesToCountWithoutACurrency(t *testing.T) {
	t.Parallel()

	repo := &repository{l: zap.NewNop()}

	_, err := repo.SumAutomaticMoney(t.Context(), &agentports.SumAutomaticMoneyRequest{
		Currency: " ",
	})
	require.ErrorIs(t, err, errMoneyCurrencyRequired)
}
