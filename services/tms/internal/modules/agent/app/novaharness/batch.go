package novaharness

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// SubmitTasks sends a batch of one kind of task to the provider's batch API,
// each request carrying Nova's identity like a task run on its own.
func (h *Harness) SubmitTasks(
	ctx context.Context,
	batch *llmports.NovaBatch,
) (*llmports.BatchHandle, error) {
	items := make([]llmports.BatchCompletionItem, 0, len(batch.Items))
	for _, item := range batch.Items {
		items = append(items, llmports.BatchCompletionItem{
			CustomID: item.CustomID,
			Request:  *request(item.Task),
		})
	}

	return h.batches.SubmitBatch(ctx, &llmports.BatchCompletionRequest{
		TenantInfo:    batch.TenantInfo,
		Task:          batch.Task,
		Items:         items,
		SpendLimitUSD: batch.SpendLimitUSD,
		Nova:          true,
	})
}

func (h *Harness) BatchStatus(
	ctx context.Context,
	req *llmports.BatchStatusRequest,
) (*llmports.BatchStatus, error) {
	nova := *req
	nova.Nova = true

	return h.batches.BatchStatus(ctx, &nova)
}

func (h *Harness) CancelBatch(ctx context.Context, req *llmports.BatchCancelRequest) error {
	nova := *req
	nova.Nova = true

	return h.batches.CancelBatch(ctx, &nova)
}

func (h *Harness) BatchResults(
	ctx context.Context,
	req *llmports.BatchResultsRequest,
) (map[string]llmports.BatchItemResult, error) {
	nova := *req
	nova.Nova = true

	return h.batches.BatchResults(ctx, &nova)
}
