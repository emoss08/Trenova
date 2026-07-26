package reportfmt

import (
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var currencySymbols = map[string]string{
	"USD": "$",
	"CAD": "C$",
	"MXN": "MX$",
	"EUR": "€",
	"GBP": "£",
	"AUD": "A$",
	"JPY": "¥",
}

var compactSuffixes = [...]struct {
	threshold decimal.Decimal
	suffix    string
}{
	{decimal.New(1, 12), "T"},
	{decimal.New(1, 9), "B"},
	{decimal.New(1, 6), "M"},
	{decimal.New(1, 3), "K"},
}

// String renders a scanned report value using the resolved display contract.
// The renderers (CSV, PDF, XLSX text fallback) and the JSON envelope all go
// through here so a column looks identical in every output format.
//
// A zero-value Resolved renders raw, lossless values: callers that never went
// through Resolve get the underlying data rather than an accidental format.
func (r *Resolved) String(value any, loc *time.Location) string {
	if value == nil {
		return r.NullText
	}
	if r.Style == StyleAuto {
		return rawString(value)
	}

	// A banded column scans the range's lower bound, never a value a reader
	// would recognize on its own, so it always renders as the range.
	if !r.Band.IsEmpty() {
		if number, ok := ruleValue(value); ok {
			return r.bandLabel(number)
		}
	}

	switch v := value.(type) {
	case decimal.Decimal:
		if r.Style == StyleDate {
			return r.date(time.Unix(v.IntPart(), 0).In(location(loc)))
		}
		return r.numeric(v)
	case int64:
		if r.Style == StyleDate {
			return r.date(time.Unix(v, 0).In(location(loc)))
		}
		return r.numeric(decimal.NewFromInt(v))
	case int:
		return r.String(int64(v), loc)
	case float64:
		return r.numeric(decimal.NewFromFloat(v))
	case bool:
		return r.boolean(v)
	case time.Time:
		return r.date(v.In(location(loc)))
	case string:
		return r.text(v)
	default:
		return r.text(stringify(v))
	}
}

func rawString(value any) string {
	switch v := value.(type) {
	case decimal.Decimal:
		return v.String()
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case string:
		return v
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return stringify(v)
	}
}

func location(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}
	return loc
}

func stringify(value any) string {
	if stringer, ok := value.(interface{ String() string }); ok {
		return stringer.String()
	}
	return ""
}

func (r *Resolved) text(value string) string {
	if value == "" {
		return r.NullText
	}
	if r.Prefix == "" && r.Suffix == "" {
		return value
	}
	return r.Prefix + value + r.Suffix
}

func (r *Resolved) boolean(value bool) string {
	switch r.BoolStyle {
	case BoolStyleTrueFalse:
		return pick(value, "True", "False")
	case BoolStyleOnOff:
		return pick(value, "On", "Off")
	case BoolStyleCheck:
		return pick(value, "✓", "✗")
	case BoolStyleOneZero:
		return pick(value, "1", "0")
	case BoolStyleAuto, BoolStyleYesNo:
		return pick(value, "Yes", "No")
	default:
		return pick(value, "Yes", "No")
	}
}

