package retrievalservice

import (
	"context"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

var testTenant = pagination.TenantInfo{OrgID: pulid.ID("org_test"), BuID: pulid.ID("bu_test")}

const (
	activeKey  = "api.example.com/embed-small@768"
	pendingKey = "api.example.com/embed-large@1024"
)

type fakeRetrievalRepo struct {
	retrievalports.AIRetrievalRepository

	mu           sync.Mutex
	availability airetrieval.Availability
	settings     *airetrieval.Settings
	updated      []*airetrieval.Settings
	paused       []retrievalports.SetAIRetrievalPausedRequest
	marked       []retrievalports.MarkAIRetrievalStaleRequest
	claim        []*airetrieval.IndexEntry
	claims       []retrievalports.ClaimIndexEntriesRequest
	hashes       map[pulid.ID][]retrievalports.EmbeddingChunkHash
	replaced     []retrievalports.ReplaceEmbeddingChunksRequest
	deleted      []retrievalports.AIRetrievalSourceRef
	indexed      []retrievalports.IndexEntryOutcome
	skipped      []retrievalports.IndexEntryOutcome
	failed       []retrievalports.IndexEntryOutcome
	stale        map[string][]pulid.ID
	counts       []retrievalports.IndexEntryCount
	swapped      []retrievalports.SwapAIRetrievalModelRequest
	purges       []retrievalports.PurgeAIRetrievalModelResult
	purged       int
	search       []retrievalports.VectorSearchHit
	searchErr    error
	searches     []retrievalports.VectorSearchRequest
}

func newFakeRetrievalRepo(settings *airetrieval.Settings) *fakeRetrievalRepo {
	return &fakeRetrievalRepo{
		availability: airetrieval.Availability{Available: true},
		settings:     settings,
		hashes:       map[pulid.ID][]retrievalports.EmbeddingChunkHash{},
		stale:        map[string][]pulid.ID{},
	}
}

func (f *fakeRetrievalRepo) VectorAvailability(context.Context) (airetrieval.Availability, error) {
	return f.availability, nil
}

func (f *fakeRetrievalRepo) GetSettings(
	context.Context,
	pagination.TenantInfo,
) (*airetrieval.Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	copied := *f.settings

	return &copied, nil
}

func (f *fakeRetrievalRepo) UpdateSettings(
	_ context.Context,
	entity *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	copied := *entity
	f.settings = &copied
	f.updated = append(f.updated, &copied)

	return entity, nil
}

func (f *fakeRetrievalRepo) SetPaused(
	_ context.Context,
	req *retrievalports.SetAIRetrievalPausedRequest,
) (*airetrieval.Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.paused = append(f.paused, *req)
	f.settings.Paused = req.Paused
	f.settings.PausedReason = req.Reason

	copied := *f.settings

	return &copied, nil
}

func (f *fakeRetrievalRepo) MarkStale(
	_ context.Context,
	req *retrievalports.MarkAIRetrievalStaleRequest,
) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.marked = append(f.marked, *req)

	return len(req.SourceIDs) * len(req.ModelKeys), nil
}

func (f *fakeRetrievalRepo) ClaimIndexEntries(
	_ context.Context,
	req *retrievalports.ClaimIndexEntriesRequest,
) ([]*airetrieval.IndexEntry, error) {
	f.claims = append(f.claims, *req)
	claimed := f.claim
	f.claim = nil

	return claimed, nil
}

func (f *fakeRetrievalRepo) ListChunkHashes(
	_ context.Context,
	req *retrievalports.ListEmbeddingChunkHashesRequest,
) ([]retrievalports.EmbeddingChunkHash, error) {
	return f.hashes[req.Source.SourceID], nil
}

func (f *fakeRetrievalRepo) ReplaceChunks(
	_ context.Context,
	req *retrievalports.ReplaceEmbeddingChunksRequest,
) (retrievalports.ReplaceEmbeddingChunksResult, error) {
	f.replaced = append(f.replaced, *req)

	return retrievalports.ReplaceEmbeddingChunksResult{Written: len(req.Chunks)}, nil
}

func (f *fakeRetrievalRepo) DeleteSource(
	_ context.Context,
	source *retrievalports.AIRetrievalSourceRef,
) (retrievalports.DeleteAIRetrievalSourceResult, error) {
	f.deleted = append(f.deleted, *source)

	return retrievalports.DeleteAIRetrievalSourceResult{IndexEntries: 1}, nil
}

func (f *fakeRetrievalRepo) MarkIndexed(
	_ context.Context,
	req retrievalports.MarkIndexEntriesRequest,
) (retrievalports.MarkIndexEntriesResult, error) {
	f.indexed = append(f.indexed, req.Outcomes...)

	return retrievalports.MarkIndexEntriesResult{Applied: len(req.Outcomes)}, nil
}

