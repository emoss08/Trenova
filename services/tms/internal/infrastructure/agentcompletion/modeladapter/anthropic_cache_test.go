package modeladapter

import (
	"testing"

	"github.com/bytedance/sonic"
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

	blocks := cachedSystem("You are an agent inside Trenova.")

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
	assert.Nil(t, cachedSystem(""))
}

// The system field has to serialise as blocks, not a string, or the breakpoint
// has nowhere to live. This is the shape that actually goes on the wire.
func TestAnthropicRequest_SendsSystemAsBlocksCarryingTheBreakpoint(t *testing.T) {
	t.Parallel()

	body := anthropicRequest{
		Model:     "claude-test",
		MaxTokens: 1024,
		System:    cachedSystem("Be brief."),
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

// Nothing marks the conversation: it grows every turn, so a mark there caches a
// prefix the next request has already moved past.
func TestAnthropicRequest_LeavesTheConversationUnmarked(t *testing.T) {
	t.Parallel()

	messages := toAnthropicMessages(nil)
	for _, message := range messages {
		for _, block := range message.Content {
			assert.Nil(t, block.CacheControl)
		}
	}
}
