package retrievalservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveModelChange(t *testing.T) {
	t.Parallel()

	blank := airetrieval.DefaultSettings(testTenant.OrgID, testTenant.BuID)
	withPending := activeSettings()
	withPending.PendingModelKey = pendingKey
	withPending.PendingDimensions = airetrieval.Dimensions1024

	tests := []struct {
		name       string
		settings   *airetrieval.Settings
		configured string
		dimensions int
		changed    bool
		active     string
		pending    string
	}{
		{
			name:       "no active model adopts the configured one",
			settings:   blank,
			configured: activeKey,
			dimensions: 768,
			changed:    true,
			active:     activeKey,
		},
		{
			name:       "the configured model is already active",
			settings:   activeSettings(),
			configured: activeKey,
			dimensions: 768,
			active:     activeKey,
		},
		{
			name:       "a new configured model becomes pending beside the active one",
			settings:   activeSettings(),
			configured: pendingKey,
			dimensions: 1024,
			changed:    true,
			active:     activeKey,
			pending:    pendingKey,
		},
		{
			name:       "the pending model is still the configured one",
			settings:   withPending,
			configured: pendingKey,
			dimensions: 1024,
			active:     activeKey,
			pending:    pendingKey,
		},
		{
			name:       "going back to the active model drops the pending one",
			settings:   withPending,
			configured: activeKey,
			dimensions: 768,
			changed:    true,
			active:     activeKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next, changed := ResolveModelChange(tt.settings, tt.configured, tt.dimensions)
			assert.Equal(t, tt.changed, changed)
			assert.Equal(t, tt.active, next.ActiveModelKey)
			assert.Equal(t, tt.pending, next.PendingModelKey)
			if tt.pending != "" {
				assert.Equal(t, tt.dimensions, next.PendingDimensions)
			}
		})
	}
}

func TestPlanAdoptsTheConfiguredModelAndListsRetiredKeys(t *testing.T) {
	t.Parallel()

	svc := newTestService(airetrieval.DefaultSettings(testTenant.OrgID, testTenant.BuID))
	svc.sources.keys = []string{activeKey, "old.example.com/embed@1536"}

	plan, err := svc.Plan(t.Context(), testTenant)
	require.NoError(t, err)
	assert.True(t, plan.Active)
	assert.Equal(t, activeKey, plan.ActiveModelKey)
	assert.Empty(t, plan.PendingModelKey)
	assert.Equal(t, []string{"old.example.com/embed@1536"}, plan.RetiredKeys)
	require.Len(t, svc.repo.updated, 1)
	assert.Equal(t, airetrieval.Dimensions768, svc.repo.updated[0].Dimensions)
}

func TestPlanStartsAModelChangeBesideTheActiveRows(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	svc.embeddings.configured = pendingKey

	plan, err := svc.Plan(t.Context(), testTenant)
	require.NoError(t, err)
	assert.Equal(t, []string{activeKey, pendingKey}, plan.ModelKeys())
	assert.Equal(t, airetrieval.Dimensions1024, svc.repo.settings.PendingDimensions)
}

func TestPlanReportsWhyItCannotIndex(t *testing.T) {
	t.Parallel()

	t.Run("no vector storage", func(t *testing.T) {
		t.Parallel()

		svc := newTestService(activeSettings())
		svc.repo.availability = airetrieval.Availability{
			Reason: airetrieval.UnavailableReasonExtensionMissing,
		}
		plan, err := svc.Plan(t.Context(), testTenant)
		require.NoError(t, err)
		assert.False(t, plan.Active)
		assert.Equal(t, airetrieval.UnavailableReasonExtensionMissing, plan.Reason)
	})

	t.Run("no embedding provider", func(t *testing.T) {
		t.Parallel()

		svc := newTestService(activeSettings())
		svc.embeddings.err = serviceports.ErrNoProviderConfigured
		plan, err := svc.Plan(t.Context(), testTenant)
		require.NoError(t, err)
		assert.Equal(t, airetrieval.UnavailableReasonNoProvider, plan.Reason)
	})

	t.Run("AI turned off", func(t *testing.T) {
		t.Parallel()

		svc := newTestService(activeSettings())
		svc.embeddings.err = errortypes.NewBusinessError("AI is turned off")
		plan, err := svc.Plan(t.Context(), testTenant)
		require.NoError(t, err)
		assert.Equal(t, airetrieval.UnavailableReasonNoProvider, plan.Reason)
	})

	t.Run("every source turned off", func(t *testing.T) {
		t.Parallel()

		settings := activeSettings()
		settings.MemoryEnabled = false
		settings.DocumentsEnabled = false
		settings.InboundMessagesEnabled = false
		plan, err := newTestService(settings).Plan(t.Context(), testTenant)
		require.NoError(t, err)
		assert.Equal(t, airetrieval.UnavailableReasonDisabled, plan.Reason)
	})

	t.Run("paused by a person", func(t *testing.T) {
		t.Parallel()

		settings := activeSettings()
		settings.Paused = true
		settings.PausedReason = airetrieval.PauseReasonManual
		plan, err := newTestService(settings).Plan(t.Context(), testTenant)
		require.NoError(t, err)
		assert.Equal(t, airetrieval.UnavailableReasonDisabled, plan.Reason)
	})

	t.Run("a transient provider error is returned for a retry", func(t *testing.T) {
		t.Parallel()

		svc := newTestService(activeSettings())
		svc.embeddings.err = errors.New("connection refused")
		_, err := svc.Plan(t.Context(), testTenant)
		require.ErrorContains(t, err, "connection refused")
	})
}

