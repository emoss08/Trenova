package quotaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func OrUnlimited(guard services.QuotaGuard) services.QuotaGuard {
	if guard == nil {
		return NewUnlimited()
	}

	return guard
}

func Enforcing(guard services.QuotaGuard) bool {
	if guard == nil {
		return false
	}

	_, unlimited := guard.(*UnlimitedQuotaGuard)

	return !unlimited
}

func DecisionError(decision *services.QuotaDecision) error {
	if decision == nil || decision.Allowed {
		return nil
	}

	return errortypes.NewQuotaExceededError(
		string(decision.Meter),
		decision.Limit,
		decision.Used,
		decision.Plan.String(),
	)
}

func Preflight(ctx context.Context, guard services.QuotaGuard, req *services.QuotaRequest) error {
	if !Enforcing(guard) {
		return nil
	}

	decision, err := guard.Check(ctx, req)
	if err != nil {
		return err
	}

	return DecisionError(decision)
}

func EnforceAll(ctx context.Context, guard services.QuotaGuard, reqs ...services.QuotaRequest) error {
	if !Enforcing(guard) {
		return nil
	}

	for i := range reqs {
		if err := guard.Enforce(ctx, &reqs[i]); err != nil {
			return err
		}
	}

	return nil
}
