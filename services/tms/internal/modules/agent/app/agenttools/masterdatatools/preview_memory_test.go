package masterdatatools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemember_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	tool := NewRememberTool(memories).(*RememberTool)
	params := agenttoolstest.MemoryParams(map[string]any{
		"content":     "Needs the POD within one day.",
		"kind":        "Instruction",
		"subjectType": "Customer",
		"subjectId":   pulid.MustNew("cus_").String(),
		"expiresOn":   "2026-12-31",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&memories.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceAgentMemory, change.Resource)
	assert.Equal(
		t,
		"Needs the POD within one day.",
		agenttoolstest.FieldByPath(t, change, "content").After,
	)
	assert.Equal(t, "Instruction", agenttoolstest.FieldByPath(t, change, "kind").After)
	assert.Equal(t, int64(1798761600), agenttoolstest.FieldByPath(t, change, "expiresAt").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, memories.created,
		toolpreview.Only(rememberedFields...),
		toolpreview.Labels(rememberedLabels),
		toolpreview.Types(rememberedTypes),
	)
}

func TestRemember_PreviewSaysWhenItIsAlreadyKnown(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{existing: &agentmemory.Memory{ID: pulid.MustNew("amem_")}}
	tool := NewRememberTool(memories).(*RememberTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&memories.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(
				t.Context(),
				agenttoolstest.MemoryParams(map[string]any{"content": "Known."}),
			)
		},
	)

	assert.Empty(t, preview.Changes)
	assert.Contains(t, preview.Summary, "already remembered")
}

func TestForgetMemory_PreviewReadsTheMemory(t *testing.T) {
	t.Parallel()

	memory := &agentmemory.Memory{
		ID:      pulid.MustNew("amem_"),
		Content: "Acme's dock closes at 3pm on Fridays.",
		Status:  agentmemory.MemoryStatusActive,
		Version: 2,
	}
	before := *memory
	memories := &fakeMemories{stored: map[pulid.ID]*agentmemory.Memory{memory.ID: memory}}
	tool := NewForgetMemoryTool(memories).(*ForgetMemoryTool)
	params := agenttoolstest.MemoryParams(map[string]any{"memoryId": memory.ID.String()})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&memories.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationArchive, change.Operation)
	assert.Equal(t, "Acme's dock closes at 3pm on Fridays.", change.Label)
	assert.Equal(t, "Active", agenttoolstest.FieldByPath(t, change, "status").Before)
	assert.Equal(t, "Retired", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.True(t, agenttoolstest.FieldByPath(t, change, "retiredAt").Volatile)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, memory,
		toolpreview.Only(retiredMemoryFields...), toolpreview.Volatile("retiredAt"))
}

func TestForgetMemory_PreviewWarnsForAMemoryAlreadyRetired(t *testing.T) {
	t.Parallel()

	memory := &agentmemory.Memory{
		ID:     pulid.MustNew("amem_"),
		Status: agentmemory.MemoryStatusRetired,
	}
	memories := &fakeMemories{stored: map[pulid.ID]*agentmemory.Memory{memory.ID: memory}}
	tool := NewForgetMemoryTool(memories).(*ForgetMemoryTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&memories.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(
				t.Context(),
				agenttoolstest.MemoryParams(map[string]any{"memoryId": memory.ID.String()}),
			)
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

// A person without the manage permission remembering something for the
// organization is offering it: the preview says so, so what they approve is
// what will happen.
func TestRemember_PreviewSaysWhenItWouldBeOffered(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{offers: true}
	tool := NewRememberTool(memories).(*RememberTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&memories.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.MemoryParams(map[string]any{
				"kind":      "Fact",
				"content":   "Granite wants the POD on the invoice.",
				"visibleTo": "organization",
			}))
		},
	)

	assert.Contains(t, preview.Summary, "Would offer a memory for an administrator to keep")
	assert.Len(t, preview.Changes, 1, "the offered memory is still the one change")
}

// Retiring a memory the person may not change fails when it runs, so the
// preview warns rather than promising the retire.
func TestForgetMemory_PreviewWarnsWhenTheServiceWouldRefuse(t *testing.T) {
	t.Parallel()

	memory := &agentmemory.Memory{
		ID:     pulid.MustNew("amem_"),
		Status: agentmemory.MemoryStatusActive,
	}
	memories := &fakeMemories{
		stored: map[pulid.ID]*agentmemory.Memory{memory.ID: memory},
		refuse: errortypes.NewAuthorizationError("shared memory is an administrator's"),
	}
	tool := NewForgetMemoryTool(memories).(*ForgetMemoryTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&memories.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(
				t.Context(),
				agenttoolstest.MemoryParams(map[string]any{"memoryId": memory.ID.String()}),
			)
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}