func (f *fakeRetrievalRepo) MarkSkipped(
	_ context.Context,
	req retrievalports.MarkIndexEntriesRequest,
) (retrievalports.MarkIndexEntriesResult, error) {
	f.skipped = append(f.skipped, req.Outcomes...)

	return retrievalports.MarkIndexEntriesResult{Applied: len(req.Outcomes)}, nil
}

func (f *fakeRetrievalRepo) MarkFailed(
	_ context.Context,
	req retrievalports.MarkIndexEntriesRequest,
) (retrievalports.MarkIndexEntriesResult, error) {
	f.failed = append(f.failed, req.Outcomes...)

	return retrievalports.MarkIndexEntriesResult{Applied: len(req.Outcomes)}, nil
}

func (f *fakeRetrievalRepo) FindStaleSources(
	_ context.Context,
	req *retrievalports.FindStaleAIRetrievalSourcesRequest,
) ([]pulid.ID, error) {
	key := string(req.SourceType) + "|" + req.ModelKey
	ids := f.stale[key]
	delete(f.stale, key)

	return ids, nil
}

func (f *fakeRetrievalRepo) CountIndexEntries(
	context.Context,
	retrievalports.CountIndexEntriesRequest,
) ([]retrievalports.IndexEntryCount, error) {
	return f.counts, nil
}

func (f *fakeRetrievalRepo) SwapModel(
	_ context.Context,
	req retrievalports.SwapAIRetrievalModelRequest,
) (*retrievalports.SwapAIRetrievalModelResult, error) {
	f.swapped = append(f.swapped, req)
	retired := f.settings.ActiveModelKey
	f.settings.ActiveModelKey = f.settings.PendingModelKey
	f.settings.Dimensions = f.settings.PendingDimensions
	f.settings.PendingModelKey = ""
	f.settings.PendingDimensions = 0

	copied := *f.settings

	return &retrievalports.SwapAIRetrievalModelResult{
		Settings:        &copied,
		RetiredModelKey: retired,
	}, nil
}

func (f *fakeRetrievalRepo) ListIndexedTenants(
	context.Context,
	retrievalports.ListIndexedRetrievalTenantsRequest,
) ([]pagination.TenantInfo, error) {
	if f.settings == nil || !f.settings.HasActiveModel() {
		return []pagination.TenantInfo{}, nil
	}

	return []pagination.TenantInfo{{
		OrgID: f.settings.OrganizationID,
		BuID:  f.settings.BusinessUnitID,
	}}, nil
}

func (f *fakeRetrievalRepo) PurgeModel(
	context.Context,
	retrievalports.PurgeAIRetrievalModelRequest,
) (retrievalports.PurgeAIRetrievalModelResult, error) {
	f.purged++
	if len(f.purges) == 0 {
		return retrievalports.PurgeAIRetrievalModelResult{}, nil
	}
	next := f.purges[0]
	f.purges = f.purges[1:]

	return next, nil
}

func (f *fakeRetrievalRepo) Search(
	_ context.Context,
	req *retrievalports.VectorSearchRequest,
) ([]retrievalports.VectorSearchHit, error) {
	f.searches = append(f.searches, *req)

	return f.search, f.searchErr
}

type fakeSources struct {
	retrievalports.RetrievalSourceRepository

	memories  []*agentmemory.Memory
	documents []*retrievalports.RetrievalDocumentSource
	messages  []*inboundmessage.InboundMessage
	keys      []string
	ids       []pulid.ID
	keyword   []retrievalports.RetrievalKeywordHit
	listed    []retrievalports.ListRetrievalSourceIDsRequest
	searched  []retrievalports.RetrievalKeywordSearchRequest
}

func (f *fakeSources) ListModelKeys(
	context.Context,
	retrievalports.ListRetrievalModelKeysRequest,
) ([]string, error) {
	return f.keys, nil
}

func (f *fakeSources) GetMemories(
	_ context.Context,
	req retrievalports.RetrievalSourcesRequest,
) ([]*agentmemory.Memory, error) {
	return slices.DeleteFunc(slices.Clone(f.memories), func(memory *agentmemory.Memory) bool {
		return !slices.Contains(req.IDs, memory.ID)
	}), nil
}

func (f *fakeSources) GetDocuments(
	_ context.Context,
	req *retrievalports.RetrievalDocumentsRequest,
) ([]*retrievalports.RetrievalDocumentSource, error) {
	return slices.DeleteFunc(slices.Clone(f.documents),
		func(source *retrievalports.RetrievalDocumentSource) bool {
			return !slices.Contains(req.IDs, source.Document.ID)
		}), nil
}

