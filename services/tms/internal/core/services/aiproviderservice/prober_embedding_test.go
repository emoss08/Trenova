package aiproviderservice

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func embeddingProbeServer(t *testing.T, returnedDims int) (*httptest.Server, *atomic.Value) {
	t.Helper()

	var path atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path.Store(r.URL.Path)
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		encoded, _ := sonic.Marshal(map[string]any{
			"model": "voyage-3.5",
			"data": []map[string]any{
				{"index": 0, "embedding": make([]float32, returnedDims)},
			},
			"usage": map[string]any{"total_tokens": 12},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded)
	}))
	t.Cleanup(server.Close)

	return server, &path
}

func newTestProber() *Prober {
	return &Prober{
		logger:   zap.NewNop(),
		cfg:      &config.AIConfig{},
		adapters: modeladapter.NewRegistry(),
		clients:  make(map[bool]*http.Client, 2),
	}
}

func embeddingOnlyProvider(baseURL string, dims int) *aiprovider.Provider {
	return &aiprovider.Provider{
		ID:                  pulid.MustNew("aiprv_"),
		Name:                "Voyage",
		Kind:                aiprovider.KindOpenAIChat,
		BaseURL:             baseURL,
		Model:               "voyage-3.5",
		AllowPrivateNetwork: true,
		Tasks:               []aiprovider.Task{aiprovider.TaskEmbedding},
		EmbeddingDimensions: &dims,
		EmbeddingInputStyle: aiprovider.EmbeddingInputStyleVoyageInputType,
		Enabled:             true,
	}
}

func TestProbe_EmbeddingOnlyProviderIsAskedForAnEmbedding(t *testing.T) {
	t.Parallel()

	server, path := embeddingProbeServer(t, 1024)

	result := newTestProber().Probe(t.Context(), embeddingOnlyProvider(server.URL, 1024), "")

	require.True(t, result.Success, result.Message)
	assert.True(t, result.SchemaHonoured, "the vector has the configured shape")
	assert.Equal(t, "/embeddings", path.Load(), "a chat probe would have hit /chat/completions")
	assert.Equal(t, "voyage-3.5", result.ModelIdentifier)
	assert.Contains(t, result.Message, "1024")
}

func TestProbe_EmbeddingOfTheWrongSizeFailsTheTest(t *testing.T) {
	t.Parallel()

	server, _ := embeddingProbeServer(t, 512)

	result := newTestProber().Probe(t.Context(), embeddingOnlyProvider(server.URL, 1024), "")

	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "different dimension")
	assert.Contains(t, result.Detail, "512")
}
