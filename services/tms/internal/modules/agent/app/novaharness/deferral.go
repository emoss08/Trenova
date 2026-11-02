package novaharness

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// Defer hands a long task to a provider that runs it in the background. Only a
// worker defers: the handle is polled from the same workflow.
func (h *Harness) Defer(
	ctx context.Context,
	task *llmports.NovaTask,
) (*llmports.BackgroundSubmission, error) {
	submission, err := h.router.SubmitBackground(ctx, request(task))
	if err != nil {
		return nil, fmt.Errorf("nova %s: %w", task.Name, err)
	}

	return submission, nil
}

// Collect reports where a deferred task stands.
func (h *Harness) Collect(
	ctx context.Context,
	req *llmports.BackgroundPollRequest,
) (*llmports.BackgroundOutcome, error) {
	nova := *req
	nova.Nova = true

	return h.router.PollBackground(ctx, &nova)
}
