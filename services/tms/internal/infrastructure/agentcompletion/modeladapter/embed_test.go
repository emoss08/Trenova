package modeladapter

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type embedCapture struct {
	mu   sync.Mutex
	path string
	auth string
	body map[string]any
}

func (c *embedCapture) snapshot() (string, string, map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.path, c.auth, c.body
}

func embedServer(t *testing.T, reply func(body map[string]any) any) (*httptest.Server, *embedCapture) {
	t.Helper()

	capture := &embedCapture{}
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

		capture.mu.Lock()
		capture.path = r.URL.Path
		capture.auth = r.Header.Get("Authorization")
		capture.body = body
		capture.mu.Unlock()

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
		Tasks:               []aiprovider.Task{aiprovider.TaskEmbedding},
		EmbeddingDimensions: &dimensions,
		EmbeddingInputStyle: style,
		Enabled:             true,
	}
}

func embedCallFor(
	provider *aiprovider.Provider,
	purpose serviceports.EmbeddingPurpose,
	inputs ...string,
) *EmbedCall {
	return &EmbedCall{
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
	resp, err := NewOpenAIResponsesAdapter().(Embedder).Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeDocument, "first", "second"),
	)
	require.NoError(t, err)

	path, auth, body := capture.snapshot()
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
		aiprovider.KindOpenAIChat, server.URL+"/v1", 8, aiprovider.EmbeddingInputStyleVoyageInputType,
	)
	adapter := NewOpenAIChatAdapter().(Embedder)

	_, err := adapter.Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeDocument, "a rate confirmation"),
	)
	require.NoError(t, err)

	path, _, body := capture.snapshot()
	assert.Equal(t, "/v1/embeddings", path, "the OpenAI-compatible base already carries its version")
	assert.Equal(t, "document", body["input_type"])
	assert.EqualValues(t, 8, body["output_dimension"])
	assert.NotContains(t, body, "dimensions", "Voyage names the size output_dimension")

	_, err = adapter.Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeQuery, "reefer loads"),
	)
	require.NoError(t, err)

	_, _, body = capture.snapshot()
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

	_, err := NewOpenAIChatAdapter().(Embedder).Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeDocument, "text"),
	)
	require.NoError(t, err)

	_, _, body := capture.snapshot()
	value, present := body["dimensions"]
	assert.True(t, present)
	assert.Nil(t, value, "a server that refuses the field is configured with an explicit null")
	assert.EqualValues(t, 512, body["truncate_prompt_tokens"])
	assert.Equal(t, "embed-model", body["model"], "the extra body cannot redirect the model")
}

func TestOllamaEmbed_UsesTheNativeEndpointWithNomicPrefixes(t *testing.T) {
	t.Parallel()

	server, capture := embedServer(t, func(body map[string]any) any {
		inputs, _ := body["input"].([]any)
		vectors := make([][]float32, 0, len(inputs))
		for range inputs {
			vectors = append(vectors, vectorOf(6, 1))
		}

		return map[string]any{
			"model":             "nomic-embed-text:latest",
			"embeddings":        vectors,
			"prompt_eval_count": 14,
		}
	})

	provider := embeddingProvider(
		aiprovider.KindOllama, server.URL, 6, aiprovider.EmbeddingInputStyleNomicPrefix,
	)
	adapter := NewOllamaAdapter().(Embedder)

	resp, err := adapter.Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeDocument, "one", "two"),
	)
	require.NoError(t, err)

	path, _, body := capture.snapshot()
	assert.Equal(t, "/api/embed", path)
	assert.EqualValues(t, 6, body["dimensions"])
	assert.Equal(
		t,
		[]string{"search_document: one", "search_document: two"},
		inputsOf(t, body),
	)
	assert.Len(t, resp.Vectors, 2)
	assert.Equal(t, 14, resp.InputTokens)
	assert.Equal(t, "nomic-embed-text:latest", resp.ModelIdentifier)

	_, err = adapter.Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeQuery, "late deliveries"),
	)
	require.NoError(t, err)

	_, _, body = capture.snapshot()
	assert.Equal(t, []string{"search_query: late deliveries"}, inputsOf(t, body))
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

	_, err := NewOpenAIChatAdapter().(Embedder).Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeDocument, "text"),
	)
	require.Error(t, err)

	assert.ErrorIs(t, err, serviceports.ErrEmbeddingDimensionMismatch)
	var business *errortypes.BusinessError
	assert.True(t, errors.As(err, &business), "a person has to change the configuration")
	assert.False(t, IsRetryable(err), "asking again returns the same size")
}

func TestEmbed_RejectsAReplyWithTheWrongNumberOfVectors(t *testing.T) {
	t.Parallel()

	server, _ := embedServer(t, func(map[string]any) any {
		return map[string]any{"embeddings": [][]float32{vectorOf(4, 1)}}
	})

	provider := embeddingProvider(
		aiprovider.KindOllama, server.URL, 4, aiprovider.EmbeddingInputStyleNone,
	)

	_, err := NewOllamaAdapter().(Embedder).Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeDocument, "one", "two"),
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, serviceports.ErrEmbeddingResponseInvalid)
	assert.False(t, IsRetryable(err))
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

	resp, err := NewOpenAIChatAdapter().(Embedder).Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeDocument, "one", "two"),
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

	_, err := NewOpenAIChatAdapter().(Embedder).Embed(
		t.Context(),
		embedCallFor(provider, serviceports.EmbeddingPurposeDocument, "text"),
	)

	var transport *TransportError
	require.ErrorAs(t, err, &transport)
	assert.Equal(t, http.StatusTooManyRequests, transport.StatusCode)
	assert.True(t, IsRetryable(err))
}

func TestRegistry_HasNoEmbedderForAnthropic(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	_, err := registry.Embedder(aiprovider.KindAnthropicMessages)
	require.Error(t, err)
	assert.ErrorIs(t, err, serviceports.ErrNoProviderConfigured)

	for _, kind := range []aiprovider.Kind{
		aiprovider.KindOpenAIResponses,
		aiprovider.KindOpenAIChat,
		aiprovider.KindOllama,
	} {
		embedder, embedErr := registry.Embedder(kind)
		require.NoError(t, embedErr, kind)
		assert.NotNil(t, embedder, kind)
	}
}
