package completionrouter

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionembed"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ollamaEndpoint struct {
	server *httptest.Server
	mu     sync.Mutex
	paths  []string
	inputs [][]string
}

func newOllamaEndpoint(t *testing.T) *ollamaEndpoint {
	t.Helper()

	endpoint := &ollamaEndpoint{}
	endpoint.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var body struct {
			Input []string `json:"input"`
		}
		if err = sonic.Unmarshal(raw, &body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		endpoint.mu.Lock()
		endpoint.paths = append(endpoint.paths, r.URL.Path)
		endpoint.inputs = append(endpoint.inputs, body.Input)
		endpoint.mu.Unlock()

		data := make([]map[string]any, 0, len(body.Input))
		for idx := range body.Input {
			vector := make([]float32, aiprovider.BuiltInEmbeddingDimensions)
			vector[0] = 1
			data = append(data, map[string]any{"index": idx, "embedding": vector})
		}
		encoded, _ := sonic.Marshal(map[string]any{
			"model": "nomic-embed-text:v1.5",
			"data":  data,
			"usage": map[string]any{"prompt_tokens": 7 * len(body.Input)},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded)
	}))
	t.Cleanup(endpoint.server.Close)

	return endpoint
}

func (e *ollamaEndpoint) calls() (paths []string, inputs [][]string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]string(nil), e.paths...), append([][]string(nil), e.inputs...)
}

func withBuiltIn(svc *Service, endpoint *ollamaEndpoint) {
	svc.BuiltIn = &fakeBuiltIn{service: &aiprovider.BuiltInEmbeddingService{
		URL:   endpoint.server.URL,
		Model: "nomic-embed-text:v1.5",
	}}
}

func TestBuiltIn_ServesAnOrganizationWithNoEmbeddingRoute(t *testing.T) {
	t.Parallel()

	endpoint := newOllamaEndpoint(t)
	usage := &fakeUsage{}
	svc := newEmbeddingService(t)
	svc.Usage = usage
	withProviders(svc)
	withBuiltIn(svc, endpoint)
	tenant := embedTenant()

	key, err := svc.ConfiguredModelKey(t.Context(), tenant)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(key, "/nomic-embed-text:v1.5@768"), key)

	result, err := svc.Embed(t.Context(), &aiusageports.EmbedRequest{
		TenantInfo: tenant,
		Purpose:    aiusageports.EmbeddingPurposeQuery,
		Inputs:     []string{"late loads"},
	})
	require.NoError(t, err)
	assert.Equal(t, key, result.ModelKey)
	assert.Equal(t, aiprovider.BuiltInEmbeddingID, result.ProviderID)

	paths, inputs := endpoint.calls()
	require.Equal(t, []string{"/v1/embeddings"}, paths, "Ollama's OpenAI-compatible endpoint")
	require.Equal(t, [][]string{{"search_query: late loads"}}, inputs,
		"nomic-embed-text is asked with its task prefix")

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, aiprovider.BuiltInEmbeddingID, rows[0].ProviderID)
	require.NotNil(t, rows[0].CostUSD, "a call to the service costs nothing; it is not unpriced")
	assert.True(t, rows[0].CostUSD.IsZero())
}

func TestBuiltIn_IsReachedOnTheInstanceWhenOrganizationProvidersMayNotUsePrivateAddresses(
	t *testing.T,
) {
	t.Parallel()

	endpoint := newOllamaEndpoint(t)
	svc := newEmbeddingService(t)
	svc.AI = &aiconfig.Config{MaxRetries: 1, PrivateNetworkProviders: false}
	withProviders(svc)
	withBuiltIn(svc, endpoint)

	_, err := svc.Embed(t.Context(), &aiusageports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    aiusageports.EmbeddingPurposeDocument,
		Inputs:     []string{"chunk #0"},
	})
	require.NoError(t, err, "the service's address is instance configuration, not an organization's provider")
}

