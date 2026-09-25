package modeladapter

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func completeWith(t *testing.T, adapter Adapter, call *Call) *Response {
	t.Helper()

	resp, err := adapter.Complete(t.Context(), call)
	require.NoError(t, err)

	return resp
}

func TestAnthropicAdapter_ReadsPromptCacheTokens(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "application/json", `{"model":"model-a","stop_reason":"end_turn",`+
		`"content":[{"type":"text","text":"Done."}],`+
		`"usage":{"input_tokens":40,"output_tokens":7,"cache_read_input_tokens":1800,"cache_creation_input_tokens":120}}`)

	resp := completeWith(t, NewAnthropicAdapter(), callFor(
		aiprovider.KindAnthropicMessages, server.URL, &Request{Messages: UserMessage("hi")},
	))

	assert.Equal(t, 40, resp.InputTokens)
	assert.Equal(t, 7, resp.OutputTokens)
	assert.Equal(t, 1800, resp.CacheReadTokens)
	assert.Equal(t, 120, resp.CacheWriteTokens)
	assert.True(t, CacheSeparateFromInput(aiprovider.KindAnthropicMessages))
}

func TestAnthropicAdapter_StreamReadsPromptCacheTokens(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"model-a","usage":{"input_tokens":12,"output_tokens":1,"cache_read_input_tokens":2048,"cache_creation_input_tokens":64}}}`,
		},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Done."}}`,
		},
		[2]string{
			"message_delta",
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":9}}`,
		},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	resp, _ := streamWith(t, NewAnthropicAdapter(), callFor(
		aiprovider.KindAnthropicMessages, server.URL, &Request{Messages: UserMessage("hi")},
	))

	assert.Equal(t, 12, resp.InputTokens)
	assert.Equal(t, 9, resp.OutputTokens)
	assert.Equal(t, 2048, resp.CacheReadTokens)
	assert.Equal(t, 64, resp.CacheWriteTokens)
}

func TestOpenAIChatAdapter_ReadsCachedPromptTokens(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "application/json", `{"model":"model-b",`+
		`"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Done."}}],`+
		`"usage":{"prompt_tokens":2400,"completion_tokens":6,"prompt_tokens_details":{"cached_tokens":2048}}}`)

	resp := completeWith(t, NewOpenAIChatAdapter(), callFor(
		aiprovider.KindOpenAIChat, server.URL, &Request{Messages: UserMessage("hi")},
	))

	assert.Equal(t, 2400, resp.InputTokens)
	assert.Equal(t, 2048, resp.CacheReadTokens)
	assert.Zero(t, resp.CacheWriteTokens, "the protocol reports no cache writes")
	assert.False(t, CacheSeparateFromInput(aiprovider.KindOpenAIChat))
}

func TestOpenAIChatAdapter_StreamReadsCachedPromptTokens(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
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

	resp, _ := streamWith(t, NewOpenAIChatAdapter(), callFor(
		aiprovider.KindOpenAIChat, server.URL, &Request{Messages: UserMessage("hi")},
	))

	assert.Equal(t, 900, resp.InputTokens)
	assert.Equal(t, 512, resp.CacheReadTokens)
}

func TestOpenAIResponsesAdapter_ReadsCachedInputTokens(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(
		t,
		"application/json",
		`{"id":"resp_1","model":"model-c","status":"completed",`+
			`"output":[{"type":"message","content":[{"type":"output_text","text":"Done."}]}],`+
			`"usage":{"input_tokens":3000,"output_tokens":5,"input_tokens_details":{"cached_tokens":2816}}}`,
	)

	resp := completeWith(t, NewOpenAIResponsesAdapter(), callFor(
		aiprovider.KindOpenAIResponses, server.URL, &Request{Messages: UserMessage("hi")},
	))

	assert.Equal(t, 3000, resp.InputTokens)
	assert.Equal(t, 2816, resp.CacheReadTokens)
	assert.False(t, CacheSeparateFromInput(aiprovider.KindOpenAIResponses))
}

func TestOpenAIResponsesAdapter_StreamReadsCachedInputTokens(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{
			"response.output_text.delta",
			`{"type":"response.output_text.delta","delta":"Done."}`,
		},
		[2]string{
			"response.completed",
			`{"type":"response.completed","response":{"model":"model-c","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Done."}]}],"usage":{"input_tokens":1500,"output_tokens":4,"input_tokens_details":{"cached_tokens":1024}}}}`,
		},
	))

	resp, _ := streamWith(t, NewOpenAIResponsesAdapter(), callFor(
		aiprovider.KindOpenAIResponses, server.URL, &Request{Messages: UserMessage("hi")},
	))

	assert.Equal(t, 1500, resp.InputTokens)
	assert.Equal(t, 1024, resp.CacheReadTokens)
}

func TestOllamaAdapter_ReportsNoPromptCache(t *testing.T) {
	t.Parallel()

	body := strings.Join([]string{
		`{"model":"model-d","message":{"role":"assistant","content":"Done."},"done":true,"done_reason":"stop","prompt_eval_count":30,"eval_count":2}`,
	}, "\n") + "\n"
	server, _ := streamServer(t, "application/x-ndjson", body)

	resp, _ := streamWith(t, NewOllamaAdapter(), callFor(
		aiprovider.KindOllama, server.URL, &Request{Messages: UserMessage("hi")},
	))

	assert.Equal(t, 30, resp.InputTokens)
	assert.Zero(t, resp.CacheReadTokens)
	assert.Zero(t, resp.CacheWriteTokens)
	assert.False(t, CacheSeparateFromInput(aiprovider.KindOllama))
}
