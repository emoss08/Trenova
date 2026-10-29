package tariffverdict

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailingTests_DescribesEachFailureWithItsCounterexample(t *testing.T) {
	t.Parallel()

	run := &rules.TariffTestRun{Tests: []rules.TariffTestResult{
		{Name: "passes", Status: rules.TariffTestPassed},
		{Name: "a plain test", Kind: rules.TariffTestKindTest, Status: rules.TariffTestFailed},
		{
			Name:   "divides",
			Status: rules.TariffTestError,
			Error: &rules.TariffEvalError{
				Message: wasmmodule.Message{Template: "Division by zero"},
			},
		},
		{
			Name:   "heavier never bills less",
			Kind:   rules.TariffTestKindProperty,
			Status: rules.TariffTestFailed,
			Counterexample: &rules.TariffCounterexample{
				Values: []rules.TariffVariedValue{{Target: "shipment.weight", Display: "999 lb"}},
				Monotone: &rules.TariffMonotone{
					Target:     "shipment.weight",
					Expression: "total",
					Before: rules.TariffMonotoneSide{
						At:     rules.TariffShownValue{Display: "999 lb"},
						Result: rules.TariffShownValue{Display: "$310.00"},
					},
					After: rules.TariffMonotoneSide{
						At:     rules.TariffShownValue{Display: "1,000 lb"},
						Result: rules.TariffShownValue{Display: "$250.00"},
					},
				},
			},
		},
		{
			Name:   "never below the minimum",
			Kind:   rules.TariffTestKindProperty,
			Status: rules.TariffTestFailed,
			Counterexample: &rules.TariffCounterexample{Values: []rules.TariffVariedValue{
				{Target: "shipment.distance", Display: "1 mi"},
				{Target: "input.rate", Display: "$0.50/mi"},
			}},
		},
		{
			Name:   "every weight rates",
			Kind:   rules.TariffTestKindProperty,
			Status: rules.TariffTestFailed,
			Counterexample: &rules.TariffCounterexample{
				Values:  []rules.TariffVariedValue{{Target: "shipment.weight", Display: "1 lb"}},
				Refusal: &rules.TariffRefusal{Text: "Too light"},
			},
		},
	}}

	assert.Equal(t, []string{
		"a plain test",
		"divides (Division by zero)",
		"heavier never bills less (total is $310.00 at shipment.weight = 999 lb but $250.00 at 1,000 lb)",
		"never below the minimum (fails at shipment.distance = 1 mi, input.rate = $0.50/mi)",
		"every weight rates (refuses to rate at shipment.weight = 1 lb: Too light)",
	}, FailingTests(i18n.Default, run))
	assert.Nil(t, FailingTests(i18n.Default, nil))
}

func TestFailingTests_NamesAtMostTen(t *testing.T) {
	t.Parallel()

	run := &rules.TariffTestRun{}
	for range 15 {
		run.Tests = append(
			run.Tests,
			rules.TariffTestResult{Name: "x", Status: rules.TariffTestFailed},
		)
	}
	assert.Len(t, FailingTests(i18n.Default, run), maxNamedFailures)
}

func TestSummarizeCoverage_PlacesUnreachedRegionsAfterNonASCIIText(t *testing.T) {
	t.Parallel()

	source := "tariff \"Café 🚚\"\n\nlet w = shipment.weight ?? 0 lb\n" +
		"if w > 999 lb {\n  charge \"Heavy °F\" = $50\n}\n"
	from := utf16Offset(source, "{\n  charge")
	charge := utf16Offset(source, "charge")
	summary := SummarizeCoverage(source, &rules.TariffCoverage{
		Regions: []rules.TariffRegion{
			{Kind: rules.TariffCoverageIf, Range: rules.TariffRange{From: from, To: from + 30}},
			{
				Kind:  rules.TariffCoverageCharge,
				Range: rules.TariffRange{From: charge, To: charge + 24},
			},
		},
		Reached: 0,
		Total:   2,
	})

	require.NotNil(t, summary)
	assert.False(t, summary.Complete())
	require.Len(t, summary.Unreached, 2)
	assert.Equal(t, SourcePosition{Line: 4, Column: 15}, summary.Unreached[0].At)
	assert.Equal(t, "if w > 999 lb {", summary.Unreached[0].Excerpt)
	assert.Equal(t, rules.TariffCoverageCharge, summary.Unreached[1].Kind)
	assert.Equal(t, `charge "Heavy °F" = $50`, summary.Unreached[1].Excerpt)
	assert.Nil(t, SummarizeCoverage(source, nil))
}

func TestSummarizeCoverage_ListsAtMostFiftyRegions(t *testing.T) {
	t.Parallel()

	source := strings.Repeat("charge \"A\" = $1\n", 60)
	coverage := &rules.TariffCoverage{Total: 60}
	for i := range 60 {
		coverage.Regions = append(coverage.Regions, rules.TariffRegion{
			Kind:  rules.TariffCoverageCharge,
			Range: rules.TariffRange{From: i * 16, To: i*16 + 15},
		})
	}
	summary := SummarizeCoverage(source, coverage)
	assert.Len(t, summary.Unreached, maxUnreachedShown)
	assert.Equal(t, 10, summary.More)
}

func TestCoverageCheck_WarnsButNeverFails(t *testing.T) {
	t.Parallel()

	partial := &CoverageSummary{Reached: 18, Total: 21}
	assert.Equal(t, CoverageVerdict{
		Status: VerdictWarn,
		Label:  "Tests reach every region",
		Detail: "Tests reach 18 of 21 regions; no test runs the rest",
	}, CoverageCheck(i18n.Default, true, partial))
	assert.Equal(t, "Tests reach all 21 regions",
		CoverageCheck(i18n.Default, true, &CoverageSummary{Reached: 21, Total: 21}).Detail)
	assert.Equal(t, VerdictPass,
		CoverageCheck(i18n.Default, true, &CoverageSummary{}).Status)
	assert.Equal(t, VerdictWarn, CoverageCheck(i18n.Default, false, nil).Status)
	assert.Equal(t, VerdictWarn, CoverageCheck(i18n.Default, true, nil).Status)
}
