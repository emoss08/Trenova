package modeladapter

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
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

	tools := cachedTools([]anthropicTool{
		{Name: "list_workers"},
		{Name: "list_shipments"},
	})

	require.Len(t, tools, 2)
	assert.Nil(t, tools[0].CacheControl, "only the last tool closes the prefix")
	require.NotNil(t, tools[1].CacheControl)
	assert.Equal(t, "ephemeral", tools[1].CacheControl.Type)
}

func TestCachedSystem_MarksTheSystemPrompt(t *testing.T) {
	t.Parallel()

	blocks := cachedSystem("You are an agent inside Trenova.", 0)

	require.Len(t, blocks, 1)
	assert.Equal(t, "text", blocks[0].Type)
	assert.Equal(t, "You are an agent inside Trenova.", blocks[0].Text)
	require.NotNil(t, blocks[0].CacheControl)
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

	body := anthropicRequest{
		Model:     "claude-test",
		MaxTokens: 1024,
		System:    cachedSystem("Be brief.", 0),
		Tools:     cachedTools([]anthropicTool{{Name: "list_workers"}}),
	}

	encoded, err := sonic.Marshal(body)
	require.NoError(t, err)

	var decoded struct {
		System []struct {
			Type         string `json:"type"`
			Text         string `json:"text"`
			CacheControl *struct {
				Type string `json:"type"`
			} `json:"cache_control"`
		} `json:"system"`
		Tools []struct {
			CacheControl *struct {
				Type string `json:"type"`
			} `json:"cache_control"`
		} `json:"tools"`
	}
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))

	require.Len(t, decoded.System, 1)
	assert.Equal(t, "Be brief.", decoded.System[0].Text)
	require.NotNil(t, decoded.System[0].CacheControl)
	assert.Equal(t, "ephemeral", decoded.System[0].CacheControl.Type)

	require.Len(t, decoded.Tools, 1)
	require.NotNil(t, decoded.Tools[0].CacheControl)
}

/*
What every turn shares is cached apart from what the turn adds. The system
prompt leads with what is the same on every turn of an agent and ends with the
turn's own context, so the mark goes at the end of the shared part: a new page
or a new memory no longer costs the rules, the tools and the instructions.
*/
func TestCachedSystem_MarksTheEndOfTheSharedPart(t *testing.T) {
	t.Parallel()

	blocks := cachedSystem("Rules that never change.\n\nToday is Friday.", len("Rules that never change."))

	require.Len(t, blocks, 2)
	assert.Equal(t, "Rules that never change.", blocks[0].Text)
	require.NotNil(t, blocks[0].CacheControl)
	assert.Equal(t, "\n\nToday is Friday.", blocks[1].Text)
	assert.Nil(t, blocks[1].CacheControl)
}

func TestCachedSystem_KeepsOneBlockWithoutASharedPart(t *testing.T) {
	t.Parallel()

	for _, stable := range []int{0, -1, len("Be brief."), 99} {
		blocks := cachedSystem("Be brief.", stable)
		require.Len(t, blocks, 1, "stable=%d", stable)
		assert.Equal(t, "Be brief.", blocks[0].Text)
		require.NotNil(t, blocks[0].CacheControl)
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

	messages := cachedConversation(toAnthropicMessages([]Message{
		{Role: RoleUser, Content: "Where is S1?"},
		{
			Role:      RoleAssistant,
			Content:   "Looking.",
			ToolCalls: []ToolCall{{ID: "call_1", Name: "get_shipment", Arguments: map[string]any{}}},
		},
		{Role: RoleTool, ToolCallID: "call_1", Content: `{"status":"InTransit"}`},
	}))

	marked := 0
	for idx, message := range messages {
		for blockIdx, block := range message.Content {
			last := idx == len(messages)-1 && blockIdx == len(message.Content)-1
			if block.CacheControl != nil {
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

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{"message_start", `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":1}}}`},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))
	req := wideRequest()
	req.System = "Rules.\n\nToday."
	req.SystemStable = len("Rules.")
	_, _ = streamWith(t, NewAnthropicAdapter(), callFor(aiprovider.KindAnthropicMessages, server.URL, req))

	encoded, err := sonic.Marshal(*captured)
	require.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(encoded), `"cache_control"`))
}