func TestBuiltIn_TheOrganizationsOwnRouteComesFirst(t *testing.T) {
	t.Parallel()

	builtIn := newOllamaEndpoint(t)
	own := newEmbeddingEndpoint(t)
	svc := newEmbeddingService(t)
	provider := embeddingRouterProvider(t, svc, "own", own.server.URL, "voyage-3.5", "", 10)
	withProviders(svc, provider)
	withBuiltIn(svc, builtIn)
	tenant := embedTenant()

	key, err := svc.ConfiguredModelKey(t.Context(), tenant)
	require.NoError(t, err)
	assert.Equal(t, provider.EmbeddingModelKey(), key)

	_, err = svc.Embed(t.Context(), &aiusageports.EmbedRequest{
		TenantInfo: tenant,
		Purpose:    aiusageports.EmbeddingPurposeDocument,
		Inputs:     []string{"chunk #0"},
	})
	require.NoError(t, err)
	paths, _ := builtIn.calls()
	assert.Empty(t, paths, "the service is never a fallback for another model")
	assert.Equal(t, int32(1), own.calls.Load())
}

func TestBuiltIn_KeepsServingSearchesPinnedToItsModelDuringAModelChange(t *testing.T) {
	t.Parallel()

	builtIn := newOllamaEndpoint(t)
	own := newEmbeddingEndpoint(t)
	svc := newEmbeddingService(t)
	withProviders(svc, embeddingRouterProvider(t, svc, "own", own.server.URL, "voyage-3.5", "", 10))
	withBuiltIn(svc, builtIn)
	tenant := embedTenant()

	builtInKey := aiprovider.BuiltInEmbedding(tenant.OrgID, tenant.BuID, aiprovider.BuiltInEmbeddingService{
		URL:   builtIn.server.URL,
		Model: "nomic-embed-text:v1.5",
	}).EmbeddingModelKey()

	result, err := svc.Embed(t.Context(), &aiusageports.EmbedRequest{
		TenantInfo: tenant,
		Purpose:    aiusageports.EmbeddingPurposeQuery,
		Inputs:     []string{"late loads"},
		ModelKey:   builtInKey,
	})
	require.NoError(t, err, "searches keep the old model until every source is indexed under the new one")
	assert.Equal(t, builtInKey, result.ModelKey)
	assert.Zero(t, own.calls.Load())
}

func TestBuiltIn_BatchesSmallForTwoCPUThreads(t *testing.T) {
	t.Parallel()

	endpoint := newOllamaEndpoint(t)
	svc := newEmbeddingService(t)
	withProviders(svc)
	withBuiltIn(svc, endpoint)

	result, err := svc.Embed(t.Context(), &aiusageports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    aiusageports.EmbeddingPurposeDocument,
		Inputs:     numberedInputs(completionembed.BuiltInEmbeddingBatchMaxInputs*2 + 1),
	})
	require.NoError(t, err)
	assert.Len(t, result.Vectors, completionembed.BuiltInEmbeddingBatchMaxInputs*2+1)

	_, inputs := endpoint.calls()
	require.Len(t, inputs, 3)
	for _, batch := range inputs {
		assert.LessOrEqual(t, len(batch), completionembed.BuiltInEmbeddingBatchMaxInputs)
	}
}

func TestBuiltIn_NoneLeavesTheTaskUnrouted(t *testing.T) {
	t.Parallel()

	svc := newEmbeddingService(t)
	withProviders(svc)

	_, err := svc.ConfiguredModelKey(t.Context(), embedTenant())
	assert.ErrorIs(t, err, llmports.ErrNoProviderConfigured)
}

func TestBuiltIn_AFailureToResolveItIsReturned(t *testing.T) {
	t.Parallel()

	svc := newEmbeddingService(t)
	withProviders(svc)
	svc.BuiltIn = &fakeBuiltIn{err: errors.New("the plan could not be read")}

	_, err := svc.ConfiguredModelKey(t.Context(), embedTenant())
	require.Error(t, err)
	assert.NotErrorIs(t, err, llmports.ErrNoProviderConfigured)
}

func TestBuiltIn_OffThePlanNothingIsEmbedded(t *testing.T) {
	t.Parallel()

	endpoint := newOllamaEndpoint(t)
	svc := newEmbeddingService(t)
	withProviders(svc)
	withBuiltIn(svc, endpoint)
	svc.Features = plantest.Without(platformcatalog.FeatureAIDesk)

	_, err := svc.ConfiguredModelKey(t.Context(), embedTenant())
	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured)
	paths, _ := endpoint.calls()
	assert.Empty(t, paths)
}
