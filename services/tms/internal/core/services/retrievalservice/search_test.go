package retrievalservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeVectorizer struct {
	vector serviceports.QueryVector
	err    error
	texts  []string
}

func (f *fakeVectorizer) Vectorize(
	_ context.Context,
	req serviceports.QueryVectorRequest,
) (serviceports.QueryVector, error) {
	f.texts = append(f.texts, req.Text)

	return f.vector, f.err
}

func (f *fakeVectorizer) Availability(
	context.Context,
	pagination.TenantInfo,
) (airetrieval.Availability, error) {
	return airetrieval.Availability{Available: true}, nil
}

func usableVector() serviceports.QueryVector {
	return serviceports.QueryVector{
		Available:  true,
		Vector:     make([]float32, 768),
		ModelKey:   activeKey,
		Dimensions: 768,
	}
}

type fakeAccess struct {
	unreadable map[permission.Resource]bool
	hidden     map[string]bool
	textHidden map[permission.Resource]bool
	records    []string
}

func (f *fakeAccess) MayReadResource(_ context.Context, resource permission.Resource) bool {
	return !f.unreadable[resource]
}

func (f *fakeAccess) MayReadRecord(
	ctx context.Context,
	resource permission.Resource,
	recordID string,
) bool {
	f.records = append(f.records, resource.String()+":"+recordID)

	return f.MayReadResource(ctx, resource)
}

func (f *fakeAccess) ShowsField(_ context.Context, resource permission.Resource, field string) bool {
	return !f.hidden[resource.String()+"."+field]
}

func (f *fakeAccess) ShowsRecordText(_ context.Context, resource permission.Resource) bool {
	return !f.textHidden[resource]
}

func newTestSearcher(
	repo *fakeRetrievalRepo,
	sources *fakeSources,
	vectorizer serviceports.QueryVectorizer,
) *Searcher {
	return NewSearcher(SearcherParams{
		Logger:     zap.NewNop(),
		Repo:       repo,
		Sources:    sources,
		Registry:   permission.NewRegistry(),
		Vectorizer: vectorizer,
	})
}

func searchRequest(query string, access serviceports.RetrievalAccess) serviceports.RetrievalSearchRequest {
	return serviceports.RetrievalSearchRequest{
		TenantInfo: testTenant,
		Query:      query,
		Limit:      5,
		Access:     access,
	}
}

func TestSearchDocumentsFusesBothLegsAndLabelsTheMatch(t *testing.T) {
	t.Parallel()

	both := documentSource("Lumper fee reimbursed with receipt.", "Detention billed hourly.")
	words := documentSource("Lumper receipt for load 4471.")
	meaning := documentSource("Unloading service paid in cash.", "Capstone crew unloaded 24 pallets.")

	repo := newFakeRetrievalRepo(activeSettings())
	repo.search = []repositories.VectorSearchHit{
		{SourceID: meaning.Document.ID, ChunkIndex: 1, Similarity: 0.82},
		{SourceID: both.Document.ID, ChunkIndex: 0, Similarity: 0.71},
	}
	sources := &fakeSources{
		documents: []*repositories.RetrievalDocumentSource{both, words, meaning},
		keyword: []repositories.RetrievalKeywordHit{
			{SourceID: both.Document.ID, Rank: 0.4},
			{SourceID: words.Document.ID, Rank: 0.3},
		},
	}
	vectorizer := &fakeVectorizer{vector: usableVector()}
	searcher := newTestSearcher(repo, sources, vectorizer)

	result, err := searcher.SearchDocuments(t.Context(), searchRequest("lumper fee", &fakeAccess{}))
	require.NoError(t, err)
	assert.True(t, result.Semantics.Used)
	require.Len(t, result.Hits, 3)

	assert.Equal(t, both.Document.ID, result.Hits[0].Document.ID)
	assert.Equal(t, serviceports.RetrievalMatchBoth, result.Hits[0].Match)
	assert.Equal(t, 1, result.Hits[0].Page)
	assert.Contains(t, result.Hits[0].Snippet, "Lumper fee")

	byID := make(map[pulid.ID]serviceports.DocumentSearchHit, len(result.Hits))
	for _, hit := range result.Hits {
		byID[hit.Document.ID] = hit
	}
	assert.Equal(t, serviceports.RetrievalMatchWords, byID[words.Document.ID].Match)
	assert.Equal(t, serviceports.RetrievalMatchMeaning, byID[meaning.Document.ID].Match)
	assert.Equal(t, 2, byID[meaning.Document.ID].Page, "a meaning match shows the chunk it matched")
	assert.Contains(t, byID[meaning.Document.ID].Snippet, "Capstone")

	require.Len(t, repo.searches, 1)
	assert.Equal(t, []airetrieval.SourceType{airetrieval.SourceTypeDocument},
		repo.searches[0].SourceTypes)
	assert.Equal(t, testTenant, repo.searches[0].TenantInfo)
	assert.Equal(t, []string{"lumper fee"}, vectorizer.texts)
}

