//go:build integration

package airetrievalrepository

import (
	"os"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/migrations"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeletingASourceRemovesItsRetrievalRows(t *testing.T) {
	h := newHarness(t)

	memory := h.insertMemory(t, "Acme closes at four on Fridays")
	doc := h.insertDocument(t)
	kept := h.insertMemory(t, "Quote in the customer's currency")

	for _, source := range []struct {
		sourceType airetrieval.SourceType
		id         pulid.ID
	}{
		{airetrieval.SourceTypeMemory, memory.ID},
		{airetrieval.SourceTypeMemory, kept.ID},
		{airetrieval.SourceTypeDocument, doc.ID},
	} {
		_, err := h.repo.MarkStale(h.ctx, repositories.MarkAIRetrievalStaleRequest{
			TenantInfo: h.tenant,
			SourceType: source.sourceType,
			SourceIDs:  []pulid.ID{source.id},
			ModelKeys:  []string{modelA},
		})
		require.NoError(t, err)
	}

	availability, err := h.repo.VectorAvailability(h.ctx)
	require.NoError(t, err)
	withVectors := availability.Available
	if !withVectors && os.Getenv(expectVectorEnv) == "true" {
		t.Fatalf("pgvector is expected on this server but reports %s", availability.Reason)
	}
	if withVectors {
		h.replace(t, h.tenant, airetrieval.SourceTypeMemory, memory.ID, modelA, 768,
			repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "m",
				Vector: axisVector(768, map[int]float32{0: 1})})
		h.replace(t, h.tenant, airetrieval.SourceTypeMemory, kept.ID, modelA, 768,
			repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "k",
				Vector: axisVector(768, map[int]float32{1: 1})})
		h.replace(t, h.tenant, airetrieval.SourceTypeDocument, doc.ID, modelA, 768,
			repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "d0",
				Vector: axisVector(768, map[int]float32{2: 1})},
			repositories.EmbeddingChunk{ChunkIndex: 1, ContentHash: "d1",
				Vector: axisVector(768, map[int]float32{3: 1})})
	}

	_, err = h.db.NewDelete().
		Model((*agent.Memory)(nil)).
		Where(buncolgen.MemoryColumns.ID.Eq(), memory.ID).
		Exec(h.ctx)
	require.NoError(t, err)
	_, err = h.db.NewDelete().
		Model((*document.Document)(nil)).
		Where(buncolgen.DocumentColumns.ID.Eq(), doc.ID).
		Exec(h.ctx)
	require.NoError(t, err)

	assert.Zero(t, h.countIndexEntries(t, memory.ID, ""))
	assert.Zero(t, h.countIndexEntries(t, doc.ID, ""))
	assert.Equal(t, 1, h.countIndexEntries(t, kept.ID, ""), "other sources are untouched")

	if withVectors {
		assert.Zero(t, h.countEmbeddings(t, memory.ID, ""))
		assert.Zero(t, h.countEmbeddings(t, doc.ID, ""))
		assert.Equal(t, 1, h.countEmbeddings(t, kept.ID, ""))
	}

	triggers := h.triggersOn(t)
	for _, table := range []string{"agent_memories", "documents", "inbound_messages"} {
		assert.Contains(t, triggers[table], "trg_"+table+"_ai_index_entries_purge")
		if withVectors {
			assert.Contains(t, triggers[table], "trg_"+table+"_ai_embeddings_purge")
		}
	}
}

func (h *harness) triggersOn(t *testing.T) map[string][]string {
	t.Helper()

	var rows []struct {
		Table   string `bun:"table_name"`
		Trigger string `bun:"trigger_name"`
	}
	require.NoError(t, h.db.NewRaw(
		`SELECT c.relname AS table_name, tg.tgname AS trigger_name
		FROM pg_trigger AS tg
		JOIN pg_class AS c ON c.oid = tg.tgrelid
		WHERE NOT tg.tgisinternal AND tg.tgname LIKE 'trg_%_purge'`,
	).Scan(h.ctx, &rows))

	byTable := make(map[string][]string, len(rows))
	for _, row := range rows {
		byTable[row.Table] = append(byTable[row.Table], row.Trigger)
	}

	return byTable
}

