package retrievalquery

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memorySharedVectors struct {
	mu      sync.Mutex
	vectors map[agentports.QueryVectorCacheKey]agentports.QueryVector
	ttls    []time.Duration
	failGet bool
}

func newMemorySharedVectors() *memorySharedVectors {
	return &memorySharedVectors{vectors: map[agentports.QueryVectorCacheKey]agentports.QueryVector{}}
}

func (m *memorySharedVectors) Get(
	_ context.Context,
	key agentports.QueryVectorCacheKey,
) (agentports.QueryVector, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failGet {
		return agentports.QueryVector{}, false, errors.New("redis is down")
	}
	vector, ok := m.vectors[key]

	return vector, ok, nil
}

func (m *memorySharedVectors) Set(
	_ context.Context,
	key agentports.QueryVectorCacheKey,
	vector agentports.QueryVector,
	ttl time.Duration,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vectors[key] = vector
	m.ttls = append(m.ttls, ttl)

	return nil
}

func sharedVectorizer(
	repo *fakeRepository,
	embeddings *fakeEmbeddings,
	shared agentports.QueryVectorCache,
) *Service {
	return New(Params{Logger: zap.NewNop(), Repository: repo, Embeddings: embeddings, Shared: shared})
}

func TestVectorize_AnotherProcessReusesTheSharedVector(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	shared := newMemorySharedVectors()
	apiEmbeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(2) }}
	workerEmbeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(5) }}
	api := sharedVectorizer(newFakeRepository(activeSettings(tenant)), apiEmbeddings, shared)
	worker := sharedVectorizer(newFakeRepository(activeSettings(tenant)), workerEmbeddings, shared)

	first, err := api.Vectorize(t.Context(), queryRequest(tenant, "which invoices are past due?"))
	require.NoError(t, err)
	second, err := worker.Vectorize(t.Context(), queryRequest(tenant, "which invoices are past due?"))
	require.NoError(t, err)

	assert.Len(t, apiEmbeddings.calls(), 1)
	assert.Empty(t, workerEmbeddings.calls(), "the worker reads the vector the API already paid for")
	assert.Equal(t, first.Vector, second.Vector)
	assert.Equal(t, []time.Duration{queryCacheTTL}, shared.ttls)
}

func TestVectorize_KeepsAnotherTenantsSharedVectorApart(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	other := testTenant()
	shared := newMemorySharedVectors()
	embeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(2) }}
	repo := newFakeRepository(activeSettings(tenant))
	service := sharedVectorizer(repo, embeddings, shared)

	_, err := service.Vectorize(t.Context(), queryRequest(tenant, "chase the shipper"))
	require.NoError(t, err)
	repo.settings = activeSettings(other)
	fresh := sharedVectorizer(repo, embeddings, shared)
	_, err = fresh.Vectorize(t.Context(), queryRequest(other, "chase the shipper"))
	require.NoError(t, err)

	assert.Len(t, embeddings.calls(), 2, "one organization never reads another's cached vector")
}

func TestVectorize_IgnoresASharedVectorOfTheWrongSize(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	shared := newMemorySharedVectors()
	embeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(2) }}
	repo := newFakeRepository(activeSettings(tenant))
	service := sharedVectorizer(repo, embeddings, shared)
	text := QueryText("chase the shipper")
	shared.vectors[sharedKey(tenant, queryKey{
		organizationID: tenant.OrgID,
		businessUnitID: tenant.BuID,
		modelKey:       testModelKey,
		textHash:       hashutils.SHA256Hex(text),
	})] = agentports.QueryVector{
		Available:  true,
		Vector:     make([]float32, airetrieval.Dimensions1536),
		ModelKey:   testModelKey,
		Dimensions: airetrieval.Dimensions1536,
	}

	vector, err := service.Vectorize(t.Context(), queryRequest(tenant, "chase the shipper"))
	require.NoError(t, err)

	assert.Len(t, embeddings.calls(), 1, "a vector the index cannot use is embedded again")
	assert.Equal(t, airetrieval.Dimensions768, vector.Dimensions)
}

func TestVectorize_EmbedsWhenTheSharedCacheFails(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	shared := newMemorySharedVectors()
	shared.failGet = true
	embeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(2) }}
	service := sharedVectorizer(newFakeRepository(activeSettings(tenant)), embeddings, shared)

	vector, err := service.Vectorize(t.Context(), queryRequest(tenant, "chase the shipper"))

	require.NoError(t, err)
	assert.True(t, vector.Usable())
	assert.Len(t, embeddings.calls(), 1)
}
