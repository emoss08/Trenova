package completionrouter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEmbeddingDimensions = 768

type embeddingEndpoint struct {
	server   *httptest.Server
	calls    atomic.Int32
	mu       sync.Mutex
	inputs   [][]string
	byAPIKey map[string]int
	delay    time.Duration
	dims     int
}

func newEmbeddingEndpoint(t *testing.T) *embeddingEndpoint {
	t.Helper()

	endpoint := &embeddingEndpoint{
		byAPIKey: make(map[string]int),
		dims:     testEmbeddingDimensions,
	}
	endpoint.server = httptest.NewServer(http.HandlerFunc(endpoint.serve))
	t.Cleanup(endpoint.server.Close)

	return endpoint
}

func (e *embeddingEndpoint) serve(w http.ResponseWriter, r *http.Request) {
	e.calls.Add(1)

	if e.delay > 0 {
		select {
		case <-time.After(e.delay):
		case <-r.Context().Done():
			return
		}
	}

	if status, failing := e.statusFor(r.Header.Get("Authorization")); failing {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream failure"}}`))
		return
	}

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

	e.mu.Lock()
	e.inputs = append(e.inputs, body.Input)
	e.mu.Unlock()

	data := make([]map[string]any, 0, len(body.Input))
	for idx, input := range body.Input {
		vector := make([]float32, e.dims)
		vector[0] = float32(positionOf(input))
		data = append(data, map[string]any{"index": idx, "embedding": vector})
	}

	encoded, _ := sonic.Marshal(map[string]any{
		"model": "embed-model",
		"data":  data,
		"usage": map[string]any{"prompt_tokens": 10 * len(body.Input)},
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(encoded)
}

func (e *embeddingEndpoint) statusFor(auth string) (int, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	status, failing := e.byAPIKey[strings.TrimPrefix(auth, "Bearer ")]

	return status, failing
}

func (e *embeddingEndpoint) failFor(apiKey string, status int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.byAPIKey[apiKey] = status
}

func (e *embeddingEndpoint) batches() [][]string {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([][]string(nil), e.inputs...)
}

func positionOf(input string) int {
	_, digits, found := strings.Cut(input, "#")
	if !found {
		return -1
	}
	position, err := strconv.Atoi(digits)
	if err != nil {
		return -1
	}

	return position
}

func embeddingRouterProvider(
	t *testing.T,
	svc *Service,
	name, baseURL, model, apiKey string,
	priority int,
) *aiprovider.Provider {
	t.Helper()

	dims := testEmbeddingDimensions
	provider := &aiprovider.Provider{
		ID:                  pulid.MustNew("aiprv_"),
		Name:                name,
		Kind:                aiprovider.KindOpenAIChat,
		BaseURL:             baseURL,
		Model:               model,
		AllowPrivateNetwork: true,
		Priority:            priority,
		Tasks:               []aiprovider.Task{aiprovider.TaskEmbedding},
		EmbeddingDimensions: &dims,
		EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNone,
		Enabled:             true,
	}
	if apiKey != "" {
		encrypted, err := svc.encryption.EncryptString(apiKey)
		require.NoError(t, err)
		provider.APIKey = encrypted
	}

	return provider
}

func newEmbeddingService(t *testing.T) *Service {
	t.Helper()

	svc := newTestService(t)
	svc.pause = func(context.Context, time.Duration) error { return nil }

	return svc
}

func withProviders(svc *Service, providers ...*aiprovider.Provider) {
	svc.repo = &fakeRepo{providers: providers}
}

func embedTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func numberedInputs(n int) []string {
	inputs := make([]string, 0, n)
	for idx := range n {
		inputs = append(inputs, "chunk #"+strconv.Itoa(idx))
	}

	return inputs
}

func TestBatchEmbeddingInputs(t *testing.T) {
	t.Parallel()

	limits := embeddingBatchLimits{inputs: 3, tokens: 10}

	tests := []struct {
		name   string
		inputs []string
		want   []int
	}{
		{name: "one input", inputs: []string{"abcd"}, want: []int{1}},
		{
			name:   "splits on the input count",
			inputs: []string{"a", "b", "c", "d", "e", "f", "g"},
			want:   []int{3, 3, 1},
		},
		{
			name:   "splits on the token budget",
			inputs: []string{strings.Repeat("x", 24), strings.Repeat("x", 24), "y"},
			want:   []int{1, 2},
		},
		{
			name:   "an input over the budget travels alone",
			inputs: []string{"a", strings.Repeat("x", 400), "b"},
			want:   []int{1, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			batches := batchEmbeddingInputs(tt.inputs, limits)
			sizes := make([]int, 0, len(batches))
			total := 0
			for _, batch := range batches {
				sizes = append(sizes, len(batch.inputs))
				total += len(batch.inputs)
			}
			assert.Equal(t, tt.want, sizes)
			assert.Equal(t, len(tt.inputs), total, "every input is sent exactly once")
		})
	}
}

func TestEmbeddingLimitsFor(t *testing.T) {
	t.Parallel()

	openAI := &aiprovider.Provider{Kind: aiprovider.KindOpenAIResponses}
	gemini := &aiprovider.Provider{
		Kind:    aiprovider.KindOpenAIChat,
		BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
	}

	assert.Equal(t, EmbeddingBatchMaxInputs, embeddingLimitsFor(openAI).inputs)
	assert.Equal(t, EmbeddingBatchMaxTokens, embeddingLimitsFor(openAI).tokens)
	assert.Equal(t, GeminiEmbeddingBatchMaxInputs, embeddingLimitsFor(gemini).inputs)
}

func TestEmbed_BatchesAndKeepsInputOrder(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	svc := newEmbeddingService(t)
	withProviders(svc, embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10))

	inputs := numberedInputs(200)
	result, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     inputs,
	})
	require.NoError(t, err)

	batches := endpoint.batches()
	require.Len(t, batches, 3)
	assert.Len(t, batches[0], EmbeddingBatchMaxInputs)
	assert.Len(t, batches[1], EmbeddingBatchMaxInputs)
	assert.Len(t, batches[2], 200-2*EmbeddingBatchMaxInputs)

	require.Len(t, result.Vectors, len(inputs))
	for idx, vector := range result.Vectors {
		require.Len(t, vector, testEmbeddingDimensions)
		assert.InDelta(t, float64(idx), float64(vector[0]), 0.0001, "vector %d is out of order", idx)
	}
	assert.Equal(t, testEmbeddingDimensions, result.Dimensions)
	assert.Equal(t, 2000, result.InputTokens)
	assert.NotEmpty(t, result.ModelKey)
	assert.True(t, strings.HasSuffix(result.ModelKey, "/m@768"), result.ModelKey)
}

