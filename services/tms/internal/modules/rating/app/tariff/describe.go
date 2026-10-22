package tariff

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
)

const maxDescribedFailures = 5

type SourcePosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func PositionOf(source string, offset int) SourcePosition {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	position := SourcePosition{Line: 1, Column: 1}
	for _, r := range source[:offset] {
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
	if at == nil || at.From < 0 || at.To > len(source) || at.From >= at.To {
		return ""
	}
	text := source[at.From:at.To]
	if len(text) <= maxExcerptBytes {
		return text
	}
	cut := maxExcerptBytes
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + "…"
}

type TestOutcome struct {
	Name     string          `json:"name"`
	Status   string          `json:"status"`
	At       *SourcePosition `json:"at,omitempty"`
	Failures []string        `json:"failures,omitempty"`
	Error    string          `json:"error,omitempty"`
}

type TestSummary struct {
	Total  int           `json:"total"`
	Passed int           `json:"passed"`
	Error  string        `json:"error,omitempty"`
	Tests  []TestOutcome `json:"tests"`
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
		outcome := TestOutcome{
			Name:   test.Name,
			Status: string(test.Status),
			At:     positionPtr(source, &test.Range),
		}
		if test.Status == rules.TariffTestPassed {
			summary.Passed++
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
		summary.Tests = append(summary.Tests, outcome)
	}
	return summary
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
