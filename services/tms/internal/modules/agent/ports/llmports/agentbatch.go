package llmports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var ErrBatchUnsupported = errors.New("the AI provider for this task cannot take a batch")

var ErrBatchOverBudget = errors.New("not one request of the batch fits the spend limit")

type BatchCompletionItem struct {
	CustomID string
	Request  StructuredCompletionRequest
}

type BatchCompletionRequest struct {
	TenantInfo    pagination.TenantInfo
	Task          aitask.Task
	Items         []BatchCompletionItem
	SpendLimitUSD *decimal.Decimal
	// Nova marks a batch of Nova's tasks, as on StructuredCompletionRequest.
	Nova bool
}

type BatchHandle struct {
	ProviderID       pulid.ID        `json:"providerId"`
	BatchID          string          `json:"batchId"`
	Model            string          `json:"model"`
	CustomIDs        []string        `json:"customIds"`
	SubmittedAt      int64           `json:"submittedAt"`
	EstimatedCostUSD decimal.Decimal `json:"estimatedCostUsd"`
}

type BatchStatusRequest struct {
	TenantInfo pagination.TenantInfo
	ProviderID pulid.ID
	BatchID    string
	Nova       bool
}

type BatchCancelRequest struct {
	TenantInfo pagination.TenantInfo
	ProviderID pulid.ID
	BatchID    string
	Nova       bool
}

type BatchStatus struct {
	Ended      bool
	RawStatus  string
	Processing int
	Succeeded  int
	Errored    int
	Canceled   int
	Expired    int
}

type BatchResultsRequest struct {
	TenantInfo   pagination.TenantInfo
	Task         aitask.Task
	ProviderID   pulid.ID
	BatchID      string
	SubmittedAt  int64
	Attributions map[string]aiusageports.AIUsageAttribution
	// Nova marks a batch of Nova's tasks, so its answers are priced as Nova's
	// model rather than the provider's own.
	Nova bool
}

type BatchItemResult struct {
	CustomID       string
	Result         *StructuredCompletionResult
	FailureCode    string
	FailureMessage string
	// NotRun is a request the batch never ran: canceled or expired.
	NotRun bool
}

func (r *BatchItemResult) Succeeded() bool {
	return r != nil && r.Result != nil
}

type BatchCompletionService interface {
	SubmitBatch(ctx context.Context, req *BatchCompletionRequest) (*BatchHandle, error)
	BatchStatus(ctx context.Context, req *BatchStatusRequest) (*BatchStatus, error)
	// CancelBatch stops the requests that have not run; the batch then ends
	// with the results of those that had.
	CancelBatch(ctx context.Context, req *BatchCancelRequest) error
	BatchResults(
		ctx context.Context,
		req *BatchResultsRequest,
	) (map[string]BatchItemResult, error)
}
