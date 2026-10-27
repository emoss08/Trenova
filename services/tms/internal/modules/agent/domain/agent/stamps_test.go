package agent_test

import (
	"testing"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/stretchr/testify/assert"
)

func TestInsertStampsUpdatedAtForTheAuditProjector(t *testing.T) {
	t.Parallel()

	for _, table := range []*schema.Table{agentdb.AgentRunsSchema(), agentdb.AgentProposalsSchema()} {
		assert.GreaterOrEqual(t, table.AutoUpdate, int32(0), table.Name)
		assert.GreaterOrEqual(t, table.AutoCreate, int32(0), table.Name)
	}
}
