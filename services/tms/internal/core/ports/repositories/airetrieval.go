package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AIRetrievalSourceRef struct {
	TenantInfo pagination.TenantInfo
	SourceType airetrieval.SourceType
	SourceID   pulid.ID
}

type ListEmbeddingChunkHashesRequest struct {
	Source   AIRetrievalSourceRef
	ModelKey string
}

type EmbeddingChunkHash struct {
	ChunkIndex  int    `bun:"chunk_index"`
	ContentHash string `bun:"content_hash"`
}

type EmbeddingChunk struct {
	ChunkIndex  int
	ContentHash string
	Vector      []float32
}

type ReplaceEmbeddingChunksRequest struct {
	Source     AIRetrievalSourceRef
	ModelKey   string
	Dimensions int
	Chunks     []EmbeddingChunk
}

type ReplaceEmbeddingChunksResult struct {
	Written   int
	Unchanged int
	Removed   int
}

type DeleteAIRetrievalSourceResult struct {
	Embeddings   int
	IndexEntries int
}

type VectorSearchRequest struct {
	TenantInfo    pagination.TenantInfo
	SourceTypes   []airetrieval.SourceType
	ModelKey      string
	Dimensions    int
	Query         []float32
	Limit         int
	Candidates    int
	MinSimilarity float64
}

type VectorSearchHit struct {
	SourceType airetrieval.SourceType `bun:"source_type"`
	SourceID   pulid.ID               `bun:"source_id"`
	ChunkIndex int                    `bun:"chunk_index"`
	Similarity float64                `bun:"similarity"`
}

type GetCatalogEmbeddingsRequest struct {
	ModelKey string
	Corpus   airetrieval.CatalogCorpus
	Items    []airetrieval.CatalogItemRef
}

type CatalogEmbeddingInput struct {
	ItemKey     string
	ContentHash string
	Vector      []float32
}

type PutCatalogEmbeddingsRequest struct {
	ModelKey   string
	Corpus     airetrieval.CatalogCorpus
	Dimensions int
	Items      []CatalogEmbeddingInput
}

type PruneCatalogEmbeddingsRequest struct {
	ModelKey string
	Corpus   airetrieval.CatalogCorpus
	Keep     []airetrieval.CatalogItemRef
}

type AIEmbeddingRepository interface {
	VectorAvailability(ctx context.Context) (airetrieval.Availability, error)
	ListChunkHashes(
		ctx context.Context,
		req ListEmbeddingChunkHashesRequest,
	) ([]EmbeddingChunkHash, error)
	ReplaceChunks(
		ctx context.Context,
		req ReplaceEmbeddingChunksRequest,
	) (ReplaceEmbeddingChunksResult, error)
	DeleteSource(
		ctx context.Context,
		source AIRetrievalSourceRef,
	) (DeleteAIRetrievalSourceResult, error)
	Search(ctx context.Context, req VectorSearchRequest) ([]VectorSearchHit, error)
	GetCatalogEmbeddings(
		ctx context.Context,
		req GetCatalogEmbeddingsRequest,
	) ([]*airetrieval.CatalogEmbedding, error)
	PutCatalogEmbeddings(ctx context.Context, req PutCatalogEmbeddingsRequest) (int, error)
	PruneCatalogEmbeddings(ctx context.Context, req PruneCatalogEmbeddingsRequest) (int, error)
}

type MarkAIRetrievalStaleRequest struct {
	TenantInfo pagination.TenantInfo
	SourceType airetrieval.SourceType
	SourceIDs  []pulid.ID
	ModelKeys  []string
	Now        int64
}

type ClaimIndexEntriesRequest struct {
	TenantInfo  pagination.TenantInfo
	ModelKey    string
	SourceTypes []airetrieval.SourceType
	Limit       int
	Lease       time.Duration
	Now         int64
}

type IndexEntryOutcome struct {
	Key        airetrieval.IndexEntryKey
	Generation int64
	ChunkCount int
	Error      string
	RetryAt    int64
}

type MarkIndexEntriesRequest struct {
	Outcomes []IndexEntryOutcome
	Now      int64
}

type MarkIndexEntriesResult struct {
	Applied    int
	Superseded int
}

type CountIndexEntriesRequest struct {
	TenantInfo pagination.TenantInfo
	ModelKey   string
}

type IndexEntryCount struct {
	SourceType    airetrieval.SourceType  `bun:"source_type"`
	Status        airetrieval.IndexStatus `bun:"status"`
	Count         int                     `bun:"count"`
	LastIndexedAt *int64                  `bun:"last_indexed_at"`
	LastAttemptAt *int64                  `bun:"last_attempt_at"`
}

type FindStaleAIRetrievalSourcesRequest struct {
	TenantInfo pagination.TenantInfo
	SourceType airetrieval.SourceType
	ModelKey   string
	AfterID    pulid.ID
	Limit      int
}

type AIIndexEntryRepository interface {
	MarkStale(ctx context.Context, req MarkAIRetrievalStaleRequest) (int, error)
	ClaimIndexEntries(
		ctx context.Context,
		req ClaimIndexEntriesRequest,
	) ([]*airetrieval.IndexEntry, error)
	MarkIndexed(ctx context.Context, req MarkIndexEntriesRequest) (MarkIndexEntriesResult, error)
	MarkFailed(ctx context.Context, req MarkIndexEntriesRequest) (MarkIndexEntriesResult, error)
	MarkSkipped(ctx context.Context, req MarkIndexEntriesRequest) (MarkIndexEntriesResult, error)
	CountIndexEntries(ctx context.Context, req CountIndexEntriesRequest) ([]IndexEntryCount, error)
	FindStaleSources(
		ctx context.Context,
		req FindStaleAIRetrievalSourcesRequest,
	) ([]pulid.ID, error)
}

type SetAIRetrievalPausedRequest struct {
	TenantInfo pagination.TenantInfo
	Paused     bool
	Reason     airetrieval.PauseReason
	Now        int64
}

type SwapAIRetrievalModelRequest struct {
	TenantInfo      pagination.TenantInfo
	PendingModelKey string
}

type SwapAIRetrievalModelResult struct {
	Settings        *airetrieval.Settings
	RetiredModelKey string
}

type PurgeAIRetrievalModelRequest struct {
	TenantInfo pagination.TenantInfo
	ModelKey   string
	BatchSize  int
}

type PurgeAIRetrievalModelResult struct {
	Embeddings   int
	IndexEntries int
}

func (r PurgeAIRetrievalModelResult) Done() bool {
	return r.Embeddings == 0 && r.IndexEntries == 0
}

type AIRetrievalSettingsRepository interface {
	GetSettings(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*airetrieval.Settings, error)
	UpdateSettings(ctx context.Context, entity *airetrieval.Settings) (*airetrieval.Settings, error)
	SetPaused(ctx context.Context, req SetAIRetrievalPausedRequest) (*airetrieval.Settings, error)
	SwapModel(
		ctx context.Context,
		req SwapAIRetrievalModelRequest,
	) (*SwapAIRetrievalModelResult, error)
	PurgeModel(
		ctx context.Context,
		req PurgeAIRetrievalModelRequest,
	) (PurgeAIRetrievalModelResult, error)
}

type AIRetrievalRepository interface {
	AIEmbeddingRepository
	AIIndexEntryRepository
	AIRetrievalSettingsRepository
}
