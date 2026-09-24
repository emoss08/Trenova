//go:build integration

package airetrievalrepository

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

const (
	modelA = "api.example.test/embed-small/768"
	modelB = "api.example.test/embed-large/1024"
)

func TestSearch_FiltersByTenantAndModelAndKeepsTheBestChunk(t *testing.T) {
	h := newHarness(t)
	h.requireVector(t)

	other := seedtest.SeedAdditionalTenant(t, h.ctx, h.db, "RB")
	otherTenant := pagination.TenantInfo{OrgID: other.Organization.ID, BuID: other.BusinessUnit.ID}

	exact := pulid.MustNew("amem_")
	near := pulid.MustNew("amem_")
	far := pulid.MustNew("amem_")
	otherTenants := pulid.MustNew("amem_")
	otherModel := pulid.MustNew("amem_")

	h.replace(t, h.tenant, airetrieval.SourceTypeMemory, exact, modelA, 768,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "h-exact-0",
			Vector: axisVector(768, map[int]float32{1: 1})},
		repositories.EmbeddingChunk{ChunkIndex: 1, ContentHash: "h-exact-1",
			Vector: axisVector(768, map[int]float32{0: 1})},
	)
	h.replace(t, h.tenant, airetrieval.SourceTypeMemory, near, modelA, 768,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "h-near-0",
			Vector: axisVector(768, map[int]float32{0: 0.8, 1: 0.6})},
	)
	h.replace(t, h.tenant, airetrieval.SourceTypeMemory, far, modelA, 768,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "h-far-0",
			Vector: axisVector(768, map[int]float32{2: 1})},
	)
	h.replace(t, otherTenant, airetrieval.SourceTypeMemory, otherTenants, modelA, 768,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "h-other-0",
			Vector: axisVector(768, map[int]float32{0: 1})},
	)
	h.replace(t, h.tenant, airetrieval.SourceTypeMemory, otherModel, modelB, 1024,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "h-model-0",
			Vector: axisVector(1024, map[int]float32{0: 1})},
	)

	hits, err := h.repo.Search(h.ctx, repositories.VectorSearchRequest{
		TenantInfo:  h.tenant,
		SourceTypes: []airetrieval.SourceType{airetrieval.SourceTypeMemory},
		ModelKey:    modelA,
		Dimensions:  768,
		Query:       axisVector(768, map[int]float32{0: 1}),
		Limit:       10,
	})
	require.NoError(t, err)
	require.Len(t, hits, 3, "only this tenant's rows for this model, one per source")

	assert.Equal(t, exact, hits[0].SourceID)
	assert.Equal(t, 1, hits[0].ChunkIndex, "a source is represented by its best chunk")
	assert.InDelta(t, 1.0, hits[0].Similarity, 0.01)
	assert.Equal(t, near, hits[1].SourceID)
	assert.InDelta(t, 0.8, hits[1].Similarity, 0.01)
	assert.Equal(t, far, hits[2].SourceID)
	assert.InDelta(t, 0.0, hits[2].Similarity, 0.01)
	for _, hit := range hits {
		assert.NotEqual(t, otherTenants, hit.SourceID, "another tenant's embedding never matches")
		assert.NotEqual(t, otherModel, hit.SourceID, "another model's embedding never matches")
		assert.Equal(t, airetrieval.SourceTypeMemory, hit.SourceType)
	}

	floored, err := h.repo.Search(h.ctx, repositories.VectorSearchRequest{
		TenantInfo:    h.tenant,
		ModelKey:      modelA,
		Dimensions:    768,
		Query:         axisVector(768, map[int]float32{0: 1}),
		MinSimilarity: 0.5,
	})
	require.NoError(t, err)
	assert.Len(t, floored, 2, "the similarity floor drops the unrelated source")

	documents, err := h.repo.Search(h.ctx, repositories.VectorSearchRequest{
		TenantInfo:  h.tenant,
		SourceTypes: []airetrieval.SourceType{airetrieval.SourceTypeDocument},
		ModelKey:    modelA,
		Dimensions:  768,
		Query:       axisVector(768, map[int]float32{0: 1}),
	})
	require.NoError(t, err)
	assert.Empty(t, documents, "the source type filter is applied")

	_, err = h.repo.Search(h.ctx, repositories.VectorSearchRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
		Dimensions: 768,
		Query:      axisVector(1024, nil),
	})
	require.ErrorIs(t, err, airetrieval.ErrEmbeddingLength)
}

