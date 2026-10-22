package formulatemplateports

import (
	"context"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

type FormulaCalculator interface {
	Calculate(
		ctx context.Context,
		req *formulatemplatetypes.CalculateRequest,
	) (*formulatemplatetypes.CalculateResponse, error)
}

type EvaluatePredicatesRequest struct {
	Expressions []string
	SchemaID    string
	Entity      any
	TenantInfo  pagination.TenantInfo
}

type PredicateOutcome struct {
	Holds bool
	Err   error
}

type FormulaPredicateEvaluator interface {
	EvaluatePredicates(
		ctx context.Context,
		req *EvaluatePredicatesRequest,
	) ([]PredicateOutcome, error)
}
