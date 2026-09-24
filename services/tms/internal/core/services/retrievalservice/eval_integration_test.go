//go:build integration

package retrievalservice_test

import (
	"context"
	"errors"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/emoss08/trenova/internal/core/services/agentmemoryservice"
	"github.com/emoss08/trenova/internal/core/services/retrievalservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/agentmemoryrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/airetrievalrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/retrievalsourcerepository"
	"github.com/emoss08/trenova/internal/testutil/retrievaltest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

var update = flag.Bool("update", false, "pin the retrieval eval floors to what this run measures")

const (
	evalLimit          = 10
	requireFloorsEnv   = "TRENOVA_EVAL_REQUIRE_FLOORS"
	requireHybridEnv   = agentevalgate.RequireHybridEnv
	expectVectorEnv    = "TRENOVA_TEST_EXPECT_VECTOR"
	secondTenantSuffix = "RX"
)

type evalSuites struct {
	documents *agentevalgate.RetrievalSuite
	inbox     *agentevalgate.RetrievalSuite
	memories  *agentevalgate.RetrievalSuite
}

type evalHarness struct {
	ctx      context.Context
	db       *bun.DB
	conn     *postgres.Connection
	repo     repositories.AIRetrievalRepository
	sources  repositories.RetrievalSourceRepository
	memories repositories.AgentMemoryRepository
	registry *permission.Registry
	data     *seedtest.TestData
	tenant   pagination.TenantInfo
}

func newEvalHarness(t *testing.T) (*evalHarness, evalSuites) {
	t.Helper()

	documents, inbox, memories := loadRetrievalSuites(t)

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	h := &evalHarness{
		ctx:  ctx,
		db:   db,
		conn: conn,
		repo: airetrievalrepository.New(airetrievalrepository.Params{
			DB:     conn,
			Probe:  postgres.NewTestCapabilityProbe(conn),
			Logger: zap.NewNop(),
		}),
		sources: retrievalsourcerepository.New(retrievalsourcerepository.Params{
			DB:     conn,
			Logger: zap.NewNop(),
		}),
		memories: agentmemoryrepository.New(agentmemoryrepository.Params{
			DB:     conn,
			Logger: zap.NewNop(),
		}),
		registry: permission.NewRegistry(),
		data:     data,
		tenant:   pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID},
	}

	return h, evalSuites{documents: documents, inbox: inbox, memories: memories}
}

func (h *evalHarness) searcher(vectorizer serviceports.QueryVectorizer) *retrievalservice.Searcher {
	return retrievalservice.NewSearcher(retrievalservice.SearcherParams{
		Logger:     zap.NewNop(),
		Repo:       h.repo,
		Sources:    h.sources,
		Registry:   h.registry,
		Vectorizer: vectorizer,
	})
}

func (h *evalHarness) memoryService(
	vectors serviceports.MemoryVectorSearcher,
) serviceports.AgentMemoryService {
	return agentmemoryservice.New(agentmemoryservice.Params{
		Logger:  zap.NewNop(),
		Repo:    h.memories,
		Vectors: vectors,
	})
}

func (h *evalHarness) vectorReady(t *testing.T) bool {
	t.Helper()

	availability, err := h.repo.VectorAvailability(h.ctx)
	require.NoError(t, err)
	if availability.Available {
		return true
	}
	if os.Getenv(expectVectorEnv) == "true" {
		t.Fatalf("pgvector is expected on this server but reports %s", availability.Reason)
	}

	return false
}

type seeded struct {
	documents *retrievaltest.Corpus
	inbox     *retrievaltest.Corpus
	memories  *retrievaltest.Corpus
}

