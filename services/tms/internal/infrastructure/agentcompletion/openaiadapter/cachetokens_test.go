package openaiadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
)

func TestOpenAIChatAdapter_ReadsCachedPromptTokens(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "application/json", `{"model":"model-b",`+
		`"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Done."}}],`+
		`"usage":{"prompt_tokens":2400,"completion_tokens":6,"prompt_tokens_details":{"cached_tokens":2048}}}`)

	resp := modeladaptertest.CompleteWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	))

	assert.Equal(t, 2400, resp.InputTokens)
	assert.Equal(t, 2048, resp.CacheReadTokens)
	assert.Zero(t, resp.CacheWriteTokens, "the protocol reports no cache writes")
	assert.False(t, aiprovider.KindOpenAIChat.CachesOutsideInput())
}

func TestOpenAIChatAdapter_StreamReadsCachedPromptTokens(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"model-b","choices":[{"index":0,"delta":{"role":"assistant","content":"Done."},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"model-b","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		},
		[2]string{
			"",
			`{"model":"model-b","choices":[],"usage":{"prompt_tokens":900,"completion_tokens":3,"prompt_tokens_details":{"cached_tokens":512}}}`,
		},
		[2]string{"", `[DONE]`},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	))

	assert.Equal(t, 900, resp.InputTokens)
	assert.Equal(t, 512, resp.CacheReadTokens)
}

func TestOpenAIResponsesAdapter_ReadsCachedInputTokens(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(
		t,
		"application/json",
		`{"id":"resp_1","model":"model-c","status":"completed",`+
			`"output":[{"type":"message","content":[{"type":"output_text","text":"Done."}]}],`+
			`"usage":{"input_tokens":3000,"output_tokens":5,"input_tokens_details":{"cached_tokens":2816}}}`,
	)

	resp := modeladaptertest.CompleteWith(t, NewOpenAIResponsesAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIResponses,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	))

	assert.Equal(t, 3000, resp.InputTokens)
	assert.Equal(t, 2816, resp.CacheReadTokens)
	assert.False(t, aiprovider.KindOpenAIResponses.CachesOutsideInput())
}

func TestOpenAIResponsesAdapter_StreamReadsCachedInputTokens(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"response.output_text.delta",
			`{"type":"response.output_text.delta","delta":"Done."}`,
		},
		[2]string{
			"response.completed",
			`{"type":"response.completed","response":{"model":"model-c","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Done."}]}],"usage":{"input_tokens":1500,"output_tokens":4,"input_tokens_details":{"cached_tokens":1024}}}}`,
		},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIResponsesAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIResponses,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	))

	assert.Equal(t, 1500, resp.InputTokens)
	assert.Equal(t, 1024, resp.CacheReadTokens)
}
