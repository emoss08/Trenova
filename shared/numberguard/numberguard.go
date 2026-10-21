package numberguard

import (
	"slices"
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
	// smallNumberCeiling is the value below which a bare count is treated as
	// ordinary prose rather than a citation. "the top 3 customers", "a second
	// driver", "one of the stops" are all sentence furniture. A small figure
	// written as money, a percentage or a rate is a claim all the same: "$3.10 a
	// mile", "a 4% surcharge" and "2.15 per mile" are what a person acts on.
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
	return CheckNumbersAs(ReadCodesApart, prose, supported)
}

// CheckNumbersAs is CheckNumbers in the given reading.
func CheckNumbersAs(how Reading, prose string, supported []decimal.Decimal) NumberCheck {
	cited := scan(prose, how)
	if len(cited) == 0 {
		return NumberCheck{OK: true}
	}

	unsupported := make([]string, 0, len(cited))
	for _, citation := range cited {
		if !citation.supportedBy(supported) && !slices.Contains(unsupported, citation.text) {
			unsupported = append(unsupported, citation.text)
		}
	}

	return NumberCheck{OK: len(unsupported) == 0, Unsupported: unsupported}
}

// CheckAmounts is CheckNumbers held to the amounts in prose: money and
// percentages. Those are the figures a person acts on, and the ones a model
// that totals, averages or divides in its head writes; a count, a year, a time
// or a limit it knows from the trade is left alone, because a reply may state
// one that nothing it read printed.
func CheckAmounts(prose string, supported []decimal.Decimal) NumberCheck {
	return CheckAmountsAs(ReadCodesApart, prose, supported)
}

// CheckAmountsAs is CheckAmounts in the given reading.
func CheckAmountsAs(how Reading, prose string, supported []decimal.Decimal) NumberCheck {
	unsupported := make([]string, 0, 2)
	for _, cited := range scan(prose, how) {
		if !cited.money && !cited.percent {
			continue
		}
		if cited.supportedBy(supported) {
			continue
		}
		if !slices.Contains(unsupported, cited.text) {
			unsupported = append(unsupported, cited.text)
		}
	}

	return NumberCheck{OK: len(unsupported) == 0, Unsupported: unsupported}
}

// supportedBy reports whether a computed figure supports the citation: one
// within rounding of it, or within half a unit of its last written digit, as
// "$57K" is of 57,323.83. A percentage is also the fraction it renders: 52.3%
// is a computed 0.523.
func (c citation) supportedBy(supported []decimal.Decimal) bool {
	if c.bareCount() && c.value.Abs().LessThanOrEqual(decimal.NewFromInt(smallNumberCeiling)) {
		return true
	}
	return isSupported(c.value, supported) || c.readIn(supported)
}

// bareCount reports whether the citation is a plain count, the only kind of
// figure the small-number exemption covers.
func (c citation) bareCount() bool {
	return !c.money && !c.percent && !c.rate
}

// readIn reports whether a figure in supported is the one cited, leaving out
// the small numbers supportedBy lets pass as sentence furniture.
func (c citation) readIn(supported []decimal.Decimal) bool {
	thousand := decimal.NewFromInt(1000)
	for _, candidate := range supported {
		if withinTolerance(c.value, candidate) ||
			withinTolerance(c.value.Mul(thousand), candidate) ||
			c.value.Sub(candidate).Abs().LessThanOrEqual(c.half) {
			return true
		}
		if c.percent && (withinTolerance(c.value.Shift(-2), candidate) ||
			c.value.Shift(-2).Sub(candidate).Abs().LessThanOrEqual(c.half.Shift(-2))) {
			return true
		}
	}

	return false
}

// maxWorkingOperands bounds the figures of a reply that are paired up.
const maxWorkingOperands = 16

