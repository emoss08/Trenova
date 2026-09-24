package retrievaljobs

import (
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/shopspring/decimal"
)

type PlanInput struct {
	temporaljobs.TenantWorkItem
}

type ModelChangeInput struct {
	temporaljobs.TenantWorkItem
	PendingModelKey string `json:"pendingModelKey"`
}

type PurgeInput struct {
	temporaljobs.TenantWorkItem
	ModelKey string `json:"modelKey"`
}

type OrganizationSweepInput struct {
	temporaljobs.TenantWorkItem
	Wake bool `json:"wake"`
}

type ListOrganizationsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

type SweepInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
}

type SweepResult struct {
	OrganizationsSwept  int      `json:"organizationsSwept"`
	Marked              int      `json:"marked"`
	FailedOrganizations []string `json:"failedOrganizations"`
}

type IndexOrganizationResult struct {
	Batches        int                           `json:"batches"`
	Indexed        int                           `json:"indexed"`
	Skipped        int                           `json:"skipped"`
	Failed         int                           `json:"failed"`
	ChunksEmbedded int                           `json:"chunksEmbedded"`
	CostUSD        decimal.Decimal               `json:"costUsd"`
	Swapped        bool                          `json:"swapped"`
	Purged         int                           `json:"purged"`
	Stopped        airetrieval.UnavailableReason `json:"stopped,omitempty"`
}

func (r *IndexOrganizationResult) absorb(batch *serviceports.RetrievalIndexBatchResult) {
	r.Batches++
	r.Indexed += batch.Indexed
	r.Skipped += batch.Skipped
	r.Failed += batch.Failed
	r.ChunksEmbedded += batch.ChunksEmbedded
	r.CostUSD = r.CostUSD.Add(batch.CostUSD)
}

type ReindexResult struct {
	Marked int  `json:"marked"`
	Done   bool `json:"done"`
}