func TestPlanPausesAtTheMonthlyBudgetAndResumesUnderIt(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	svc.usage.spent = airetrieval.DefaultMonthlyIndexingBudgetUSD

	plan, err := svc.Plan(t.Context(), testTenant)
	require.NoError(t, err)
	assert.Equal(t, airetrieval.UnavailableReasonBudgetPaused, plan.Reason)
	require.Len(t, svc.repo.paused, 1)
	assert.True(t, svc.repo.paused[0].Paused)
	assert.Equal(t, airetrieval.PauseReasonBudget, svc.repo.paused[0].Reason)

	svc.usage.spent = decimal.RequireFromString("2.50")
	plan, err = svc.Plan(t.Context(), testTenant)
	require.NoError(t, err)
	assert.True(t, plan.Active)
	require.Len(t, svc.repo.paused, 2)
	assert.False(t, svc.repo.paused[1].Paused)
}

func TestIndexBatchEmbedsOnlyChangedChunks(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	memory := &agent.Memory{
		ID:     pulid.MustNew("amem_"),
		Kind:   agent.MemoryKindFact,
		Status: agent.MemoryStatusActive,
	}
	memory.Content = "Acme closes at 3 PM on Fridays."
	unchanged := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindFact,
		Status:  agent.MemoryStatusActive,
		Content: "Globex allows two hours of free time.",
	}
	svc.sources.memories = []*agent.Memory{memory, unchanged}
	svc.repo.hashes[unchanged.ID] = []repositories.EmbeddingChunkHash{
		{ChunkIndex: 0, ContentHash: ChunkMemory(unchanged)[0].Hash},
	}
	svc.repo.claim = []*airetrieval.IndexEntry{
		entry(airetrieval.SourceTypeMemory, memory.ID),
		entry(airetrieval.SourceTypeMemory, unchanged.ID),
	}

	result, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   activeKey,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Claimed)
	assert.Equal(t, 2, result.Indexed)
	assert.Equal(t, 1, result.ChunksEmbedded)
	assert.Equal(t, []string{ChunkMemory(memory)[0].Text}, svc.embeddings.inputs())

	request := svc.embeddings.requests[0]
	assert.Equal(t, serviceports.EmbeddingPurposeDocument, request.Purpose)
	assert.Equal(t, activeKey, request.ModelKey)
	assert.Equal(t, "Indexing", string(request.Surface))

	require.Len(t, svc.repo.replaced, 2)
	for _, replaced := range svc.repo.replaced {
		require.Len(t, replaced.Chunks, 1)
		if replaced.Source.SourceID == unchanged.ID {
			assert.Nil(t, replaced.Chunks[0].Vector, "an unchanged chunk is not embedded again")
		} else {
			assert.Len(t, replaced.Chunks[0].Vector, 768)
		}
	}
	require.Len(t, svc.repo.indexed, 2)
	assert.Equal(t, int64(3), svc.repo.indexed[0].Generation)
}