func (f *fakeSources) GetInboundMessages(
	_ context.Context,
	req retrievalports.RetrievalSourcesRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return slices.DeleteFunc(slices.Clone(f.messages),
		func(message *inboundmessage.InboundMessage) bool {
			return !slices.Contains(req.IDs, message.ID)
		}), nil
}

func (f *fakeSources) ListSourceIDs(
	_ context.Context,
	req *retrievalports.ListRetrievalSourceIDsRequest,
) ([]pulid.ID, error) {
	f.listed = append(f.listed, *req)

	return f.ids, nil
}

func (f *fakeSources) SearchDocuments(
	context.Context,
	*retrievalports.RetrievalKeywordSearchRequest,
) ([]retrievalports.RetrievalKeywordHit, error) {
	return f.keyword, nil
}

func (f *fakeSources) SearchInboundMessages(
	_ context.Context,
	req *retrievalports.RetrievalKeywordSearchRequest,
) ([]retrievalports.RetrievalKeywordHit, error) {
	f.searched = append(f.searched, *req)

	return f.keyword, nil
}

type fakeUsage struct {
	aiusageports.AIUsageRepository

	spent decimal.Decimal
}

func (f *fakeUsage) SurfaceCost(
	context.Context,
	aiusageports.AIUsageSurfaceCostRequest,
) (*aiusageports.AIUsageCost, error) {
	return &aiusageports.AIUsageCost{CostUSD: f.spent}, nil
}

type fakeEmbeddings struct {
	configured string
	err        error
	embedErr   error
	cost       decimal.Decimal
	requests   []aiusageports.EmbedRequest
}

func (f *fakeEmbeddings) ConfiguredModelKey(
	context.Context,
	pagination.TenantInfo,
) (string, error) {
	return f.configured, f.err
}

func (f *fakeEmbeddings) Embed(
	_ context.Context,
	req *aiusageports.EmbedRequest,
) (aiusageports.EmbedResult, error) {
	f.requests = append(f.requests, *req)
	if f.embedErr != nil {
		return aiusageports.EmbedResult{}, f.embedErr
	}

	dimensions, _ := airetrieval.ModelKeyDimensions(req.ModelKey)
	vectors := make([][]float32, len(req.Inputs))
	for idx := range vectors {
		vectors[idx] = make([]float32, dimensions)
		vectors[idx][idx%dimensions] = 1
	}
	cost := f.cost

	return aiusageports.EmbedResult{Vectors: vectors, CostUSD: &cost, Dimensions: dimensions}, nil
}

func (f *fakeEmbeddings) inputs() []string {
	inputs := make([]string, 0, len(f.requests))
	for _, req := range f.requests {
		inputs = append(inputs, req.Inputs...)
	}

	return inputs
}

type fakeSignals struct {
	err     error
	signals []string
}

func (f *fakeSignals) SignalWithStartWorkflow(
	_ context.Context,
	workflowID, _ string,
	_ any,
	_ client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	f.signals = append(f.signals, workflowID)

	return nil, f.err
}

type testService struct {
	*Service

	repo       *fakeRetrievalRepo
	sources    *fakeSources
	usage      *fakeUsage
	embeddings *fakeEmbeddings
	signals    *fakeSignals
}

func newTestService(settings *airetrieval.Settings) *testService {
	repo := newFakeRetrievalRepo(settings)
	sources := &fakeSources{}
	usage := &fakeUsage{spent: decimal.Zero}
	embeddings := &fakeEmbeddings{configured: activeKey, cost: decimal.RequireFromString("0.01")}
	signals := &fakeSignals{}

	service := New(Params{
		Logger:     zap.NewNop(),
		Repo:       repo,
		Sources:    sources,
		Usage:      usage,
		Registry:   permissioncatalog.NewRegistry(),
		Embeddings: embeddings,
		Signals:    signals,
	})
	service.now = func() int64 { return 1_800_000_000 }

	return &testService{
		Service:    service,
		repo:       repo,
		sources:    sources,
		usage:      usage,
		embeddings: embeddings,
		signals:    signals,
	}
}

func activeSettings() *airetrieval.Settings {
	settings := airetrieval.DefaultSettings(testTenant.OrgID, testTenant.BuID)
	settings.ActiveModelKey = activeKey
	settings.Dimensions = airetrieval.Dimensions768

	return settings
}

func entry(sourceType airetrieval.SourceType, id pulid.ID) *airetrieval.IndexEntry {
	return &airetrieval.IndexEntry{
		OrganizationID: testTenant.OrgID,
		BusinessUnitID: testTenant.BuID,
		SourceType:     sourceType,
		SourceID:       id,
		ModelKey:       activeKey,
		Status:         airetrieval.IndexStatusPending,
		Generation:     3,
		Attempts:       1,
	}
}
