package retrievalquery

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var catalogAxes = map[string]int{
	"list workers":   0,
	"list time off":  1,
	"send reminder":  2,
	"edited_tool v2": 4,
}

func catalogItems() []serviceports.EmbeddingCatalogItem {
	return []serviceports.EmbeddingCatalogItem{
		serviceports.NewEmbeddingCatalogItem("list_workers", "list workers"),
		serviceports.NewEmbeddingCatalogItem("list_time_off", "list time off"),
		serviceports.NewEmbeddingCatalogItem("send_reminder", "send reminder"),
	}
}

func textEmbeddings() *fakeEmbeddings {
	return &fakeEmbeddings{vectorFor: func(input string) []float32 {
		if index, ok := catalogAxes[input]; ok {
			return axis(index)
		}
		return axis(100)
	}}
}

func catalogQuery() serviceports.QueryVector {
	return serviceports.QueryVector{
		Available:  true,
		Vector:     blend(map[int]float32{1: 0.8, 0: 0.6}),
		ModelKey:   testModelKey,
		Dimensions: airetrieval.Dimensions768,
	}
}

func newCatalogIndex(repo *fakeRepository, embeddings serviceports.EmbeddingService) *CatalogIndex {
	return NewCatalogIndex(CatalogParams{
		Logger:     zap.NewNop(),
		Repository: repo,
		Embeddings: embeddings,
	})
}

func similarityRequest(
	items []serviceports.EmbeddingCatalogItem,
) serviceports.CatalogSimilarityRequest {
	return serviceports.CatalogSimilarityRequest{
		TenantInfo: testTenant(),
		Corpus:     airetrieval.CatalogCorpusTools,
		Items:      items,
		Query:      catalogQuery(),
	}
}

func TestSimilarities_EmbedsWhatIsMissingOnceAndKeepsIt(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository(nil)
	embeddings := textEmbeddings()
	index := newCatalogIndex(repo, embeddings)

	result, err := index.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)
	require.True(t, result.Available)
	assert.InDelta(t, 0.8, result.ByKey["list_time_off"], 1e-6)
	assert.InDelta(t, 0.6, result.ByKey["list_workers"], 1e-6)
	assert.InDelta(t, 0, result.ByKey["send_reminder"], 1e-6)

	calls := embeddings.calls()
	require.Len(t, calls, 1)
	assert.Equal(t, serviceports.EmbeddingPurposeDocument, calls[0].Purpose)
	assert.Equal(t, aiusage.SurfaceIndexing, calls[0].ResolvedSurface())
	assert.Equal(t, testModelKey, calls[0].ModelKey)
	assert.Len(t, calls[0].Inputs, 3)

	require.Len(t, repo.putRequests, 1)
	assert.Equal(t, airetrieval.CatalogCorpusTools, repo.putRequests[0].Corpus)
	assert.Equal(t, airetrieval.Dimensions768, repo.putRequests[0].Dimensions)
	assert.Len(t, repo.putRequests[0].Items, 3)
	require.Len(t, repo.getRequests, 1)
	assert.Len(t, repo.pruneRequests, 1)

	for range 3 {
		again, againErr := index.Similarities(t.Context(), similarityRequest(catalogItems()))
		require.NoError(t, againErr)
		assert.Equal(t, result.ByKey, again.ByKey)
	}
	assert.Len(t, embeddings.calls(), 1, "a warm cache embeds nothing")
	assert.Len(t, repo.getRequests, 1, "a warm cache never reads the table again")
	assert.Len(t, repo.pruneRequests, 1, "retired rows are pruned once per process")
}

func TestSimilarities_ReadsWhatAnotherProcessStored(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository(nil)
	first := newCatalogIndex(repo, textEmbeddings())
	_, err := first.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)

	embeddings := textEmbeddings()
	second := newCatalogIndex(repo, embeddings)
	result, err := second.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)

	require.True(t, result.Available)
	assert.InDelta(t, 0.8, result.ByKey["list_time_off"], 1e-3)
	assert.Empty(t, embeddings.calls(), "stored vectors are not embedded again")
}