func TestIndexBatchSkipsWhatMayNotBeEmbedded(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	retired := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindFact,
		Status:  agent.MemoryStatusRetired,
		Content: "Old fact.",
	}
	workerDoc := documentSource("Medical examiner's certificate for J. Doe.")
	workerDoc.Document.ResourceType = "worker"
	oldVersion := documentSource("Superseded rate confirmation.")
	oldVersion.Document.IsCurrentVersion = false
	unknownOwner := documentSource("Attached somewhere unregistered.")
	unknownOwner.Document.ResourceType = "mystery_record"
	unread := documentSource()

	svc.sources.memories = []*agent.Memory{retired}
	svc.sources.documents = []*repositories.RetrievalDocumentSource{
		workerDoc, oldVersion, unknownOwner, unread,
	}
	missing := pulid.MustNew("imsg_")
	svc.repo.claim = []*airetrieval.IndexEntry{
		entry(airetrieval.SourceTypeMemory, retired.ID),
		entry(airetrieval.SourceTypeDocument, workerDoc.Document.ID),
		entry(airetrieval.SourceTypeDocument, oldVersion.Document.ID),
		entry(airetrieval.SourceTypeDocument, unknownOwner.Document.ID),
		entry(airetrieval.SourceTypeDocument, unread.Document.ID),
		entry(airetrieval.SourceTypeInboundMessage, missing),
	}

	result, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   activeKey,
	})
	require.NoError(t, err)
	assert.Equal(t, 6, result.Skipped)
	assert.Empty(t, svc.embeddings.requests, "nothing that may not be sent is embedded")
	require.Len(t, svc.repo.skipped, 5)
	for _, replaced := range svc.repo.replaced {
		assert.Empty(t, replaced.Chunks, "a skipped source keeps no embeddings")
	}
	require.Len(t, svc.repo.deleted, 1)
	assert.Equal(t, missing, svc.repo.deleted[0].SourceID)

	reasons := make(map[pulid.ID]string, len(svc.repo.skipped))
	for _, skipped := range svc.repo.skipped {
		reasons[skipped.Key.SourceID] = skipped.Error
	}
	assert.Equal(t, skipSensitiveDocument, reasons[workerDoc.Document.ID])
	assert.Equal(t, skipSensitiveDocument, reasons[unknownOwner.Document.ID])
	assert.Equal(t, skipUnsearchableDoc, reasons[oldVersion.Document.ID])
	assert.Equal(t, skipUnreadDocument, reasons[unread.Document.ID])
	assert.Equal(t, skipInactiveMemory, reasons[retired.ID])
}

func TestIndexBatchIndexesAShipmentDocumentAndMail(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	doc := documentSource("Linehaul $2,450 for load 4471.", "Detention $75 per hour.")
	message := &inboundmessage.InboundMessage{
		ID:          pulid.MustNew("imsg_"),
		FromAddress: "ops@globex.test",
		Subject:     "Where is load 4471?",
		TextBody:    "Please send an ETA.\n> quoted",
	}
	svc.sources.documents = []*repositories.RetrievalDocumentSource{doc}
	svc.sources.messages = []*inboundmessage.InboundMessage{message}
	svc.repo.claim = []*airetrieval.IndexEntry{
		entry(airetrieval.SourceTypeDocument, doc.Document.ID),
		entry(airetrieval.SourceTypeInboundMessage, message.ID),
	}

	result, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   activeKey,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Indexed)
	assert.Equal(t, 3, result.ChunksEmbedded)
	assert.NotContains(t, svc.embeddings.inputs()[2], "quoted")
	assert.True(t, result.CostUSD.Equal(decimal.RequireFromString("0.01")))
}

func TestIndexBatchRecordsAnEmbeddingFailureForARetry(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	memory := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindFact,
		Status:  agent.MemoryStatusActive,
		Content: "Acme closes at 3 PM.",
	}
	svc.sources.memories = []*agent.Memory{memory}
	svc.repo.claim = []*airetrieval.IndexEntry{entry(airetrieval.SourceTypeMemory, memory.ID)}
	svc.embeddings.embedErr = errors.New("provider timed out")

	result, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   activeKey,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, svc.repo.failed, 1)
	assert.Contains(t, svc.repo.failed[0].Error, "provider timed out")
	assert.Equal(t, RetryAt(1_800_000_000, 1), svc.repo.failed[0].RetryAt)
	assert.Empty(t, svc.repo.replaced, "nothing is written for a failed embed")
}

func TestIndexBatchFailsAConfigurationErrorForGood(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	memory := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindFact,
		Status:  agent.MemoryStatusActive,
		Content: "Acme closes at 3 PM.",
	}
	svc.sources.memories = []*agent.Memory{memory}
	svc.repo.claim = []*airetrieval.IndexEntry{entry(airetrieval.SourceTypeMemory, memory.ID)}
	svc.embeddings.embedErr = errortypes.NewBusinessError("the provider returned 512 dimensions").
		WithInternal(serviceports.ErrEmbeddingDimensionMismatch)

	_, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   activeKey,
	})
	require.NoError(t, err)
	require.Len(t, svc.repo.failed, 1)
	assert.Zero(t, svc.repo.failed[0].RetryAt)
}

