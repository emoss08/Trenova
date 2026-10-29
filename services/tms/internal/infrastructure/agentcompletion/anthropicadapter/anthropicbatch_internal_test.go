package anthropicadapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	batchCreatedReply = `{"id":"msgbatch_01","type":"message_batch","processing_status":"in_progress",` +
		`"request_counts":{"processing":3,"succeeded":0,"errored":0,"canceled":0,"expired":0},` +
		`"created_at":"2026-10-27T03:00:00Z","expires_at":"2026-10-28T03:00:00Z"}`
	batchEndedReply = `{"id":"msgbatch_01","type":"message_batch","processing_status":"ended",` +
		`"request_counts":{"processing":0,"succeeded":1,"errored":1,"canceled":0,"expired":1},` +
		`"created_at":"2026-10-27T03:00:00Z","expires_at":"2026-10-28T03:00:00Z",` +
		`"ended_at":"2026-10-27T03:20:00Z"}`
	batchResultsReply = `{"custom_id":"aeval_1","result":{"type":"succeeded","message":` +
		`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5-5",` +
		`"stop_reason":"end_turn","content":[{"type":"text","text":"{\"score\":0.8,\"rationale\":\"Right.\"}"}],` +
		`"usage":{"input_tokens":1200,"output_tokens":30,"cache_read_input_tokens":0,` +
		`"cache_creation_input_tokens":0}}}}` + "\n" +
		`{"custom_id":"aeval_2","result":{"type":"errored","error":{"type":"error",` +
		`"error":{"type":"invalid_request_error","message":"max_tokens is too large"}}}}` + "\n" +
		`{"custom_id":"aeval_3","result":{"type":"expired"}}` + "\n"
)

type batchRequestSeen struct {
	method string
	path   string
	beta   string
	body   map[string]any
}

func batchServer(t *testing.T) (*httptest.Server, func() []batchRequestSeen) {
	t.Helper()

	var (
		mu   sync.Mutex
		seen []batchRequestSeen
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		if r.Method == http.MethodPost {
			_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
		}
		mu.Lock()
		seen = append(seen, batchRequestSeen{
			method: r.Method,
			path:   r.URL.Path,
			beta:   strings.Join(r.Header.Values("Anthropic-Beta"), ","),
			body:   body,
		})
		mu.Unlock()

		switch {
		case r.URL.Path == "/v1/messages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(anthropicOKReply))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/messages/batches":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(batchCreatedReply))
		case strings.HasSuffix(r.URL.Path, "/results"):
			w.Header().Set("Content-Type", "application/x-jsonl")
			_, _ = w.Write([]byte(batchResultsReply))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(batchEndedReply))
		}
	}))
	t.Cleanup(server.Close)

	return server, func() []batchRequestSeen {
		mu.Lock()
		defer mu.Unlock()
		return append([]batchRequestSeen(nil), seen...)
	}
}

func judgeRequest(question string) *modeladapter.Request {
	return &modeladapter.Request{
		System:   "You grade one answer.",
		Messages: modeladapter.UserMessage(question),
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"score":     map[string]any{"type": "number", "minimum": 0, "maximum": 1},
				"rationale": map[string]any{"type": "string"},
			},
			"required":             []string{"score", "rationale"},
			"additionalProperties": false,
		},
		SchemaName: "evaluation_judgement",
		MaxTokens:  600,
	}
}

func TestAnthropicBatch_SendsEachRequestExactlyAsOneCallWould(t *testing.T) {
	t.Parallel()

	server, seen := batchServer(t)
	adapter := NewAnthropicAdapter().(anthropicAdapter)

	single := anthropicCall(server.URL)
	single.Request = judgeRequest("Was the invoice sent?")
	_, err := adapter.Complete(t.Context(), single)
	require.NoError(t, err)

	batched := anthropicCall(server.URL)
	submitted, err := adapter.SubmitBatch(t.Context(), batched, []modeladapter.BatchRequest{
		{CustomID: "aeval_1", Request: judgeRequest("Was the invoice sent?")},
		{CustomID: "aeval_2", Request: judgeRequest("Which load is late?")},
	})
	require.NoError(t, err)
	assert.Equal(t, "msgbatch_01", submitted.ID)
	assert.Equal(t, "in_progress", submitted.Status)
	assert.Positive(t, submitted.ExpiresAt)

	requests := seen()
	require.Len(t, requests, 2)
	assert.Equal(t, "/v1/messages", requests[0].path)
	assert.Equal(t, "/v1/messages/batches", requests[1].path)
	assert.Contains(t, requests[1].beta, "message-batches-2024-09-24")

	items, ok := requests[1].body["requests"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)

	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "aeval_1", first["custom_id"])
	assert.Equal(t, requests[0].body, first["params"],
		"a batched request carries the same model, prompt, schema and limits as one call")

	second, ok := items[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "aeval_2", second["custom_id"])
	params, ok := second["params"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, params, "output_config", "the schema is enforced on the wire")
}

