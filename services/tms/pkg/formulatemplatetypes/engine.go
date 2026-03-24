package formulatemplatetypes

import (
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type EvaluationRequest struct {
	Template  *formulatemplate.FormulaTemplate
	Entity    any
	Variables map[string]any
}

type EvaluationResult struct {
	Value     decimal.Decimal
	RawValue  any
	Variables map[string]any
}

type CalculateRequest struct {
	TemplateID pulid.ID
	Entity     any
	Variables  map[string]any
	TenantInfo pagination.TenantInfo
}

type CalculateResponse struct {
	Amount    decimal.Decimal
	Variables map[string]any
}