func TestIndexOutbox_ClaimsOnceAndKeepsLateWritesPending(t *testing.T) {
	h := newHarness(t)

	first := h.insertMemory(t, "Detention starts after two free hours")
	second := h.insertMemory(t, "Carrier Blue Ox needs a lumper receipt")
	now := time.Now().Unix()

	marked, err := h.repo.MarkStale(h.ctx, repositories.MarkAIRetrievalStaleRequest{
		TenantInfo: h.tenant,
		SourceType: airetrieval.SourceTypeMemory,
		SourceIDs:  []pulid.ID{first.ID, second.ID, first.ID},
		ModelKeys:  []string{modelA},
		Now:        now,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, marked, "a repeated id is marked once")

	claimed, err := h.repo.ClaimIndexEntries(h.ctx, repositories.ClaimIndexEntriesRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
		Limit:      10,
		Lease:      time.Minute,
		Now:        now,
	})
	require.NoError(t, err)
	require.Len(t, claimed, 2)
	for _, entry := range claimed {
		assert.Equal(t, 1, entry.Attempts)
		require.NotNil(t, entry.LeaseExpiresAt)
		assert.Equal(t, now+60, *entry.LeaseExpiresAt)
	}

	again, err := h.repo.ClaimIndexEntries(h.ctx, repositories.ClaimIndexEntriesRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
		Now:        now + 1,
	})
	require.NoError(t, err)
	assert.Empty(t, again, "a leased entry is not claimed twice")

	_, err = h.repo.MarkStale(h.ctx, repositories.MarkAIRetrievalStaleRequest{
		TenantInfo: h.tenant,
		SourceType: airetrieval.SourceTypeMemory,
		SourceIDs:  []pulid.ID{second.ID},
		ModelKeys:  []string{modelA},
		Now:        now + 2,
	})
	require.NoError(t, err)

	outcomes := make([]repositories.IndexEntryOutcome, 0, len(claimed))
	for _, entry := range claimed {
		outcomes = append(outcomes, repositories.IndexEntryOutcome{
			Key:        entry.Key(),
			Generation: entry.Generation,
			ChunkCount: 1,
		})
	}
	result, err := h.repo.MarkIndexed(h.ctx, repositories.MarkIndexEntriesRequest{
		Outcomes: outcomes,
		Now:      now + 3,
	})
	require.NoError(t, err)
	assert.Equal(t, repositories.MarkIndexEntriesResult{Applied: 1, Superseded: 1}, result,
		"a source written while it was being indexed stays pending")

	retry, err := h.repo.ClaimIndexEntries(h.ctx, repositories.ClaimIndexEntriesRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
		Now:        now + 4,
	})
	require.NoError(t, err)
	require.Len(t, retry, 1)
	assert.Equal(t, second.ID, retry[0].SourceID)
	assert.Equal(t, int64(2), retry[0].Generation)

	failed, err := h.repo.MarkFailed(h.ctx, repositories.MarkIndexEntriesRequest{
		Outcomes: []repositories.IndexEntryOutcome{{
			Key:        retry[0].Key(),
			Generation: retry[0].Generation,
			Error:      "provider returned 503",
			RetryAt:    now + 600,
		}},
		Now: now + 5,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, failed.Applied)

	early, err := h.repo.ClaimIndexEntries(h.ctx, repositories.ClaimIndexEntriesRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
		Now:        now + 6,
	})
	require.NoError(t, err)
	assert.Empty(t, early, "a retry waits for its backoff")

	counts, err := h.repo.CountIndexEntries(h.ctx, repositories.CountIndexEntriesRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
	})
	require.NoError(t, err)
	byStatus := make(map[airetrieval.IndexStatus]int, len(counts))
	for _, count := range counts {
		assert.Equal(t, airetrieval.SourceTypeMemory, count.SourceType)
		byStatus[count.Status] = count.Count
	}
	assert.Equal(t, map[airetrieval.IndexStatus]int{
		airetrieval.IndexStatusIndexed: 1,
		airetrieval.IndexStatusPending: 1,
	}, byStatus)

	stale, err := h.repo.FindStaleSources(h.ctx, repositories.FindStaleAIRetrievalSourcesRequest{
		TenantInfo: h.tenant,
		SourceType: airetrieval.SourceTypeMemory,
		ModelKey:   modelA,
	})
	require.NoError(t, err)
	assert.Empty(
		t,
		stale,
		"an indexed source that has not changed since, and a pending one, are not stale",
	)

	third := h.insertMemory(t, "Receiver needs appointments a day ahead")
	_, err = h.db.NewUpdate().
		Model((*agent.Memory)(nil)).
		Set(buncolgen.MemoryColumns.UpdatedAt.Set(), now+100).
		Where(buncolgen.MemoryColumns.ID.Eq(), first.ID).
		Exec(h.ctx)
	require.NoError(t, err)

	stale, err = h.repo.FindStaleSources(h.ctx, repositories.FindStaleAIRetrievalSourcesRequest{
		TenantInfo: h.tenant,
		SourceType: airetrieval.SourceTypeMemory,
		ModelKey:   modelA,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{first.ID, third.ID}, stale,
		"a source edited after it was indexed, and one never indexed, are stale")
}

