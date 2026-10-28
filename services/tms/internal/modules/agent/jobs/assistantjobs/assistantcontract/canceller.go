package assistantcontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.temporal.io/api/serviceerror"
)

// turnCanceller cancels the executions StartTurnWorkflow starts. It is where
// Temporal's "no such execution" becomes the port's ErrNoTurnExecution, so
// every caller that stops a turn tells a stopped turn from one it has to
// close itself the same way.
type turnCanceller struct {
	workflows kernelports.WorkflowStarter
}

// NewTurnCanceller is the canceller for turns carried by AssistantTurnWorkflow.
func NewTurnCanceller(workflows kernelports.WorkflowStarter) assistantports.AssistantTurnCanceller {
	return &turnCanceller{workflows: workflows}
}

func (c *turnCanceller) CancelTurn(ctx context.Context, workflowID string) error {
	err := c.workflows.CancelWorkflow(ctx, workflowID, "")
	var gone *serviceerror.NotFound
	if errors.As(err, &gone) {
		return fmt.Errorf("%w: %w", assistantports.ErrNoTurnExecution, err)
	}

	return err
}
