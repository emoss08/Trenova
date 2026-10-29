package tariffverdict

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func utf16Offset(source, text string) int {
	return len(utf16.Encode([]rune(source[:strings.Index(source, text)])))
}

func TestPositionOf_CountsLinesAndCharactersFromUTF16Offsets(t *testing.T) {
	t.Parallel()

	source := "tariff \"Süd 🚚\"\n\ninput rate: Money/mi\n"
	assert.Equal(t, SourcePosition{Line: 1, Column: 1}, PositionOf(source, 0))
	assert.Equal(t, SourcePosition{Line: 1, Column: 12}, PositionOf(source, 11))
	assert.Equal(t, SourcePosition{Line: 1, Column: 14}, PositionOf(source, 14),
		"the emoji is two code units and one character")
	assert.Equal(
		t,
		SourcePosition{Line: 3, Column: 7},
		PositionOf(source, utf16Offset(source, "rate")),
	)
	assert.Equal(t, SourcePosition{Line: 4, Column: 1}, PositionOf(source, len(source)+10))
	assert.Equal(t, SourcePosition{Line: 1, Column: 1}, PositionOf(source, -3))
}

func TestFindings_ReadRangesAsUTF16AfterNonASCIIText(t *testing.T) {
	t.Parallel()

	source := "tariff \"Café 🚚\"\n\nlet t = -5 °F\nlet bad = nope\n"
	from := utf16Offset(source, "nope")
	findings := Findings(i18n.Locale("en"), source, []rules.TariffDiagnostic{{
		Severity: rules.SeverityError,
		Code:     "unknown-name",
		Message:  wasmmodule.Message{Template: "Nothing is named {0}", Args: []any{"nope"}},
		Range:    &rules.TariffRange{From: from, To: from + 4},
	}, {
		Severity: rules.SeverityWarning,
		Code:     "unused",
		Message:  wasmmodule.Message{Template: "Unused"},
		Range: &rules.TariffRange{
			From: utf16Offset(source, "°F"),
			To:   utf16Offset(source, "°F") + 2,
		},
	}, {
		Severity: rules.SeverityWarning,
		Code:     "doc",
		Message:  wasmmodule.Message{Template: "The name"},
		Range: &rules.TariffRange{
			From: utf16Offset(source, "Café"),
			To:   utf16Offset(source, "\"\n"),
		},
	}})

	require.Len(t, findings, 3)
	assert.Equal(t, "nope", findings[0].Excerpt)
	assert.Equal(t, &SourcePosition{Line: 4, Column: 11}, findings[0].At)
	assert.Equal(t, "°F", findings[1].Excerpt)
	assert.Equal(t, "Café 🚚", findings[2].Excerpt)
}

func TestFindings_PlaceSourceFindingsAndLeaveLibraryOnesUnplaced(t *testing.T) {
	t.Parallel()

	source := "tariff \"Per Mile\"\n\ncharge \"" + strings.Repeat("x", 200) + "\" = $1\n"
	findings := Findings(i18n.Locale("en"), source, []rules.TariffDiagnostic{
		{
			Severity: rules.SeverityError,
			Code:     "unknown-code",
			Message:  wasmmodule.Message{Template: "There is no code {0}", Args: []any{"HAZ"}},
			Range:    &rules.TariffRange{From: 19, To: len(source) - 1},
			Fixes:    []rules.TariffFix{{Label: wasmmodule.Message{Template: "Remove the code"}}},
		},
		{
			Severity: rules.SeverityError,
			Code:     "type",
			Message:  wasmmodule.Message{Template: "A weight is not a distance"},
			Range:    &rules.TariffRange{From: 3, To: 9},
			Module:   "org.accessorials",
		},
		{
			Severity: rules.SeverityError,
			Code:     "input-value",
			Message:  wasmmodule.Message{Template: "Expected money"},
			Path:     "rule.rate",
		},
	})

	require.Len(t, findings, 3)
	assert.Equal(t, &SourcePosition{Line: 3, Column: 1}, findings[0].At)
	assert.Equal(t, "There is no code HAZ", findings[0].Message)
	assert.Equal(t, []string{"Remove the code"}, findings[0].Fixes)
	assert.True(t, strings.HasSuffix(findings[0].Excerpt, "…"))
	assert.LessOrEqual(t, len(findings[0].Excerpt), maxExcerptBytes+len("…"))
	assert.Nil(t, findings[1].At, "a library's range is in the library, not this source")
	assert.Empty(t, findings[1].Excerpt)
	assert.Equal(t, "org.accessorials", findings[1].Module)
	assert.Equal(t, "rule.rate", findings[2].Path)
	assert.Nil(t, findings[2].At)
}

