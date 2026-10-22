package tariff

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPositionOf_CountsLinesAndCharacters(t *testing.T) {
	t.Parallel()

	source := "tariff \"Süd\"\n\ninput rate: Money/mi\n"
	assert.Equal(t, SourcePosition{Line: 1, Column: 1}, PositionOf(source, 0))
	assert.Equal(t, SourcePosition{Line: 1, Column: 12}, PositionOf(source, 12))
	assert.Equal(t, SourcePosition{Line: 3, Column: 7}, PositionOf(source, strings.Index(source, "rate")))
	assert.Equal(t, SourcePosition{Line: 4, Column: 1}, PositionOf(source, len(source)+10))
	assert.Equal(t, SourcePosition{Line: 1, Column: 1}, PositionOf(source, -3))
}

func TestFindings_PlaceSourceFindingsAndLeaveLibraryOnesUnplaced(t *testing.T) {
	t.Parallel()

	source := "tariff \"Per Mile\"\n\ncharge \"" + strings.Repeat("x", 200) + "\" = $1\n"
	findings := Findings(i18n.Locale("en"), source, []rules.TariffDiagnostic{
		{
			Severity: rules.SeverityError,
			Code:     "unknown-code",
			Message:  rules.Message{Template: "There is no code {0}", Args: []any{"HAZ"}},
			Range:    &rules.TariffRange{From: 19, To: len(source) - 1},
			Fixes:    []rules.TariffFix{{Label: rules.Message{Template: "Remove the code"}}},
		},
		{
			Severity: rules.SeverityError,
			Code:     "type",
			Message:  rules.Message{Template: "A weight is not a distance"},
			Range:    &rules.TariffRange{From: 3, To: 9},
			Module:   "org.accessorials",
		},
		{
			Severity: rules.SeverityError,
			Code:     "input-value",
			Message:  rules.Message{Template: "Expected money"},
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
			Message: rules.Message{Template: "total was wrong"},
			Values:  []rules.TariffTestValue{{Display: "$200"}},
		})
	}
	summary := SummarizeTests(i18n.Locale("en"), "tariff \"T\"\ntest \"a\" {}\n", &rules.TariffTestRun{
		Tests: []rules.TariffTestResult{
			{Name: "a", Status: rules.TariffTestPassed, Range: rules.TariffRange{From: 11}},
			{Name: "b", Status: rules.TariffTestFailed, Failures: failures},
			{
				Name:   "c",
				Status: rules.TariffTestError,
				Error:  &rules.TariffEvalError{Code: "none", Message: rules.Message{Template: "no weight"}},
			},
		},
	})

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
