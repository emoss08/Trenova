package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLocation_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	tool := fixture.tool.(*createLocationTool)
	params := fixture.params(nil)
	params.IdempotencyKey = ""

	preview := previewWithoutWrites(t, &fixture.creator.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceLocation, change.Resource)
	assert.Equal(t, "Reno Cold Storage", change.Label)
	assert.Equal(t, "NV", fieldByPath(t, change, "state").After)
	assert.Equal(t, "Warehouse", fieldByPath(t, change, "category").After)
	assert.Equal(t, "Dock 4", fieldByPath(t, change, "addressLine2").After)
	assert.Contains(t, preview.Summary, "NV 89502")
	assert.Nil(t, fixture.creator.created)

	require.NoError(t, tool.Execute(t.Context(), fixture.params(nil)))
	requireCreateParity(t, change,
		locationViewOf(fixture.creator.created, fixture.state, fixture.category))
}

func TestCreateLocation_PreviewWarnsOnAnUnknownState(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	tool := fixture.tool.(*createLocationTool)

	preview := previewWithoutWrites(t, &fixture.creator.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), fixture.params(map[string]any{"state": "ZZ"}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
