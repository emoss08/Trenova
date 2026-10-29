package openaiadapter

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func embedServer(
	t *testing.T,
	reply func(body map[string]any) any,
) (*httptest.Server, *modeladaptertest.EmbedCapture) {
	t.Helper()

	capture := &modeladaptertest.EmbedCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body := make(map[string]any)
		if err = sonic.Unmarshal(raw, &body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		capture.Mu.Lock()
		capture.Path = r.URL.Path
		capture.Auth = r.Header.Get("Authorization")
		capture.Body = body
		capture.Mu.Unlock()

		encoded, err := sonic.Marshal(reply(body))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded)
	}))
	t.Cleanup(server.Close)

	return server, capture
}

func vectorOf(dimensions int, seed float32) []float32 {
	vector := make([]float32, dimensions)
	for idx := range vector {
		vector[idx] = seed
	}

	return vector
}

func embeddingProvider(
	kind aiprovider.Kind,
	baseURL string,
	dimensions int,
	style aiprovider.EmbeddingInputStyle,
) *aiprovider.Provider {
	return &aiprovider.Provider{
		ID:                  pulid.MustNew("aiprv_"),
		Name:                "embedder",
		Kind:                kind,
		BaseURL:             baseURL,
		Model:               "embed-model",
		Tasks:               []aitask.Task{aitask.TaskEmbedding},
		EmbeddingDimensions: &dimensions,
		EmbeddingInputStyle: style,
		Enabled:             true,
	}
}

func embedCallFor(
	provider *aiprovider.Provider,
	purpose aiusageports.EmbeddingPurpose,
	inputs ...string,
) *modeladapter.EmbedCall {
	return &modeladapter.EmbedCall{
		Provider: provider,
		APIKey:   "secret",
		Client:   &http.Client{Timeout: 5 * time.Second},
		Purpose:  purpose,
		Inputs:   inputs,
	}
}

func inputsOf(t *testing.T, body map[string]any) []string {
	t.Helper()

	raw, ok := body["input"].([]any)
	require.True(t, ok, "input is sent as an array: %v", body["input"])

	out := make([]string, 0, len(raw))
	for _, value := range raw {
		text, isText := value.(string)
		require.True(t, isText)
		out = append(out, text)
	}

	return out
}

func TestOpenAIResponsesEmbed_UsesTheVersionedPathAndOrdersByIndex(t *testing.T) {
	t.Parallel()

	server, capture := embedServer(t, func(map[string]any) any {
		return map[string]any{
			"model": "text-embedding-3-small",
			"data": []map[string]any{
				{"index": 1, "embedding": vectorOf(4, 2)},
				{"index": 0, "embedding": vectorOf(4, 1)},
			},
			"usage": map[string]any{"prompt_tokens": 9, "total_tokens": 9},
		}
	})

	provider := embeddingProvider(
		aiprovider.KindOpenAIResponses, server.URL, 4, aiprovider.EmbeddingInputStyleNone,
	)
	resp, err := NewOpenAIResponsesAdapter().(modeladapter.Embedder).Embed(
		t.Context(),
		embedCallFor(provider, aiusageports.EmbeddingPurposeDocument, "first", "second"),
	)
	require.NoError(t, err)

	path, auth, body := capture.Snapshot()
	assert.Equal(t, "/v1/embeddings", path)
	assert.Equal(t, "Bearer secret", auth)
	assert.Equal(t, "embed-model", body["model"])
	assert.EqualValues(t, 4, body["dimensions"])
	assert.NotContains(t, body, "input_type")
	assert.Equal(t, []string{"first", "second"}, inputsOf(t, body))

	require.Len(t, resp.Vectors, 2)
	assert.InDelta(t, 1, resp.Vectors[0][0], 0.0001, "vectors follow the index, not arrival order")
	assert.InDelta(t, 2, resp.Vectors[1][0], 0.0001)
	assert.Equal(t, 9, resp.InputTokens)
	assert.Equal(t, "text-embedding-3-small", resp.ModelIdentifier)
}

func TestOpenAIChatEmbed_SendsVoyageInputTypeAndOutputDimension(t *testing.T) {
	t.Parallel()

	server, capture := embedServer(t, func(map[string]any) any {
		return map[string]any{
			"data":  []map[string]any{{"index": 0, "embedding": vectorOf(8, 1)}},
			"usage": map[string]any{"total_tokens": 3},
		}
	})

	provider := embeddingProvider(
		aiprovider.KindOpenAIChat,
		server.URL+"/v1",
		8,
		aiprovider.EmbeddingInputStyleVoyageInputType,
	)
	adapter := NewOpenAIChatAdapter().(modeladapter.Embedder)

	_, err := adapter.Embed(
		t.Context(),
		embedCallFor(provider, aiusageports.EmbeddingPurposeDocument, "a rate confirmation"),
	)
	require.NoError(t, err)

	path, _, body := capture.Snapshot()
	assert.Equal(
		t,
		"/v1/embeddings",
		path,
		"the OpenAI-compatible base already carries its version",
	)
	assert.Equal(t, "document", body["input_type"])
	assert.EqualValues(t, 8, body["output_dimension"])
	assert.NotContains(t, body, "dimensions", "Voyage names the size output_dimension")

	_, err = adapter.Embed(
		t.Context(),
		embedCallFor(provider, aiusageports.EmbeddingPurposeQuery, "reefer loads"),
	)
	require.NoError(t, err)

	_, _, body = capture.Snapshot()
	assert.Equal(t, "query", body["input_type"])
	assert.Equal(t, []string{"reefer loads"}, inputsOf(t, body), "Voyage takes no text prefix")
}

