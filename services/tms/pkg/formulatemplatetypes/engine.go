package formulatemplatetypes

import (
	"context"

	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ContextVariableProvider interface {
	ContextVariables(ctx context.Context, tenantInfo pagination.TenantInfo) (map[string]any, error)
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

type RoundingResult struct {
	Mode            string          `json:"mode"`
	Precision       int32           `json:"precision"`
	Applied         bool            `json:"applied"`
	UnroundedAmount decimal.Decimal `json:"unroundedAmount"`
}

type CalculateRequest struct {
	TemplateID        pulid.ID
	Entity            any
	Variables         map[string]any
	Overrides         map[string]any
	TenantInfo        pagination.TenantInfo
	RatingDate        int64
	UnavailableInputs map[string]string
}

type CalculateResponse struct {
	Amount              decimal.Decimal
	Variables           map[string]any
	FormulaTemplateID   string
	FormulaTemplateName string
	Expression          string
	Breakdown           []BreakdownAmount
	Guardrail           *GuardrailResult
	Rounding            *RoundingResult
	VersionNumber       int64
	Receipt             *formulatypes.Receipt
}