func (h *evalHarness) seed(
	t *testing.T,
	tenant pagination.TenantInfo,
	userID pulid.ID,
	suites evalSuites,
) seeded {
	t.Helper()

	mailbox := retrievaltest.SeedMailbox(t, h.ctx, h.db, tenant)

	return seeded{
		documents: retrievaltest.SeedDocuments(t, h.ctx, h.db, retrievaltest.DocumentSeed{
			Tenant:       tenant,
			UploadedByID: userID,
			Items:        suites.documents.Corpus,
		}),
		inbox: retrievaltest.SeedMessages(t, h.ctx, h.db, retrievaltest.MessageSeed{
			Tenant:    tenant,
			MailboxID: mailbox,
			Items:     suites.inbox.Corpus,
		}),
		memories: retrievaltest.SeedMemories(t, h.ctx, h.db, retrievaltest.MemorySeed{
			Tenant: tenant,
			Items:  suites.memories.Corpus,
		}),
	}
}

func (h *evalHarness) activateModel(t *testing.T, tenant pagination.TenantInfo) {
	t.Helper()

	settings := airetrieval.DefaultSettings(tenant.OrgID, tenant.BuID)
	settings.ActiveModelKey = retrievaltest.ModelKey
	settings.Dimensions = airetrieval.Dimensions768
	_, err := h.repo.UpdateSettings(h.ctx, settings)
	require.NoError(t, err)
}

func (h *evalHarness) embed(
	t *testing.T,
	fixture *agentevalgate.EmbeddingFixture,
	corpora seeded,
) {
	t.Helper()

	for sourceType, corpus := range map[airetrieval.SourceType]*retrievaltest.Corpus{
		airetrieval.SourceTypeDocument:       corpora.documents,
		airetrieval.SourceTypeInboundMessage: corpora.inbox,
		airetrieval.SourceTypeMemory:         corpora.memories,
	} {
		retrievaltest.SeedEmbeddings(t, h.ctx, retrievaltest.EmbedSeed{
			Repo:       h.repo,
			Sources:    h.sources,
			Fixture:    fixture,
			ModelKey:   retrievaltest.ModelKey,
			SourceType: sourceType,
			Corpus:     corpus,
		})
	}
}

func loadFixture(t *testing.T) (*agentevalgate.EmbeddingFixture, bool) {
	t.Helper()

	fixture, err := agentevalgate.LoadEmbeddingFixture(retrievaltest.FixturePath)
	if errors.Is(err, agentevalgate.ErrEmbeddingFixtureMissing) {
		return nil, false
	}
	require.NoError(t, err)

	return fixture, true
}

func access(registry *permission.Registry) retrievaltest.Access {
	return retrievaltest.Access{Registry: registry}
}

type retrievers struct {
	documents agentevalgate.Retriever
	inbox     agentevalgate.Retriever
	memories  agentevalgate.Retriever
}

func (h *evalHarness) retrievers(
	searcher *retrievalservice.Searcher,
	memories serviceports.AgentMemoryService,
	corpora seeded,
) retrievers {
	request := func(query string) serviceports.RetrievalSearchRequest {
		return serviceports.RetrievalSearchRequest{
			TenantInfo: h.tenant,
			Query:      query,
			Limit:      evalLimit,
			Access:     access(h.registry),
		}
	}

	return retrievers{
		documents: func(query string) ([]string, error) {
			result, err := searcher.SearchDocuments(h.ctx, request(query))
			if err != nil {
				return nil, err
			}
			ids := make([]pulid.ID, 0, len(result.Hits))
			for _, hit := range result.Hits {
				ids = append(ids, hit.Document.ID)
			}
			return corpora.documents.KeysOf(ids), nil
		},
		inbox: func(query string) ([]string, error) {
			result, err := searcher.SearchInboundMessages(h.ctx, request(query))
			if err != nil {
				return nil, err
			}
			ids := make([]pulid.ID, 0, len(result.Hits))
			for _, hit := range result.Hits {
				ids = append(ids, hit.Message.ID)
			}
			return corpora.inbox.KeysOf(ids), nil
		},
		memories: func(query string) ([]string, error) {
			recalled, err := memories.Recall(h.ctx, serviceports.RecallAgentMemoriesRequest{
				TenantInfo: h.tenant,
				Query:      query,
				Limit:      evalLimit,
			})
			if err != nil {
				return nil, err
			}
			ids := make([]pulid.ID, 0, len(recalled))
			for _, memory := range recalled {
				ids = append(ids, memory.Memory.ID)
			}
			return corpora.memories.KeysOf(ids), nil
		},
	}
}

