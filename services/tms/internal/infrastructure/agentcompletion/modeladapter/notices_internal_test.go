package modeladapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	worldNotice = "[Notice from the system, not from the person: S-1 changed.]"
	steerNotice = "[Notice from the system, not from the person: they added this.]"
)

func noticeConversation() []Message {
	return []Message{
		{Role: RoleUser, Content: "Where is S-1?"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID:        "toolu_1",
			Name:      "get_shipment",
			Arguments: map[string]any{"number": "S-1"},
		}}},
		{Role: RoleTool, ToolCallID: "toolu_1", ToolName: "get_shipment", Content: `{"s":"X"}`},
		{Role: RoleUser, Content: "Only the late ones."},
		{Role: RoleSystem, Content: steerNotice},
		{Role: RoleSystem, Content: worldNotice},
	}
}

func noticeBody(t *testing.T, model string, platform aiplatform.Platform) []any {
	t.Helper()

	server, captured := wireServer(t, "text/event-stream", wireStreamReply)
	request := Request{Messages: noticeConversation(), Tools: wireTools()}
	call := reasoningCall(aiprovider.KindAnthropicMessages, server.URL,
		aireasoning.ReasoningOff, &request)
	call.Provider.Model = model
	call.Provider.Platform = platform
	call.APIKey = "sk-test"
	_, _ = streamWith(t, NewAnthropicAdapter(), call)

	return captured().Body["messages"].([]any)
}

func textsOf(message map[string]any) []string {
	texts := make([]string, 0, 4)
	for _, raw := range message["content"].([]any) {
		block := raw.(map[string]any)
		if block["type"] == "text" {
			texts = append(texts, block["text"].(string))
		}
	}

	return texts
}

func TestAnthropicAdapter_SendsNoticesAsOneSystemMessageAfterThePersonsWords(t *testing.T) {
	t.Parallel()

	messages := noticeBody(t, "claude-haiku-5-5", aiplatform.PlatformDirect)

	require.Len(t, messages, 4)
	results := messages[2].(map[string]any)
	assert.Equal(t, "user", results["role"])
	assert.Equal(t, []string{"Only the late ones."}, textsOf(results),
		"the person's words follow the tool results in the same message")
	notice := messages[3].(map[string]any)
	assert.Equal(t, "system", notice["role"])
	assert.Equal(t, []string{steerNotice, worldNotice}, textsOf(notice),
		"notices in a row are one system message, which the API takes only once")
	for _, raw := range notice["content"].([]any) {
		assert.NotContains(t, raw.(map[string]any), "cache_control")
	}
	last := results["content"].([]any)
	assert.Contains(t, last[len(last)-1].(map[string]any), "cache_control",
		"the conversation's breakpoint stays on the last message the cache can hold")
}

func TestAnthropicAdapter_SendsNoticesAsTextWhereSystemMessagesAreNotTaken(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		model    string
		platform aiplatform.Platform
	}{
		{"claude-sonnet-5", aiplatform.PlatformDirect},
		{"claude-haiku-4-5", aiplatform.PlatformDirect},
		{"claude-haiku-5-5", aiplatform.PlatformFoundry},
	} {
		messages := noticeBody(t, tc.model, tc.platform)

		require.Len(t, messages, 3, tc.model)
		results := messages[2].(map[string]any)
		assert.Equal(t, "user", results["role"], tc.model)
		assert.Equal(t, []string{"Only the late ones.", steerNotice, worldNotice},
			textsOf(results), tc.model)
	}

	for platform, want := range map[aiplatform.Platform]bool{
		aiplatform.PlatformDirect:            true,
		aiplatform.PlatformVertexAI:          true,
		aiplatform.PlatformClaudeAWS:         true,
		aiplatform.PlatformClaudeGoogleCloud: true,
		aiplatform.PlatformBedrock:           false,
		aiplatform.PlatformFoundry:           false,
	} {
		call := &Call{Provider: &aiprovider.Provider{
			Kind:     aiprovider.KindAnthropicMessages,
			Model:    "claude-haiku-5-5",
			Platform: platform,
		}}
		assert.Equal(t, want, anthropicNotices(call), platform)
	}
}

func TestAnthropicConversation_PlacesANoticeOnlyWhereTheAPITakesOne(t *testing.T) {
	t.Parallel()

	afterReply := anthropicConversation([]Message{
		{Role: RoleUser, Content: "How many are late?"},
		{Role: RoleAssistant, Content: "About half."},
		{Role: RoleSystem, Content: "Say only what a tool returned."},
	}, nil, true)
	require.Len(t, afterReply, 3)
	assert.Equal(t, "user", string(afterReply[2].Role),
		"a notice cannot follow an assistant message, so it is sent as text")

	beforePerson := anthropicConversation([]Message{
		{Role: RoleUser, Content: "Where is S-1?"},
		{Role: RoleSystem, Content: worldNotice},
		{Role: RoleUser, Content: "And S-2?"},
	}, nil, true)
	require.Len(t, beforePerson, 3)
	for _, message := range beforePerson {
		assert.Equal(t, "user", string(message.Role),
			"a notice the person's next message follows is sent as text")
	}

	answered := anthropicConversation([]Message{
		{Role: RoleUser, Content: "Where is S-1?"},
		{Role: RoleSystem, Content: worldNotice},
		{Role: RoleAssistant, Content: "In transit."},
	}, nil, true)
	require.Len(t, answered, 3)
	assert.Equal(t, "system", string(answered[1].Role))
}

func TestOpenAIResponses_SendsANoticeAsADeveloperMessage(t *testing.T) {
	t.Parallel()

	items := encodedItems(t, toResponsesInput("", noticeConversation(), nil))

	require.Len(t, items, 6)
	assert.Equal(t, "user", items[3]["role"])
	assert.Equal(t, "developer", items[4]["role"])
	assert.Equal(t, "developer", items[5]["role"])
}

func TestOpenAIChat_SendsANoticeAsText(t *testing.T) {
	t.Parallel()

	out := toChatMessages("", noticeConversation(), pulid.Nil, "gpt-5.2")

	require.Len(t, out, 6)
	assert.Equal(t, "user", out[4].Role,
		"an OpenAI-compatible host may refuse a system message mid-conversation")
	assert.Equal(t, steerNotice, out[4].Content)
}
