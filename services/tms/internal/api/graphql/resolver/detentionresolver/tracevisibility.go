package detentionresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
)

const (
	traceStepsKey  = "steps"
	traceKindKey   = "kind"
	traceAmountKey = "amount"
)

// visibleTrace drops the driver's pay from a calculation trace for a reader
// without the margins field group, as detention.CalculationTrace.WithholdFields
// does on REST: beside the billed amount the pay gives the margin away.
func (r *Deps) visibleTrace(ctx context.Context, trace map[string]any) map[string]any {
	if trace == nil {
		return nil
	}
	authCtx, ok := gqlctx.AuthContext(ctx)
	if ok && r.VisibleFieldGroups(ctx, authCtx).Has(permission.FieldGroupMargins) {
		return trace
	}
	steps, ok := trace[traceStepsKey].([]any)
	if !ok {
		return trace
	}
	for _, raw := range steps {
		step, isStep := raw.(map[string]any)
		if isStep && step[traceKindKey] == string(detention.TraceStepDriverPay) {
			delete(step, traceAmountKey)
		}
	}

	return trace
}
