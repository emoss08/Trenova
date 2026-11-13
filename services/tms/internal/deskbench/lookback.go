package deskbench

import (
	"fmt"

	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
)

// LookBack runs the conversation's look back in this process: what the
// reflection workflow does once a conversation goes quiet, without the wait.
// A look back with nothing to go on is skipped, as it is in the product, and
// that is not an error: the scenario's checks say what should have been kept.
func (c *Conversation) LookBack(ctx context.Context) error {
	bench := c.session.bench
	ctx = c.session.Context(ctx)

	plan, err := bench.Reflections.PrepareThread(ctx, &agentreflectionports.ReflectOnThreadRequest{
		TenantInfo: c.session.Tenant,
		ThreadID:   c.Thread.ID,
		UserID:     c.session.Actor.UserID,
	})
	if err != nil {
		return fmt.Errorf("prepare the look back: %w", err)
	}
	if !plan.Ready() {
		return nil
	}

	completion, err := bench.Completion.CompleteStructured(ctx, plan.Request)
	if err != nil {
		if failErr := bench.Reflections.Fail(ctx, &agentreflectionports.FailReflectionRequest{
			Plan:    plan,
			Message: err.Error(),
		}); failErr != nil {
			return fmt.Errorf("look back: %w (and it could not be recorded: %w)", err, failErr)
		}

		return fmt.Errorf("look back: %w", err)
	}
	if _, err = bench.Reflections.Finish(ctx, plan, completion); err != nil {
		return fmt.Errorf("keep what the look back learned: %w", err)
	}

	return nil
}
