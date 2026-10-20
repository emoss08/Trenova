package retrievalservice

import (
	"context"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
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
	agentports.AIRetrievalRepository

	mu           sync.Mutex
	availability airetrieval.Availability
	settings     *airetrieval.Settings
	updated      []*airetrieval.Settings
	paused       []agentports.SetAIRetrievalPausedRequest
	marked       []agentports.MarkAIRetrievalStaleRequest
	claim        []*airetrieval.IndexEntry
	claims       []agentports.ClaimIndexEntriesRequest
	hashes       map[pulid.ID][]agentports.EmbeddingChunkHash
	replaced     []agentports.ReplaceEmbeddingChunksRequest
	deleted      []agentports.AIRetrievalSourceRef
	indexed      []agentports.IndexEntryOutcome
	skipped      []agentports.IndexEntryOutcome
	failed       []agentports.IndexEntryOutcome
	stale        map[string][]pulid.ID
	counts       []agentports.IndexEntryCount
	swapped      []agentports.SwapAIRetrievalModelRequest
	purges       []agentports.PurgeAIRetrievalModelResult
	purged       int
	search       []agentports.VectorSearchHit
	searchErr    error
	searches     []agentports.VectorSearchRequest
}

func newFakeRetrievalRepo(settings *airetrieval.Settings) *fakeRetrievalRepo {
	return &fakeRetrievalRepo{
		availability: airetrieval.Availability{Available: true},
		settings:     settings,
		hashes:       map[pulid.ID][]agentports.EmbeddingChunkHash{},
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
	req *agentports.SetAIRetrievalPausedRequest,
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
	req *agentports.MarkAIRetrievalStaleRequest,
) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.marked = append(f.marked, *req)

	return len(req.SourceIDs) * len(req.ModelKeys), nil
}

func (f *fakeRetrievalRepo) ClaimIndexEntries(
	_ context.Context,
	req *agentports.ClaimIndexEntriesRequest,
) ([]*airetrieval.IndexEntry, error) {
	f.claims = append(f.claims, *req)
	claimed := f.claim
	f.claim = nil

	return claimed, nil
}

func (f *fakeRetrievalRepo) ListChunkHashes(
	_ context.Context,
	req *agentports.ListEmbeddingChunkHashesRequest,
) ([]agentports.EmbeddingChunkHash, error) {
	return f.hashes[req.Source.SourceID], nil
}

func (f *fakeRetrievalRepo) ReplaceChunks(
	_ context.Context,
	req *agentports.ReplaceEmbeddingChunksRequest,
) (agentports.ReplaceEmbeddingChunksResult, error) {
	f.replaced = append(f.replaced, *req)

	return agentports.ReplaceEmbeddingChunksResult{Written: len(req.Chunks)}, nil
}

func (f *fakeRetrievalRepo) DeleteSource(
	_ context.Context,
	source *agentports.AIRetrievalSourceRef,
) (agentports.DeleteAIRetrievalSourceResult, error) {
	f.deleted = append(f.deleted, *source)

	return agentports.DeleteAIRetrievalSourceResult{IndexEntries: 1}, nil
}

func (f *fakeRetrievalRepo) MarkIndexed(
	_ context.Context,
	req agentports.MarkIndexEntriesRequest,
) (agentports.MarkIndexEntriesResult, error) {
	f.indexed = append(f.indexed, req.Outcomes...)

	return agentports.MarkIndexEntriesResult{Applied: len(req.Outcomes)}, nil
}

func (f *fakeRetrievalRepo) MarkSkipped(
	_ context.Context,
	req agentports.MarkIndexEntriesRequest,
) (agentports.MarkIndexEntriesResult, error) {
	f.skipped = append(f.skipped, req.Outcomes...)

	return agentports.MarkIndexEntriesResult{Applied: len(req.Outcomes)}, nil
}

func (f *fakeRetrievalRepo) MarkFailed(
	_ context.Context,
	req agentports.MarkIndexEntriesRequest,
) (agentports.MarkIndexEntriesResult, error) {
	f.failed = append(f.failed, req.Outcomes...)

	return agentports.MarkIndexEntriesResult{Applied: len(req.Outcomes)}, nil
}

