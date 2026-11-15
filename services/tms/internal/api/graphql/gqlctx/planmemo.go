package gqlctx

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

type planMemoKey struct{}

// PlanMemo is one request's answers about the organization's plan: each
// feature is asked of the gate at most once, however many rows read it.
type PlanMemo interface {
	FeatureOff(ctx context.Context, feature platformcatalog.FeatureKey) (bool, error)
}

func WithPlanMemo(ctx context.Context, memo PlanMemo) context.Context {
	return context.WithValue(ctx, planMemoKey{}, memo)
}

func PlanMemoFrom(ctx context.Context) (PlanMemo, bool) {
	memo, ok := ctx.Value(planMemoKey{}).(PlanMemo)
	return memo, ok && memo != nil
}