func TestFallback_ReportsUnavailableWithoutTheExtension(t *testing.T) {
	h := newHarness(t)

	initial, err := h.repo.VectorAvailability(h.ctx)
	require.NoError(t, err)
	if os.Getenv(expectVectorEnv) == "false" {
		require.False(t, initial.Available)
		require.Equal(t, airetrieval.UnavailableReasonExtensionMissing, initial.Reason,
			"the fallback job must run on a server without pgvector")
	}

	for _, statement := range []string{
		"DROP TABLE IF EXISTS ai_catalog_embeddings",
		"DROP TABLE IF EXISTS ai_embeddings",
		"DROP FUNCTION IF EXISTS ai_embeddings_purge_source() CASCADE",
		"DROP EXTENSION IF EXISTS vector",
	} {
		_, err = h.db.ExecContext(h.ctx, statement)
		require.NoError(t, err)
	}

	repo := newRepository(h.conn)
	availability, err := repo.VectorAvailability(h.ctx)
	require.NoError(t, err)
	assert.Equal(t, airetrieval.Availability{
		Reason: airetrieval.UnavailableReasonExtensionMissing,
	}, availability)

	_, err = repo.Search(h.ctx, repositories.VectorSearchRequest{
		TenantInfo: h.tenant,
		ModelKey:   modelA,
		Dimensions: 768,
		Query:      axisVector(768, map[int]float32{0: 1}),
	})
	require.ErrorIs(t, err, airetrieval.ErrVectorUnavailable)
	var unavailable *airetrieval.UnavailableError
	require.ErrorAs(t, err, &unavailable)
	assert.Equal(t, airetrieval.UnavailableReasonExtensionMissing, unavailable.Reason)

	_, err = repo.ReplaceChunks(h.ctx, repositories.ReplaceEmbeddingChunksRequest{
		Source: repositories.AIRetrievalSourceRef{
			TenantInfo: h.tenant,
			SourceType: airetrieval.SourceTypeMemory,
			SourceID:   pulid.MustNew("amem_"),
		},
		ModelKey:   modelA,
		Dimensions: 768,
		Chunks: []repositories.EmbeddingChunk{{
			ChunkIndex:  0,
			ContentHash: "x",
			Vector:      axisVector(768, map[int]float32{0: 1}),
		}},
	})
	require.ErrorIs(t, err, airetrieval.ErrVectorUnavailable)

	_, err = repo.GetCatalogEmbeddings(h.ctx, repositories.GetCatalogEmbeddingsRequest{
		ModelKey: modelA,
		Corpus:   airetrieval.CatalogCorpusTools,
	})
	require.ErrorIs(t, err, airetrieval.ErrVectorUnavailable)

	memory := h.insertMemory(t, "Keyword search keeps working")
	source := repositories.AIRetrievalSourceRef{
		TenantInfo: h.tenant,
		SourceType: airetrieval.SourceTypeMemory,
		SourceID:   memory.ID,
	}
	marked, err := repo.MarkStale(h.ctx, repositories.MarkAIRetrievalStaleRequest{
		TenantInfo: h.tenant,
		SourceType: airetrieval.SourceTypeMemory,
		SourceIDs:  []pulid.ID{memory.ID},
		ModelKeys:  []string{modelA},
	})
	require.NoError(t, err, "the outbox does not need the extension")
	assert.Equal(t, 1, marked)

	deleted, err := repo.DeleteSource(h.ctx, source)
	require.NoError(t, err)
	assert.Equal(t, repositories.DeleteAIRetrievalSourceResult{IndexEntries: 1}, deleted)

	settings, err := repo.GetSettings(h.ctx, h.tenant)
	require.NoError(t, err)
	assert.True(t, settings.ID.IsNil(), "an organization without a row reads the defaults")
	assert.True(t, settings.MemoryEnabled)

	_, err = h.db.NewDelete().
		Model((*agent.Memory)(nil)).
		Where(buncolgen.MemoryColumns.ID.Eq(), memory.ID).
		Exec(h.ctx)
	require.NoError(t, err, "deleting a source works without the vector triggers")
}

