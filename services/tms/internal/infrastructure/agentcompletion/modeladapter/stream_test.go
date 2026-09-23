package modeladapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// streamServer replies with a canned streaming body and records the request so
// the stream flag the adapter set can be asserted.
func streamServer(
	t *testing.T,
	contentType string,
	body string,
) (*httptest.Server, *map[string]any) {
	t.Helper()

	captured := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&decoded)
		for k, v := range decoded {
			captured[k] = v
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server, &captured
}

func sse(events ...[2]string) string {
	var builder strings.Builder
	for _, event := range events {
		if event[0] != "" {
			builder.WriteString("event: " + event[0] + "\n")
		}
		builder.WriteString("data: " + event[1] + "\n\n")
	}

	return builder.String()
}

func collectDeltas() (StreamSink, *[]string) {
	deltas := []string{}
	return func(delta string) { deltas = append(deltas, delta) }, &deltas
}

func streamWith(t *testing.T, adapter Adapter, call *Call) (*Response, []string) {
	t.Helper()

	streamer, ok := adapter.(Streamer)
	require.True(t, ok, "adapter %s must stream", adapter.Kind())

	sink, deltas := collectDeltas()
	resp, err := streamer.Stream(t.Context(), call, sink)
	require.NoError(t, err)

	return resp, *deltas
}

func TestOpenAIChatAdapter_StreamsTextThenAssemblesToolCallFragments(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"Look"},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"ing."},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"lookup_shipment","arguments":"{\"num"}}]},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ber\":\"S1\"}"}}]},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[],"usage":{"prompt_tokens":7,"completion_tokens":4}}`,
		},
		[2]string{"", `[DONE]`},
	))

	resp, deltas := streamWith(t, NewOpenAIChatAdapter(), callFor(
		aiprovider.KindOpenAIChat, server.URL,
		&Request{Messages: UserMessage("Where is S1?"), Tools: lookupTool()},
	))

	assert.Equal(t, []string{"Look", "ing."}, deltas)
	assert.Equal(t, "Looking.", resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "lookup_shipment", resp.ToolCalls[0].Name)
	assert.Equal(t, "S1", resp.ToolCalls[0].Arguments["number"])
	assert.Equal(t, 7, resp.InputTokens)
	assert.Equal(t, 4, resp.OutputTokens)
	assert.Equal(t, true, (*captured)["stream"])
}

func TestOpenAIChatAdapter_StreamReportsContentFilterAsRefusal(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"I"},"finish_reason":"content_filter"}]}`,
		},
		[2]string{"", `[DONE]`},
	))

	resp, _ := streamWith(t, NewOpenAIChatAdapter(), callFor(
		aiprovider.KindOpenAIChat, server.URL, &Request{Messages: UserMessage("hi")},
	))

	assert.True(t, resp.Refused)
}

func TestAnthropicAdapter_StreamsTextBlocksAndToolUseInput(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":9,"output_tokens":1}}}`,
		},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Looking"}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" that up."}}`,
		},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":0}`},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"lookup_shipment","input":{}}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"number\":"}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"S12345\"}"}}`,
		},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":1}`},
		[2]string{
			"message_delta",
			`{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":12}}`,
		},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	resp, deltas := streamWith(t, NewAnthropicAdapter(), callFor(
		aiprovider.KindAnthropicMessages, server.URL,
		&Request{System: "sys", Messages: UserMessage("Where is S12345?"), Tools: lookupTool()},
	))

	assert.Equal(t, []string{"Looking", " that up."}, deltas)
	assert.Equal(t, "Looking that up.", resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "toolu_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "S12345", resp.ToolCalls[0].Arguments["number"])
	assert.Equal(t, "claude-x", resp.ModelIdentifier)
	assert.Equal(t, 9, resp.InputTokens)
	assert.Equal(t, 12, resp.OutputTokens)
	assert.Equal(t, true, (*captured)["stream"])
}