func TestIndexBatchStopsAtTheBudget(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	svc.usage.spent = decimal.RequireFromString("9.995")
	memory := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindFact,
		Status:  agent.MemoryStatusActive,
		Content: "Acme closes at 3 PM.",
	}
	svc.sources.memories = []*agent.Memory{memory}
	svc.repo.claim = []*airetrieval.IndexEntry{entry(airetrieval.SourceTypeMemory, memory.ID)}

	result, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   activeKey,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Indexed, "work already paid for is kept")
	assert.True(t, result.BudgetReached)
	require.Len(t, svc.repo.paused, 1)
	assert.Equal(t, airetrieval.PauseReasonBudget, svc.repo.paused[0].Reason)

	next, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   activeKey,
	})
	require.NoError(t, err)
	assert.True(t, next.BudgetReached)
	assert.Zero(t, next.Claimed, "a paused organization claims nothing")
}

func TestIndexBatchClaimsOnlyEnabledSources(t *testing.T) {
	t.Parallel()

	settings := activeSettings()
	settings.DocumentsEnabled = false
	svc := newTestService(settings)

	_, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   activeKey,
	})
	require.NoError(t, err)
	require.Len(t, svc.repo.claims, 1)
	assert.Equal(t, []airetrieval.SourceType{
		airetrieval.SourceTypeMemory,
		airetrieval.SourceTypeInboundMessage,
	}, svc.repo.claims[0].SourceTypes)
}

func TestIndexBatchIgnoresARetiredModelKey(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	result, err := svc.IndexBatch(t.Context(), serviceports.RetrievalIndexBatchRequest{
		TenantInfo: testTenant,
		ModelKey:   "old.example.com/embed@1536",
	})
	require.NoError(t, err)
	assert.Zero(t, result.Claimed)
	assert.Empty(t, svc.repo.claims)
}

func TestCompleteModelChangeWaitsForTheWholeCorpus(t *testing.T) {
	t.Parallel()

	settings := activeSettings()
	settings.PendingModelKey = pendingKey
	settings.PendingDimensions = airetrieval.Dimensions1024
	svc := newTestService(settings)
	svc.repo.stale[string(airetrieval.SourceTypeDocument)+"|"+pendingKey] = []pulid.ID{
		pulid.MustNew("doc_"),
	}

	result, err := svc.CompleteModelChange(t.Context(), testTenant, pendingKey)
	require.NoError(t, err)
	assert.False(t, result.Swapped, "a source not yet indexed under the new model holds the swap")

	svc.repo.counts = []repositories.IndexEntryCount{
		{
			SourceType: airetrieval.SourceTypeMemory,
			Status:     airetrieval.IndexStatusPending,
			Count:      2,
		},
	}
	result, err = svc.CompleteModelChange(t.Context(), testTenant, pendingKey)
	require.NoError(t, err)
	assert.False(t, result.Swapped, "pending entries hold the swap")

	svc.repo.counts = []repositories.IndexEntryCount{
		{
			SourceType: airetrieval.SourceTypeMemory,
			Status:     airetrieval.IndexStatusIndexed,
			Count:      2,
		},
	}
	result, err = svc.CompleteModelChange(t.Context(), testTenant, pendingKey)
	require.NoError(t, err)
	assert.True(t, result.Swapped)
	assert.Equal(t, activeKey, result.RetiredModelKey)
	assert.Equal(t, pendingKey, svc.repo.settings.ActiveModelKey)

	again, err := svc.CompleteModelChange(t.Context(), testTenant, pendingKey)
	require.NoError(t, err)
	assert.False(t, again.Swapped, "a key that is no longer pending is not swapped twice")
}

func TestPurgeRetiredModelRunsBatchesUntilDone(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	svc.repo.purges = []repositories.PurgeAIRetrievalModelResult{
		{Embeddings: 1000, IndexEntries: 400},
		{Embeddings: 250},
	}

	result, err := svc.PurgeRetiredModel(t.Context(), testTenant, "old.example.com/embed@1536")
	require.NoError(t, err)
	assert.True(t, result.Done)
	assert.Equal(t, 1250, result.Embeddings)
	assert.Equal(t, 400, result.IndexEntries)
	assert.Equal(t, 3, svc.repo.purged)
}

func TestMarkStaleQueuesEveryIndexedModelAndWakesTheIndexer(t *testing.T) {
	t.Parallel()

	settings := activeSettings()
	settings.PendingModelKey = pendingKey
	settings.PendingDimensions = airetrieval.Dimensions1024
	svc := newTestService(settings)
	id := pulid.MustNew("doc_")

	require.NoError(t, svc.MarkStale(t.Context(), testTenant, airetrieval.SourceTypeDocument, id))
	require.Len(t, svc.repo.marked, 1)
	assert.Equal(t, []string{activeKey, pendingKey}, svc.repo.marked[0].ModelKeys)
	assert.Equal(t, []pulid.ID{id}, svc.repo.marked[0].SourceIDs)
	assert.Equal(t, []string{"ai-index:" + testTenant.OrgID.String()}, svc.signals.signals)
}

