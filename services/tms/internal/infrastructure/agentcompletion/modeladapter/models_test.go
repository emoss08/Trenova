package modeladapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serveModels(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		body, ok := routes[key]
		if !ok {
			body, ok = routes[r.URL.Path]
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"no such route"}}`))
			return
		}
		if r.Header.Get("Authorization") == "Bearer refused" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

func listFrom(t *testing.T, server *httptest.Server, kind aiprovider.Kind, key string) ([]ModelInfo, error) {
	t.Helper()

	return ListModels(t.Context(), &ModelsCall{
		Provider: &aiprovider.Provider{Kind: kind, BaseURL: server.URL},
		APIKey:   key,
		Client:   server.Client(),
	})
}

func TestListModels_ReadsOpenRouterPricesPerMillion(t *testing.T) {
	t.Parallel()

	server := serveModels(t, map[string]string{"/models": `{"data":[
		{"id":"anthropic/claude-sonnet-4.5","name":"Claude Sonnet 4.5","context_length":1000000,
		 "pricing":{"prompt":"0.000003","completion":"0.000015"}},
		{"id":"openai/text-embedding-3-small","name":"Embedding",
		 "architecture":{"output_modalities":["embeddings"]}},
		{"id":"anthropic/claude-sonnet-4.5"}
	]}`})

	models, err := listFrom(t, server, aiprovider.KindOpenAIChat, "")
	require.NoError(t, err)
	require.Len(t, models, 2, "a model listed twice is offered once")

	chat := models[0]
	assert.Equal(t, "Claude Sonnet 4.5", chat.DisplayName)
	assert.Equal(t, 1_000_000, chat.ContextWindow)
	require.NotNil(t, chat.InputCostPerMillion)
	assert.Equal(t, "3", chat.InputCostPerMillion.String())
	assert.Equal(t, "15", chat.OutputCostPerMillion.String())
	assert.False(t, chat.Embedding)
	assert.True(t, models[1].Embedding)
}

func TestListModels_ReadsABareArrayWithPricesAlreadyPerMillion(t *testing.T) {
	t.Parallel()

	server := serveModels(t, map[string]string{"/models": `[
		{"id":"meta-llama/Llama-3.3-70B","display_name":"Llama 3.3 70B","context_length":131072,
		 "pricing":{"input":0.88,"output":0.88}},
		{"id":"BAAI/bge-large-en-v1.5","type":"embedding"}
	]`})

	models, err := listFrom(t, server, aiprovider.KindOpenAIChat, "")
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "0.88", models[0].InputCostPerMillion.String())
	assert.True(t, models[1].Embedding)
}

func TestListModels_MarksOllamaModelsInMemory(t *testing.T) {
	t.Parallel()

	server := serveModels(t, map[string]string{
		"/api/tags": `{"models":[
			{"name":"qwen3:32b","model":"qwen3:32b","size":20000000000,"details":{"family":"qwen3"}},
			{"name":"nomic-embed-text:latest","model":"nomic-embed-text:latest","size":274000000,
			 "details":{"family":"nomic-bert"}}
		]}`,
		"/api/ps": `{"models":[{"name":"qwen3:32b","model":"qwen3:32b","context_length":40960}]}`,
	})

	models, err := listFrom(t, server, aiprovider.KindOllama, "")
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.True(t, models[0].Loaded)
	assert.Equal(t, 40960, models[0].ContextWindow)
	assert.Equal(t, int64(20000000000), models[0].SizeBytes)
	assert.False(t, models[1].Loaded)
	assert.True(t, models[1].Embedding)
}

func TestListModels_FollowsAnthropicPages(t *testing.T) {
	t.Parallel()

	server := serveModels(t, map[string]string{
		"/v1/models?limit=1000": `{"data":[{"id":"claude-opus-4-1","display_name":"Claude Opus 4.1"}],
			"has_more":true,"last_id":"claude-opus-4-1"}`,
		"/v1/models?after_id=claude-opus-4-1&limit=1000": `{"data":[{"id":"claude-haiku-4-5",
			"display_name":"Claude Haiku 4.5"}],"has_more":false,"last_id":"claude-haiku-4-5"}`,
	})

	models, err := listFrom(t, server, aiprovider.KindAnthropicMessages, "sk-ant-x")
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "Claude Haiku 4.5", models[1].DisplayName)
	assert.Positive(t, models[0].ContextWindow, "a known family's window is filled in")
}

func TestListModels_ReportsARefusedKeyAsTheProvidersStatus(t *testing.T) {
	t.Parallel()

	server := serveModels(t, map[string]string{"/models": `{"data":[]}`})

	_, err := listFrom(t, server, aiprovider.KindOpenAIChat, "refused")
	var transport *TransportError
	require.ErrorAs(t, err, &transport)
	assert.Equal(t, http.StatusUnauthorized, transport.StatusCode)
}

func TestIsEmbeddingModelID(t *testing.T) {
	t.Parallel()

	for id, want := range map[string]bool{
		"text-embedding-3-small": true,
		"nomic-embed-text":       true,
		"voyage-3-large":         true,
		"BAAI/bge-m3":            true,
		"intfloat/e5-mistral":    true,
		"voyage-rerank-2":        false,
		"gpt-4.1":                false,
		"qwen3:32b":              false,
	} {
		assert.Equal(t, want, IsEmbeddingModelID(id), id)
	}
}

func TestListModels_ReadsWhenEachModelWasCreated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		kind   aiprovider.Kind
		routes map[string]string
		want   map[string]int64
	}{
		{
			name: "anthropic created_at",
			kind: aiprovider.KindAnthropicMessages,
			routes: map[string]string{"/v1/models": `{"data":[
				{"id":"claude-opus-4-1","display_name":"Opus","created_at":"2025-08-05T00:00:00Z"},
				{"id":"claude-old","display_name":"Old"},
				{"id":"claude-bad","display_name":"Bad","created_at":"yesterday"}
			],"has_more":false}`},
			want: map[string]int64{"claude-opus-4-1": 1754352000, "claude-old": 0, "claude-bad": 0},
		},
		{
			name: "openai created seconds",
			kind: aiprovider.KindOpenAIResponses,
			routes: map[string]string{"/v1/models": `{"data":[
				{"id":"gpt-5","created":1754352000},
				{"id":"gpt-float","created":1754352000.0},
				{"id":"gpt-text","created":"1754352000"},
				{"id":"gpt-none"},
				{"id":"gpt-neg","created":-5},
				{"id":"gpt-junk","created":"soon"},
				{"id":"gpt-obj","created":{"at":1}}
			]}`},
			want: map[string]int64{
				"gpt-5": 1754352000, "gpt-float": 1754352000, "gpt-text": 1754352000,
				"gpt-none": 0, "gpt-neg": 0, "gpt-junk": 0, "gpt-obj": 0,
			},
		},
		{
			name: "ollama modified_at",
			kind: aiprovider.KindOllama,
			routes: map[string]string{
				"/api/tags": `{"models":[
					{"name":"qwen3:32b","model":"qwen3:32b","modified_at":"2025-08-05T02:30:00.123456789+02:30"},
					{"name":"llama3:8b","model":"llama3:8b"},
					{"name":"bad:1b","model":"bad:1b","modified_at":"not a time"}
				]}`,
				"/api/ps": `{"models":[]}`,
			},
			want: map[string]int64{"qwen3:32b": 1754352000, "llama3:8b": 0, "bad:1b": 0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := serveModels(t, tc.routes)
			models, err := listFrom(t, server, tc.kind, "key")
			require.NoError(t, err)
			require.Len(t, models, len(tc.want))
			for _, model := range models {
				assert.Equal(t, tc.want[model.ID], model.CreatedAt, model.ID)
			}
		})
	}
}