func TestAnthropicAdapter_StreamErrorEventFails(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":1}}}`,
		},
		[2]string{
			"error",
			`{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`,
		},
	))

	streamer := NewAnthropicAdapter().(Streamer)
	_, err := streamer.Stream(t.Context(), callFor(
		aiprovider.KindAnthropicMessages, server.URL, &Request{Messages: UserMessage("hi")},
	), func(string) {})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Overloaded")
	assert.True(t, IsRetryable(err), "an overloaded stream is worth retrying")
}

func TestOpenAIResponsesAdapter_StreamsDeltasAndTakesFinalFromCompleted(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{"response.created", `{"type":"response.created","response":{"model":"gpt-x"}}`},
		[2]string{
			"response.output_text.delta",
			`{"type":"response.output_text.delta","delta":"Sure"}`,
		},
		[2]string{
			"response.output_text.delta",
			`{"type":"response.output_text.delta","delta":", one moment."}`,
		},
		[2]string{
			"response.function_call_arguments.delta",
			`{"type":"response.function_call_arguments.delta","delta":"{\"number\":\"S9\"}"}`,
		},
		[2]string{
			"response.completed",
			`{"type":"response.completed","response":{"model":"gpt-x","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Sure, one moment."}]},{"type":"function_call","call_id":"call_9","name":"lookup_shipment","arguments":"{\"number\":\"S9\"}"}],"usage":{"input_tokens":3,"output_tokens":8}}}`,
		},
	))

	resp, deltas := streamWith(t, NewOpenAIResponsesAdapter(), callFor(
		aiprovider.KindOpenAIResponses, server.URL,
		&Request{Messages: UserMessage("Where is S9?"), Tools: lookupTool()},
	))

	assert.Equal(t, []string{"Sure", ", one moment."}, deltas)
	assert.Equal(t, "Sure, one moment.", resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_9", resp.ToolCalls[0].ID)
	assert.Equal(t, "S9", resp.ToolCalls[0].Arguments["number"])
	assert.Equal(t, 3, resp.InputTokens)
	assert.Equal(t, 8, resp.OutputTokens)
	assert.Equal(t, true, (*captured)["stream"])
}

func TestOpenAIResponsesAdapter_StreamFailedEventSurfacesMessage(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{
			"response.failed",
			`{"type":"response.failed","response":{"status":"failed","error":{"code":"server_error","message":"upstream exploded"}}}`,
		},
	))

	streamer := NewOpenAIResponsesAdapter().(Streamer)
	_, err := streamer.Stream(t.Context(), callFor(
		aiprovider.KindOpenAIResponses, server.URL, &Request{Messages: UserMessage("hi")},
	), func(string) {})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "upstream exploded")
}

func TestOllamaAdapter_StreamsNDJSONAndCollectsToolCalls(t *testing.T) {
	t.Parallel()

	body := strings.Join([]string{
		`{"model":"qwen","message":{"role":"assistant","content":"Let me "},"done":false}`,
		`{"model":"qwen","message":{"role":"assistant","content":"check."},"done":false}`,
		`{"model":"qwen","message":{"role":"assistant","content":"","tool_calls":[{"function":{"name":"lookup_shipment","arguments":{"number":"S7"}}}]},"done":false}`,
		`{"model":"qwen","message":{"role":"assistant","content":""},"done":true,"done_reason":"stop","prompt_eval_count":11,"eval_count":6}`,
	}, "\n") + "\n"
	server, captured := streamServer(t, "application/x-ndjson", body)

	resp, deltas := streamWith(t, NewOllamaAdapter(), callFor(
		aiprovider.KindOllama, server.URL,
		&Request{Messages: UserMessage("Where is S7?"), Tools: lookupTool()},
	))

	assert.Equal(t, []string{"Let me ", "check."}, deltas)
	assert.Equal(t, "Let me check.", resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "lookup_shipment", resp.ToolCalls[0].Name)
	assert.Equal(t, "S7", resp.ToolCalls[0].Arguments["number"])
	assert.Equal(t, 11, resp.InputTokens)
	assert.Equal(t, 6, resp.OutputTokens)
	assert.Equal(t, true, (*captured)["stream"])
}

