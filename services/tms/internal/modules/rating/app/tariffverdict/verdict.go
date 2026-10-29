package tariffverdict

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxNamedFailures  = 10
	maxUnreachedShown = 50
)

func FailingTests(locale i18n.Locale, run *rules.TariffTestRun) []string {
	if run == nil {
		return nil
	}
	out := make([]string, 0, min(len(run.Tests), maxNamedFailures))
	for i := range run.Tests {
		test := &run.Tests[i]
		if test.Status == rules.TariffTestPassed {
			continue
		}
		if len(out) == maxNamedFailures {
			break
		}
		out = append(out, DescribeFailure(locale, test))
	}
	return out
}

func DescribeFailure(locale i18n.Locale, test *rules.TariffTestResult) string {
	if example := test.Counterexample; example != nil {
		return describeCounterexample(locale, test.Name, example)
	}
	if test.Error != nil {
		return i18n.Translate(locale, "{0} ({1})", test.Name, test.Error.Message.Localize(locale))
	}
	return test.Name
}

func describeCounterexample(
	locale i18n.Locale,
	name string,
	example *rules.TariffCounterexample,
) string {
	switch {
	case example.Monotone != nil:
		monotone := example.Monotone
		return i18n.Translate(
			locale,
			"{0} ({1} is {2} at {3} = {4} but {5} at {6})",
			name,
			monotone.Expression,
			monotone.Before.Result.Display,
			monotone.Target,
			monotone.Before.At.Display,
			monotone.After.Result.Display,
			monotone.After.At.Display,
		)
	case example.Refusal != nil:
		return i18n.Translate(
			locale,
			"{0} (refuses to rate at {1}: {2})",
			name,
			VariedValues(example.Values),
			example.Refusal.Localize(locale),
		)
	case len(example.Values) > 0:
		return i18n.Translate(locale, "{0} (fails at {1})", name, VariedValues(example.Values))
	default:
		return name
	}
}

func VariedValues(values []rules.TariffVariedValue) string {
	parts := make([]string, 0, len(values))
	for i := range values {
		parts = append(parts, values[i].Target+" = "+values[i].Display)
	}
	return strings.Join(parts, ", ")
}

type UnreachedRegion struct {
	Kind    rules.TariffCoverageKind `json:"kind"`
	Range   rules.TariffRange        `json:"range"`
	At      SourcePosition           `json:"at"`
	Excerpt string                   `json:"excerpt,omitempty"`
}

type CoverageSummary struct {
	Reached   int               `json:"reached"`
	Total     int               `json:"total"`
	Unreached []UnreachedRegion `json:"unreached"`
	More      int               `json:"more,omitempty"`
}

func (s *CoverageSummary) Complete() bool {
	return s.Reached == s.Total
}

func SummarizeCoverage(source string, coverage *rules.TariffCoverage) *CoverageSummary {
	if coverage == nil {
		return nil
	}
	unreached := coverage.Unreached()
	out := &CoverageSummary{
		Reached:   coverage.Reached,
		Total:     coverage.Total,
		Unreached: make([]UnreachedRegion, 0, min(len(unreached), maxUnreachedShown)),
	}
	for i := range unreached {
		if i == maxUnreachedShown {
			out.More = len(unreached) - maxUnreachedShown
			break
		}
		region := &unreached[i]
		out.Unreached = append(out.Unreached, UnreachedRegion{
			Kind:    region.Kind,
			Range:   region.Range,
			At:      PositionOf(source, region.Range.From),
			Excerpt: lineAt(source, region.Range.From),
		})
	}
	return out
}

func lineAt(source string, offset int) string {
	at := stringutils.ByteOffsetOfUTF16(source, offset)
	start := strings.LastIndexByte(source[:at], '\n') + 1
	end := len(source)
	if cut := strings.IndexByte(source[at:], '\n'); cut >= 0 {
		end = at + cut
	}
	line := strings.TrimSpace(source[start:end])
	if len(line) <= maxExcerptBytes {
		return line
	}
	cut := maxExcerptBytes
	for cut > 0 && !utf8.RuneStart(line[cut]) {
		cut--
	}
	return line[:cut] + "…"
}

const (
	VerdictPass = "pass"
	VerdictWarn = "warn"
)

type CoverageVerdict struct {
	Status string
	Label  string
	Detail string
}

func CoverageCheck(locale i18n.Locale, checks bool, coverage *CoverageSummary) CoverageVerdict {
	verdict := CoverageVerdict{
		Status: VerdictWarn,
		Label:  i18n.Translate(locale, "Tests reach every region"),
	}
	switch {
	case !checks:
		verdict.Detail = i18n.Translate(locale, "Coverage is measured once the source checks")
	case coverage == nil:
		verdict.Detail = i18n.Translate(locale, "Coverage could not be measured")
	case coverage.Total == 0:
		verdict.Status = VerdictPass
		verdict.Detail = i18n.Translate(
			locale,
			"The file has no branches, charges or refusals to reach",
		)
	case coverage.Complete():
		verdict.Status = VerdictPass
		verdict.Detail = i18n.Translate(
			locale,
			"Tests reach {0, plural, one {the file's one region} other {all # regions}}",
			coverage.Total,
		)
	default:
		verdict.Detail = i18n.Translate(
			locale,
			"Tests reach {0} of {1} regions; no test runs the rest",
			coverage.Reached,
			coverage.Total,
		)
	}
	return verdict
}