// Worked is what the figures prose states work out to in one step: the sum,
// difference, product and each quotient of every pair of them that supported
// holds. A reply that gives both sides of "$6,050,147.60 open, $50,147.60
// without the $6,000,000 invoice" or "112 gallons at $3.89 come to $435.68"
// shows its working; the person can follow it and the guard checks the
// result, so a result that is wrong is still unsupported.
func Worked(prose string, supported []decimal.Decimal) []decimal.Decimal {
	operands := make([]decimal.Decimal, 0, maxWorkingOperands)
	for _, cited := range extractNumbers(prose) {
		if len(operands) == maxWorkingOperands {
			break
		}
		if cited.readIn(supported) && !slices.ContainsFunc(operands, cited.value.Equal) {
			operands = append(operands, cited.value)
		}
	}

	return Combinations(operands)
}

// Combinations is the sum, difference, product and each quotient of every
// pair of values.
func Combinations(values []decimal.Decimal) []decimal.Decimal {
	if len(values) < 2 {
		return nil
	}

	combined := make([]decimal.Decimal, 0, len(values)*(len(values)-1)*5/2)
	for i := range values {
		for j := i + 1; j < len(values); j++ {
			a, b := values[i], values[j]
			combined = append(combined, a.Add(b), a.Sub(b).Abs(), a.Mul(b))
			if !b.IsZero() {
				combined = append(combined, a.Div(b))
			}
			if !a.IsZero() {
				combined = append(combined, b.Div(a))
			}
		}
	}

	return combined
}

func matchesAny(value decimal.Decimal, supported []decimal.Decimal) bool {
	for _, candidate := range supported {
		if withinTolerance(value, candidate) {
			return true
		}
	}

	return false
}

type citation struct {
	text    string
	value   decimal.Decimal
	money   bool
	percent bool
	rate    bool
	// half is half a unit of the last digit written, in the figure's own
	// units: "$57K" is 57,000 to the nearest thousand, so 500 either way,
	// and "43%" is 43 to the nearest point. Zero in the joined reading.
	half decimal.Decimal
}

func isSupported(value decimal.Decimal, supported []decimal.Decimal) bool {
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
	return scan(prose, ReadCodesApart)
}

// Reading is how a scan tells one figure from the next. The agent loop's
// grounding guard narrows a reply with ReadSeparated before it counts the
// reply's working; everything else reads with ReadCodesApart.
type Reading int

const (
	// ReadCodesApart is ReadSeparated that also leaves out what is a code or a
	// year rather than a quantity: a number written with a leading zero
	// ("083"), one listed after an identifier as a shortened form of it
	// ("INV2610000103, 104, 099"), and a bare year ("Dates (2027)").
	ReadCodesApart Reading = iota
	// ReadSeparated takes a comma as a thousands separator only before a group
	// of three digits, and a point as a decimal point only before a digit, so
	// "[172.5,12937.5,0.708]" in a tool's JSON is three figures. A number joined
	// to another by a hyphen, slash or colon (2026-10-25, 10/26, 14:30,
	// 214-555-0142, a 31-60 bucket) and one that is part of an identifier
	// (INV2610000001, HL-30117) or an ordinal (31st) is not a figure.
	ReadSeparated
)

var ordinalSuffixes = []string{"st", "nd", "rd", "th"}

