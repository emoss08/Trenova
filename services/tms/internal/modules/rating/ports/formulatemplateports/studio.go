package formulatemplateports

import (
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
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
	RefusedCount        int             `json:"refusedCount"`
	AdjustedCount       int             `json:"adjustedCount"`
	CurrentTotal        decimal.Decimal `json:"currentTotal"`
	CandidateTotal      decimal.Decimal `json:"candidateTotal"`
	TotalDelta          decimal.Decimal `json:"totalDelta"`
	TotalDeltaPct       decimal.Decimal `json:"totalDeltaPct"`
	MaxIncrease         decimal.Decimal `json:"maxIncrease"`
	MaxDecrease         decimal.Decimal `json:"maxDecrease"`
}

type PreviewError struct {
	Code    string             `json:"code"`
	Message string             `json:"message"`
	Range   *rules.TariffRange `json:"range"`
}

type PreviewResult struct {
	Valid       bool                     `json:"valid"`
	Diagnostics []rules.TariffDiagnostic `json:"diagnostics"`
	Inputs      []tarifftypes.InputDecl  `json:"inputs"`
	Imports     []tarifftypes.Import     `json:"imports"`
	Charges     []rules.TariffChargeRef  `json:"charges"`
	Bill        *rules.TariffBill        `json:"bill"`
	RatedBill   *tarifftypes.Bill        `json:"-"`
	Refusal     *rules.TariffRefusal     `json:"refusal"`
	Error       *PreviewError            `json:"error"`
	Trace       rules.RawJSON            `json:"trace"`
	Lookups     rules.RawJSON            `json:"lookups"`
	Tests       []rules.TariffTestResult `json:"tests"`
}

func (r *PreviewResult) FailingTests() []string {
	names := make([]string, 0, len(r.Tests))
	for i := range r.Tests {
		if r.Tests[i].Status != rules.TariffTestPassed {
			names = append(names, r.Tests[i].Name)
		}
	}
	return names
}