func TestEmbed_FailsOverOnlyToProvidersWithTheSameModelKey(t *testing.T) {
	t.Parallel()

	shared := newEmbeddingEndpoint(t)
	shared.failFor("primary-key", http.StatusServiceUnavailable)
	shared.failFor("secondary-key", http.StatusServiceUnavailable)
	other := newEmbeddingEndpoint(t)

	svc := newEmbeddingService(t)
	primary := embeddingRouterProvider(t, svc, "primary", shared.server.URL, "m", "primary-key", 10)
	secondary := embeddingRouterProvider(
		t, svc, "secondary", shared.server.URL, "m", "secondary-key", 20,
	)
	differentModel := embeddingRouterProvider(t, svc, "other", other.server.URL, "n", "", 30)
	withProviders(svc, primary, secondary, differentModel)

	require.Equal(t, primary.EmbeddingModelKey(), secondary.EmbeddingModelKey())
	require.NotEqual(t, primary.EmbeddingModelKey(), differentModel.EmbeddingModelKey())

	_, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     []string{"chunk #0"},
	})
	require.Error(t, err)
	assert.Zero(t, other.calls.Load(),
		"a provider with another model writes vectors the index cannot compare")
	assert.Positive(t, shared.calls.Load())
}

func TestEmbed_FallsThroughToTheNextProviderWithTheSameModelKey(t *testing.T) {
	t.Parallel()

	shared := newEmbeddingEndpoint(t)
	shared.failFor("primary-key", http.StatusInternalServerError)

	svc := newEmbeddingService(t)
	primary := embeddingRouterProvider(t, svc, "primary", shared.server.URL, "m", "primary-key", 10)
	secondary := embeddingRouterProvider(
		t, svc, "secondary", shared.server.URL, "m", "secondary-key", 20,
	)
	withProviders(svc, primary, secondary)

	result, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     []string{"chunk #0"},
	})
	require.NoError(t, err)
	assert.Equal(t, secondary.ID, result.ProviderID)
	assert.Len(t, result.Vectors, 1)
}