func TestEnableVector_InstallsTheSchemaAMigrationSkipped(t *testing.T) {
	h := newHarness(t)
	h.requireVector(t)

	for _, statement := range []string{
		"DROP TABLE IF EXISTS ai_catalog_embeddings",
		"DROP TABLE IF EXISTS ai_embeddings",
		"DROP FUNCTION IF EXISTS ai_embeddings_purge_source() CASCADE",
	} {
		_, err := h.db.ExecContext(h.ctx, statement)
		require.NoError(t, err)
	}

	repo := newRepository(h.conn)
	availability, err := repo.VectorAvailability(h.ctx)
	require.NoError(t, err)
	assert.False(t, availability.Available)
	assert.Equal(t, airetrieval.UnavailableReasonSchemaMissing, availability.Reason)
	assert.True(t, availability.ExtensionInstalled)

	result, err := migrations.EnableVector(h.ctx, h.db)
	require.NoError(t, err)
	assert.Equal(t, dbdialect.VectorSchemaMissing, result.Before.State)
	assert.Equal(t, dbdialect.VectorReady, result.After.State)

	again, err := migrations.EnableVector(h.ctx, h.db)
	require.NoError(t, err, "enabling twice is harmless")
	assert.Equal(t, dbdialect.VectorReady, again.Before.State)

	repo = newRepository(h.conn)
	availability, err = repo.VectorAvailability(h.ctx)
	require.NoError(t, err)
	assert.True(t, availability.Available)

	memory := h.insertMemory(t, "Indexed after enabling")
	h.repo = repo
	h.replace(t, h.tenant, airetrieval.SourceTypeMemory, memory.ID, modelA, 768,
		repositories.EmbeddingChunk{ChunkIndex: 0, ContentHash: "e",
			Vector: axisVector(768, map[int]float32{0: 1})})
	_, err = h.db.NewDelete().
		Model((*agent.Memory)(nil)).
		Where(buncolgen.MemoryColumns.ID.Eq(), memory.ID).
		Exec(h.ctx)
	require.NoError(t, err)
	assert.Zero(t, h.countEmbeddings(t, memory.ID, ""), "the re-created trigger removes rows")
}
