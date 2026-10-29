package openaiadapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatAdapter_StreamsTextThenAssemblesToolCallFragments(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
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

	resp, deltas := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{
			Messages: modeladapter.UserMessage("Where is S1?"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))

	assert.Equal(t, []string{"Look", "ing."}, deltas)
	assert.Equal(t, "Looking.", resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_1", resp.ToolCalls[0].ID)
	assert.False(t, resp.ToolCalls[0].SynthesizedID, "the provider's own id is its own")
	assert.Equal(t, "lookup_shipment", resp.ToolCalls[0].Name)
	assert.Equal(t, "S1", resp.ToolCalls[0].Arguments["number"])
	assert.Equal(t, 7, resp.InputTokens)
	assert.Equal(t, 4, resp.OutputTokens)
	assert.Equal(t, true, (*captured)["stream"])
}

func TestOpenAIChatAdapter_StreamReportsContentFilterAsRefusal(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"I"},"finish_reason":"content_filter"}]}`,
		},
		[2]string{"", `[DONE]`},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	))

	assert.True(t, resp.Refused)
}

func TestOpenAIResponsesAdapter_StreamsDeltasAndTakesFinalFromCompleted(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
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

	resp, deltas := modeladaptertest.StreamWith(
		t,
		NewOpenAIResponsesAdapter(),
		modeladaptertest.CallFor(
			aiprovider.KindOpenAIResponses,
			server.URL,
			&modeladapter.Request{
				Messages: modeladapter.UserMessage("Where is S9?"),
				Tools:    modeladaptertest.LookupTool(),
			},
		),
	)

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

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"response.failed",
			`{"type":"response.failed","response":{"status":"failed","error":{"code":"server_error","message":"upstream exploded"}}}`,
		},
	))

	streamer := NewOpenAIResponsesAdapter().(modeladapter.Streamer)
	_, err := streamer.Stream(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIResponses,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	), func(string) {})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "upstream exploded")
}

/*
A streamed call the provider gave no id is numbered by its position, so the
first call of every response is call_0. It is marked as made up, so the loop
gives it an id of its own rather than letting call_0 of one turn overwrite the
artifacts and pairing of call_0 of another.
*/
func TestOpenAIChatAdapter_MarksAPositionalIDAsSynthesized(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"type":"function","function":{"name":"lookup_shipment","arguments":"{\"number\":\"S1\"}"}}]},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		},
		[2]string{"", `[DONE]`},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{
			Messages: modeladapter.UserMessage("Where is S1?"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_0", resp.ToolCalls[0].ID)
	assert.True(t, resp.ToolCalls[0].SynthesizedID)
	assert.Equal(t, "S1", resp.ToolCalls[0].Arguments["number"])
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

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
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

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{
			Messages: modeladapter.UserMessage("Where is S1?"),
			Tools:    modeladaptertest.LookupTool(),
		},
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
	assert.Equal(t, 7*time.Second, modeladapter.RetryAfterFrom(header, nil))

	dated := http.Header{}
	dated.Set("Retry-After", time.Now().Add(90*time.Second).UTC().Format(http.TimeFormat))
	assert.InDelta(t, 90, modeladapter.RetryAfterFrom(dated, nil).Seconds(), 2)

	google := []byte(
		`{"error":{"code":429,"message":"You exceeded your current quota","status":"RESOURCE_EXHAUSTED",` +
			`"details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"34s"}]}}`,
	)
	assert.Equal(t, 34*time.Second, modeladapter.RetryAfterFrom(http.Header{}, google))

	assert.Equal(
		t,
		time.Duration(0),
		modeladapter.RetryAfterFrom(http.Header{}, []byte(`{"error":{"message":"overloaded"}}`)),
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(
			[]byte(`{"error":{"message":"The model is overloaded. Please try again later."}}`),
		)
	}))
	t.Cleanup(server.Close)

	_, err := NewOpenAIChatAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	))
	var transport *modeladapter.TransportError
	require.ErrorAs(t, err, &transport)
	assert.Equal(t, 3*time.Second, transport.RetryAfter)
	assert.True(t, transport.Retryable)
}