func TestSummarizeTests_CountsAndBoundsFailures(t *testing.T) {
	t.Parallel()

	failures := make([]rules.TariffTestFailure, 0, 8)
	for range 8 {
		failures = append(failures, rules.TariffTestFailure{
			Message: wasmmodule.Message{Template: "total was wrong"},
			Values:  []rules.TariffTestValue{{Display: "$200"}},
		})
	}
	summary := SummarizeTests(
		i18n.Locale("en"),
		"tariff \"T\"\ntest \"a\" {}\n",
		&rules.TariffTestRun{
			Tests: []rules.TariffTestResult{
				{Name: "a", Status: rules.TariffTestPassed, Range: rules.TariffRange{From: 11}},
				{Name: "b", Status: rules.TariffTestFailed, Failures: failures},
				{
					Name:   "c",
					Status: rules.TariffTestError,
					Error: &rules.TariffEvalError{
						Code:    "none",
						Message: wasmmodule.Message{Template: "no weight"},
					},
				},
			},
		},
	)

	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 1, summary.Passed)
	assert.Equal(t, &SourcePosition{Line: 2, Column: 1}, summary.Tests[0].At)
	assert.Len(t, summary.Tests[1].Failures, maxDescribedFailures)
	assert.Equal(t, "total was wrong; $200", summary.Tests[1].Failures[0])
	assert.Equal(t, "no weight", summary.Tests[2].Error)

	empty := SummarizeTests(i18n.Locale("en"), "", nil)
	assert.NotNil(t, empty.Tests)
	assert.Zero(t, empty.Total)
}

func TestSummarizeTests_ReportsPropertiesAndCoverage(t *testing.T) {
	t.Parallel()

	source := "tariff \"Café\"\n\nlet w = shipment.weight ?? 0 lb\n" +
		"if w > 999 lb {\n  charge \"Heavy\" = -$50\n}\ncharge \"Base\" = $200\n\n" +
		"property \"heavier never bills less\" {\n" +
		"  vary shipment.weight in 1 lb..2_000 lb\n" +
		"  expect total nondecreasing in shipment.weight\n}\n\n" +
		"property \"light loads bill the base\" {\n" +
		"  vary shipment.weight in 1 lb..900 lb\n  expect total == $200\n}\n"
	run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
		Source:  source,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)

	summary := SummarizeTests(i18n.Default, source, run)
	require.Len(t, summary.Tests, 2)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.Passed)

	failed := summary.Tests[0]
	assert.Equal(t, "property", failed.Kind)
	assert.Equal(t, "failed", failed.Status)
	assert.Positive(t, failed.Cases)
	require.NotNil(t, failed.Counterexample)
	assert.Equal(t,
		[]VariedValue{{Target: "shipment.weight", Value: "999 lb"}},
		failed.Counterexample.Values,
	)
	require.NotNil(t, failed.Counterexample.Monotone)
	assert.Equal(t, MonotoneFailure{
		Target:     "shipment.weight",
		Expression: "total",
		Direction:  "nondecreasing",
		Before:     MonotoneSide{At: "999 lb", Result: "$200.00"},
		After:      MonotoneSide{At: "1000 lb", Result: "$150.00"},
	}, *failed.Counterexample.Monotone)
	assert.Contains(t, failed.Counterexample.Test, "given shipment.weight = 999 lb")

	passed := summary.Tests[1]
	assert.Equal(t, "passed", passed.Status)
	assert.Nil(t, passed.Counterexample)

	require.NotNil(t, summary.Coverage)
	assert.Equal(t, summary.Coverage.Total, summary.Coverage.Reached)
	assert.Equal(t,
		"Tests reach "+strconv.Itoa(summary.Coverage.Reached)+" of "+
			strconv.Itoa(summary.Coverage.Total)+" regions",
		summary.Coverage.Summary,
	)
	assert.Empty(t, summary.Coverage.Unreached)
}
