package modeladapter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const anthropicOKReply = `{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"ok"}],` +
	`"usage":{"input_tokens":1,"output_tokens":1}}`

type seenRequest struct {
	path    string
	headers http.Header
}

func recordingServer(
	t *testing.T,
	status int,
	contentType string,
	reply string,
) (*httptest.Server, func() []seenRequest) {
	t.Helper()

	var (
		mu   sync.Mutex
		seen []seenRequest
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, seenRequest{path: r.URL.Path, headers: r.Header.Clone()})
		mu.Unlock()
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Retry-After", "4")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)

	return server, func() []seenRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]seenRequest(nil), seen...)
	}
}

func anthropicCall(url string) *Call {
	call := callFor(aiprovider.KindAnthropicMessages, url, &Request{Messages: UserMessage("hi")})
	call.Provider.Model = "claude-opus-4-8"
	call.APIKey = "sk-provider"

	return call
}

/*
A server hosts many organizations, each with its own provider and key. The
SDK's default client reads Anthropic credentials, an address and extra headers
from the process; any of those reaching a request would send the server's
credentials to whatever address an operator configured. A request carries the
provider's key, goes to the provider's address, and nothing else.
*/
func TestAnthropicAdapter_TakesNothingFromTheEnvironment(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  string
		env  map[string]string
	}{
		{
			name: "server key beside a provider key",
			key:  "sk-provider",
			env:  map[string]string{"ANTHROPIC_API_KEY": "sk-from-env"},
		},
		{
			name: "server token beside a provider without a key",
			env:  map[string]string{"ANTHROPIC_AUTH_TOKEN": "token-from-env"},
		},
		{
			name: "server headers and profile beside a provider without a key",
			env: map[string]string{
				"ANTHROPIC_CUSTOM_HEADERS": "X-Leak: yes",
				"ANTHROPIC_PROFILE":        "missing-profile",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			elsewhere, elsewhereSeen := recordingServer(t, http.StatusOK, "application/json", anthropicOKReply)
			t.Setenv("ANTHROPIC_BASE_URL", elsewhere.URL)
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			server, seen := recordingServer(t, http.StatusOK, "application/json", anthropicOKReply)
			call := anthropicCall(server.URL)
			call.APIKey = tc.key
			_, err := NewAnthropicAdapter().Complete(t.Context(), call)
			require.NoError(t, err)

			requests := seen()
			require.Len(t, requests, 1)
			assert.Equal(t, tc.key, requests[0].headers.Get("X-Api-Key"))
			assert.Empty(t, requests[0].headers.Get("Authorization"))
			assert.Empty(t, requests[0].headers.Get("X-Leak"))
			assert.Empty(t, elsewhereSeen())
		})
	}
}

/*
Retries belong to the router and the workflow, which count every attempt and
rest a provider that keeps failing. A client that retried underneath them would
turn one failure into three and hide it from the breaker, so a failed call is
one request, and its error carries the status, the provider's message and how
long it asked to be left alone.
*/
func TestAnthropicAdapter_MakesOneAttemptAndReportsTheFailure(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		status      int
		contentType string
		reply       string
		retryable   bool
		message     string
	}{
		{
			name:        "overloaded",
			status:      529,
			contentType: "application/json",
			reply:       `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`,
			retryable:   true,
			message:     "Overloaded",
		},
		{
			name:        "rate limited",
			status:      http.StatusTooManyRequests,
			contentType: "application/json",
			reply:       `{"type":"error","error":{"type":"rate_limit_error","message":"Slow down"}}`,
			retryable:   true,
			message:     "Slow down",
		},
		{
			name:        "invalid request",
			status:      http.StatusBadRequest,
			contentType: "application/json",
			reply:       `{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: too large"}}`,
			retryable:   false,
			message:     "max_tokens: too large",
		},
		{
			name:        "gateway error page",
			status:      http.StatusBadGateway,
			contentType: "text/html",
			reply:       "<html><body>Bad Gateway</body></html>",
			retryable:   true,
			message:     "Bad Gateway",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, stream := range []bool{false, true} {
				server, seen := recordingServer(t, tc.status, tc.contentType, tc.reply)
				call := anthropicCall(server.URL)

				var err error
				if stream {
					_, err = NewAnthropicAdapter().(Streamer).Stream(t.Context(), call, func(string) {})
				} else {
					_, err = NewAnthropicAdapter().Complete(t.Context(), call)
				}

				var transport *TransportError
				require.ErrorAs(t, err, &transport, "stream=%v", stream)
				assert.Equal(t, tc.status, transport.StatusCode, "stream=%v", stream)
				assert.Equal(t, tc.retryable, transport.Retryable, "stream=%v", stream)
				assert.Equal(t, tc.retryable, IsRetryable(err), "stream=%v", stream)
				assert.Contains(t, transport.Message, tc.message, "stream=%v", stream)
				assert.Equal(t, 4*time.Second, transport.RetryAfter, "stream=%v", stream)
				assert.Len(t, seen(), 1, "stream=%v made more than one attempt", stream)
			}
		})
	}
}

