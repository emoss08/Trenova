package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type RetrievalIndexer interface {
	MarkStale(
		ctx context.Context,
		tenant pagination.TenantInfo,
		sourceType airetrieval.SourceType,
		ids ...pulid.ID,
	) error
	DeleteSource(
		ctx context.Context,
		tenant pagination.TenantInfo,
		sourceType airetrieval.SourceType,
		id pulid.ID,
	) error
	Reindex(
		ctx context.Context,
		tenant pagination.TenantInfo,
		sourceType airetrieval.SourceType,
	) error
}

type RetrievalIndexPlan struct {
	Active          bool                          `json:"active"`
	Reason          airetrieval.UnavailableReason `json:"reason,omitempty"`
	ActiveModelKey  string                        `json:"activeModelKey,omitempty"`
	PendingModelKey string                        `json:"pendingModelKey,omitempty"`
	SourceTypes     []airetrieval.SourceType      `json:"sourceTypes,omitempty"`
	RetiredKeys     []string                      `json:"retiredKeys,omitempty"`
}

func (p RetrievalIndexPlan) ModelKeys() []string {
	keys := make([]string, 0, 2)
	if p.ActiveModelKey != "" {
		keys = append(keys, p.ActiveModelKey)
	}
	if p.PendingModelKey != "" {
		keys = append(keys, p.PendingModelKey)
	}

	return keys
}

type RetrievalIndexBatchRequest struct {
	TenantInfo  pagination.TenantInfo    `json:"tenantInfo"`
	ModelKey    string                   `json:"modelKey"`
	SourceTypes []airetrieval.SourceType `json:"sourceTypes"`
	Limit       int                      `json:"limit"`
}

type RetrievalIndexBatchResult struct {
	Claimed        int             `json:"claimed"`
	Indexed        int             `json:"indexed"`
	Skipped        int             `json:"skipped"`
	Failed         int             `json:"failed"`
	Superseded     int             `json:"superseded"`
	ChunksEmbedded int             `json:"chunksEmbedded"`
	CostUSD        decimal.Decimal `json:"costUsd"`
	BudgetReached  bool            `json:"budgetReached"`
}

func (r RetrievalIndexBatchResult) Drained() bool { return r.Claimed == 0 }

type RetrievalModelChangeResult struct {
	Swapped         bool   `json:"swapped"`
	RetiredModelKey string `json:"retiredModelKey,omitempty"`
}

type RetrievalPurgeResult struct {
	Embeddings   int  `json:"embeddings"`
	IndexEntries int  `json:"indexEntries"`
	Done         bool `json:"done"`
}

type RetrievalSweepResult struct {
	Marked  int `json:"marked"`
	Waiting int `json:"waiting"`
}

func (r RetrievalSweepResult) HasWork() bool { return r.Marked > 0 || r.Waiting > 0 }

type RetrievalReindexPageRequest struct {
	TenantInfo pagination.TenantInfo  `json:"tenantInfo"`
	SourceType airetrieval.SourceType `json:"sourceType"`
	AfterID    pulid.ID               `json:"afterId"`
	Limit      int                    `json:"limit"`
}

type RetrievalReindexPage struct {
	Marked int      `json:"marked"`
	Next   pulid.ID `json:"next"`
	Done   bool     `json:"done"`
}

type RetrievalIndexPipeline interface {
	Plan(ctx context.Context, tenant pagination.TenantInfo) (RetrievalIndexPlan, error)
	IndexBatch(
		ctx context.Context,
		req RetrievalIndexBatchRequest,
	) (RetrievalIndexBatchResult, error)
	CompleteModelChange(
		ctx context.Context,
		tenant pagination.TenantInfo,
		pendingModelKey string,
	) (RetrievalModelChangeResult, error)
	PurgeRetiredModel(
		ctx context.Context,
		tenant pagination.TenantInfo,
		modelKey string,
	) (RetrievalPurgeResult, error)
	Sweep(ctx context.Context, tenant pagination.TenantInfo) (RetrievalSweepResult, error)
	Wake(ctx context.Context, tenant pagination.TenantInfo)
	ReindexPage(ctx context.Context, req RetrievalReindexPageRequest) (RetrievalReindexPage, error)
}

type RetrievalMatch string

const (
	RetrievalMatchWords   = RetrievalMatch("words")
	RetrievalMatchMeaning = RetrievalMatch("meaning")
	RetrievalMatchBoth    = RetrievalMatch("both")
)

type RetrievalAccess interface {
	MayReadResource(ctx context.Context, resource permission.Resource) bool
	MayReadRecord(ctx context.Context, resource permission.Resource, recordID string) bool
	ShowsField(ctx context.Context, resource permission.Resource, field string) bool
	ShowsRecordText(ctx context.Context, resource permission.Resource) bool
}

type RetrievalSearchRequest struct {
	TenantInfo  pagination.TenantInfo
	Query       string
	Limit       int
	Access      RetrievalAccess
	Attribution AIUsageAttribution
}

type RetrievalSemantics struct {
	Used   bool
	Reason airetrieval.UnavailableReason
}

type DocumentSearchHit struct {
	Document      *document.Document
	Page          int
	Snippet       string
	ShowsFileName bool
	Match         RetrievalMatch
	Score         float64
	Similarity    float64
}

type DocumentSearchResult struct {
	Hits      []DocumentSearchHit
	Semantics RetrievalSemantics
}

type InboundMessageSearchHit struct {
	Message    *inboundmessage.InboundMessage
	Subject    string
	From       string
	Snippet    string
	Match      RetrievalMatch
	Score      float64
	Similarity float64
}

type InboundMessageSearchResult struct {
	Hits      []InboundMessageSearchHit
	Semantics RetrievalSemantics
}

type RetrievalSearcher interface {
	SearchDocuments(ctx context.Context, req RetrievalSearchRequest) (*DocumentSearchResult, error)
	SearchInboundMessages(
		ctx context.Context,
		req RetrievalSearchRequest,
	) (*InboundMessageSearchResult, error)
}

type SimilarMemoriesRequest struct {
	TenantInfo  pagination.TenantInfo
	Text        string
	Query       QueryVector
	Limit       int
	Attribution AIUsageAttribution
}

type MemorySimilarity struct {
	MemoryID   pulid.ID
	Similarity float64
}

type SimilarMemories struct {
	Memories  []MemorySimilarity
	Semantics RetrievalSemantics
	Floor     float64
}

type MemoryVectorSearcher interface {
	SimilarMemories(ctx context.Context, req SimilarMemoriesRequest) (SimilarMemories, error)
}

type ContextQuery struct {
	Actor        *RequestActor
	DefinitionID pulid.ID
	ThreadID     pulid.ID
	RunID        pulid.ID
	Input        string
	History      []conversation.Message
}

func (q ContextQuery) Request() QueryVectorRequest {
	if q.Actor == nil {
		return QueryVectorRequest{}
	}

	return QueryVectorRequest{
		TenantInfo: q.Actor.TenantInfo(),
		Text:       TurnQueryText(q.Input, q.History),
		Attribution: AIUsageAttribution{
			UserID:            q.Actor.UserID,
			AgentDefinitionID: q.DefinitionID,
			ThreadID:          q.ThreadID,
			RunID:             q.RunID,
		},
	}
}