func TestSimilarities_EmbedsOnlyAnEditedItem(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository(nil)
	embeddings := textEmbeddings()
	index := newCatalogIndex(repo, embeddings)
	_, err := index.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)

	edited := catalogItems()
	edited[2] = serviceports.NewEmbeddingCatalogItem("send_reminder", "edited_tool v2")
	result, err := index.Similarities(t.Context(), similarityRequest(edited))
	require.NoError(t, err)
	require.True(t, result.Available)

	calls := embeddings.calls()
	require.Len(t, calls, 2)
	assert.Equal(t, []string{"edited_tool v2"}, calls[1].Inputs)
	assert.InDelta(t, 0, result.ByKey["send_reminder"], 1e-6)
}

func TestSimilarities_KeepsEachModelApart(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository(nil)
	embeddings := textEmbeddings()
	index := newCatalogIndex(repo, embeddings)
	_, err := index.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)

	other := similarityRequest(catalogItems())
	other.Query.ModelKey = "api.voyageai.com/voyage-3.5@768"
	_, err = index.Similarities(t.Context(), other)
	require.NoError(t, err)

	calls := embeddings.calls()
	require.Len(t, calls, 2)
	assert.Equal(t, "api.voyageai.com/voyage-3.5@768", calls[1].ModelKey)

	guide := similarityRequest(catalogItems())
	guide.Corpus = airetrieval.CatalogCorpusProductGuide
	_, err = index.Similarities(t.Context(), guide)
	require.NoError(t, err)
	assert.Len(t, embeddings.calls(), 3, "each corpus is its own set of rows")
}

func TestSimilarities_BacksOffAfterAFailedEmbed(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository(nil)
	embeddings := textEmbeddings()
	embeddings.err = errors.New("503 service unavailable")
	index := newCatalogIndex(repo, embeddings)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	index.now = func() time.Time { return now }

	result, err := index.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)
	assert.False(t, result.Available)
	assert.Equal(t, airetrieval.UnavailableReasonProviderFailed, result.Reason)

	result, err = index.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)
	assert.False(t, result.Available)
	assert.Len(t, embeddings.calls(), 1, "a failed catalog is not retried on every turn")

	now = now.Add(catalogRetryBackoff + time.Second)
	embeddings.err = nil
	result, err = index.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)
	assert.True(t, result.Available)
	assert.Len(t, embeddings.calls(), 2)
}

func TestSimilarities_WithoutAnEmbeddingServiceHasNoProvider(t *testing.T) {
	t.Parallel()

	index := newCatalogIndex(newFakeRepository(nil), nil)
	result, err := index.Similarities(t.Context(), similarityRequest(catalogItems()))
	require.NoError(t, err)
	assert.False(t, result.Available)
	assert.Equal(t, airetrieval.UnavailableReasonNoProvider, result.Reason)
}

func TestSimilarities_RefusesAMalformedRequest(t *testing.T) {
	t.Parallel()

	index := newCatalogIndex(newFakeRepository(nil), textEmbeddings())

	unusable := similarityRequest(catalogItems())
	unusable.Query = serviceports.UnavailableQueryVector(airetrieval.UnavailableReasonNotIndexed)
	_, err := index.Similarities(t.Context(), unusable)
	require.ErrorIs(t, err, serviceports.ErrCatalogQueryNotUsable)

	empty := similarityRequest(nil)
	_, err = index.Similarities(t.Context(), empty)
	require.ErrorIs(t, err, serviceports.ErrCatalogRequestInvalid)

	repeated := similarityRequest(append(catalogItems(), catalogItems()[0]))
	_, err = index.Similarities(t.Context(), repeated)
	require.ErrorIs(t, err, serviceports.ErrCatalogItemKeyRepeated)

	zero := similarityRequest(catalogItems())
	zero.Query.Vector = make([]float32, airetrieval.Dimensions768)
	_, err = index.Similarities(t.Context(), zero)
	require.ErrorIs(t, err, serviceports.ErrCatalogQueryNotUsable)
}
