package formulatemplateports

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type BacktestSummary struct {
	ShipmentCount       int             `json:"shipmentCount"`
	EvaluatedCount      int             `json:"evaluatedCount"`
	ChangedCount        int             `json:"changedCount"`
	IncreasedCount      int             `json:"increasedCount"`
	DecreasedCount      int             `json:"decreasedCount"`
	ErrorCount          int             `json:"errorCount"`
	CurrentErrorCount   int             `json:"currentErrorCount"`
	CandidateErrorCount int             `json:"candidateErrorCount"`
	GuardrailCount      int             `json:"guardrailCount"`
	CurrentTotal        decimal.Decimal `json:"currentTotal"`
	CandidateTotal      decimal.Decimal `json:"candidateTotal"`
	TotalDelta          decimal.Decimal `json:"totalDelta"`
	TotalDeltaPct       decimal.Decimal `json:"totalDeltaPct"`
	MaxIncrease         decimal.Decimal `json:"maxIncrease"`
	MaxDecrease         decimal.Decimal `json:"maxDecrease"`
}

type ScenarioVerdict string

const (
	ScenarioVerdictIntended = ScenarioVerdict("intended")
	ScenarioVerdictBug      = ScenarioVerdict("bug")
	ScenarioVerdictUnclear  = ScenarioVerdict("unclear")
)

func (v ScenarioVerdict) IsValid() bool {
	switch v {
	case ScenarioVerdictIntended, ScenarioVerdictBug, ScenarioVerdictUnclear:
		return true
	default:
		return false
	}
}

type ScenarioTraceRowKind string

const (
	ScenarioRowInput   = ScenarioTraceRowKind("input")
	ScenarioRowTerm    = ScenarioTraceRowKind("term")
	ScenarioRowFormula = ScenarioTraceRowKind("formula")
	ScenarioRowPolicy  = ScenarioTraceRowKind("policy")
	ScenarioRowBilled  = ScenarioTraceRowKind("billed")
)

type ScenarioTraceRow struct {
	Kind   ScenarioTraceRowKind `json:"kind"`
	Name   string               `json:"name"`
	Before *string              `json:"before"`
	After  *string              `json:"after"`
}

type ScenarioRecorder struct {
	ID   pulid.ID `json:"id"`
	Name string   `json:"name"`
}

type ScenarioTrace struct {
	Expected          string             `json:"expected"`
	Actual            *string            `json:"actual"`
	Delta             *string            `json:"delta"`
	BaselineLabel     string             `json:"baselineLabel"`
	RecordedOnVersion *int64             `json:"recordedOnVersion"`
	RecordedBy        *ScenarioRecorder  `json:"recordedBy"`
	Error             string             `json:"error,omitempty"`
	Rows              []ScenarioTraceRow `json:"rows"`

	ScenarioName        string          `json:"-"`
	ScenarioInputs      map[string]any  `json:"-"`
	ExpressionChanged   bool            `json:"-"`
	PolicyChanged       bool            `json:"-"`
	RecordedRaw         *string         `json:"-"`
	RecordedBilled      *string         `json:"-"`
	RecordedClamp       string          `json:"-"`
	CandidateRaw        *string         `json:"-"`
	CandidateClamp      string          `json:"-"`
	CandidateExpression string          `json:"-"`
	FallbackVerdict     ScenarioVerdict `json:"-"`
	FallbackWhy         string          `json:"-"`
}
