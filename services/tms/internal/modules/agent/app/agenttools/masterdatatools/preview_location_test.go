package masterdatatools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLocation_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	tool := fixture.tool.(*CreateLocationTool)
	params := fixture.params(nil)
	params.IdempotencyKey = ""

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&fixture.creator.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceLocation, change.Resource)
	assert.Equal(t, "Reno Cold Storage", change.Label)
	assert.Equal(t, "NV", agenttoolstest.FieldByPath(t, change, "state").After)
	assert.Equal(t, "Warehouse", agenttoolstest.FieldByPath(t, change, "category").After)
	assert.Equal(t, "Dock 4", agenttoolstest.FieldByPath(t, change, "addressLine2").After)
	assert.Contains(t, preview.Summary, "NV 89502")
	assert.Nil(t, fixture.creator.created)

	require.NoError(t, tool.Execute(t.Context(), fixture.params(nil)))
	agenttoolstest.RequireCreateParity(t, change,
		locationViewOf(fixture.creator.created, fixture.state, fixture.category))
}

func TestCreateLocation_PreviewWarnsOnAnUnknownState(t *testing.T) {
	t.Parallel()

	fixture := newLocationFixture()
	tool := fixture.tool.(*CreateLocationTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&fixture.creator.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), fixture.params(map[string]any{"state": "ZZ"}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}