// An error the provider reports inside a stream is worth another attempt only
// when the fault is on its side.
func TestAnthropicAdapter_StreamErrorEventKeepsItsRetryability(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{"message_start", `{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":1}}}`},
		[2]string{"error", `{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long"}}`},
	))

	_, err := NewAnthropicAdapter().(Streamer).Stream(t.Context(), anthropicCall(server.URL), func(string) {})

	var transport *TransportError
	require.ErrorAs(t, err, &transport)
	assert.False(t, transport.Retryable)
	assert.Contains(t, transport.Message, "prompt is too long")
	assert.Equal(t, "claude-x", ServedModel(err, ""), "a stream that failed keeps the model that served it")
}

// A provider that opens a stream and then goes silent is abandoned after the
// call's idle timeout, rather than holding the turn open until it is killed.
func TestAnthropicAdapter_AbandonsAStreamThatGoesSilent(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse(
			[2]string{"message_start", `{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":1}}}`},
		)))
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	call := anthropicCall(server.URL)
	call.Client = &http.Client{}
	call.StreamIdle = 100 * time.Millisecond

	started := time.Now()
	_, err := NewAnthropicAdapter().(Streamer).Stream(t.Context(), call, func(string) {})

	require.ErrorIs(t, err, ErrStreamStalled)
	assert.Less(t, time.Since(started), 5*time.Second)
	assert.True(t, IsRetryable(err))
}

// The SDK refuses a blocking request it expects to run long unless the call
// has a deadline. Every client the router builds has one, and a call that
// brings a client without one still gets through.
func TestAnthropicAdapter_BlockingCallTakesALargeReplyCeiling(t *testing.T) {
	t.Parallel()

	server, _ := recordingServer(t, http.StatusOK, "application/json", anthropicOKReply)
	for _, client := range []*http.Client{{Timeout: 5 * time.Second}, {}} {
		call := anthropicCall(server.URL)
		call.Client = client
		call.Request.MaxTokens = 128_000

		resp, err := NewAnthropicAdapter().Complete(t.Context(), call)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp.Text)
		assert.Equal(t, 128_000, resp.OutputLimit)
	}
}

// A gateway serves the Messages API under a path of its own, and a provider
// whose endpoint takes no key sends none.
func TestAnthropicAdapter_KeepsAGatewayPathAndSendsNoEmptyKey(t *testing.T) {
	t.Parallel()

	server, seen := recordingServer(t, http.StatusOK, "application/json", anthropicOKReply)
	call := anthropicCall(server.URL + "/gateway/anthropic/")
	call.APIKey = ""

	_, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	requests := seen()
	require.Len(t, requests, 1)
	assert.Equal(t, "/gateway/anthropic/v1/messages", requests[0].path)
	assert.NotContains(t, requests[0].headers, "X-Api-Key")
	assert.Empty(t, requests[0].headers.Get("Authorization"))
}

// A reply whose tool input is not an object is reported on the call, the same
// whether it streamed or not, so the runtime refuses it instead of running the
// tool on no arguments.
func TestAnthropicAdapter_ReportsToolInputThatDoesNotParse(t *testing.T) {
	t.Parallel()

	server, _ := recordingServer(t, http.StatusOK, "application/json",
		`{"model":"m","stop_reason":"tool_use","content":[`+
			`{"type":"tool_use","id":"toolu_1","name":"lookup_shipment","input":"S1"}],`+
			`"usage":{"input_tokens":1,"output_tokens":1}}`)

	resp, err := NewAnthropicAdapter().Complete(t.Context(), anthropicCall(server.URL))
	require.NoError(t, err)
	require.Len(t, resp.ToolCalls, 1)
	assert.NotEmpty(t, resp.ToolCalls[0].ArgumentsError)
	assert.Empty(t, resp.ToolCalls[0].Arguments)
}

// Requests are independent: one in flight does not hold up another, and none
// shares state with the next.
func TestAnthropicAdapter_ServesConcurrentCalls(t *testing.T) {
	t.Parallel()

	var served atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(anthropicOKReply))
	}))
	t.Cleanup(server.Close)

	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for range 16 {
		wg.Go(func() {
			_, err := NewAnthropicAdapter().Complete(t.Context(), anthropicCall(server.URL))
			errs <- err
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, int32(16), served.Load())
}
