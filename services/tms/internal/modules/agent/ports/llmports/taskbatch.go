package llmports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

// NovaBatchItem is one task of a batch, named by an id the caller reads its
// answer back under.
type NovaBatchItem struct {
	CustomID string
	Task     *NovaTask
}

// NovaBatch is many tasks of one kind sent to the provider's batch API, which
// answers within a day at about half the price of asking one at a time.
type NovaBatch struct {
	TenantInfo    pagination.TenantInfo
	Task          aitask.Task
	Items         []NovaBatchItem
	SpendLimitUSD *decimal.Decimal
}

// TaskBatcher sends Nova tasks that nobody is waiting on through the batch API.
// A caller submits, checks the status on a timer and decodes each answer with
// DecodeAnswer once the batch has ended.
type TaskBatcher interface {
	SubmitTasks(ctx context.Context, batch *NovaBatch) (*BatchHandle, error)
	BatchStatus(ctx context.Context, req *BatchStatusRequest) (*BatchStatus, error)
	// CancelBatch stops the requests that have not run, for work someone is
	// now waiting on; the batch then ends with what had already run.
	CancelBatch(ctx context.Context, req *BatchCancelRequest) error
	BatchResults(
		ctx context.Context,
		req *BatchResultsRequest,
	) (map[string]BatchItemResult, error)
}

// TaskDeferrer hands a long Nova task to a provider that runs it in the
// background, so an activity does not hold its heartbeat open for minutes. A
// provider that cannot defer answers inline, on the submission.
type TaskDeferrer interface {
	Defer(ctx context.Context, task *NovaTask) (*BackgroundSubmission, error)
	Collect(ctx context.Context, req *BackgroundPollRequest) (*BackgroundOutcome, error)
}
