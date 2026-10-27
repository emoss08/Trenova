package agentproposalrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestExpirePendingByDefinitionExpiresOnlyThePendingProposalsOfTheAgentsRuns(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	run := agentdb.AgentRuns

	sql, args := keeltest.Render(t, pendingOfRuns(run.Pick(run.ID).Where(
		run.AgentDefinitionID.Eq(agentID),
	)))

	assert.Contains(t, sql, `FROM "agent_proposals" ap WHERE`)
	assert.Contains(t, sql, `ap."status" = 'Pending'`)
	assert.Contains(t, sql, `ap."run_id" IN (SELECT ar."id" FROM "agent_runs" ar WHERE`)
	assert.Contains(t, sql, `ar."agent_definition_id" = `)
	assert.Contains(t, args, agentID.String())
}
