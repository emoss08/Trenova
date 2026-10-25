package agentproposalbaselinerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/stretchr/testify/assert"
)

func TestOrphans_AreOnlyOldBaselinesNoProposalNames(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithSystem(t.Context(), "test")
	sql, args := keeltest.RenderIn(t, ctx, orphans(1767225600, 250))

	assert.Contains(t, sql, `SELECT orphan."proposal_id"`)
	assert.Contains(t, sql, `FROM "agent_proposal_baselines" orphan`)
	assert.Contains(t, sql, `orphan."created_at" < $1`)
	assert.Contains(t, sql, `NOT EXISTS (SELECT`)
	assert.Contains(t, sql, `"agent_proposals" ap`)
	assert.Contains(t, sql, `ap."id" = orphan."proposal_id"`)
	assert.Contains(t, sql, `ap."organization_id" = orphan."organization_id"`)
	assert.Contains(t, sql, `ap."business_unit_id" = orphan."business_unit_id"`)
	assert.Contains(t, sql, `ORDER BY orphan."created_at" ASC LIMIT 250`)
	assert.Equal(t, []any{int64(1767225600)}, args)
}