type leg struct {
	label   string
	suite   *agentevalgate.RetrievalSuite
	floors  string
	keyword agentevalgate.Retriever
	hybrid  agentevalgate.Retriever
}

func TestRetrievalAgainstFloors(t *testing.T) {
	h, suites := newEvalHarness(t)
	corpora := h.seed(t, h.tenant, h.data.User.ID, suites)

	keyword := h.retrievers(h.searcher(nil), h.memoryService(nil), corpora)

	var hybrid *retrievers
	fixture, recorded := loadFixture(t)
	switch {
	case recorded && h.vectorReady(t):
		h.activateModel(t, h.tenant)
		h.embed(t, fixture, corpora)
		searcher := h.searcher(retrievaltest.FixtureVectorizer{
			Fixture:  fixture,
			ModelKey: retrievaltest.ModelKey,
		})
		found := h.retrievers(searcher, h.memoryService(searcher), corpora)
		hybrid = &found
	case strings.EqualFold(os.Getenv(requireHybridEnv), "true"):
		t.Fatalf("the hybrid retrieval gate needs pgvector and %s; record it with:\n  %s",
			retrievaltest.FixturePath, retrievaltest.RecordCommand)
	default:
		t.Logf("hybrid retrieval not measured: it needs pgvector and %s (record it with: %s)",
			retrievaltest.FixturePath, retrievaltest.RecordCommand)
	}

	legs := []leg{
		{label: "search_documents", suite: suites.documents, floors: retrievaltest.DocumentFloors,
			keyword: keyword.documents},
		{label: "search_inbound_messages", suite: suites.inbox, floors: retrievaltest.InboxFloors,
			keyword: keyword.inbox},
		{label: "recall_memory", suite: suites.memories, floors: retrievaltest.MemoryFloors,
			keyword: keyword.memories},
	}
	if hybrid != nil {
		legs[0].hybrid = hybrid.documents
		legs[1].hybrid = hybrid.inbox
		legs[2].hybrid = hybrid.memories
	}

	for _, current := range legs {
		t.Run(current.label, func(t *testing.T) {
			measureLeg(t, current)
		})
	}
}

func measureLeg(t *testing.T, current leg) {
	t.Helper()

	keyword, err := agentevalgate.EvaluateRetrieval(current.suite, current.keyword)
	require.NoError(t, err)
	logReport(t, current.label+" by keyword", keyword)

	var hybrid *agentevalgate.RetrievalReport
	if current.hybrid != nil {
		report, hybridErr := agentevalgate.EvaluateRetrieval(current.suite, current.hybrid)
		require.NoError(t, hybridErr)
		logReport(t, current.label+" by keyword and meaning", report)
		hybrid = &report
	}

	floors := readRetrievalFloors(t, current.floors)
	if *update {
		keywordFloors := keyword.Floors()
		floors.Keyword = &keywordFloors
		if hybrid != nil {
			hybridFloors := hybrid.Floors()
			floors.Hybrid = &hybridFloors
		}
		encoded, encodeErr := agentevalgate.MarshalJSON(floors)
		require.NoError(t, encodeErr)
		require.NoError(t, agentevalgate.WriteFile(current.floors, encoded))
		return
	}

	enforceRetrieval(t, current.label+" by keyword", keyword, floors.Keyword)
	if hybrid != nil {
		enforceRetrieval(t, current.label+" by keyword and meaning", *hybrid, floors.Hybrid)
	}
}

func logReport(t *testing.T, label string, report agentevalgate.RetrievalReport) {
	t.Helper()

	t.Logf("%s: recall@5 %.2f, MRR %.2f, nDCG@10 %.2f over %d queries",
		label, report.RecallAt5, report.MRR, report.NDCGAt10, len(report.Outcomes))
	if misses := report.Misses(); misses != "" {
		t.Logf("%s, queries whose first result was not a relevant one:\n%s", label, misses)
	}
}