func TestSearchDocumentsDropsWhatTheCallerMayNotRead(t *testing.T) {
	t.Parallel()

	shipment := documentSource("Linehaul rate for load 4471.")
	worker := documentSource("Medical card for the driver of load 4471.")
	worker.Document.ResourceType = "worker"
	mystery := documentSource("Load 4471 notes.")
	mystery.Document.ResourceType = "mystery_record"
	rejected := documentSource("Rejected load 4471 paperwork.")
	rejected.Document.Status = "Rejected"

	sources := &fakeSources{
		documents: []*repositories.RetrievalDocumentSource{shipment, worker, mystery, rejected},
		keyword: []repositories.RetrievalKeywordHit{
			{SourceID: worker.Document.ID}, {SourceID: mystery.Document.ID},
			{SourceID: rejected.Document.ID}, {SourceID: shipment.Document.ID},
			{SourceID: pulid.MustNew("doc_")},
		},
	}
	access := &fakeAccess{unreadable: map[permission.Resource]bool{permission.ResourceWorker: true}}
	searcher := newTestSearcher(newFakeRetrievalRepo(activeSettings()), sources, nil)

	result, err := searcher.SearchDocuments(t.Context(), searchRequest("load 4471", access))
	require.NoError(t, err)
	require.Len(t, result.Hits, 1)
	assert.Equal(t, shipment.Document.ID, result.Hits[0].Document.ID)
	assert.Contains(t, access.records, "worker:"+worker.Document.ResourceID)
	assert.Equal(t, airetrieval.UnavailableReasonNoProvider, result.Semantics.Reason)
}

func TestSearchDocumentsBuildsSnippetsOnlyFromVisibleFields(t *testing.T) {
	t.Parallel()

	doc := documentSource("Hourly detention after two hours of free time.")
	sources := &fakeSources{
		documents: []*repositories.RetrievalDocumentSource{doc},
		keyword:   []repositories.RetrievalKeywordHit{{SourceID: doc.Document.ID}},
	}
	searcher := newTestSearcher(newFakeRetrievalRepo(activeSettings()), sources, nil)

	hidden := &fakeAccess{
		hidden:     map[string]bool{"document.originalName": true},
		textHidden: map[permission.Resource]bool{permission.ResourceShipment: true},
	}
	result, err := searcher.SearchDocuments(t.Context(), searchRequest("detention", hidden))
	require.NoError(t, err)
	require.Len(t, result.Hits, 1)
	assert.Empty(t, result.Hits[0].Snippet, "the owning record's text is above the ceiling")
	assert.Zero(t, result.Hits[0].Page)
	assert.False(t, result.Hits[0].ShowsFileName)

	visible, err := searcher.SearchDocuments(t.Context(), searchRequest("detention", &fakeAccess{}))
	require.NoError(t, err)
	assert.Equal(t, "Hourly detention after two hours of free time.", visible.Hits[0].Snippet)
	assert.True(t, visible.Hits[0].ShowsFileName)
}

func TestSearchDocumentsWithoutDocumentAccessFindsNothing(t *testing.T) {
	t.Parallel()

	doc := documentSource("Anything.")
	sources := &fakeSources{
		documents: []*repositories.RetrievalDocumentSource{doc},
		keyword:   []repositories.RetrievalKeywordHit{{SourceID: doc.Document.ID}},
	}
	searcher := newTestSearcher(newFakeRetrievalRepo(activeSettings()), sources, nil)

	result, err := searcher.SearchDocuments(t.Context(), searchRequest("anything",
		&fakeAccess{unreadable: map[permission.Resource]bool{permission.ResourceDocument: true}}))
	require.NoError(t, err)
	assert.Empty(t, result.Hits)

	_, err = searcher.SearchDocuments(t.Context(), searchRequest("anything", nil))
	require.Error(t, err, "a search that cannot tell who is asking is refused")

	_, err = searcher.SearchDocuments(t.Context(), searchRequest("   ", &fakeAccess{}))
	require.Error(t, err)
}