func scan(prose string, how Reading) []citation {
	runes := []rune(prose)
	citations := make([]citation, 0, 8)
	codeEnd := -1

	for index := 0; index < len(runes); index++ {
		if !unicode.IsDigit(runes[index]) {
			continue
		}

		start := index
		grouped := !jsonBefore(runes, start)
		for index < len(runes) && continuesNumber(runes, index, grouped) {
			index++
		}
		// "[50,150,250]" is three numbers, not 150,250 with a stray 50.
		if grouped && jsonAfter(runes, index) &&
			slices.Contains(runes[start:index], ',') {
			index = start
			for index < len(runes) && continuesNumber(runes, index, false) {
				index++
			}
		}

		raw := string(runes[start:index])
		// A trailing separator belongs to the sentence, not the number: "12,400,"
		// at the end of a clause is the figure plus a comma.
		raw = strings.TrimRight(raw, ".,")
		end := start + len([]rune(raw))
		if notAFigure(runes, start, end) {
			if inIdentifier(runes, start) {
				codeEnd = end
			}
			continue
		}

		value, err := parseNumber(raw, runes, index)
		if err != nil {
			continue
		}

		cited := citation{
			text:    raw,
			value:   value,
			money:   isMoney(runes, start, end),
			percent: isPercent(runes, end),
			rate:    isRate(runes, end),
		}
		if how == ReadCodesApart && !cited.money && !cited.percent &&
			(leadingZero(raw) || listedAfter(runes, codeEnd, start) || isYear(raw)) {
			codeEnd = end
			continue
		}
		cited.half = halfUnit(raw, value)
		citations = append(citations, cited)
	}

	return citations
}

var (
	currencySymbols = []rune{'$', '€', '£', '¥'}
	currencyCodes   = []string{"USD", "CAD", "MXN", "EUR", "GBP"}
	percentWords    = []string{"percent", "pct"}
)

// isMoney reports whether the figure at runes[start:end] is written as money:
// a currency symbol before it ("$12,400", "-$80", "($80)", "US$ 80") or a
// currency code on either side ("USD 12,400", "12,400 CAD").
func isMoney(runes []rune, start, end int) bool {
	before := start - 1
	for before >= 0 && (runes[before] == ' ' || runes[before] == '-' || runes[before] == '(') &&
		start-before <= 2 {
		before--
	}
	if before >= 0 && slices.Contains(currencySymbols, runes[before]) {
		return true
	}
	if word := wordBefore(runes, start); slices.Contains(currencyCodes, word) {
		return true
	}

	after := end
	if after < len(runes) && strings.ContainsRune("kKmM", runes[after]) {
		after++
	}

	return slices.Contains(currencyCodes, wordAfter(runes, after))
}

var rateWords = []string{"per", "a", "an", "each"}

var rateUnits = []string{
	"mile", "miles", "mi", "km", "hour", "hours", "hr", "hrs", "day", "days", "lb", "lbs",
	"pound", "pounds", "cwt", "kg", "ton", "tons", "gallon", "gallons", "gal", "liter",
	"liters", "load", "loads", "stop", "stops", "pallet", "pallets", "unit", "units",
	"piece", "pieces", "week", "month", "shipment", "shipments", "trip", "trips",
}

// isRate reports whether the figure ending at end is a per-unit rate: "2.15 per
// mile", "2.15/mi", "45 an hour", "1.10 a gallon". A unit suffix such as k is
// skipped first, as isMoney does.
func isRate(runes []rune, end int) bool {
	after := end
	if after < len(runes) && strings.ContainsRune("kKmM", runes[after]) &&
		(after+1 >= len(runes) || !unicode.IsLetter(runes[after+1])) {
		after++
	}
	if after+1 < len(runes) && runes[after] == '/' && unicode.IsLetter(runes[after+1]) {
		return true
	}

	word := strings.ToLower(wordAfter(runes, after))
	if !slices.Contains(rateWords, word) {
		return false
	}
	next := after
	for next < len(runes) && runes[next] == ' ' {
		next++
	}
	next += len([]rune(word))
	unit := strings.ToLower(wordAfter(runes, next))
	return word == "per" || slices.Contains(rateUnits, unit)
}

// isPercent reports whether the figure ending at end is written as a share:
// "52.3%", "52.3 %", "52.3 percent".
func isPercent(runes []rune, end int) bool {
	if end < len(runes) && runes[end] == '%' {
		return true
	}
	if end+1 < len(runes) && runes[end] == ' ' && runes[end+1] == '%' {
		return true
	}

	return slices.Contains(percentWords, strings.ToLower(wordAfter(runes, end)))
}

