package insighttools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDismissInsight_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	finding := &insight.Insight{
		ID:       pulid.MustNew("inst_"),
		Headline: "Acme's margin fell 9 points this month",
		Status:   insight.StatusActive,
		Version:  3,
	}
	before := *finding
	insights := &fakeInsightDismisser{finding: finding}
	tool := NewDismissInsightTool(insights).(*DismissInsightTool)
	params := agenttoolstest.MemoryParams(map[string]any{
		"insightId": finding.ID.String(),
		"reason":    "Known seasonal pattern for this customer.",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &insights.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationArchive, change.Operation)
	assert.Equal(t, "Acme's margin fell 9 points this month", change.Label)
	assert.Equal(t, "Active", agenttoolstest.FieldByPath(t, change, "status").Before)
	assert.Equal(t, "Dismissed", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.Equal(t, "Known seasonal pattern for this customer.",
		agenttoolstest.FieldByPath(t, change, "dismissReason").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, finding,
		toolpreview.Only(dismissedInsightFields...), toolpreview.Volatile("dismissedAt"))
}

func TestDismissInsight_PreviewWarnsForAFindingNoLongerActive(t *testing.T) {
	t.Parallel()

	finding := &insight.Insight{ID: pulid.MustNew("inst_"), Status: insight.StatusResolved}
	insights := &fakeInsightDismisser{finding: finding}
	tool := NewDismissInsightTool(insights).(*DismissInsightTool)

	preview := agenttoolstest.PreviewWithoutWrites(t, &insights.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agenttoolstest.MemoryParams(map[string]any{
			"insightId": finding.ID.String(),
			"reason":    "Handled.",
		}))
	})

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}