func TestSearchFallsBackToWordsWhenMeaningIsUnavailable(t *testing.T) {
	t.Parallel()

	doc := documentSource("Detention after two hours.")
	sources := &fakeSources{
		documents: []*repositories.RetrievalDocumentSource{doc},
		keyword:   []repositories.RetrievalKeywordHit{{SourceID: doc.Document.ID}},
	}

	cases := []struct {
		name       string
		settings   func(*airetrieval.Settings)
		vectorizer *fakeVectorizer
		searchErr  error
		reason     airetrieval.UnavailableReason
	}{
		{
			name:       "documents turned off",
			settings:   func(s *airetrieval.Settings) { s.DocumentsEnabled = false },
			vectorizer: &fakeVectorizer{vector: usableVector()},
			reason:     airetrieval.UnavailableReasonDisabled,
		},
		{
			name: "the query timed out",
			vectorizer: &fakeVectorizer{vector: serviceports.UnavailableQueryVector(
				airetrieval.UnavailableReasonQueryTimeout)},
			reason: airetrieval.UnavailableReasonQueryTimeout,
		},
		{
			name:       "the vectorizer failed",
			vectorizer: &fakeVectorizer{err: errors.New("boom")},
			reason:     airetrieval.UnavailableReasonProviderFailed,
		},
		{
			name:       "the vector tables are missing",
			vectorizer: &fakeVectorizer{vector: usableVector()},
			searchErr: &airetrieval.UnavailableError{
				Reason: airetrieval.UnavailableReasonSchemaMissing,
			},
			reason: airetrieval.UnavailableReasonSchemaMissing,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			settings := activeSettings()
			if tc.settings != nil {
				tc.settings(settings)
			}
			repo := newFakeRetrievalRepo(settings)
			repo.searchErr = tc.searchErr

			result, err := newTestSearcher(repo, sources, tc.vectorizer).
				SearchDocuments(t.Context(), searchRequest("detention", &fakeAccess{}))
			require.NoError(t, err)
			assert.False(t, result.Semantics.Used)
			assert.Equal(t, tc.reason, result.Semantics.Reason)
			require.Len(t, result.Hits, 1)
			assert.Equal(t, serviceports.RetrievalMatchWords, result.Hits[0].Match)
		})
	}
}

func TestSearchInboundMessagesRespectsFieldVisibility(t *testing.T) {
	t.Parallel()

	message := &inboundmessage.InboundMessage{
		ID:             pulid.MustNew("imsg_"),
		OrganizationID: testTenant.OrgID,
		FromName:       "Dana",
		FromAddress:    "dana@globex.test",
		Subject:        "Driver never showed",
		TextBody:       "Your driver missed pickup.\n> quoted text about lumber",
	}
	sources := &fakeSources{
		messages: []*inboundmessage.InboundMessage{message},
		keyword:  []repositories.RetrievalKeywordHit{{SourceID: message.ID}},
	}
	searcher := newTestSearcher(newFakeRetrievalRepo(activeSettings()), sources, nil)

	result, err := searcher.SearchInboundMessages(t.Context(), searchRequest("pickup", &fakeAccess{}))
	require.NoError(t, err)
	require.Len(t, result.Hits, 1)
	assert.Equal(t, "Driver never showed", result.Hits[0].Subject)
	assert.Equal(t, "Dana <dana@globex.test>", result.Hits[0].From)
	assert.Equal(t, "Your driver missed pickup.", result.Hits[0].Snippet)
	assert.NotContains(t, result.Hits[0].Snippet, "lumber")

	restricted := &fakeAccess{hidden: map[string]bool{
		"inbound_message.subject":     true,
		"inbound_message.fromAddress": true,
		"inbound_message.textBody":    true,
	}}
	hidden, err := searcher.SearchInboundMessages(t.Context(), searchRequest("pickup", restricted))
	require.NoError(t, err)
	require.Len(t, hidden.Hits, 1)
	assert.Empty(t, hidden.Hits[0].Subject)
	assert.Empty(t, hidden.Hits[0].From)
	assert.Empty(t, hidden.Hits[0].Snippet)

	denied, err := searcher.SearchInboundMessages(t.Context(), searchRequest("pickup",
		&fakeAccess{unreadable: map[permission.Resource]bool{
			permission.ResourceInboundMessage: true,
		}}))
	require.NoError(t, err)
	assert.Empty(t, denied.Hits)
}

func TestSnippetCentresOnTheFirstTermAndStaysWithinTheLimit(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("filler ", 200) + "the lumper fee was reimbursed " + strings.Repeat("tail ", 200)
	snippet := Snippet(body, QueryTerms("lumper fee"))

	assert.LessOrEqual(t, len([]rune(snippet)), MaxSnippetChars)
	assert.Contains(t, snippet, "lumper fee")
	assert.True(t, strings.HasPrefix(snippet, "…"))
	assert.True(t, strings.HasSuffix(snippet, "…"))

	assert.Equal(t, "short text", Snippet("short   text", nil))
}

func TestQueryTermsDropStopWordsAndShortWords(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"lumper", "fee", "load", "4471"},
		QueryTerms("What is the lumper fee on load 4471? on the load"))
}