// wordBefore is the word that ends one space before start, or "".
func wordBefore(runes []rune, start int) string {
	end := start
	if end > 0 && runes[end-1] == ' ' {
		end--
	}
	begin := end
	for begin > 0 && unicode.IsLetter(runes[begin-1]) {
		begin--
	}

	return string(runes[begin:end])
}

// wordAfter is the word that starts one space after end, or "".
func wordAfter(runes []rune, end int) string {
	begin := end
	if begin < len(runes) && runes[begin] == ' ' {
		begin++
	}
	stop := begin
	for stop < len(runes) && unicode.IsLetter(runes[stop]) {
		stop++
	}

	return string(runes[begin:stop])
}

func continuesNumber(runes []rune, index int, grouped bool) bool {
	r := runes[index]
	if unicode.IsDigit(r) {
		return true
	}

	switch r {
	case '.':
		return isDigitAt(runes, index+1)
	case ',':
		return grouped && isDigitAt(runes, index+1) && isDigitAt(runes, index+2) &&
			isDigitAt(runes, index+3) && !isDigitAt(runes, index+4)
	default:
		return false
	}
}

// inIdentifier reports whether the number starting at start is the tail of an
// identifier: glued to a letter, or joined to one ("HL-30117").
func inIdentifier(runes []rune, start int) bool {
	if start == 0 {
		return false
	}
	before := runes[start-1]

	return unicode.IsLetter(before) ||
		(strings.ContainsRune(joiners+"_", before) && start > 1 && unicode.IsLetter(runes[start-2]))
}

// leadingZero reports whether raw is a whole number written with a leading
// zero, as a code is ("083") and a quantity never is.
func leadingZero(raw string) bool {
	return len(raw) > 1 && raw[0] == '0' && raw[1] != '.'
}

// listedAfter reports whether the number starting at start follows the code
// ending at codeEnd in one list: "INV2610000103, 104 and 099".
func listedAfter(runes []rune, codeEnd, start int) bool {
	if codeEnd < 0 || codeEnd > start {
		return false
	}
	between := strings.TrimSpace(strings.Trim(string(runes[codeEnd:start]), " ,"))

	return between == "" || between == "and" || between == "or"
}

// isYear reports whether raw is a year written alone: four digits from 1900
// to 2199 with no thousands separator, as a count of that size never is.
func isYear(raw string) bool {
	if len(raw) != 4 {
		return false
	}
	year, err := strconv.Atoi(raw)

	return err == nil && year >= 1900 && year < 2200
}

// jsonBefore reports whether the number starting at start opens a JSON
// value, where a comma never groups digits.
func jsonBefore(runes []rune, start int) bool {
	return start > 0 && strings.ContainsRune("[:,", runes[start-1])
}

// jsonAfter reports whether the run ending at end closes a JSON value or is
// followed by the next one with no space between.
func jsonAfter(runes []rune, end int) bool {
	if end >= len(runes) {
		return false
	}
	switch runes[end] {
	case ']', '}':
		return true
	case ',':
		return end+1 < len(runes) && !unicode.IsSpace(runes[end+1])
	default:
		return false
	}
}

// joiners join two numbers into a date, a time, a range or a code.
const joiners = "-–—/:"

var months = []string{
	"jan", "january", "feb", "february", "mar", "march", "apr", "april", "may", "jun", "june",
	"jul", "july", "aug", "august", "sep", "sept", "september", "oct", "october", "nov",
	"november", "dec", "december",
}

// dateConnectors join the days of one written date: "September 29 and 30".
var dateConnectors = []string{"and", "or", "to", "through", "thru"}