func TestMarkStaleSurvivesAnUnreachableTemporal(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	svc.signals.err = errors.New("temporal is down")

	err := svc.MarkStale(
		t.Context(),
		testTenant,
		airetrieval.SourceTypeMemory,
		pulid.MustNew("amem_"),
	)
	require.NoError(t, err)
	assert.Len(t, svc.repo.marked, 1, "the outbox keeps the source for the hourly sweep")
}

func TestMarkStaleLeavesDisabledSourcesAndUnconfiguredOrganizationsAlone(t *testing.T) {
	t.Parallel()

	settings := activeSettings()
	settings.InboundMessagesEnabled = false
	svc := newTestService(settings)
	require.NoError(t, svc.MarkStale(t.Context(), testTenant,
		airetrieval.SourceTypeInboundMessage, pulid.MustNew("imsg_")))
	assert.Empty(t, svc.repo.marked)

	unconfigured := newTestService(airetrieval.DefaultSettings(testTenant.OrgID, testTenant.BuID))
	require.NoError(t, unconfigured.MarkStale(t.Context(), testTenant,
		airetrieval.SourceTypeMemory, pulid.MustNew("amem_")))
	assert.Empty(t, unconfigured.repo.marked)
	assert.Empty(t, unconfigured.signals.signals)

	require.Error(t, svc.MarkStale(t.Context(), testTenant, airetrieval.SourceType("Shipment"),
		pulid.MustNew("shp_")))
}

func TestSweepMarksStaleSourcesAndCountsWaitingOnes(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	stale := []pulid.ID{pulid.MustNew("doc_"), pulid.MustNew("doc_")}
	svc.repo.stale[string(airetrieval.SourceTypeDocument)+"|"+activeKey] = stale
	svc.repo.counts = []repositories.IndexEntryCount{
		{Status: airetrieval.IndexStatusPending, Count: 4},
		{Status: airetrieval.IndexStatusIndexed, Count: 9},
	}

	result, err := svc.Sweep(t.Context(), testTenant)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Marked)
	assert.Equal(t, 4, result.Waiting)
	assert.True(t, result.HasWork())
	assert.Empty(t, svc.signals.signals, "the sweep leaves waking to its caller")
}

func TestReindexPageMarksAPageAndSaysWhereToGoOn(t *testing.T) {
	t.Parallel()

	svc := newTestService(activeSettings())
	svc.sources.ids = []pulid.ID{pulid.MustNew("doc_"), pulid.MustNew("doc_")}

	page, err := svc.ReindexPage(t.Context(), serviceports.RetrievalReindexPageRequest{
		TenantInfo: testTenant,
		SourceType: airetrieval.SourceTypeDocument,
		Limit:      2,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, page.Marked)
	assert.False(t, page.Done)
	assert.Equal(t, svc.sources.ids[1], page.Next)
	assert.Len(t, svc.signals.signals, 1)

	page, err = svc.ReindexPage(t.Context(), serviceports.RetrievalReindexPageRequest{
		TenantInfo: testTenant,
		SourceType: airetrieval.SourceTypeDocument,
		AfterID:    page.Next,
		Limit:      5,
	})
	require.NoError(t, err)
	assert.True(t, page.Done)
}

func TestRetryAtBacksOffAndGivesUp(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(160), RetryAt(100, 1))
	assert.Equal(t, int64(220), RetryAt(100, 2))
	assert.Equal(t, int64(100+64*60), RetryAt(100, 7))
	assert.Zero(t, RetryAt(100, maxAttempts))
}

func TestDocumentTextEmbeddable(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	assert.True(t, DocumentTextEmbeddable(registry, document.OwnerResource("shipment")))
	assert.True(t, DocumentTextEmbeddable(registry, document.OwnerResource("Customer")))
	assert.False(t, DocumentTextEmbeddable(registry, document.OwnerResource("worker")))
	assert.False(t, DocumentTextEmbeddable(registry, document.OwnerResource("invoice_adjustment")))
	assert.False(t, DocumentTextEmbeddable(registry, document.OwnerResource("user")))
	assert.False(t, DocumentTextEmbeddable(registry, document.OwnerResource("unknown_thing")))
	assert.False(t, DocumentTextEmbeddable(nil, permission.ResourceShipment))
}
