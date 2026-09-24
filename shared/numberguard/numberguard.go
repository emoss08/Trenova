package numberguard

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
)

// The guard's tolerances.
//
// A model told that on-time delivery is 82.4% may reasonably write "about 82%"
// or "82.4%". Both are the same fact rendered for a reader, and rejecting them
// would leave every insight wearing the detector's stiffer wording for no gain.
// What must be rejected is a number nobody computed — "costing you $48,000" when
// the finding says $12,400 — because that is the sentence a person acts on.
const (
	// roundingTolerance is how far a cited number may sit from a computed one and
	// still be read as the same figure. It is proportional, so it scales from
	// percentages to six-figure sums.
	roundingTolerance = 0.005
	// smallNumberCeiling is the value below which a number is treated as ordinary
	// prose rather than a citation. "the top 3 customers", "a second driver", "one
	// of the stops" are all sentence furniture, and a model cannot mislead anyone
	// about money with them.
	smallNumberCeiling = 24
)

// NumberCheck is the outcome of scanning generated prose for figures.
type NumberCheck struct {
	// OK is false when the prose cites a number the finding does not support.
	OK bool
	// Unsupported lists the offending figures exactly as they appeared, so a log
	// line can say what the model claimed rather than only that it failed.
	Unsupported []string
}

// CheckNumbers reports whether generated prose only cites figures the detector
// actually computed.
//
// This is the guard that lets narration be trusted at all. The model is handed
// the finding's numbers and asked to write a sentence about them; nothing stops
// it writing a different, more dramatic number, and a wrong figure on a card
// headed "insight" is worse than no card, because it will be repeated in a
// meeting. Any figure in the prose that does not match something computed —
// within rounding, and ignoring small conversational numbers — fails the check,
// and the caller falls back to the detector's own wording.
//
// Supported values are the metric values, their baselines, and any extra
// figures the caller states are legitimate, such as the window length.
func CheckNumbers(prose string, supported []decimal.Decimal) NumberCheck {
	cited := extractNumbers(prose)
	if len(cited) == 0 {
		return NumberCheck{OK: true}
	}

	unsupported := make([]string, 0, len(cited))
	for _, citation := range cited {
		if !isSupported(citation.value, supported) {
			unsupported = append(unsupported, citation.text)
		}
	}

	return NumberCheck{OK: len(unsupported) == 0, Unsupported: unsupported}
}

type citation struct {
	text  string
	value decimal.Decimal
}

func isSupported(value decimal.Decimal, supported []decimal.Decimal) bool {
	// Small numbers are sentence furniture rather than claims: "the top 3
	// customers" cites nothing, and demanding a detector compute a 3 to permit
	// the phrase would reject perfectly honest prose.
	if value.Abs().LessThanOrEqual(decimal.NewFromInt(smallNumberCeiling)) {
		return true
	}

	for _, candidate := range supported {
		if withinTolerance(value, candidate) {
			return true
		}
		// A model given 12440.50 will often write "$12,400" or "12.4". The first
		// is covered by rounding; the second is the same figure in thousands, and
		// a detector reporting money in dollars should not force prose to spell
		// out every digit.
		if withinTolerance(value.Mul(decimal.NewFromInt(1000)), candidate) {
			return true
		}
	}

	return false
}

func withinTolerance(value, candidate decimal.Decimal) bool {
	if value.Equal(candidate) {
		return true
	}

	difference := value.Sub(candidate).Abs()
	scale := decimal.Max(value.Abs(), candidate.Abs())
	if scale.IsZero() {
		return difference.IsZero()
	}

	// A proportional band covers "about 82%" for 82.4 and "$12,400" for 12,437
	// without admitting a figure that is simply different.
	return difference.Div(scale).LessThanOrEqual(decimal.NewFromFloat(roundingTolerance))
}

// extractNumbers pulls every figure out of prose.
//
// It reads the text rune by rune rather than with a regular expression because
// the shapes that matter are "$12,400", "82.4%", "1,234.56" and "12.4k", and a
// pattern covering those without also matching the "30" inside an identifier
// ends up longer and harder to reason about than the scan.
func extractNumbers(prose string) []citation {
	runes := []rune(prose)
	citations := make([]citation, 0, 8)

	for index := 0; index < len(runes); index++ {
		if !unicode.IsDigit(runes[index]) {
			continue
		}

		start := index
		for index < len(runes) && isNumericRune(runes[index]) {
			index++
		}

		raw := string(runes[start:index])
		// A trailing separator belongs to the sentence, not the number: "12,400,"
		// at the end of a clause is the figure plus a comma.
		raw = strings.TrimRight(raw, ".,")

		value, err := parseNumber(raw, runes, index)
		if err != nil {
			continue
		}

		citations = append(citations, citation{text: raw, value: value})
	}

	return citations
}

func isNumericRune(r rune) bool {
	return unicode.IsDigit(r) || r == ',' || r == '.'
}

func parseNumber(raw string, runes []rune, after int) (decimal.Decimal, error) {
	normalized := strings.ReplaceAll(raw, ",", "")

	value, err := decimal.NewFromString(normalized)
	if err != nil {
		return decimal.Zero, err
	}

	// "12.4k" and "1.2M" are the same claim as the full figure, and a model
	// writing for a dashboard uses them constantly.
	if after < len(runes) {
		switch runes[after] {
		case 'k', 'K':
			return value.Mul(decimal.NewFromInt(1000)), nil
		case 'm', 'M':
			return value.Mul(decimal.NewFromInt(1_000_000)), nil
		}
	}

	return value, nil
}

// SupportedValues collects every figure prose may legitimately cite: each
// metric value, each baseline, and the extras the caller vouches for.
func SupportedValues(
	metricValues []decimal.Decimal,
	baselines []decimal.Decimal,
	extras ...decimal.Decimal,
) []decimal.Decimal {
	supported := make(
		[]decimal.Decimal,
		0,
		len(metricValues)+len(baselines)+len(extras)+len(metricValues),
	)
	supported = append(supported, metricValues...)
	supported = append(supported, baselines...)
	supported = append(supported, extras...)

	// A change between a value and its baseline is a figure the model is
	// positively encouraged to state — "fell 11 points" is the useful sentence —
	// and it is arithmetic on computed numbers rather than an invention.
	for index, value := range metricValues {
		if index < len(baselines) {
			supported = append(supported, value.Sub(baselines[index]).Abs())
		}
	}

	return supported
}

func SupportedFromText(texts ...string) []decimal.Decimal {
	supported := make([]decimal.Decimal, 0, len(texts)*4)
	seen := make(map[string]struct{}, len(texts)*4)
	for _, text := range texts {
		if text == "" {
			continue
		}
		for _, cited := range extractNumbers(text) {
			key := cited.value.String()
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			supported = append(supported, cited.value)
		}
	}

	return supported
}

// FormatForPrompt renders a value the way the model should cite it, so the
// prompt and the guard agree on what the number looks like.
func FormatForPrompt(value decimal.Decimal) string {
	rounded := value.Round(2)
	if rounded.Equal(rounded.Truncate(0)) {
		return strconv.FormatInt(rounded.IntPart(), 10)
	}

	return rounded.String()
}
