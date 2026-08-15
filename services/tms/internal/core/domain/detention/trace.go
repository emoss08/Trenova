package detention

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// TraceStepKind names each stage of the derivation so the UI can render a
// waterfall and the customer-facing receipt can cite the governing term.
type TraceStepKind string

const (
	TraceStepPolicyResolved TraceStepKind = "PolicyResolved"
	TraceStepClockStart     TraceStepKind = "ClockStart"
	TraceStepLateArrival    TraceStepKind = "LateArrival"
	TraceStepClockStop      TraceStepKind = "ClockStop"
	TraceStepRawDwell       TraceStepKind = "RawDwell"
	TraceStepFreeTime       TraceStepKind = "FreeTime"
	TraceStepMinimum        TraceStepKind = "MinimumThreshold"
	TraceStepRounding       TraceStepKind = "Rounding"
	TraceStepMinutesCap     TraceStepKind = "MinutesCap"
	TraceStepTier           TraceStepKind = "TierApplied"
	TraceStepFlatRate       TraceStepKind = "FlatRateApplied"
	TraceStepAmountCap      TraceStepKind = "AmountCap"
	TraceStepLayover        TraceStepKind = "LayoverConversion"
	TraceStepNotification   TraceStepKind = "NotificationGate"
	TraceStepDriverPay      TraceStepKind = "DriverPay"
	TraceStepFinal          TraceStepKind = "Final"
)

// TraceStep is one line of the calculation receipt. Every step carries both the
// machine values and a sentence a billing clerk can paste into an email.
type TraceStep struct {
	Kind        TraceStepKind    `json:"kind"`
	Label       string           `json:"label"`
	Detail      string           `json:"detail"`
	Term        string           `json:"term,omitempty"`
	Minutes     *int32           `json:"minutes,omitempty"`
	Amount      *decimal.Decimal `json:"amount,omitempty"`
	Timestamp   *int64           `json:"timestamp,omitempty"`
	IsReduction bool             `json:"isReduction"`
}

// CalculationTrace is the complete, replayable derivation of a detention
// charge. Storing it turns "your invoice is wrong" from an argument into a
// document.
type CalculationTrace struct {
	Steps         []TraceStep `json:"steps"`
	EngineVer     string      `json:"engineVersion"`
	ComputedAt    int64       `json:"computedAt"`
	Deterministic bool        `json:"deterministic"`
}

const EngineVersion = "1.0.0"

func NewTrace(computedAt int64) *CalculationTrace {
	return &CalculationTrace{
		Steps:         make([]TraceStep, 0, 12),
		EngineVer:     EngineVersion,
		ComputedAt:    computedAt,
		Deterministic: true,
	}
}

func (t *CalculationTrace) Add(step TraceStep) {
	if t == nil {
		return
	}
	t.Steps = append(t.Steps, step)
}

func (t *CalculationTrace) AddMinutes(
	kind TraceStepKind,
	label, detail, term string,
	minutes int32,
	isReduction bool,
) {
	m := minutes
	t.Add(TraceStep{
		Kind:        kind,
		Label:       label,
		Detail:      detail,
		Term:        term,
		Minutes:     &m,
		IsReduction: isReduction,
	})
}

func (t *CalculationTrace) AddAmount(
	kind TraceStepKind,
	label, detail, term string,
	amount decimal.Decimal,
	isReduction bool,
) {
	a := amount
	t.Add(TraceStep{
		Kind:        kind,
		Label:       label,
		Detail:      detail,
		Term:        term,
		Amount:      &a,
		IsReduction: isReduction,
	})
}

func (t *CalculationTrace) AddTimestamp(
	kind TraceStepKind,
	label, detail, term string,
	timestamp int64,
) {
	ts := timestamp
	t.Add(TraceStep{
		Kind:      kind,
		Label:     label,
		Detail:    detail,
		Term:      term,
		Timestamp: &ts,
	})
}

// Receipt renders the trace as plain text suitable for pasting into a dispute
// response. Each line names the term that produced it.
func (t *CalculationTrace) Receipt() string {
	if t == nil || len(t.Steps) == 0 {
		return ""
	}

	var b strings.Builder
	for _, step := range t.Steps {
		b.WriteString("• ")
		b.WriteString(step.Label)
		if step.Detail != "" {
			b.WriteString(": ")
			b.WriteString(step.Detail)
		}
		if step.Term != "" {
			fmt.Fprintf(&b, " [%s]", step.Term)
		}
		b.WriteString("\n")
	}

	return b.String()
}