func TestBestChunkPrefersTermsThenTheVectorChunk(t *testing.T) {
	t.Parallel()

	chunks := []Chunk{{Body: "nothing here"}, {Body: "detention detention"}, {Body: "other"}}
	idx, ok := BestChunk(chunks, []string{"detention"}, 2)
	require.True(t, ok)
	assert.Equal(t, 1, idx)

	idx, ok = BestChunk(chunks, []string{"absent"}, 2)
	require.True(t, ok)
	assert.Equal(t, 2, idx)

	_, ok = BestChunk([]Chunk{{Body: ""}}, nil, -1)
	assert.False(t, ok)
}

func TestSimilarMemoriesEmbedsTheTextUnlessGivenAVector(t *testing.T) {
	t.Parallel()

	repo := newFakeRetrievalRepo(activeSettings())
	id := pulid.MustNew("amem_")
	repo.search = []repositories.VectorSearchHit{{SourceID: id, Similarity: 0.7}}
	vectorizer := &fakeVectorizer{vector: usableVector()}
	searcher := newTestSearcher(repo, &fakeSources{}, vectorizer)

	similar, err := searcher.SimilarMemories(t.Context(), serviceports.SimilarMemoriesRequest{
		TenantInfo: testTenant,
		Text:       "free time at Globex",
	})
	require.NoError(t, err)
	assert.True(t, similar.Semantics.Used)
	assert.Equal(t, []serviceports.MemorySimilarity{{MemoryID: id, Similarity: 0.7}},
		similar.Memories)
	assert.Equal(t, DefaultSimilarityFloor, similar.Floor)
	assert.Equal(t, []airetrieval.SourceType{airetrieval.SourceTypeMemory},
		repo.searches[0].SourceTypes)

	_, err = searcher.SimilarMemories(t.Context(), serviceports.SimilarMemoriesRequest{
		TenantInfo: testTenant,
		Query:      usableVector(),
	})
	require.NoError(t, err)
	assert.Len(t, vectorizer.texts, 1, "a turn's vector is reused, not embedded again")

	settings := activeSettings()
	settings.MemoryEnabled = false
	off, err := newTestSearcher(newFakeRetrievalRepo(settings), &fakeSources{}, vectorizer).
		SimilarMemories(t.Context(), serviceports.SimilarMemoriesRequest{
			TenantInfo: testTenant,
			Text:       "anything",
		})
	require.NoError(t, err)
	assert.Equal(t, airetrieval.UnavailableReasonDisabled, off.Semantics.Reason)
}

func memoriesByAge(count int) []*agent.Memory {
	memories := make([]*agent.Memory, 0, count)
	for idx := range count {
		memories = append(memories, &agent.Memory{
			ID:        pulid.MustNew("amem_"),
			Kind:      agent.MemoryKindFact,
			Content:   "memory",
			CreatedAt: int64(1_000 + idx*100),
			UpdatedAt: int64(1_000 + idx*100),
		})
	}

	return memories
}

func TestMemoryRankerLiftsSimilarMemoriesAboveRecency(t *testing.T) {
	t.Parallel()

	memories := memoriesByAge(4)
	recency := recencyOrder(t, memories)
	require.Equal(t, memories[3].ID, recency[0].ID)

	repo := newFakeRetrievalRepo(activeSettings())
	repo.search = []repositories.VectorSearchHit{
		{SourceID: memories[0].ID, Similarity: 0.91},
		{SourceID: memories[1].ID, Similarity: 0.2},
	}
	ranker := NewMemoryRanker(newTestSearcher(repo, &fakeSources{}, nil))

	ranked, err := ranker.RankMemories(t.Context(), serviceports.RankMemoriesRequest{
		TenantInfo: testTenant,
		Now:        2_000,
		Memories:   memories,
		Query:      usableVector(),
	})
	require.NoError(t, err)
	assert.Equal(t, memories[0].ID, ranked[0].ID, "the most similar memory leads")
	assert.Len(t, ranked, 4)

	unranked, err := ranker.RankMemories(t.Context(), serviceports.RankMemoriesRequest{
		TenantInfo: testTenant,
		Now:        2_000,
		Memories:   memories,
	})
	require.NoError(t, err)
	assert.Equal(t, memories[3].ID, unranked[0].ID, "without a vector the order is recency")
}

func recencyOrder(t *testing.T, memories []*agent.Memory) []*agent.Memory {
	t.Helper()

	ranked, err := NewMemoryRankerFrom(nil, DefaultSearchTuning(), zap.NewNop()).
		RankMemories(t.Context(), serviceports.RankMemoriesRequest{
			Now:      2_000,
			Memories: memories,
		})
	require.NoError(t, err)

	return ranked
}