func TestSearch_UsesThePartialHNSWIndexForItsDimensions(t *testing.T) {
	h := newHarness(t)
	h.requireVector(t)

	for i := range 20 {
		h.replace(
			t,
			h.tenant,
			airetrieval.SourceTypeMemory,
			pulid.MustNew("amem_"),
			modelA,
			768,
			repositories.EmbeddingChunk{
				ChunkIndex:  0,
				ContentHash: "h-" + pulid.MustNew("x_").String(),
				Vector:      axisVector(768, map[int]float32{i % 768: 1}),
			},
		)
	}

	req := repositories.VectorSearchRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
		Dimensions: 768,
		Query:      axisVector(768, map[int]float32{0: 1}),
		Limit:      5,
	}
	plan, err := planSearch(req)
	require.NoError(t, err)

	var explained []string
	err = h.db.RunInTx(h.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, setting := range []string{
			"SET LOCAL enable_seqscan = off",
			"SET LOCAL enable_sort = off",
			setIterativeScan,
		} {
			if _, execErr := tx.ExecContext(ctx, setting); execErr != nil {
				return execErr
			}
		}

		query := searchQuery(tx, req, plan).String()
		return tx.NewRaw("EXPLAIN "+query).Scan(ctx, &explained)
	})
	require.NoError(t, err)

	plainText := strings.Join(explained, "\n")
	assert.Contains(t, plainText, "idx_ai_embeddings_hnsw_768",
		"the cast and the dimensions predicate must match the partial index:\n%s", plainText)
}

func TestReplaceChunks_KeepsUnchangedHashesAndDropsRemovedChunks(t *testing.T) {
	h := newHarness(t)
	h.requireVector(t)

	sourceID := pulid.MustNew("doc_")
	first := h.replace(t, h.tenant, airetrieval.SourceTypeDocument, sourceID, modelA, 768,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "page-1",
			Vector: axisVector(768, map[int]float32{0: 1})},
		repositories.EmbeddingChunk{ChunkIndex: 1, ContentHash: "page-2",
			Vector: axisVector(768, map[int]float32{1: 1})},
		repositories.EmbeddingChunk{ChunkIndex: 2, ContentHash: "page-3",
			Vector: axisVector(768, map[int]float32{2: 1})},
	)
	assert.Equal(t, repositories.ReplaceEmbeddingChunksResult{Written: 3}, first)

	before := h.rowVersion(t, sourceID, 0)

	second := h.replace(t, h.tenant, airetrieval.SourceTypeDocument, sourceID, modelA, 768,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "page-1",
			Vector: axisVector(768, map[int]float32{5: 1})},
		repositories.EmbeddingChunk{ChunkIndex: 1, ContentHash: "page-2-edited",
			Vector: axisVector(768, map[int]float32{3: 1})},
	)
	assert.Equal(t, repositories.ReplaceEmbeddingChunksResult{
		Written:   1,
		Unchanged: 1,
		Removed:   1,
	}, second)
	assert.Equal(t, before, h.rowVersion(t, sourceID, 0),
		"a chunk whose hash did not change is not written again")

	hashes, err := h.repo.ListChunkHashes(h.ctx, repositories.ListEmbeddingChunkHashesRequest{
		Source: repositories.AIRetrievalSourceRef{
			TenantInfo: h.tenant,
			SourceType: airetrieval.SourceTypeDocument,
			SourceID:   sourceID,
		},
		ModelKey: modelA,
	})
	require.NoError(t, err)
	assert.Equal(t, []repositories.EmbeddingChunkHash{
		{ChunkIndex: 0, ContentHash: "page-1"},
		{ChunkIndex: 1, ContentHash: "page-2-edited"},
	}, hashes)

	unchanged := h.replace(t, h.tenant, airetrieval.SourceTypeDocument, sourceID, modelA, 768,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "page-1"},
		repositories.EmbeddingChunk{ChunkIndex: 1, ContentHash: "page-2-edited"},
	)
	assert.Equal(t, repositories.ReplaceEmbeddingChunksResult{Unchanged: 2}, unchanged,
		"an unchanged chunk needs no embedding")

	_, err = h.repo.ReplaceChunks(h.ctx, repositories.ReplaceEmbeddingChunksRequest{
		Source: repositories.AIRetrievalSourceRef{
			TenantInfo: h.tenant,
			SourceType: airetrieval.SourceTypeDocument,
			SourceID:   sourceID,
		},
		ModelKey:   modelA,
		Dimensions: 768,
		Chunks:     []repositories.EmbeddingChunk{{ChunkIndex: 0, ContentHash: "page-1-edited"}},
	})
	require.ErrorIs(t, err, airetrieval.ErrChunkEmbeddingMissing)
	assert.Equal(t, 2, h.countEmbeddings(t, sourceID, modelA), "a refused replace changes nothing")
}

