package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemember_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	tool := newRememberTool(memories).(*rememberTool)
	params := memoryParams(map[string]any{
		"content":     "Needs the POD within one day.",
		"kind":        "Instruction",
		"subjectType": "Customer",
		"subjectId":   pulid.MustNew("cus_").String(),
		"expiresOn":   "2026-12-31",
	})

	preview := previewWithoutWrites(t, &memories.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceAgentMemory, change.Resource)
	assert.Equal(t, "Needs the POD within one day.", fieldByPath(t, change, "content").After)
	assert.Equal(t, "Instruction", fieldByPath(t, change, "kind").After)
	assert.Equal(t, int64(1798761600), fieldByPath(t, change, "expiresAt").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireCreateParity(t, change, memories.created,
		toolpreview.Only(rememberedFields...),
		toolpreview.Labels(rememberedLabels),
		toolpreview.Types(rememberedTypes),
	)
}

func TestRemember_PreviewSaysWhenItIsAlreadyKnown(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{existing: &agent.Memory{ID: pulid.MustNew("amem_")}}
	tool := newRememberTool(memories).(*rememberTool)

	preview := previewWithoutWrites(t, &memories.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), memoryParams(map[string]any{"content": "Known."}))
	})

	assert.Empty(t, preview.Changes)
	assert.Contains(t, preview.Summary, "already remembered")
}

func TestForgetMemory_PreviewReadsTheMemory(t *testing.T) {
	t.Parallel()

	memory := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Content: "Acme's dock closes at 3pm on Fridays.",
		Status:  agent.MemoryStatusActive,
		Version: 2,
	}
	before := *memory
	memories := &fakeMemories{stored: map[pulid.ID]*agent.Memory{memory.ID: memory}}
	tool := newForgetMemoryTool(memories).(*forgetMemoryTool)
	params := memoryParams(map[string]any{"memoryId": memory.ID.String()})

	preview := previewWithoutWrites(t, &memories.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationArchive, change.Operation)
	assert.Equal(t, "Acme's dock closes at 3pm on Fridays.", change.Label)
	assert.Equal(t, "Active", fieldByPath(t, change, "status").Before)
	assert.Equal(t, "Retired", fieldByPath(t, change, "status").After)
	assert.True(t, fieldByPath(t, change, "retiredAt").Volatile)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, memory,
		toolpreview.Only(retiredMemoryFields...), toolpreview.Volatile("retiredAt"))
}

func TestForgetMemory_PreviewWarnsForAMemoryAlreadyRetired(t *testing.T) {
	t.Parallel()

	memory := &agent.Memory{ID: pulid.MustNew("amem_"), Status: agent.MemoryStatusRetired}
	memories := &fakeMemories{stored: map[pulid.ID]*agent.Memory{memory.ID: memory}}
	tool := newForgetMemoryTool(memories).(*forgetMemoryTool)

	preview := previewWithoutWrites(t, &memories.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), memoryParams(map[string]any{"memoryId": memory.ID.String()}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