func TestEmbed_HonoursAPinnedModelKey(t *testing.T) {
	t.Parallel()

	current := newEmbeddingEndpoint(t)
	pending := newEmbeddingEndpoint(t)

	svc := newEmbeddingService(t)
	first := embeddingRouterProvider(t, svc, "current", current.server.URL, "m", "", 10)
	second := embeddingRouterProvider(t, svc, "pending", pending.server.URL, "n", "", 20)
	withProviders(svc, first, second)

	result, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeQuery,
		Inputs:     []string{"chunk #0"},
		ModelKey:   second.EmbeddingModelKey(),
	})
	require.NoError(t, err)
	assert.Equal(t, second.EmbeddingModelKey(), result.ModelKey)
	assert.Zero(t, current.calls.Load())
	assert.Equal(t, int32(1), pending.calls.Load())
}

func TestEmbed_PinnedModelKeyWithNoProviderIsNotConfigured(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	svc := newEmbeddingService(t)
	withProviders(svc, embeddingRouterProvider(t, svc, "current", endpoint.server.URL, "m", "", 10))

	_, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeQuery,
		Inputs:     []string{"chunk #0"},
		ModelKey:   "retired.example/old@1536",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, serviceports.ErrNoProviderConfigured)
	assert.Zero(t, endpoint.calls.Load())
}

func TestEmbed_NeverRoutesToAProtocolWithoutEmbeddings(t *testing.T) {
	t.Parallel()

	svc := newEmbeddingService(t)
	dims := testEmbeddingDimensions
	withProviders(svc, &aiprovider.Provider{
		ID:                  pulid.MustNew("aiprv_"),
		Name:                "anthropic",
		Kind:                aiprovider.KindAnthropicMessages,
		Model:               "claude-opus-5",
		Tasks:               []aiprovider.Task{aiprovider.TaskEmbedding},
		EmbeddingDimensions: &dims,
		Enabled:             true,
	})

	_, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     []string{"text"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, serviceports.ErrNoProviderConfigured)
}

func TestEmbed_WrongDimensionIsNotRetried(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	endpoint.dims = 1024
	svc := newEmbeddingService(t)
	withProviders(svc, embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10))

	_, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     []string{"chunk #0"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, serviceports.ErrEmbeddingDimensionMismatch)
	var business *errortypes.BusinessError
	assert.True(t, errors.As(err, &business), "modelcall classifies this as a rejection")
	assert.Equal(t, int32(1), endpoint.calls.Load())
}

func TestEmbed_ScrubsDocumentsButNotQueries(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	svc := newEmbeddingService(t)
	withProviders(svc, embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10))

	_, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     []string{"Driver SSN 123-45-6789 on file"},
	})
	require.NoError(t, err)

	_, err = svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeQuery,
		Inputs:     []string{"account 12345678"},
	})
	require.NoError(t, err)

	batches := endpoint.batches()
	require.Len(t, batches, 2)
	assert.Equal(t, []string{"Driver SSN [SSN] on file"}, batches[0])
	assert.Equal(t, []string{"account 12345678"}, batches[1],
		"a query is the person's own words and is compared, not stored")
}

func TestEmbed_QueryGivesUpAtItsDeadline(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	endpoint.delay = 2 * time.Second
	svc := newEmbeddingService(t)
	withProviders(svc, embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10))

	started := time.Now()
	_, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo:   embedTenant(),
		Purpose:      serviceports.EmbeddingPurposeQuery,
		Inputs:       []string{"late reefer loads"},
		QueryTimeout: 50 * time.Millisecond,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), time.Second, "keyword search takes over instead of waiting")
}

func TestEmbed_DefaultQueryDeadline(t *testing.T) {
	t.Parallel()

	req := serviceports.EmbedRequest{Purpose: serviceports.EmbeddingPurposeQuery}
	assert.Equal(t, serviceports.DefaultQueryEmbeddingTimeout, req.ResolvedQueryTimeout())
	assert.Equal(t, 1500*time.Millisecond, serviceports.DefaultQueryEmbeddingTimeout)
}

