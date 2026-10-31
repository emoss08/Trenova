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
	if h == nil || h.batches == nil {
		return nil, llmports.ErrBatchUnsupported
	}

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
	})
}

func (h *Harness) BatchStatus(
	ctx context.Context,
	req *llmports.BatchStatusRequest,
) (*llmports.BatchStatus, error) {
	if h == nil || h.batches == nil {
		return nil, llmports.ErrBatchUnsupported
	}

	return h.batches.BatchStatus(ctx, req)
}

func (h *Harness) CancelBatch(ctx context.Context, req *llmports.BatchCancelRequest) error {
	if h == nil || h.batches == nil {
		return llmports.ErrBatchUnsupported
	}

	return h.batches.CancelBatch(ctx, req)
}

func (h *Harness) BatchResults(
	ctx context.Context,
	req *llmports.BatchResultsRequest,
) (map[string]llmports.BatchItemResult, error) {
	if h == nil || h.batches == nil {
		return nil, llmports.ErrBatchUnsupported
	}

	return h.batches.BatchResults(ctx, req)
}
