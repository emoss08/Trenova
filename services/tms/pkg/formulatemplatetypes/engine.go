package formulatemplatetypes

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// ErrRateTableMiss marks a lookup whose table exists but holds no entry, band,
// or cell for the requested key. It is the only lookup failure the *Or
// variants absorb: a missing table, a wrong arity, or an unusable key is an
// authoring mistake and must surface rather than quietly price at the fallback.
var ErrRateTableMiss = errors.New("rate table has no matching entry")

// ErrRateTableUnavailable is returned when an expression reaches for a rate
// table in a context that never loaded any.
var ErrRateTableUnavailable = errors.New(
	"rate table lookups are not available in this context",
)

type RateTableLookup interface {
	Lookup(table string, key any) (float64, error)
	Has(table string) bool
	Lookup2(table string, rowKey, colKey any) (float64, error)
	Has2(table string) bool
}

// BandedLookup is implemented by providers whose single-axis tables are bands
// with numeric floors, so a formula can read between floors or rate a weight at
// the next break instead of only landing in one band.
type BandedLookup interface {
	LookupInterp(table string, key any) (float64, error)
	DeficitWeight(table string, weight any) (float64, error)
}

// LookupExplainer is implemented by providers that can say which row or band
// answered a lookup, so a receipt shows more than the number that came back.
type LookupExplainer interface {
	ExplainLookup(table string, key any) (formulatypes.LookupMatch, bool)
	ExplainLookup2(table string, rowKey, colKey any) (formulatypes.LookupMatch, bool)
}

// EnvEvaluationRequest evaluates one expression against an already-built
// environment, the shape the Studio preview and saved scenarios use.
type EnvEvaluationRequest struct {
	Expression string
	Env        map[string]any
	Lookup     RateTableLookup
}

// ContextVariableProvider supplies variables no record carries — market data
// such as the tenant's latest fuel price — so a formula can read them like any
// other field. Providers are asked per tenant, once per batch.
type ContextVariableProvider interface {
	ContextVariables(ctx context.Context, tenantInfo pagination.TenantInfo) (map[string]any, error)
}

type EvaluationRequest struct {
	Template  *formulatemplate.FormulaTemplate
	Entity    any
	Variables map[string]any
	// Provided are tenant-level values from external feeds. They sit beneath
	// caller variables, so a scenario can pin a fuel price the feed disagrees with.
	Provided map[string]any
	// Overrides are engine-supplied bindings that may shadow schema fields —
	// the rate engine binding a rule's rate or a matrix cell as baseRate.
	// Caller-supplied variables can never shadow a field; overrides exist so
	// the system itself can.
	Overrides map[string]any
	Lookup    RateTableLookup
}

type ExpressionEvaluationRequest struct {
	Expression string
	Entity     any
	SchemaID   string
	Variables  map[string]any
	Provided   map[string]any
	Breakdowns []*formulatypes.BreakdownDefinition
	Lookup     RateTableLookup
	// AllowBoolean lets a yes-or-no expression evaluate to true or false. A
	// charge never may: a comparison that slipped in as a whole formula would
	// otherwise price every shipment at one dollar.
	AllowBoolean bool
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

// RoundingResult records how the charge policy rounded an amount, so a
// preview can show the raw figure beside the billable one and a reviewer can
// see which mode produced the cents.
type RoundingResult struct {
	Mode            string          `json:"mode"`
	Precision       int32           `json:"precision"`
	Applied         bool            `json:"applied"`
	UnroundedAmount decimal.Decimal `json:"unroundedAmount"`
}

type EvaluationResult struct {
	Value     decimal.Decimal
	RawValue  any
	Variables map[string]any
	Breakdown []BreakdownAmount
	Receipt   *formulatypes.Receipt
}

type CalculateRequest struct {
	TemplateID pulid.ID
	Entity     any
	Variables  map[string]any
	// Overrides carry engine-supplied bindings that may shadow schema fields.
	// See EvaluationRequest.Overrides.
	Overrides  map[string]any
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
	Rounding            *RoundingResult
	VersionNumber       int64
	Receipt             *formulatypes.Receipt
}
