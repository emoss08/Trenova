package assistantartifactrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact/assistantartifactdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityNamesTheRowAnUpsertReplaces(t *testing.T) {
	t.Parallel()

	a := assistantartifactdb.Artifacts
	proposalID, planID := pulid.MustNew("aprop_"), pulid.MustNew("apl_")

	for name, tc := range map[string]struct {
		artifact *assistantartifact.Artifact
		columns  []string
	}{
		"proposal": {
			artifact: &assistantartifact.Artifact{ProposalID: proposalID, SourceToolCallID: "call_1"},
			columns:  []string{`"proposal_id" = `},
		},
		"plan": {
			artifact: &assistantartifact.Artifact{PlanID: planID, SourceToolCallID: "call_1"},
			columns:  []string{`"plan_id" = `},
		},
		"tool call": {
			artifact: &assistantartifact.Artifact{
				SourceToolCallID: "call_1",
				Kind:             assistantartifact.KindTableView,
			},
			columns: []string{`"source_tool_call_id" = `, `"kind" = `},
		},
	} {
		id, err := identityOf(tc.artifact)
		require.NoError(t, err, name)

		sql, _ := keeltest.Render(t, a.Where(id.match...))
		for _, column := range tc.columns {
			assert.Contains(t, sql, column, name)
		}
	}

	_, err := identityOf(&assistantartifact.Artifact{})
	require.ErrorIs(t, err, ErrNoIdentity)
}