func readRetrievalFloors(t *testing.T, path string) agentevalgate.RetrievalFloorSet {
	t.Helper()

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return agentevalgate.RetrievalFloorSet{}
	}
	require.NoError(t, err)

	var floors agentevalgate.RetrievalFloorSet
	require.NoError(t, sonic.Unmarshal(raw, &floors))

	return floors
}

func enforceRetrieval(
	t *testing.T,
	label string,
	report agentevalgate.RetrievalReport,
	floors *agentevalgate.RetrievalFloors,
) {
	t.Helper()

	if floors == nil {
		message := label + " has no floors yet; pin them with:\n  " + retrievaltest.FloorsCommand
		if strings.EqualFold(os.Getenv(requireFloorsEnv), "true") {
			t.Fatal(message)
		}
		t.Log(message)
		return
	}

	require.Emptyf(t, report.Below(*floors),
		"%s fell below its floors: %s\n%s",
		label, strings.Join(report.Below(*floors), ", "), report.Misses())
}

func TestRetrievalNeverLeaks(t *testing.T) {
	h, suites := newEvalHarness(t)
	corpora := h.seed(t, h.tenant, h.data.User.ID, suites)

	other := seedtest.SeedAdditionalTenant(t, h.ctx, h.db, secondTenantSuffix)
	otherTenant := pagination.TenantInfo{OrgID: other.Organization.ID, BuID: other.BusinessUnit.ID}
	foreign := h.seed(t, otherTenant, other.User.ID, suites)

	sensitive := retrievaltest.SeedDocuments(t, h.ctx, h.db, retrievaltest.DocumentSeed{
		Tenant:       h.tenant,
		UploadedByID: h.data.User.ID,
		OwnerType:    "worker",
		Items:        suites.documents.Corpus,
	})
	agentScoped := retrievaltest.SeedMemories(t, h.ctx, h.db, retrievaltest.MemorySeed{
		Tenant:            h.tenant,
		Items:             suites.memories.Corpus,
		Scope:             agent.MemoryScopeAgent,
		AgentDefinitionID: pulid.MustNew("agdef_"),
	})
	retired := retrievaltest.SeedMemories(t, h.ctx, h.db, retrievaltest.MemorySeed{
		Tenant: h.tenant,
		Items:  suites.memories.Corpus,
		Status: agent.MemoryStatusRetired,
	})
	expiredAt := int64(1)
	expired := retrievaltest.SeedMemories(t, h.ctx, h.db, retrievaltest.MemorySeed{
		Tenant:    h.tenant,
		Items:     suites.memories.Corpus,
		ExpiresAt: &expiredAt,
	})

	searchers := map[string]*retrievalservice.Searcher{"keyword": h.searcher(nil)}
	memoryServices := map[string]serviceports.AgentMemoryService{"keyword": h.memoryService(nil)}
	if fixture, recorded := loadFixture(t); recorded && h.vectorReady(t) {
		for _, tenant := range []pagination.TenantInfo{h.tenant, otherTenant} {
			h.activateModel(t, tenant)
		}
		h.embed(t, fixture, corpora)
		h.embed(t, fixture, foreign)
		h.plantVectors(t, fixture, suites.documents, sensitive, airetrieval.SourceTypeDocument)
		for _, planted := range []*retrievaltest.Corpus{agentScoped, retired, expired} {
			h.plantVectors(t, fixture, suites.memories, planted, airetrieval.SourceTypeMemory)
		}
		searcher := h.searcher(retrievaltest.FixtureVectorizer{
			Fixture:  fixture,
			ModelKey: retrievaltest.ModelKey,
		})
		searchers["hybrid"] = searcher
		memoryServices["hybrid"] = h.memoryService(searcher)
	}

	denyWorker := retrievaltest.Access{
		Registry:   h.registry,
		Unreadable: map[permission.Resource]bool{permission.ResourceWorker: true},
	}
	forbidden := forbiddenIDs(foreign.documents, foreign.inbox, foreign.memories,
		sensitive, agentScoped, retired, expired)

	for name, searcher := range searchers {
		t.Run(name, func(t *testing.T) {
			for _, query := range allQueries(suites) {
				documents, err := searcher.SearchDocuments(
					h.ctx,
					serviceports.RetrievalSearchRequest{
						TenantInfo: h.tenant, Query: query, Limit: 50, Access: denyWorker,
					},
				)
				require.NoError(t, err)
				for _, hit := range documents.Hits {
					assert.NotContains(t, forbidden, hit.Document.ID, "document for %q", query)
					assert.Equal(t, h.tenant.OrgID, hit.Document.OrganizationID)
				}

				messages, err := searcher.SearchInboundMessages(h.ctx,
					serviceports.RetrievalSearchRequest{
						TenantInfo: h.tenant, Query: query, Limit: 50, Access: denyWorker,
					})
				require.NoError(t, err)
				for _, hit := range messages.Hits {
					assert.NotContains(t, forbidden, hit.Message.ID, "message for %q", query)
					assert.Equal(t, h.tenant.OrgID, hit.Message.OrganizationID)
				}

				recalled, err := memoryServices[name].Recall(h.ctx,
					serviceports.RecallAgentMemoriesRequest{
						TenantInfo: h.tenant, Query: query, Limit: agent.MaxMemoryRecallLimit,
					})
				require.NoError(t, err)
				for _, memory := range recalled {
					assert.NotContains(t, forbidden, memory.Memory.ID, "memory for %q", query)
				}
			}

			quoted, err := searcher.SearchInboundMessages(
				h.ctx,
				serviceports.RetrievalSearchRequest{
					TenantInfo: h.tenant, Query: "lumber order", Limit: 50, Access: denyWorker,
				},
			)
			require.NoError(t, err)
			for _, hit := range quoted.Hits {
				assert.NotEqual(t, corpora.inbox.IDs["late-pickup-complaint"], hit.Message.ID,
					"quoted history is never searched")
			}
		})
	}
}