// notAFigure reports whether the digits at runes[start:end] are a date, a
// time, a range, a code or an ordinal rather than a quantity.
func notAFigure(runes []rune, start, end int) bool {
	if start > 0 {
		before := runes[start-1]
		if unicode.IsLetter(before) {
			return true
		}
		if strings.ContainsRune(joiners+"_", before) && start > 1 &&
			(unicode.IsLetter(runes[start-2]) || unicode.IsDigit(runes[start-2])) {
			return true
		}
	}
	if end < len(runes) && strings.ContainsRune(joiners, runes[end]) && isDigitAt(runes, end+1) {
		return true
	}
	if slices.Contains(ordinalSuffixes, strings.ToLower(wordAfterNoSpace(runes, end))) {
		return true
	}

	return inCalendarDate(runes, start, end)
}

// inCalendarDate reports whether the number is the day or year of a date
// written with its month: "September 29", "Dec 26–31, 2026", "29 Sep".
func inCalendarDate(runes []rune, start, end int) bool {
	value, err := strconv.Atoi(string(runes[start:end]))
	if err != nil {
		return false
	}
	day := end-start <= 2 && value >= 1 && value <= 31
	year := end-start == 4 && value >= 1900 && value < 2200
	if !day && !year {
		return false
	}

	begin := start
	for begin > 0 && start-begin < 24 {
		r := runes[begin-1]
		if unicode.IsLetter(r) {
			word := wordBefore(runes, begin)
			if !slices.Contains(dateConnectors, strings.ToLower(word)) {
				break
			}
			begin -= len([]rune(word))
			continue
		}
		if r != ' ' && r != ',' && r != '.' && !unicode.IsDigit(r) &&
			!strings.ContainsRune(joiners, r) {
			break
		}
		begin--
	}
	if slices.Contains(months, strings.ToLower(wordBefore(runes, begin))) {
		return true
	}

	return day && slices.Contains(months, strings.ToLower(wordAfter(runes, end)))
}

func isDigitAt(runes []rune, index int) bool {
	return index < len(runes) && unicode.IsDigit(runes[index])
}

// wordAfterNoSpace is the word that starts right at end, or "".
func wordAfterNoSpace(runes []rune, end int) string {
	stop := end
	for stop < len(runes) && unicode.IsLetter(runes[stop]) {
		stop++
	}

	return string(runes[end:stop])
}

// halfUnit is half a unit of the last digit of raw, scaled as value is: "57"
// written as 57,000 is half of a thousand.
func halfUnit(raw string, value decimal.Decimal) decimal.Decimal {
	written, err := decimal.NewFromString(strings.ReplaceAll(raw, ",", ""))
	if err != nil || written.IsZero() {
		return decimal.Zero
	}
	places := int32(0)
	if dot := strings.IndexByte(raw, '.'); dot >= 0 {
		places = int32(len(raw) - dot - 1)
	}

	return decimal.New(5, -places-1).Mul(value.Div(written).Abs())
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
	return SupportedFromTextAs(ReadCodesApart, texts...)
}

// SupportedFromTextAs is SupportedFromText in the given reading.
func SupportedFromTextAs(how Reading, texts ...string) []decimal.Decimal {
	supported := make([]decimal.Decimal, 0, len(texts)*4)
	seen := make(map[string]struct{}, len(texts)*4)
	for _, text := range texts {
		if text == "" {
			continue
		}
		for _, cited := range scan(text, how) {
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

func Cites(prose string, value decimal.Decimal) bool {
	for _, citation := range extractNumbers(prose) {
		if withinTolerance(citation.value, value) {
			return true
		}
	}

	return false
}

// OnlyCounts reports whether every number in prose is exactly one of the
// counts. It is the strict form of CheckNumbers for prose made of counts,
// where no number is conversational: "9 proposals" for four is a lie however
// small nine is.
func OnlyCounts(prose string, counts []int) bool {
	allowed := make(map[int64]struct{}, len(counts))
	for _, count := range counts {
		allowed[int64(count)] = struct{}{}
	}
	for _, citation := range extractNumbers(prose) {
		if !citation.value.IsInteger() {
			return false
		}
		if _, ok := allowed[citation.value.IntPart()]; !ok {
			return false
		}
	}
	return true
}
