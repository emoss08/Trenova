package assistantjobs

import (
	"context"
	"errors"
	"fmt"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.temporal.io/api/serviceerror"
)

// turnCanceller cancels the executions StartTurnWorkflow starts. It is where
// Temporal's "no such execution" becomes the port's ErrNoTurnExecution, so
// every caller that stops a turn tells a stopped turn from one it has to
// close itself the same way.
type turnCanceller struct {
	workflows serviceports.WorkflowStarter
}

// NewTurnCanceller is the canceller for turns carried by AssistantTurnWorkflow.
func NewTurnCanceller(workflows serviceports.WorkflowStarter) serviceports.AssistantTurnCanceller {
	return &turnCanceller{workflows: workflows}
}

func (c *turnCanceller) CancelTurn(ctx context.Context, workflowID string) error {
	err := c.workflows.CancelWorkflow(ctx, workflowID, "")
	var gone *serviceerror.NotFound
	if errors.As(err, &gone) {
		return fmt.Errorf("%w: %w", serviceports.ErrNoTurnExecution, err)
	}

	return err
}