func (h *evalHarness) plantVectors(
	t *testing.T,
	fixture *agentevalgate.EmbeddingFixture,
	suite *agentevalgate.RetrievalSuite,
	corpus *retrievaltest.Corpus,
	sourceType airetrieval.SourceType,
) {
	t.Helper()

	for _, item := range suite.Corpus {
		var chunks []retrievalservice.Chunk
		switch sourceType {
		case airetrieval.SourceTypeDocument:
			chunks = retrievalservice.ChunkDocument(retrievaltest.DocumentFor(item, ""))
		case airetrieval.SourceTypeMemory:
			chunks = retrievalservice.ChunkMemory(retrievaltest.MemoryFor(item))
		}

		embedded := make([]repositories.EmbeddingChunk, 0, len(chunks))
		for _, chunk := range chunks {
			vector, ok := fixture.Document(chunk.Hash)
			require.True(t, ok, item.Key)
			embedded = append(embedded, repositories.EmbeddingChunk{
				ChunkIndex:  chunk.Index,
				ContentHash: chunk.Hash,
				Vector:      vector,
			})
		}
		_, err := h.repo.ReplaceChunks(h.ctx, repositories.ReplaceEmbeddingChunksRequest{
			Source: repositories.AIRetrievalSourceRef{
				TenantInfo: corpus.Tenant,
				SourceType: sourceType,
				SourceID:   corpus.IDs[item.Key],
			},
			ModelKey:   retrievaltest.ModelKey,
			Dimensions: fixture.Dimensions,
			Chunks:     embedded,
		})
		require.NoError(t, err)
	}
}

func forbiddenIDs(corpora ...*retrievaltest.Corpus) []pulid.ID {
	ids := make([]pulid.ID, 0, 128)
	for _, corpus := range corpora {
		for id := range corpus.Keys {
			ids = append(ids, id)
		}
	}

	return ids
}

func allQueries(suites evalSuites) []string {
	queries := make([]string, 0, 64)
	for _, suite := range []*agentevalgate.RetrievalSuite{
		suites.documents, suites.inbox, suites.memories,
	} {
		queries = append(queries, suite.Queries()...)
	}

	return queries
}
