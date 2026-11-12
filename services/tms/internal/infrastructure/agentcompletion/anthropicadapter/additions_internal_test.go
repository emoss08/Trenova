package anthropicadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openedBody(t *testing.T, model string, platform aiplatform.Platform) map[string]any {
	t.Helper()

	server, captured := modeladaptertest.WireServer(t, "text/event-stream", wireStreamReply)
	request := modeladaptertest.DeferredWireRequest()
	request.Messages = []modeladapter.Message{{
		Role:           modeladapter.RoleUser,
		Content:        "Which trailers are due?",
		ToolReferences: []string{"list_trailers", "get_shipment", "not_declared"},
	}}
	call := modeladaptertest.ReasoningCall(aiprovider.KindAnthropicMessages, server.URL,
		aireasoning.ReasoningOff, &request)
	call.Provider.Model = model
	call.Provider.Platform = platform
	call.APIKey = "sk-test"
	_, _ = modeladaptertest.StreamWith(t, NewAnthropicAdapter(), call)

	return captured().Body
}

func toolsByName(t *testing.T, body map[string]any) map[string]map[string]any {
	t.Helper()

	tools, ok := body["tools"].([]any)
	require.True(t, ok)
	byName := make(map[string]map[string]any, len(tools))
	for _, raw := range tools {
		tool, isTool := raw.(map[string]any)
		require.True(t, isTool)
		byName[tool["name"].(string)] = tool
	}

	return byName
}

/*
The tools a question needs used to be declared in full at the front of the
request, so every new question changed the tools and the whole prompt, tens of
thousands of tokens, was written to the cache again. They are named after the
person's message instead: the request's tools are what every turn declares,
and a tool_addition surfaces the question's own.
*/
func TestAnthropicAdapter_SurfacesAQuestionsToolsAfterItsMessage(t *testing.T) {
	t.Parallel()

	body := openedBody(t, "claude-haiku-5-5", aiplatform.PlatformDirect)

	byName := toolsByName(t, body)
	assert.Equal(t, true, byName["list_trailers"]["defer_loading"],
		"the opened tool stays deferred, so the tools are the same bytes on every turn")

	messages := body["messages"].([]any)
	require.Len(t, messages, 2)
	added := messages[1].(map[string]any)
	assert.Equal(t, "system", added["role"])
	assert.Equal(
		t,
		[]any{map[string]any{
			"type": "tool_addition",
			"tool": map[string]any{"type": "tool_reference", "name": "list_trailers"},
		}},
		added["content"],
		"only a declared deferred tool is added",
	)
	asked := messages[0].(map[string]any)["content"].([]any)
	assert.Contains(t, asked[len(asked)-1], "cache_control",
		"the conversation's mark stays on the person's message")
}

// A model that takes no tool change mid-conversation is sent the question's
// tools in full, as every turn was before, and so is a platform that takes
// none.
func TestAnthropicAdapter_DeclaresAQuestionsToolsWhereAdditionsAreNotTaken(t *testing.T) {
	t.Parallel()

	body := openedBody(t, "claude-sonnet-5", aiplatform.PlatformDirect)

	byName := toolsByName(t, body)
	assert.NotContains(t, byName["list_trailers"], "defer_loading")
	assert.Equal(t, true, byName["list_workers"]["defer_loading"])
	assert.Len(t, body["messages"].([]any), 1)

	request := modeladaptertest.DeferredWireRequest()
	opened := []modeladapter.Message{{
		Role: modeladapter.RoleUser, Content: "due?", ToolReferences: []string{"list_trailers"},
	}}
	call := modeladaptertest.CallFor(aiprovider.KindAnthropicMessages, "http://unused", &request)
	call.Provider.Model = "claude-haiku-5-5"
	assert.True(t, anthropicAdds(call, opened))
	assert.False(t, anthropicAdds(call, request.Messages), "nothing was opened on a message")

	call.Provider.Platform = aiplatform.PlatformBedrock
	assert.False(t, anthropicAdds(call, opened))
	assert.False(t, anthropicKeepsAnHour(call))
}

func TestAdditionFits_OnlyAheadOfAnAssistantTurnOrLast(t *testing.T) {
	t.Parallel()

	asked := modeladapter.Message{Role: modeladapter.RoleUser, ToolReferences: []string{"a"}}
	notice := modeladapter.Message{Role: modeladapter.RoleSystem}
	reply := modeladapter.Message{Role: modeladapter.RoleAssistant}

	assert.True(t, additionFits([]modeladapter.Message{asked}, 0))
	assert.True(t, additionFits([]modeladapter.Message{asked, notice, reply}, 0))
	assert.False(t, additionFits([]modeladapter.Message{asked, asked}, 0))
}

// What every turn shares is kept for an hour where the platform takes the
// longer lifetime; the conversation's own mark keeps the default.
func TestCachedPrefix_KeepsTheSharedPartAnHour(t *testing.T) {
	t.Parallel()

	tools := cachedTools(toAnthropicTools([]modeladapter.ToolSpec{{Name: "list_workers"}}), true)
	assert.Equal(t, "1h", string(tools[0].GetCacheControl().TTL))

	system := cachedSystem("Rules.\n\nToday is Friday.", len("Rules."), true)
	require.Len(t, system, 2)
	assert.Equal(t, "1h", string(system[0].CacheControl.TTL))

	short := cachedTools(toAnthropicTools([]modeladapter.ToolSpec{{Name: "list_workers"}}), false)
	assert.Empty(t, string(short[0].GetCacheControl().TTL))
}
