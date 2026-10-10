package quotaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func OrUnlimited(guard platformplanports.QuotaGuard) platformplanports.QuotaGuard {
	if guard == nil {
		return NewUnlimited()
	}

	return guard
}

func Enforcing(guard platformplanports.QuotaGuard) bool {
	if guard == nil {
		return false
	}

	_, unlimited := guard.(*UnlimitedQuotaGuard)

	return !unlimited
}

func DecisionError(decision *platformplanports.QuotaDecision) error {
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

func Preflight(ctx context.Context, guard platformplanports.QuotaGuard, req *platformplanports.QuotaRequest) error {
	if !Enforcing(guard) {
		return nil
	}

	decision, err := guard.Check(ctx, req)
	if err != nil {
		return err
	}

	return DecisionError(decision)
}

func EnforceAll(ctx context.Context, guard platformplanports.QuotaGuard, reqs ...platformplanports.QuotaRequest) error {
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