func (h *harness) rowVersion(t *testing.T, sourceID pulid.ID, chunkIndex int) string {
	t.Helper()

	cols := buncolgen.EmbeddingColumns
	var version string
	err := h.db.NewSelect().
		Model((*airetrieval.Embedding)(nil)).
		ColumnExpr(buncolgen.EmbeddingTable.Alias+".xmin::text").
		Where(cols.SourceID.Eq(), sourceID).
		Where(cols.ChunkIndex.Eq(), chunkIndex).
		Scan(h.ctx, &version)
	require.NoError(t, err)

	return version
}

func TestCatalogEmbeddings_AreKeyedByModelCorpusItemAndHash(t *testing.T) {
	h := newHarness(t)
	h.requireVector(t)

	written, err := h.repo.PutCatalogEmbeddings(h.ctx, repositories.PutCatalogEmbeddingsRequest{
		ModelKey:   modelA,
		Corpus:     airetrieval.CatalogCorpusTools,
		Dimensions: 768,
		Items: []repositories.CatalogEmbeddingInput{
			{
				ItemKey:     "list_shipments",
				ContentHash: "v1",
				Vector:      axisVector(768, map[int]float32{0: 1}),
			},
			{
				ItemKey:     "get_customer",
				ContentHash: "v1",
				Vector:      axisVector(768, map[int]float32{1: 1}),
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, written)

	again, err := h.repo.PutCatalogEmbeddings(h.ctx, repositories.PutCatalogEmbeddingsRequest{
		ModelKey:   modelA,
		Corpus:     airetrieval.CatalogCorpusTools,
		Dimensions: 768,
		Items: []repositories.CatalogEmbeddingInput{
			{
				ItemKey:     "list_shipments",
				ContentHash: "v1",
				Vector:      axisVector(768, map[int]float32{0: 1}),
			},
			{
				ItemKey:     "list_shipments",
				ContentHash: "v2",
				Vector:      axisVector(768, map[int]float32{2: 1}),
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, again, "an item already stored for this hash is not written again")

	found, err := h.repo.GetCatalogEmbeddings(h.ctx, repositories.GetCatalogEmbeddingsRequest{
		ModelKey: modelA,
		Corpus:   airetrieval.CatalogCorpusTools,
		Items: []airetrieval.CatalogItemRef{
			{ItemKey: "list_shipments", ContentHash: "v2"},
			{ItemKey: "get_customer", ContentHash: "v1"},
			{ItemKey: "get_customer", ContentHash: "v9"},
		},
	})
	require.NoError(t, err)
	require.Len(t, found, 2)
	assert.Equal(t, "get_customer", found[0].ItemKey)
	assert.Equal(t, "list_shipments", found[1].ItemKey)
	assert.Equal(t, "v2", found[1].ContentHash)
	assert.InDelta(t, 1.0, found[1].Vector()[2], 0.001)

	pruned, err := h.repo.PruneCatalogEmbeddings(h.ctx, repositories.PruneCatalogEmbeddingsRequest{
		ModelKey: modelA,
		Corpus:   airetrieval.CatalogCorpusTools,
		Keep: []airetrieval.CatalogItemRef{
			{ItemKey: "list_shipments", ContentHash: "v2"},
			{ItemKey: "get_customer", ContentHash: "v1"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, pruned, "only the superseded hash is pruned")
}

func TestModelSwap_PromotesThePendingModelAndPurgesTheOldOneInBatches(t *testing.T) {
	h := newHarness(t)
	h.requireVector(t)

	settings := airetrieval.DefaultSettings(h.tenant.OrgID, h.tenant.BuID)
	settings.ActiveModelKey = modelA
	settings.Dimensions = 768
	settings.PendingModelKey = modelB
	settings.PendingDimensions = 1024
	_, err := h.repo.UpdateSettings(h.ctx, settings)
	require.NoError(t, err)

	sources := make([]pulid.ID, 0, 5)
	for i := range 5 {
		sourceID := pulid.MustNew("amem_")
		sources = append(sources, sourceID)
		h.replace(t, h.tenant, airetrieval.SourceTypeMemory, sourceID, modelA, 768,
			repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "a",
				Vector: axisVector(768, map[int]float32{i: 1})})
		h.replace(t, h.tenant, airetrieval.SourceTypeMemory, sourceID, modelB, 1024,
			repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "a",
				Vector: axisVector(1024, map[int]float32{i: 1})})
	}
	_, err = h.repo.MarkStale(h.ctx, repositories.MarkAIRetrievalStaleRequest{
		TenantInfo: h.tenant,
		SourceType: airetrieval.SourceTypeMemory,
		SourceIDs:  sources,
		ModelKeys:  []string{modelA, modelB},
	})
	require.NoError(t, err)

	_, err = h.repo.PurgeModel(h.ctx, repositories.PurgeAIRetrievalModelRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
	})
	require.ErrorIs(t, err, airetrieval.ErrModelInUse, "the active model is never purged")

	_, err = h.repo.SwapModel(h.ctx, repositories.SwapAIRetrievalModelRequest{
		TenantInfo:      h.tenant,
		PendingModelKey: "api.example.test/something-else/768",
	})
	require.ErrorIs(t, err, airetrieval.ErrPendingModelMoved)

	swapped, err := h.repo.SwapModel(h.ctx, repositories.SwapAIRetrievalModelRequest{
		TenantInfo:      h.tenant,
		PendingModelKey: modelB,
	})
	require.NoError(t, err)
	assert.Equal(t, modelA, swapped.RetiredModelKey)
	assert.Equal(t, modelB, swapped.Settings.ActiveModelKey)
	assert.Equal(t, 1024, swapped.Settings.Dimensions)
	assert.False(t, swapped.Settings.HasPendingModel())

	stored, err := h.repo.GetSettings(h.ctx, h.tenant)
	require.NoError(t, err)
	assert.Equal(t, modelB, stored.ActiveModelKey)
	assert.Empty(t, stored.PendingModelKey)

	_, err = h.repo.PurgeModel(h.ctx, repositories.PurgeAIRetrievalModelRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelB,
	})
	require.ErrorIs(t, err, airetrieval.ErrModelInUse)

	rounds := 0
	for {
		purged, purgeErr := h.repo.PurgeModel(h.ctx, repositories.PurgeAIRetrievalModelRequest{
			TenantInfo: h.tenant,
			ModelKey:   modelA,
			BatchSize:  2,
		})
		require.NoError(t, purgeErr)
		if purged.Done() {
			break
		}
		assert.LessOrEqual(t, purged.Embeddings, 2)
		assert.LessOrEqual(t, purged.IndexEntries, 2)
		rounds++
		require.Less(t, rounds, 10, "purging must finish")
	}
	assert.Equal(t, 3, rounds, "five rows at two per batch take three batches")

	for _, sourceID := range sources {
		assert.Zero(t, h.countEmbeddings(t, sourceID, modelA))
		assert.Zero(t, h.countIndexEntries(t, sourceID, modelA))
		assert.Equal(t, 1, h.countEmbeddings(t, sourceID, modelB), "the new model's rows stay")
		assert.Equal(t, 1, h.countIndexEntries(t, sourceID, modelB))
	}

	_, err = h.repo.SwapModel(h.ctx, repositories.SwapAIRetrievalModelRequest{
		TenantInfo:      h.tenant,
		PendingModelKey: modelB,
	})
	require.ErrorIs(t, err, airetrieval.ErrNoPendingModel)
}
