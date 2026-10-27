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

func TestListByPlansReadsTheStepsOfEveryPlanInStepOrder(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("apl_"), pulid.MustNew("apl_")

	sql, args := keeltest.Render(t, byPlansQuery(first, second))

	assert.Contains(t, sql, `FROM "agent_proposals" ap WHERE`)
	assert.Contains(t, sql, `ap."plan_id" = ANY($3)`)
	assert.Contains(t, sql, `ORDER BY ap."plan_step" ASC, ap."created_at" ASC, ap."id" ASC`)
	assert.Contains(t, args, []string{first.String(), second.String()})
}

func TestListByPlansAsksNothingForNoPlans(t *testing.T) {
	t.Parallel()

	repo := &repository{l: zap.NewNop()}

	steps, err := repo.ListByPlans(t.Context(), agentports.ListAgentProposalsByPlansRequest{})

	require.NoError(t, err)
	assert.Empty(t, steps)
}
