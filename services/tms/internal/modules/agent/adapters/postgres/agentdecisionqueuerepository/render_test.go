package agentdecisionqueuerepository

import (
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionqueueports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func renderQueue(
	t *testing.T,
	req *agentdecisionqueueports.ListPendingDecisionsQuery,
) (string, []any) {
	t.Helper()

	q := pendingQueue(req)

	return keeltest.Render(t, keel.RowsOf(q.TableRef()).With(q).Limit(1))
}

func TestPendingQueueSearchesWhatEachDecisionSaysOnTheWatchtower(t *testing.T) {
	t.Parallel()

	sql, args := renderQueue(t, &agentdecisionqueueports.ListPendingDecisionsQuery{
		Search: `50%_off`,
	})

	for _, alias := range []string{"psaid", "plsaid"} {
		assert.Contains(t, sql, alias+`."title" ILIKE $`, sql)
		assert.Contains(t, sql, ` OR `+alias+`."summary" ILIKE $`, sql)
		assert.Contains(t, sql, ` OR `+alias+`."record_label" ILIKE $`, sql)
		assert.Contains(t, sql, alias+`."source_id" = `, sql)
		assert.Contains(
			t,
			sql,
			alias+`."organization_id" = `,
			"the items read are the reader's tenant's",
		)
	}
	assert.Equal(t, 2, strings.Count(sql, "EXISTS"), "one for proposals and one for plans: %s", sql)
	assert.Contains(t, args, `%50\%\_off%`, "a wildcard a person types matches itself")
	assert.Contains(t, args, "AgentProposal")
	assert.Contains(t, args, "AgentPlan")
}

func TestPendingQueueWithoutASearchReadsNoWatchtowerItem(t *testing.T) {
	t.Parallel()

	sql, _ := renderQueue(t, &agentdecisionqueueports.ListPendingDecisionsQuery{})

	assert.NotContains(t, sql, "ILIKE", sql)
	assert.NotContains(t, sql, "watchtower_items", sql)
}
