package llmports

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// ErrTaskWithoutFeature reports a task that names no plan feature, which no
// plan can pay for.
var ErrTaskWithoutFeature = errors.New("the AI task names no plan feature")

// RequireTask refuses a task the organization's plan does not pay for: none
// of its features (aitask.Task.Features) is on. The refusal is the plan's
// (PLAN_RESTRICTED, naming the add-on), and it is also ErrNoProviderConfigured,
// because nothing may be attempted: work that is skipped when no provider
// serves it is skipped the same way off the plan.
func RequireTask(
	ctx context.Context,
	gate platformcatalogports.FeatureGate,
	task aitask.Task,
) error {
	features := task.Features()
	if len(features) == 0 {
		return fmt.Errorf("%w: %q", ErrTaskWithoutFeature, task)
	}

	var refusal *errortypes.PlanRestrictionError
	for _, feature := range features {
		err := platformcatalogports.RequireFeature(ctx, gate, feature)
		if err == nil {
			return nil
		}
		restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
		if !ok {
			return err
		}
		if refusal == nil {
			refusal = restriction
		}
	}

	return refusal.WithInternal(ErrNoProviderConfigured)
}
