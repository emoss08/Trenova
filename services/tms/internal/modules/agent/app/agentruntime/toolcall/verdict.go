package toolcall

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
)

func derivedVerdict(failed bool, action *agentruntimeports.PendingAction) string {
	switch {
	case action != nil && action.Simulated:
		return aitrace.OutcomeSimulated
	case action != nil && action.Executed && action.ExecutionError != "":
		return aitrace.OutcomeFailed
	case action != nil && action.Executed:
		return aitrace.OutcomeRan
	case action != nil:
		return aitrace.OutcomeProposed
	case failed:
		return aitrace.OutcomeFailed
	default:
		return aitrace.OutcomeRan
	}
}
