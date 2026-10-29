package tariffverdict

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxDescribedFailures = 5

type SourcePosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func PositionOf(source string, offset int) SourcePosition {
	position := SourcePosition{Line: 1, Column: 1}
	for _, r := range source[:stringutils.ByteOffsetOfUTF16(source, offset)] {
		if r == '\n' {
			position.Line++
			position.Column = 1
			continue
		}
		position.Column++
	}
	return position
}

func positionPtr(source string, at *rules.TariffRange) *SourcePosition {
	if at == nil {
		return nil
	}
	position := PositionOf(source, at.From)
	return &position
}

type Finding struct {
	Severity string          `json:"severity"`
	Code     string          `json:"code"`
	Message  string          `json:"message"`
	At       *SourcePosition `json:"at,omitempty"`
	Excerpt  string          `json:"excerpt,omitempty"`
	Module   string          `json:"module,omitempty"`
	Path     string          `json:"path,omitempty"`
	Fixes    []string        `json:"fixes,omitempty"`
}

func Findings(locale i18n.Locale, source string, diagnostics []rules.TariffDiagnostic) []Finding {
	out := make([]Finding, 0, len(diagnostics))
	for i := range diagnostics {
		diagnostic := &diagnostics[i]
		finding := Finding{
			Severity: diagnostic.Severity,
			Code:     diagnostic.Code,
			Message:  diagnostic.Message.Localize(locale),
			Module:   diagnostic.Module,
			Path:     diagnostic.Path,
		}
		if diagnostic.Module == "" {
			finding.At = positionPtr(source, diagnostic.Range)
			finding.Excerpt = excerpt(source, diagnostic.Range)
		}
		for j := range diagnostic.Fixes {
			finding.Fixes = append(finding.Fixes, diagnostic.Fixes[j].Label.Localize(locale))
		}
		out = append(out, finding)
	}
	return out
}

const maxExcerptBytes = 120

func excerpt(source string, at *rules.TariffRange) string {
	if at == nil {
		return ""
	}
	from := stringutils.ByteOffsetOfUTF16(source, at.From)
	to := stringutils.ByteOffsetOfUTF16(source, at.To)
	if from >= to {
		return ""
	}
	text := source[from:to]
	if len(text) <= maxExcerptBytes {
		return text
	}
	cut := maxExcerptBytes
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + "…"
}

type MonotoneSide struct {
	At     string `json:"at"`
	Result string `json:"result"`
}

type MonotoneFailure struct {
	Target     string       `json:"target"`
	Expression string       `json:"expression"`
	Direction  string       `json:"direction"`
	Before     MonotoneSide `json:"before"`
	After      MonotoneSide `json:"after"`
}

type VariedValue struct {
	Target string `json:"target"`
	Value  string `json:"value"`
}

type Counterexample struct {
	Values   []VariedValue    `json:"values"`
	Monotone *MonotoneFailure `json:"monotone,omitempty"`
	Refusal  string           `json:"refusal,omitempty"`
	Test     string           `json:"test"`
}

type TestOutcome struct {
	Name           string          `json:"name"`
	Kind           string          `json:"kind"`
	Status         string          `json:"status"`
	At             *SourcePosition `json:"at,omitempty"`
	Cases          int             `json:"cases,omitempty"`
	Skipped        int             `json:"skipped,omitempty"`
	Failures       []string        `json:"failures,omitempty"`
	Error          string          `json:"error,omitempty"`
	Counterexample *Counterexample `json:"counterexample,omitempty"`
}

type CoverageReport struct {
	*CoverageSummary
	Summary string `json:"summary"`
}

type TestSummary struct {
	Total    int             `json:"total"`
	Passed   int             `json:"passed"`
	Error    string          `json:"error,omitempty"`
	Tests    []TestOutcome   `json:"tests"`
	Coverage *CoverageReport `json:"coverage,omitempty"`
}

func SummarizeTests(locale i18n.Locale, source string, run *rules.TariffTestRun) TestSummary {
	summary := TestSummary{Tests: []TestOutcome{}}
	if run == nil {
		return summary
	}
	if run.Error != nil {
		summary.Error = run.Error.Message.Localize(locale)
	}
	for i := range run.Tests {
		test := &run.Tests[i]
		summary.Total++
		if test.Status == rules.TariffTestPassed {
			summary.Passed++
		}
		summary.Tests = append(summary.Tests, testOutcome(locale, source, test))
	}
	summary.Coverage = coverageReport(locale, source, run.Coverage)
	return summary
}

func testOutcome(locale i18n.Locale, source string, test *rules.TariffTestResult) TestOutcome {
	outcome := TestOutcome{
		Name:   test.Name,
		Kind:   string(test.Kind),
		Status: string(test.Status),
		At:     positionPtr(source, &test.Range),
	}
	if test.IsProperty() {
		outcome.Cases = test.Cases
		outcome.Skipped = test.Skipped
	}
	if test.Error != nil {
		outcome.Error = test.Error.Message.Localize(locale)
	}
	for j := range test.Failures {
		if j == maxDescribedFailures {
			break
		}
		outcome.Failures = append(outcome.Failures, failureText(locale, &test.Failures[j]))
	}
	if test.Counterexample != nil {
		outcome.Counterexample = counterexampleOf(locale, test.Counterexample)
	}
	return outcome
}

func counterexampleOf(locale i18n.Locale, example *rules.TariffCounterexample) *Counterexample {
	out := &Counterexample{
		Values: make([]VariedValue, 0, len(example.Values)),
		Test:   example.Test,
	}
	for i := range example.Values {
		out.Values = append(out.Values, VariedValue{
			Target: example.Values[i].Target,
			Value:  example.Values[i].Display,
		})
	}
	if monotone := example.Monotone; monotone != nil {
		out.Monotone = &MonotoneFailure{
			Target:     monotone.Target,
			Expression: monotone.Expression,
			Direction:  string(monotone.Direction),
			Before: MonotoneSide{
				At:     monotone.Before.At.Display,
				Result: monotone.Before.Result.Display,
			},
			After: MonotoneSide{
				At:     monotone.After.At.Display,
				Result: monotone.After.Result.Display,
			},
		}
	}
	if example.Refusal != nil {
		out.Refusal = example.Refusal.Localize(locale)
	}
	return out
}

func coverageReport(
	locale i18n.Locale,
	source string,
	coverage *rules.TariffCoverage,
) *CoverageReport {
	summary := SummarizeCoverage(source, coverage)
	if summary == nil {
		return nil
	}
	return &CoverageReport{
		CoverageSummary: summary,
		Summary: i18n.Translate(
			locale,
			"Tests reach {0} of {1} regions",
			summary.Reached,
			summary.Total,
		),
	}
}

func failureText(locale i18n.Locale, failure *rules.TariffTestFailure) string {
	var text strings.Builder
	text.WriteString(failure.Message.Localize(locale))
	for _, value := range failure.Values {
		text.WriteString("; ")
		text.WriteString(value.Display)
	}
	return text.String()
}
