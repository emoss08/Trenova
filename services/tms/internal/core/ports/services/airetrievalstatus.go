package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type AIRetrievalAvailability struct {
	Available          bool                           `json:"available"`
	Reason             *airetrieval.UnavailableReason `json:"reason"`
	ExtensionInstalled bool                           `json:"extensionInstalled"`
	ExtensionVersion   *string                        `json:"extensionVersion"`
}

type AIRetrievalSettings struct {
	MemoryEnabled            bool                     `json:"memoryEnabled"`
	DocumentsEnabled         bool                     `json:"documentsEnabled"`
	InboundMessagesEnabled   bool                     `json:"inboundMessagesEnabled"`
	MonthlyIndexingBudgetUSD string                   `json:"monthlyIndexingBudgetUsd"`
	Paused                   bool                     `json:"paused"`
	PausedReason             *airetrieval.PauseReason `json:"pausedReason"`
	PausedAt                 *int64                   `json:"pausedAt"`
	ActiveModelKey           *string                  `json:"activeModelKey"`
	Dimensions               *int                     `json:"dimensions"`
	PendingModelKey          *string                  `json:"pendingModelKey"`
	PendingDimensions        *int                     `json:"pendingDimensions"`
	Version                  int64                    `json:"version"`
	UpdatedAt                *int64                   `json:"updatedAt"`
}

type AIRetrievalSourceStatus struct {
	SourceType    airetrieval.SourceType `json:"sourceType"`
	Enabled       bool                   `json:"enabled"`
	Total         int                    `json:"total"`
	Indexed       int                    `json:"indexed"`
	Pending       int                    `json:"pending"`
	Failed        int                    `json:"failed"`
	Skipped       int                    `json:"skipped"`
	LastIndexedAt *int64                 `json:"lastIndexedAt"`
	LastAttemptAt *int64                 `json:"lastAttemptAt"`
}

type AIRetrievalModelChange struct {
	FromModelKey string `json:"fromModelKey"`
	ToModelKey   string `json:"toModelKey"`
	Dimensions   int    `json:"dimensions"`
	Total        int    `json:"total"`
	Indexed      int    `json:"indexed"`
	Pending      int    `json:"pending"`
	Failed       int    `json:"failed"`
}

type AIRetrievalStatus struct {
	Availability           *AIRetrievalAvailability   `json:"availability"`
	Settings               *AIRetrievalSettings       `json:"settings"`
	Sources                []*AIRetrievalSourceStatus `json:"sources"`
	MonthStartedAt         int64                      `json:"monthStartedAt"`
	IndexingCostMonthUSD   string                     `json:"indexingCostMonthUsd"`
	IndexingUnpricedCalls  int                        `json:"indexingUnpricedCalls"`
	RetrievalCostMonthUSD  string                     `json:"retrievalCostMonthUsd"`
	RetrievalUnpricedCalls int                        `json:"retrievalUnpricedCalls"`
	LastIndexedAt          *int64                     `json:"lastIndexedAt"`
	ModelChange            *AIRetrievalModelChange    `json:"modelChange"`
	ConfiguredModelKey     *string                    `json:"configuredModelKey"`
	ConfiguredModelDiffers bool                       `json:"configuredModelDiffers"`
}

type AIRetrievalReindexEstimate struct {
	SourceType             airetrieval.SourceType `json:"sourceType"`
	ModelKey               *string                `json:"modelKey"`
	Sources                int                    `json:"sources"`
	AverageChunks          float64                `json:"averageChunks"`
	ChunksMeasured         bool                   `json:"chunksMeasured"`
	AverageTokensPerChunk  float64                `json:"averageTokensPerChunk"`
	EstimatedTokens        int                    `json:"estimatedTokens"`
	InputCostPerMillionUSD *string                `json:"inputCostPerMillionUsd"`
	EstimatedCostUSD       *string                `json:"estimatedCostUsd"`
	RemainingBudgetUSD     string                 `json:"remainingBudgetUsd"`
}

type AIRetrievalFailedEntry struct {
	ID            string                  `json:"id"`
	SourceType    airetrieval.SourceType  `json:"sourceType"`
	SourceID      pulid.ID                `json:"sourceId"`
	ModelKey      string                  `json:"modelKey"`
	Status        airetrieval.IndexStatus `json:"status"`
	Attempts      int                     `json:"attempts"`
	Error         string                  `json:"error"`
	LastAttemptAt *int64                  `json:"lastAttemptAt"`
	NextAttemptAt *int64                  `json:"nextAttemptAt"`
}

type AIRetrievalFailedEntryEdge struct {
	Node   *AIRetrievalFailedEntry `json:"node"`
	Cursor string                  `json:"cursor"`
}

type AIRetrievalFailedEntryPage struct {
	Edges       []*AIRetrievalFailedEntryEdge `json:"edges"`
	HasNextPage bool                          `json:"hasNextPage"`
	TotalCount  *int                          `json:"totalCount"`
}

type ListAIRetrievalFailedEntriesRequest struct {
	TenantInfo pagination.TenantInfo
	SourceType airetrieval.SourceType
	Table      memtable.Request
}

type UpdateAIRetrievalSettingsRequest struct {
	TenantInfo               pagination.TenantInfo
	MemoryEnabled            *bool
	DocumentsEnabled         *bool
	InboundMessagesEnabled   *bool
	MonthlyIndexingBudgetUSD *decimal.Decimal
	Paused                   *bool
	Actor                    *RequestActor
}

func (r *UpdateAIRetrievalSettingsRequest) Empty() bool {
	return r.MemoryEnabled == nil &&
		r.DocumentsEnabled == nil &&
		r.InboundMessagesEnabled == nil &&
		r.MonthlyIndexingBudgetUSD == nil &&
		r.Paused == nil
}

type ReindexAIRetrievalSourceRequest struct {
	TenantInfo pagination.TenantInfo
	SourceType airetrieval.SourceType
	Actor      *RequestActor
}

type AIRetrievalStatusService interface {
	Status(ctx context.Context, tenant pagination.TenantInfo) (*AIRetrievalStatus, error)
	UpdateSettings(
		ctx context.Context,
		req *UpdateAIRetrievalSettingsRequest,
	) (*AIRetrievalStatus, error)
	Reindex(ctx context.Context, req *ReindexAIRetrievalSourceRequest) (*AIRetrievalStatus, error)
	ReindexEstimate(
		ctx context.Context,
		tenant pagination.TenantInfo,
		sourceType airetrieval.SourceType,
	) (*AIRetrievalReindexEstimate, error)
	ListFailedEntries(
		ctx context.Context,
		req *ListAIRetrievalFailedEntriesRequest,
	) (*AIRetrievalFailedEntryPage, error)
}
