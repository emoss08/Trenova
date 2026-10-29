package anthropicadapter

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The two boundaries worth caching.

Tools and the system prompt are most of what a turn sends and every iteration
of a tool loop resends them byte for byte — a two-tool turn read the whole
prompt and every schema twice. A mark at the end of each means the second read
is a cache hit.

The mark goes at the end because what is cached is everything up to it.
*/
func TestCachedTools_MarksTheEndOfTheSchemas(t *testing.T) {
	t.Parallel()

	tools := cachedTools(toAnthropicTools([]modeladapter.ToolSpec{
		{Name: "list_workers"},
		{Name: "list_shipments"},
	}))

	require.Len(t, tools, 2)
	assert.NotContains(
		t,
		wireMap(t, tools[0]),
		"cache_control",
		"only the last tool closes the prefix",
	)
	assert.Equal(t, map[string]any{"type": "ephemeral"}, wireMap(t, tools[1])["cache_control"])
}

func TestCachedSystem_MarksTheSystemPrompt(t *testing.T) {
	t.Parallel()

	blocks := cachedSystem("You are an agent inside Trenova.", 0)

	require.Len(t, blocks, 1)
	block := wireMap(t, blocks[0])
	assert.Equal(t, "text", block["type"])
	assert.Equal(t, "You are an agent inside Trenova.", block["text"])
	assert.Equal(t, map[string]any{"type": "ephemeral"}, block["cache_control"])
}

// A breakpoint on nothing still costs a cache write, so an empty tool list or
// system prompt gets no mark at all.
func TestCachedPrefix_MarksNothingWhenThereIsNothingToCache(t *testing.T) {
	t.Parallel()

	assert.Empty(t, cachedTools(nil))
	assert.Nil(t, cachedSystem("", 0))
}

// The system field has to serialise as blocks, not a string, or the breakpoint
// has nowhere to live. This is the shape that actually goes on the wire.
func TestAnthropicRequest_SendsSystemAsBlocksCarryingTheBreakpoint(t *testing.T) {
	t.Parallel()

	body := wireMap(t, anthropic.BetaMessageNewParams{
		Model:     "claude-test",
		MaxTokens: 1024,
		System:    cachedSystem("Be brief.", 0),
		Tools:     cachedTools(toAnthropicTools([]modeladapter.ToolSpec{{Name: "list_workers"}})),
	})

	system, ok := body["system"].([]any)
	require.True(t, ok)
	require.Len(t, system, 1)
	block, ok := system[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Be brief.", block["text"])
	assert.Equal(t, map[string]any{"type": "ephemeral"}, block["cache_control"])

	tools, ok := body["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, tool, "cache_control")
}

/*
What every turn shares is cached apart from what the turn adds. The system
prompt leads with what is the same on every turn of an agent and ends with the
turn's own context, so the mark goes at the end of the shared part: a new page
or a new memory no longer costs the rules, the tools and the instructions.
*/
func TestCachedSystem_MarksTheEndOfTheSharedPart(t *testing.T) {
	t.Parallel()

	blocks := cachedSystem(
		"Rules that never change.\n\nToday is Friday.",
		len("Rules that never change."),
	)

	require.Len(t, blocks, 2)
	assert.Equal(t, "Rules that never change.", blocks[0].Text)
	assert.Contains(t, wireMap(t, blocks[0]), "cache_control")
	assert.Equal(t, "\n\nToday is Friday.", blocks[1].Text)
	assert.NotContains(t, wireMap(t, blocks[1]), "cache_control")
}

func TestCachedSystem_KeepsOneBlockWithoutASharedPart(t *testing.T) {
	t.Parallel()

	for _, stable := range []int{0, -1, len("Be brief."), 99} {
		blocks := cachedSystem("Be brief.", stable)
		require.Len(t, blocks, 1, "stable=%d", stable)
		assert.Equal(t, "Be brief.", blocks[0].Text)
		assert.Contains(t, wireMap(t, blocks[0]), "cache_control", "stable=%d", stable)
	}
}

/*
A tool loop resends the whole conversation on every call, one tool result
longer each time. With a mark on its last block, the next call reads back
everything before the new result instead of paying for the conversation again;
an earlier mark stays a valid place to read from as the conversation grows.
*/
func TestCachedConversation_MarksOnlyTheLastBlock(t *testing.T) {
	t.Parallel()

	messages := cachedConversation(toAnthropicMessages([]modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "Where is S1?"},
		{
			Role:    modeladapter.RoleAssistant,
			Content: "Looking.",
			ToolCalls: []modeladapter.ToolCall{
				{ID: "call_1", Name: "get_shipment", Arguments: map[string]any{}},
			},
		},
		{Role: modeladapter.RoleTool, ToolCallID: "call_1", Content: `{"status":"InTransit"}`},
	}))

	marked := 0
	for idx, message := range messages {
		for blockIdx, block := range message.Content {
			last := idx == len(messages)-1 && blockIdx == len(message.Content)-1
			if _, ok := wireMap(t, block)["cache_control"]; ok {
				marked++
				assert.True(t, last, "only the last block of the conversation is marked")
			}
		}
	}
	assert.Equal(t, 1, marked)
	assert.Empty(t, cachedConversation(nil))
}

// Anthropic allows four marks a request: the tools, the shared system part
// and the conversation's end are three.
func TestAnthropicRequest_StaysWithinTheMarkLimit(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"m","usage":{"input_tokens":1}}}`,
		},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))
	req := modeladaptertest.WideRequest()
	req.System = "Rules.\n\nToday."
	req.SystemStable = len("Rules.")
	_, _ = modeladaptertest.StreamWith(
		t,
		NewAnthropicAdapter(),
		modeladaptertest.CallFor(aiprovider.KindAnthropicMessages, server.URL, req),
	)

	encoded, err := sonic.Marshal(*captured)
	require.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(encoded), `"cache_control"`))
}

// wireMap is v as it goes on the wire.
func wireMap(t *testing.T, v any) map[string]any {
	t.Helper()

	encoded, err := sonic.Marshal(v)
	require.NoError(t, err)
	decoded := map[string]any{}
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))

	return decoded
}
