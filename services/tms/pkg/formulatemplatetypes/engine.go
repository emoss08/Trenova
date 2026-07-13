package formulatemplatetypes

import (
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type RateTableLookup interface {
	Lookup(table string, key any) (float64, error)
	Has(table string) bool
}

type EvaluationRequest struct {
	Template  *formulatemplate.FormulaTemplate
	Entity    any
	Variables map[string]any
	Lookup    RateTableLookup
}

type ExpressionEvaluationRequest struct {
	Expression string
	Entity     any
	SchemaID   string
	Variables  map[string]any
	Breakdowns []*formulatypes.BreakdownDefinition
	Lookup     RateTableLookup
}

type BreakdownAmount struct {
	Name   string          `json:"name"`
	Label  string          `json:"label,omitempty"`
	Amount decimal.Decimal `json:"amount"`
	Error  string          `json:"error,omitempty"`
}

type GuardrailResult struct {
	Applied   bool             `json:"applied"`
	Bound     string           `json:"bound,omitempty"`
	RawAmount decimal.Decimal  `json:"rawAmount"`
	MinCharge *decimal.Decimal `json:"minCharge,omitempty"`
	MaxCharge *decimal.Decimal `json:"maxCharge,omitempty"`
}

type EvaluationResult struct {
	Value     decimal.Decimal
	RawValue  any
	Variables map[string]any
	Breakdown []BreakdownAmount
}

type CalculateRequest struct {
	TemplateID pulid.ID
	Entity     any
	Variables  map[string]any
	TenantInfo pagination.TenantInfo
	RatingDate int64
}

type CalculateResponse struct {
	Amount              decimal.Decimal
	Variables           map[string]any
	FormulaTemplateID   string
	FormulaTemplateName string
	Expression          string
	Breakdown           []BreakdownAmount
	Guardrail           *GuardrailResult
	VersionNumber       int64
}
