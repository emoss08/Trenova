package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
)

type FormulaCalculator interface {
	Calculate(
		ctx context.Context,
		req *formulatemplatetypes.CalculateRequest,
	) (*formulatemplatetypes.CalculateResponse, error)
}

// EvaluatePredicateRequest asks whether a condition holds for one entity.
type EvaluatePredicateRequest struct {
	Expression string
	SchemaID   string
	Entity     any
}

// FormulaPredicateEvaluator answers yes-or-no questions written in the same
// expression language formulas use.
//
// It is separate from FormulaCalculator because a condition is not a price: it
// takes no rate tables, produces no breakdown, and runs far more often — once
// per accessorial on a contract, on every save. Sharing the calculator's entry
// point would drag a full rate-table load onto that path for a question like
// "does this load have more than two stops".
type FormulaPredicateEvaluator interface {
	EvaluatePredicate(ctx context.Context, req *EvaluatePredicateRequest) (bool, error)
}