func TestEmbed_RefusesAnEmptyOrBlankRequest(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	svc := newEmbeddingService(t)
	withProviders(svc, embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10))

	for name, req := range map[string]*serviceports.EmbedRequest{
		"no inputs": {Purpose: serviceports.EmbeddingPurposeDocument},
		"blank input": {
			Purpose: serviceports.EmbeddingPurposeDocument,
			Inputs:  []string{"text", "   "},
		},
		"unknown purpose": {Purpose: "Summary", Inputs: []string{"text"}},
		"wrong surface": {
			Purpose: serviceports.EmbeddingPurposeDocument,
			Inputs:  []string{"text"},
			Surface: aiusage.SurfaceChat,
		},
	} {
		req.TenantInfo = embedTenant()
		_, err := svc.Embed(t.Context(), req)
		require.Error(t, err, name)
		assert.True(t, errortypes.IsMultiError(err), name)
	}
	_, err := svc.Embed(t.Context(), nil)
	require.Error(t, err, "a missing request")
	assert.Zero(t, endpoint.calls.Load())
}

func TestEmbed_RecordsOneUsageRowPerBatchPricedOnInput(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	usage := &fakeUsage{}
	svc := newEmbeddingService(t)
	svc.usage = usage

	provider := embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10)
	inputCost := decimal.RequireFromString("0.02")
	provider.InputCostPerMillion = &inputCost
	withProviders(svc, provider)

	attribution := serviceports.AIUsageAttribution{
		UserID:            pulid.MustNew("usr_"),
		AgentDefinitionID: pulid.MustNew("agd_"),
		ThreadID:          pulid.MustNew("ath_"),
		RunID:             pulid.MustNew("ar_"),
	}
	result, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo:  embedTenant(),
		Purpose:     serviceports.EmbeddingPurposeQuery,
		Inputs:      numberedInputs(EmbeddingBatchMaxInputs + 4),
		Attribution: attribution,
	})
	require.NoError(t, err)

	rows := usage.recorded(t, 2)
	require.Len(t, rows, 2)
	for _, row := range rows {
		assert.Equal(t, aiprovider.TaskEmbedding, row.Task)
		assert.Equal(t, aiusage.SurfaceRetrieval, row.Surface)
		assert.True(t, row.Succeeded)
		assert.Zero(t, row.OutputTokens)
		assert.Equal(t, attribution.UserID, row.UserID)
		assert.Equal(t, attribution.AgentDefinitionID, row.AgentDefinitionID)
		assert.Equal(t, attribution.ThreadID, row.ThreadID)
		assert.Equal(t, attribution.RunID, row.RunID)
		require.NotNil(t, row.CostUSD, "an embedding is priced on its input alone")
	}

	total := decimal.Zero
	tokens := 0
	for _, row := range rows {
		total = total.Add(*row.CostUSD)
		tokens += row.InputTokens
	}
	assert.Equal(t, 10*(EmbeddingBatchMaxInputs+4), tokens)
	require.NotNil(t, result.CostUSD)
	assert.True(t, result.CostUSD.Equal(total), result.CostUSD.String())
	assert.True(
		t,
		result.CostUSD.Equal(decimal.RequireFromString("0.00002")),
		result.CostUSD.String(),
	)
}

func TestEmbed_DocumentUsageIsIndexingAndUnpricedStaysUnknown(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	usage := &fakeUsage{}
	svc := newEmbeddingService(t)
	svc.usage = usage
	withProviders(svc, embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10))

	result, err := svc.Embed(t.Context(), &serviceports.EmbedRequest{
		TenantInfo: embedTenant(),
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     []string{"chunk #0"},
	})
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, aiusage.SurfaceIndexing, rows[0].Surface)
	assert.Nil(t, rows[0].CostUSD, "an unknown price is not a free one")
	assert.Nil(t, result.CostUSD)
}

func TestConfiguredModelKey(t *testing.T) {
	t.Parallel()

	endpoint := newEmbeddingEndpoint(t)
	svc := newEmbeddingService(t)
	provider := embeddingRouterProvider(t, svc, "embedder", endpoint.server.URL, "m", "", 10)
	withProviders(svc, provider)

	key, err := svc.ConfiguredModelKey(t.Context(), embedTenant())
	require.NoError(t, err)
	assert.Equal(t, provider.EmbeddingModelKey(), key)

	withProviders(svc)
	_, err = svc.ConfiguredModelKey(t.Context(), embedTenant())
	assert.ErrorIs(t, err, serviceports.ErrNoProviderConfigured)
}