func TestOpenAIChatEmbed_LeavesDimensionsToTheExtraBodyWhenItNamesThem(t *testing.T) {
	t.Parallel()

	server, capture := embedServer(t, func(map[string]any) any {
		return map[string]any{
			"data": []map[string]any{{"index": 0, "embedding": vectorOf(4, 1)}},
		}
	})

	provider := embeddingProvider(
		aiprovider.KindOpenAIChat, server.URL, 4, aiprovider.EmbeddingInputStyleNone,
	)
	provider.ExtraBody = map[string]any{"dimensions": nil, "truncate_prompt_tokens": 512}

	_, err := NewOpenAIChatAdapter().(modeladapter.Embedder).Embed(
		t.Context(),
		embedCallFor(provider, aiusageports.EmbeddingPurposeDocument, "text"),
	)
	require.NoError(t, err)

	_, _, body := capture.Snapshot()
	value, present := body["dimensions"]
	assert.True(t, present)
	assert.Nil(t, value, "a server that refuses the field is configured with an explicit null")
	assert.EqualValues(t, 512, body["truncate_prompt_tokens"])
	assert.Equal(t, "embed-model", body["model"], "the extra body cannot redirect the model")
}

func TestEmbed_WrongDimensionIsANonRetryableConfigurationError(t *testing.T) {
	t.Parallel()

	server, _ := embedServer(t, func(map[string]any) any {
		return map[string]any{
			"data": []map[string]any{{"index": 0, "embedding": vectorOf(3072, 1)}},
		}
	})

	provider := embeddingProvider(
		aiprovider.KindOpenAIChat, server.URL, 1536, aiprovider.EmbeddingInputStyleNone,
	)

	_, err := NewOpenAIChatAdapter().(modeladapter.Embedder).Embed(
		t.Context(),
		embedCallFor(provider, aiusageports.EmbeddingPurposeDocument, "text"),
	)
	require.Error(t, err)

	assert.ErrorIs(t, err, aiusageports.ErrEmbeddingDimensionMismatch)
	var business *errortypes.BusinessError
	assert.True(t, errors.As(err, &business), "a person has to change the configuration")
	assert.False(t, modeladapter.IsRetryable(err), "asking again returns the same size")
}

func TestEmbed_RejectsAReplyWithTheWrongNumberOfVectors(t *testing.T) {
	t.Parallel()

	server, _ := embedServer(t, func(map[string]any) any {
		return map[string]any{
			"data": []map[string]any{{"index": 0, "embedding": vectorOf(4, 1)}},
		}
	})

	provider := embeddingProvider(
		aiprovider.KindOpenAIChat, server.URL, 4, aiprovider.EmbeddingInputStyleNone,
	)

	_, err := NewOpenAIChatAdapter().(modeladapter.Embedder).Embed(
		t.Context(),
		embedCallFor(provider, aiusageports.EmbeddingPurposeDocument, "one", "two"),
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, aiusageports.ErrEmbeddingResponseInvalid)
	assert.False(t, modeladapter.IsRetryable(err))
}

func TestEmbed_ReadsAReplyWithoutIndexesInArrivalOrder(t *testing.T) {
	t.Parallel()

	server, _ := embedServer(t, func(map[string]any) any {
		return map[string]any{
			"data": []map[string]any{
				{"embedding": vectorOf(2, 1)},
				{"embedding": vectorOf(2, 2)},
			},
		}
	})

	provider := embeddingProvider(
		aiprovider.KindOpenAIChat, server.URL, 2, aiprovider.EmbeddingInputStyleNone,
	)

	resp, err := NewOpenAIChatAdapter().(modeladapter.Embedder).Embed(
		t.Context(),
		embedCallFor(provider, aiusageports.EmbeddingPurposeDocument, "one", "two"),
	)
	require.NoError(t, err)
	assert.InDelta(t, 1, resp.Vectors[0][0], 0.0001)
	assert.InDelta(t, 2, resp.Vectors[1][0], 0.0001)
}

func TestEmbed_ProviderRefusalIsATransportError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"slow down"}}`))
	}))
	t.Cleanup(server.Close)

	provider := embeddingProvider(
		aiprovider.KindOpenAIChat, server.URL, 4, aiprovider.EmbeddingInputStyleNone,
	)

	_, err := NewOpenAIChatAdapter().(modeladapter.Embedder).Embed(
		t.Context(),
		embedCallFor(provider, aiusageports.EmbeddingPurposeDocument, "text"),
	)

	var transport *modeladapter.TransportError
	require.ErrorAs(t, err, &transport)
	assert.Equal(t, http.StatusTooManyRequests, transport.StatusCode)
	assert.True(t, modeladapter.IsRetryable(err))
}