func TestAnthropicBatch_RefusesAnEmptyBatch(t *testing.T) {
	t.Parallel()

	server, seen := batchServer(t)
	adapter := NewAnthropicAdapter().(anthropicAdapter)

	_, err := adapter.SubmitBatch(t.Context(), anthropicCall(server.URL), nil)
	require.ErrorIs(t, err, errEmptyBatch)
	assert.Empty(t, seen())
}

func TestAnthropicBatch_ReadsProgress(t *testing.T) {
	t.Parallel()

	server, seen := batchServer(t)
	adapter := NewAnthropicAdapter().(anthropicAdapter)

	progress, err := adapter.PollBatch(t.Context(), anthropicCall(server.URL), "msgbatch_01")
	require.NoError(t, err)
	assert.Equal(t, &modeladapter.BatchProgress{
		Ended:     true,
		RawStatus: "ended",
		Succeeded: 1,
		Errored:   1,
		Expired:   1,
	}, progress)

	requests := seen()
	require.Len(t, requests, 1)
	assert.Equal(t, http.MethodGet, requests[0].method)
	assert.Equal(t, "/v1/messages/batches/msgbatch_01", requests[0].path)
}

func TestAnthropicBatch_ReadsEveryResultByItsCustomID(t *testing.T) {
	t.Parallel()

	server, _ := batchServer(t)
	adapter := NewAnthropicAdapter().(anthropicAdapter)

	items := map[string]*modeladapter.BatchItem{}
	err := adapter.ReadBatch(t.Context(), anthropicCall(server.URL), "msgbatch_01",
		func(item *modeladapter.BatchItem) error {
			items[item.CustomID] = item
			return nil
		})
	require.NoError(t, err)
	require.Len(t, items, 3)

	succeeded := items["aeval_1"]
	require.NotNil(t, succeeded.Response)
	assert.JSONEq(t, `{"score":0.8,"rationale":"Right."}`, succeeded.Response.Text)
	assert.Equal(t, "claude-sonnet-5-5", succeeded.Response.ModelIdentifier)
	assert.Equal(t, 1200, succeeded.Response.InputTokens)
	assert.Equal(t, 30, succeeded.Response.OutputTokens)
	assert.Empty(t, succeeded.FailureCode)

	errored := items["aeval_2"]
	assert.Nil(t, errored.Response)
	assert.Equal(t, "invalid_request_error", errored.FailureCode)
	assert.Equal(t, "max_tokens is too large", errored.FailureMessage)

	expired := items["aeval_3"]
	assert.Nil(t, expired.Response)
	assert.Equal(t, "expired", expired.FailureCode)
	assert.True(t, expired.NotRun, "an expired request was never run and is not billed")
	assert.False(t, errored.NotRun, "an errored request was run")
}

func TestAnthropicBatch_OnlyTheClaudeAPITakesBatches(t *testing.T) {
	t.Parallel()

	adapter := NewAnthropicAdapter().(anthropicAdapter)
	claude := &aiprovider.Provider{
		Kind:    aiprovider.KindAnthropicMessages,
		BaseURL: "https://api.anthropic.com",
		Model:   "claude-sonnet-5-5",
	}
	assert.True(t, adapter.Batches(claude))

	gateway := &aiprovider.Provider{
		Kind:    aiprovider.KindAnthropicMessages,
		BaseURL: "https://llm-gateway.example.com",
		Model:   "claude-sonnet-5-5",
	}
	assert.False(t, adapter.Batches(gateway), "a gateway speaking Messages may have no batch API")
	assert.False(t, adapter.Batches(nil))
}