func pick(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func (r *Resolved) date(t time.Time) string {
	switch r.DateStyle {
	case DateStyleDate:
		return t.Format("2006-01-02")
	case DateStyleDateLong:
		return t.Format("Jan 2, 2006")
	case DateStyleTime:
		return t.Format("15:04")
	case DateStyleMonthYear:
		return t.Format("Jan 2006")
	case DateStyleQuarter:
		return "Q" + strconv.Itoa((int(t.Month())-1)/3+1) + " " + t.Format("2006")
	case DateStyleYear:
		return t.Format("2006")
	case DateStyleWeekday:
		return t.Format("Mon 2006-01-02")
	case DateStyleISO:
		return t.Format(time.RFC3339)
	case DateStyleAuto, DateStyleDateTime:
		return t.Format("2006-01-02 15:04")
	default:
		return t.Format("2006-01-02 15:04")
	}
}

func (r *Resolved) numeric(value decimal.Decimal) string {
	if r.Style == StyleDuration {
		return r.duration(value)
	}

	scaled := value
	if r.Style == StylePercent {
		scaled = scaled.Shift(2)
	}

	negative := scaled.IsNegative()
	magnitude := scaled.Abs()

	digits, compactSuffix := r.compact(magnitude)
	body := r.group(digits)

	var sb strings.Builder
	sb.Grow(len(body) + len(r.Prefix) + len(r.Suffix) + 6)
	sb.WriteString(r.Prefix)
	if r.Style == StyleCurrency {
		sb.WriteString(currencySymbol(r.Currency))
	}
	sb.WriteString(body)
	sb.WriteString(compactSuffix)
	if r.Style == StylePercent {
		sb.WriteString("%")
	}
	sb.WriteString(r.Suffix)

	return r.signed(sb.String(), negative)
}

func (r *Resolved) compact(magnitude decimal.Decimal) (digits, suffix string) {
	places := decimalPlaces(r.Decimals)
	if r.Notation == NotationCompact {
		for _, step := range compactSuffixes {
			if magnitude.GreaterThanOrEqual(step.threshold) {
				return magnitude.Div(step.threshold).StringFixed(places), step.suffix
			}
		}
	}
	return magnitude.StringFixed(places), ""
}

// decimalPlaces narrows the clamped 0..MaxDecimals range to the exponent type
// shopspring expects.
func decimalPlaces(decimals int) int32 {
	switch {
	case decimals < 0:
		return 0
	case decimals > MaxDecimals:
		return MaxDecimals
	default:
		return int32(decimals)
	}
}

func (r *Resolved) signed(body string, negative bool) string {
	if !negative || body == "" {
		return body
	}
	if r.Negative == NegativeParentheses {
		return "(" + body + ")"
	}
	return "-" + body
}

func (r *Resolved) group(digits string) string {
	if !r.Grouping {
		return digits
	}

	intPart, fracPart, hasFrac := strings.Cut(digits, ".")
	if len(intPart) <= 3 {
		return digits
	}

	var sb strings.Builder
	sb.Grow(len(digits) + len(intPart)/3)
	lead := len(intPart) % 3
	if lead > 0 {
		sb.WriteString(intPart[:lead])
	}
	for i := lead; i < len(intPart); i += 3 {
		if sb.Len() > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(intPart[i : i+3])
	}
	if hasFrac {
		sb.WriteByte('.')
		sb.WriteString(fracPart)
	}

	return sb.String()
}

func (r *Resolved) duration(value decimal.Decimal) string {
	seconds := value.Mul(decimal.NewFromInt(r.DurationUnit.seconds()))
	negative := seconds.IsNegative()
	total := seconds.Abs()

	var body string
	switch r.DurationStyle {
	case DurationStyleDecimal:
		hours := total.Div(decimal.NewFromInt(3600))
		body = r.group(hours.StringFixed(decimalPlaces(max(r.Decimals, 1)))) + " h"
	case DurationStyleCompact:
		body = compactDuration(total.IntPart(), false)
	case DurationStyleVerbose:
		body = compactDuration(total.IntPart(), true)
	case DurationStyleAuto, DurationStyleClock:
		body = clockDuration(total.IntPart())
	default:
		body = clockDuration(total.IntPart())
	}

	return r.signed(r.Prefix+body+r.Suffix, negative)
}

func clockDuration(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	var sb strings.Builder
	sb.WriteString(strconv.FormatInt(hours, 10))
	sb.WriteByte(':')
	if minutes < 10 {
		sb.WriteByte('0')
	}
	sb.WriteString(strconv.FormatInt(minutes, 10))
	return sb.String()
}

func compactDuration(seconds int64, verbose bool) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	units := []struct {
		value int64
		short string
		long  string
	}{
		{days, "d", "day"},
		{hours, "h", "hour"},
		{minutes, "m", "minute"},
	}

	parts := make([]string, 0, 3)
	for _, unit := range units {
		if unit.value == 0 {
			continue
		}
		if verbose {
			parts = append(
				parts,
				strconv.FormatInt(unit.value, 10)+" "+plural(unit.long, unit.value),
			)
			continue
		}
		parts = append(parts, strconv.FormatInt(unit.value, 10)+unit.short)
	}
	if len(parts) == 0 {
		if verbose {
			return "0 minutes"
		}
		return "0m"
	}

	return strings.Join(parts, " ")
}

func plural(unit string, value int64) string {
	if value == 1 {
		return unit
	}
	return unit + "s"
}

func currencySymbol(code string) string {
	if symbol, ok := currencySymbols[code]; ok {
		return symbol
	}
	if code == "" {
		return "$"
	}
	return code + " "
}