func (f *fakeRetrievalRepo) FindStaleSources(
	_ context.Context,
	req *agentports.FindStaleAIRetrievalSourcesRequest,
) ([]pulid.ID, error) {
	key := string(req.SourceType) + "|" + req.ModelKey
	ids := f.stale[key]
	delete(f.stale, key)

	return ids, nil
}

func (f *fakeRetrievalRepo) CountIndexEntries(
	context.Context,
	agentports.CountIndexEntriesRequest,
) ([]agentports.IndexEntryCount, error) {
	return f.counts, nil
}

func (f *fakeRetrievalRepo) SwapModel(
	_ context.Context,
	req agentports.SwapAIRetrievalModelRequest,
) (*agentports.SwapAIRetrievalModelResult, error) {
	f.swapped = append(f.swapped, req)
	retired := f.settings.ActiveModelKey
	f.settings.ActiveModelKey = f.settings.PendingModelKey
	f.settings.Dimensions = f.settings.PendingDimensions
	f.settings.PendingModelKey = ""
	f.settings.PendingDimensions = 0

	copied := *f.settings

	return &agentports.SwapAIRetrievalModelResult{
		Settings:        &copied,
		RetiredModelKey: retired,
	}, nil
}

func (f *fakeRetrievalRepo) ListIndexedTenants(
	context.Context,
	agentports.ListIndexedRetrievalTenantsRequest,
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
	agentports.PurgeAIRetrievalModelRequest,
) (agentports.PurgeAIRetrievalModelResult, error) {
	f.purged++
	if len(f.purges) == 0 {
		return agentports.PurgeAIRetrievalModelResult{}, nil
	}
	next := f.purges[0]
	f.purges = f.purges[1:]

	return next, nil
}

func (f *fakeRetrievalRepo) Search(
	_ context.Context,
	req *agentports.VectorSearchRequest,
) ([]agentports.VectorSearchHit, error) {
	f.searches = append(f.searches, *req)

	return f.search, f.searchErr
}

type fakeSources struct {
	agentports.RetrievalSourceRepository

	memories  []*agent.Memory
	documents []*agentports.RetrievalDocumentSource
	messages  []*inboundmessage.InboundMessage
	keys      []string
	ids       []pulid.ID
	keyword   []agentports.RetrievalKeywordHit
	listed    []agentports.ListRetrievalSourceIDsRequest
	searched  []agentports.RetrievalKeywordSearchRequest
}

func (f *fakeSources) ListModelKeys(
	context.Context,
	agentports.ListRetrievalModelKeysRequest,
) ([]string, error) {
	return f.keys, nil
}

func (f *fakeSources) GetMemories(
	_ context.Context,
	req agentports.RetrievalSourcesRequest,
) ([]*agent.Memory, error) {
	return slices.DeleteFunc(slices.Clone(f.memories), func(memory *agent.Memory) bool {
		return !slices.Contains(req.IDs, memory.ID)
	}), nil
}

func (f *fakeSources) GetDocuments(
	_ context.Context,
	req *agentports.RetrievalDocumentsRequest,
) ([]*agentports.RetrievalDocumentSource, error) {
	return slices.DeleteFunc(slices.Clone(f.documents),
		func(source *agentports.RetrievalDocumentSource) bool {
			return !slices.Contains(req.IDs, source.Document.ID)
		}), nil
}

func (f *fakeSources) GetInboundMessages(
	_ context.Context,
	req agentports.RetrievalSourcesRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return slices.DeleteFunc(slices.Clone(f.messages),
		func(message *inboundmessage.InboundMessage) bool {
			return !slices.Contains(req.IDs, message.ID)
		}), nil
}

func (f *fakeSources) ListSourceIDs(
	_ context.Context,
	req *agentports.ListRetrievalSourceIDsRequest,
) ([]pulid.ID, error) {
	f.listed = append(f.listed, *req)

	return f.ids, nil
}

func (f *fakeSources) SearchDocuments(
	context.Context,
	agentports.RetrievalKeywordSearchRequest,
) ([]agentports.RetrievalKeywordHit, error) {
	return f.keyword, nil
}

func (f *fakeSources) SearchInboundMessages(
	_ context.Context,
	req agentports.RetrievalKeywordSearchRequest,
) ([]agentports.RetrievalKeywordHit, error) {
	f.searched = append(f.searched, req)

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
		Registry:   permission.NewRegistry(),
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