// A stream that fails before the first byte is an ordinary transport failure
// and must carry the provider's message and retryability like a blocking call.
func TestStream_NonSuccessStatusIsATransportError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"model is loading"}}`))
	}))
	t.Cleanup(server.Close)

	for _, adapter := range []Adapter{
		NewOpenAIChatAdapter(), NewAnthropicAdapter(), NewOpenAIResponsesAdapter(), NewOllamaAdapter(),
	} {
		streamer := adapter.(Streamer)
		_, err := streamer.Stream(t.Context(), callFor(
			adapter.Kind(), server.URL, &Request{Messages: UserMessage("hi")},
		), func(string) {})

		var transport *TransportError
		require.ErrorAs(t, err, &transport, "adapter %s", adapter.Kind())
		assert.Equal(t, http.StatusServiceUnavailable, transport.StatusCode)
		assert.True(t, transport.Retryable)
		assert.Contains(t, transport.Message, "model is loading")
	}
}

func TestReadSSE_JoinsMultiLineDataAndIgnoresComments(t *testing.T) {
	t.Parallel()

	body := ": keep-alive\n" +
		"event: ping\n" +
		"data: {\"a\":\n" +
		"data: 1}\n" +
		"\n" +
		"data: solo\n\n"

	var seen [][2]string
	err := readSSE(strings.NewReader(body), func(event, data string) error {
		seen = append(seen, [2]string{event, data})
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, [][2]string{{"ping", "{\"a\":\n1}"}, {"", "solo"}}, seen)
}

// A streamed call carries the provider's extra content on one of its
// fragments, and it has to survive the reassembly like the id and the name.
func TestOpenAIChatAdapter_StreamKeepsProviderDataOnTheCall(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"lookup_shipment","arguments":"{\"num"},"extra_content":{"google":{"thought_signature":"sig-9"}}}]},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ber\":\"S1\"}"}}]},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		},
		[2]string{"", `[DONE]`},
	))

	resp, _ := streamWith(t, NewOpenAIChatAdapter(), callFor(
		aiprovider.KindOpenAIChat, server.URL,
		&Request{Messages: UserMessage("Where is S1?"), Tools: lookupTool()},
	))

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "S1", resp.ToolCalls[0].Arguments["number"])
	assert.Equal(t,
		map[string]any{"google": map[string]any{"thought_signature": "sig-9"}},
		resp.ToolCalls[0].ProviderData,
	)
}

// A provider that says how long to wait is listened to. Google puts the
// delay in a RetryInfo detail of the body; most others use the header.
func TestTransportError_CarriesHowLongTheProviderAskedToWait(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	header.Set("Retry-After", "7")
	assert.Equal(t, 7*time.Second, retryAfterFrom(header, nil))

	dated := http.Header{}
	dated.Set("Retry-After", time.Now().Add(90*time.Second).UTC().Format(http.TimeFormat))
	assert.InDelta(t, 90, retryAfterFrom(dated, nil).Seconds(), 2)

	google := []byte(
		`{"error":{"code":429,"message":"You exceeded your current quota","status":"RESOURCE_EXHAUSTED",` +
			`"details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"34s"}]}}`,
	)
	assert.Equal(t, 34*time.Second, retryAfterFrom(http.Header{}, google))

	assert.Equal(
		t,
		time.Duration(0),
		retryAfterFrom(http.Header{}, []byte(`{"error":{"message":"overloaded"}}`)),
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(
			[]byte(`{"error":{"message":"The model is overloaded. Please try again later."}}`),
		)
	}))
	t.Cleanup(server.Close)

	_, err := NewOpenAIChatAdapter().Complete(t.Context(), callFor(
		aiprovider.KindOpenAIChat, server.URL, &Request{Messages: UserMessage("hi")},
	))
	var transport *TransportError
	require.ErrorAs(t, err, &transport)
	assert.Equal(t, 3*time.Second, transport.RetryAfter)
	assert.True(t, transport.Retryable)
}
