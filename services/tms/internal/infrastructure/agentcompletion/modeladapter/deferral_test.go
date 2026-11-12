package modeladapter_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenedTools_DeclaresWhatAPersonsMessageOpened(t *testing.T) {
	t.Parallel()

	tools := []modeladapter.ToolSpec{
		{Name: "find_tools"},
		{Name: "list_trailers", Deferred: true},
		{Name: "list_workers", Deferred: true},
	}
	messages := []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "due?", ToolReferences: []string{"list_trailers"}},
		{Role: modeladapter.RoleTool, ToolReferences: []string{"list_workers"}},
	}

	declared, asked := modeladapter.OpenedTools(tools, messages)

	require.Len(t, declared, 3)
	assert.False(t, declared[1].Deferred, "the opened tool is sent in full")
	assert.True(t, declared[2].Deferred, "a tool a result loads is still loaded by that result")
	assert.Empty(t, asked[0].ToolReferences)
	assert.Equal(t, []string{"list_workers"}, asked[1].ToolReferences)
	assert.True(t, tools[1].Deferred, "the request's own tools are left alone")
	assert.Equal(t, []string{"list_trailers"}, messages[0].ToolReferences)

	same, untouched := modeladapter.OpenedTools(tools, messages[1:])
	assert.Equal(t, tools, same)
	assert.Equal(t, messages[1:], untouched)
}
