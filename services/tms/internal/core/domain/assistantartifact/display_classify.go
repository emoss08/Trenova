package assistantartifact

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

// longTextRunes is the length past which a string is prose rather than a
// value, and is read in the row's detail instead of a cell.
const longTextRunes = 120

// maxIdentifierRunes bounds an enum or status value; anything longer is a
// sentence that happens to have no spaces.
const maxIdentifierRunes = 40

const (
	labelKey = "label"
	pathKey  = "path"
)

// hiddenKeys say nothing a person can use: the tenant, the write's version,
// the detector's bookkeeping, a page's cursor. Compared lowercased.
var hiddenKeys = map[string]struct{}{
	"id":              {},
	"version":         {},
	"organizationid":  {},
	"businessunitid":  {},
	"tenantid":        {},
	"direction":       {},
	"dedupekey":       {},
	"narrated":        {},
	"modelidentifier": {},
	"truncated":       {},
	"hasmore":         {},
	"nextoffset":      {},
	"offset":          {},
}

// flagKeys are conditions worth saying only when they hold. "Stale: no" on
// every finding is a column of noise.
var flagKeys = map[string]struct{}{"stale": {}}

// Words a key ends in, each as its own camel-case word, and what they make a
// column. A short word must be its own word — "reason" is not a date, "legend"
// is not an end — so matching is on the word, never on the letters.
var (
	idWords       = []string{"Id", "ID", "Ids", "IDs"}
	dateTimeWords = []string{
		"At",
		"Time",
		"Timestamp",
		"Start",
		"End",
		"Arrival",
		"Departure",
		"Eta",
		"Since",
		"Until",
		"Cutoff",
		"For",
	}
	dateWords = []string{
		"Date",
		"Expiry",
		"Expires",
		"On",
		"Due",
		"Check",
		"From",
		"To",
		"Through",
		"Deadline",
		"Of",
		"Dob",
	}
	statusWords = []string{"Status", "Severity", "Priority", "Standing"}
	enumWords   = []string{
		"Category",
		"Type",
		"Kind",
		"Method",
		"Class",
		"Mode",
		"Side",
		"Source",
		"Channel",
		"Tier",
		"Level",
		"Role",
		"Unit",
	}
	moneyWords = []string{
		"Amount",
		"Charge",
		"Charges",
		"Cost",
		"Revenue",
		"Price",
		"Balance",
		"Fee",
		"Fees",
		"Pay",
	}
	decimalMoneyWord = []string{"Total", "Subtotal"}
	percentWords     = []string{"Percent", "Percentage", "Pct"}
	labelNumberWords = []string{"Year", "Number", "Code", "Zip", "PostalCode", "Phone"}
	proseKeys        = []string{
		"narrative",
		"recommendation",
		"body",
		"content",
		"explanation",
		"rationale",
		"instructions",
	}
)

// ClassifyValues decides how a column reads from its key and every value in
// it, or that it is not shown at all.
func ClassifyValues(
	key string,
	values []any,
	amountIsMoney bool,
) (DisplayType, bool) {
	if HiddenKey(key) {
		return "", false
	}

	dated, datedType := DatedKey(key)
	present := make([]any, 0, len(values))
	for _, value := range values {
		if !isEmptyValue(value, dated) {
			present = append(present, value)
		}
	}
	if len(present) == 0 || allIdentifiers(present) {
		return "", false
	}

	if _, flag := flagKeys[strings.ToLower(key)]; flag {
		return DisplayFlag, anyTrue(present)
	}

	switch shapeOf(present) {
	case shapeBool:
		return DisplayBoolean, true
	case shapeList:
		return listType(present)
	case shapeObject:
		for _, value := range present {
			if nested, _ := value.(map[string]any); RecordLabel(nested) == "" {
				return "", false
			}
		}
		return DisplayText, true
	case shapeScalar:
	}

	return classifyScalars(key, present, scalarContext{
		dated:         dated,
		datedType:     datedType,
		amountIsMoney: amountIsMoney,
	})
}

type scalarContext struct {
	dated         bool
	datedType     DisplayType
	amountIsMoney bool
}

// classifyScalars reads a column of plain values: an instant, an amount, a
// member of a set, a figure, prose or text.
func classifyScalars(key string, present []any, c scalarContext) (DisplayType, bool) {
	dated, datedType, amountIsMoney := c.dated, c.datedType, c.amountIsMoney
	numeric := allNumeric(present)
	switch {
	case dated && allInstants(present):
		return datedType, true
	case numeric && isMoneyKey(key, present, amountIsMoney):
		return DisplayMoney, true
	case numeric && EndsWithWord(key, percentWords):
		return DisplayPercent, true
	case EndsWithWord(key, statusWords) && allCodes(present):
		return DisplayStatus, true
	case EndsWithWord(key, enumWords) && allCodes(present):
		return DisplayEnum, true
	case allNumbers(present) && !EndsWithWord(key, labelNumberWords):
		return DisplayNumber, true
	case isProse(key, present):
		return DisplayLongText, true
	default:
		return DisplayText, true
	}
}

// ProjectValue is one value as its column draws it, or false when it has
// nothing to draw.
func ProjectValue(displayType DisplayType, value any) (any, bool) {
	switch displayType {
	case DisplayText:
		text := textOf(value)
		return text, text != ""
	case DisplayLongText,
		DisplayEnum,
		DisplayStatus:
		text := ReadableString(value)
		return text, text != ""
	case DisplayDate, DisplayDateTime:
		return instantOf(value)
	case DisplayMoney,
		DisplayPercent,
		DisplayNumber:
		return NumericOf(value)
	case DisplayBoolean:
		flag, ok := value.(bool)
		return flag, ok
	case DisplayFlag:
		flag, _ := value.(bool)
		return true, flag
	case DisplayMetrics:
		metrics := metricsOf(value)
		return metrics, len(metrics) > 0
	case DisplayLinks:
		links := linksOf(value)
		return links, len(links) > 0
	default:
		return nil, false
	}
}

// DisplayLabel is a key as a column head: "windowStart" reads "Window start".
// A date says what happened rather than when, so "detectedOn" reads
// "Detected" and "createdAt" reads "Created".
func DisplayLabel(key string, displayType DisplayType) string {
	if displayType == DisplayDate ||
		displayType == DisplayDateTime {
		for _, word := range []string{"At", "On"} {
			if trimmed, ok := strings.CutSuffix(key, word); ok && trimmed != "" &&
				EndsWithWord(key, []string{word}) {
				key = trimmed
				break
			}
		}
	}

	return stringutils.HumanizeCamelCaseSentence(key)
}

func HiddenKey(key string) bool {
	lowered := strings.ToLower(key)
	if _, hidden := hiddenKeys[lowered]; hidden {
		return true
	}

	return EndsWithWord(key, idWords) ||
		strings.HasSuffix(lowered, "_id") ||
		strings.HasSuffix(lowered, "_ids")
}

// EndsWithWord reports whether key is one of words, or ends in one of them as
// a camel-case word of its own.
func EndsWithWord(key string, words []string) bool {
	for _, word := range words {
		if strings.EqualFold(key, word) {
			return true
		}
		start := len(key) - len(word)
		if start < 1 || key[start:] != word {
			continue
		}
		if previous := key[start-1]; previous >= 'a' && previous <= 'z' ||
			previous >= '0' && previous <= '9' {
			return true
		}
	}

	return false
}

// DatedKey reports whether a key names an instant, and whether its hour
// matters.
func DatedKey(key string) (bool, DisplayType) {
	switch {
	case EndsWithWord(key, dateTimeWords):
		return true, DisplayDateTime
	case EndsWithWord(key, dateWords):
		return true, DisplayDate
	default:
		return false, ""
	}
}

// isEmptyValue is a value with nothing to read. Under a date's name a zero is
// how the rows spell "not set", and drawing it would invent 1970.
func isEmptyValue(value any, dated bool) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	case float64:
		return dated && typed == 0
	default:
		return false
	}
}

// allIdentifiers reports a column whose every value is a record id.
func allIdentifiers(values []any) bool {
	for _, value := range values {
		text, ok := value.(string)
		if !ok || !pulid.LooksLike(strings.TrimSpace(text)) {
			return false
		}
	}

	return true
}

func anyTrue(values []any) bool {
	for _, value := range values {
		if flag, ok := value.(bool); ok && flag {
			return true
		}
	}

	return false
}

type valueShape int

const (
	shapeScalar valueShape = iota
	shapeBool
	shapeList
	shapeObject
)

// shapeOf is the one shape every value shares, or scalar when they differ.
func shapeOf(values []any) valueShape {
	shape := shapeScalar
	for i, value := range values {
		var current valueShape
		switch value.(type) {
		case bool:
			current = shapeBool
		case []any:
			current = shapeList
		case map[string]any:
			current = shapeObject
		default:
			current = shapeScalar
		}
		if i > 0 && current != shape {
			return shapeScalar
		}
		shape = current
	}

	return shape
}

// listType reads a column of lists: measurements, links, or plain words.
// Lists of anything else are records nested in records, which a cell cannot
// hold and a person does not need to see.
func listType(values []any) (DisplayType, bool) {
	metrics, links, words := true, true, true
	for _, value := range values {
		entries, _ := value.([]any)
		for _, entry := range entries {
			object, isObject := entry.(map[string]any)
			metrics = metrics && isObject && isMetric(object)
			links = links && isObject && isLink(object)
			switch entry.(type) {
			case string, float64:
			default:
				words = false
			}
		}
	}

	switch {
	case metrics:
		return DisplayMetrics, true
	case links:
		return DisplayLinks, true
	case words:
		return DisplayText, true
	default:
		return "", false
	}
}

func isMetric(object map[string]any) bool {
	_, hasValue := object["value"]

	return stringOf(object[labelKey]) != "" && hasValue
}

func isLink(object map[string]any) bool {
	return stringOf(object[labelKey]) != "" && stringOf(object[pathKey]) != ""
}

// metricsOf keeps what a measurement says: its name, its value and its unit.
// Which way is worse is for colouring a trend, not for reading a table.
func metricsOf(value any) []map[string]any {
	entries, _ := value.([]any)
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		object, ok := entry.(map[string]any)
		if !ok || !isMetric(object) {
			continue
		}
		figure, ok := NumericOf(object["value"])
		if !ok {
			continue
		}
		metric := map[string]any{labelKey: stringOf(object[labelKey]), "value": figure}
		if unit := stringOf(object["unit"]); unit != "" {
			metric["unit"] = unit
		}
		out = append(out, metric)
	}

	return out
}

// linksOf keeps the links that stay inside the app.
func linksOf(value any) []map[string]any {
	entries, _ := value.([]any)
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		object, ok := entry.(map[string]any)
		if !ok || !isLink(object) || !IsAppPath(stringOf(object[pathKey])) {
			continue
		}
		link := map[string]any{
			labelKey: stringOf(object[labelKey]),
			pathKey:  stringOf(object[pathKey]),
		}
		if count, isCount := object["count"].(float64); isCount && count > 0 {
			link["count"] = count
		}
		out = append(out, link)
	}

	return out
}

// IsAppPath is a path inside the app: one leading slash, and never the
// protocol-relative or backslashed forms that browsers read as another host.
func IsAppPath(path string) bool {
	return strings.HasPrefix(path, "/") &&
		!strings.HasPrefix(path, "//") &&
		!strings.HasPrefix(path, "/\\")
}

// allInstants reports a column of dates: epoch seconds in a plausible range,
// or dates already written out the way the runtime writes them for the model
// ("2026-08-24 11:20 PDT (30 days ago)"). A phrase such as "none on file" is
// one only beside real dates, and a decimal is an amount whatever its key
// says — "amountDue" ends in a date word.
func allInstants(values []any) bool {
	stamped := false
	written := true
	for _, value := range values {
		switch typed := value.(type) {
		case float64:
			if typed != math.Trunc(typed) || !timeutils.IsPlausibleInstant(int64(typed)) {
				return false
			}
			stamped = true
		case string:
			text := strings.TrimSpace(typed)
			if isDecimalText(text) {
				return false
			}
			written = written && isWrittenDate(text)
		default:
			return false
		}
	}

	return stamped || written
}

// isWrittenDate reports text that opens with a calendar date, YYYY-MM-DD.
func isWrittenDate(text string) bool {
	if len(text) < len("2006-01-02") {
		return false
	}
	for i, r := range text[:len("2006-01-02")] {
		if i == 4 || i == 7 {
			if r != '-' {
				return false
			}
			continue
		}
		if !isDigit(r) {
			return false
		}
	}

	return true
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func instantOf(value any) (any, bool) {
	switch typed := value.(type) {
	case float64:
		seconds := int64(typed)
		return seconds, timeutils.IsPlausibleInstant(seconds)
	case string:
		text := strings.TrimSpace(typed)
		return text, text != ""
	default:
		return nil, false
	}
}

func allNumeric(values []any) bool {
	for _, value := range values {
		if _, ok := NumericOf(value); !ok {
			return false
		}
	}

	return true
}

func allNumbers(values []any) bool {
	for _, value := range values {
		if _, ok := value.(float64); !ok {
			return false
		}
	}

	return true
}

// NumericOf keeps a figure as it came: a decimal string stays a string, so a
// cent is never lost on its way to the reader.
func NumericOf(value any) (any, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, !math.IsNaN(typed) && !math.IsInf(typed, 0)
	case string:
		text := strings.TrimSpace(typed)
		return text, isDecimalText(text)
	default:
		return nil, false
	}
}

func isDecimalText(text string) bool {
	if text == "" || strings.ContainsFunc(text, func(r rune) bool {
		return !isDigit(r) && r != '.' && r != '-' && r != '+'
	}) {
		return false
	}
	_, err := strconv.ParseFloat(text, 64)

	return err == nil
}

// isMoneyKey reads an amount as money. "total" and "subtotal" are money only
// when written as decimals, because a count of stops is a total too; a bare
// "amount" beside a "method" may be a percentage, so it is only a figure.
func isMoneyKey(key string, values []any, amountIsMoney bool) bool {
	if EndsWithWord(key, decimalMoneyWord) {
		for _, value := range values {
			if _, isText := value.(string); !isText {
				return false
			}
		}

		return true
	}
	if !EndsWithWord(key, moneyWords) {
		return false
	}

	return amountIsMoney || !strings.EqualFold(key, "amount")
}

// allCodes reports values that are members of a set — "InTransit",
// "ServiceQuality" — rather than words somebody wrote.
func allCodes(values []any) bool {
	for _, value := range values {
		text, ok := value.(string)
		if !ok || !isCode(text) {
			return false
		}
	}

	return true
}

func isCode(text string) bool {
	if text == "" || utf8.RuneCountInString(text) > maxIdentifierRunes {
		return false
	}
	for i, r := range text {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case i > 0 && (r >= '0' && r <= '9' || r == '_'):
		default:
			return false
		}
	}

	return true
}

func isProse(key string, values []any) bool {
	if slices.Contains(proseKeys, strings.ToLower(key)) {
		return true
	}
	for _, value := range values {
		if text, ok := value.(string); ok && utf8.RuneCountInString(text) > longTextRunes {
			return true
		}
	}

	return false
}

// ReadableString is a string a person can read: never a record's id.
func ReadableString(value any) string {
	text, _ := value.(string)
	text = strings.TrimSpace(text)
	if pulid.LooksLike(text) {
		return ""
	}

	return text
}

// textOf is a value as words: a number as written, a list as its members, a
// nested record as its name.
func textOf(value any) string {
	switch typed := value.(type) {
	case string:
		return ReadableString(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, entry := range typed {
			switch entry.(type) {
			case string, float64:
				if text := textOf(entry); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		return RecordLabel(typed)
	default:
		return ""
	}
}

// recordLabelKeys are tried in order for the words that name a record: a
// shipment by its PRO, an invoice by its number, a person by their name, a
// unit by its number, a finding by its headline.
var recordLabelKeys = []string{
	"proNumber",
	"invoiceNumber",
	"referenceNumber",
	"name",
	"displayName",
	"fullName",
	"code",
	"unitNumber",
	"licenseNumber",
	"number",
	"title",
	"label",
	"headline",
	"subject",
}

// RecordLabel is the words that name a record, or "" when nothing does. A
// record's id is never its name: a card titled "Insight inst_01M37R…" says
// nothing a person can use.
func RecordLabel(record map[string]any) string {
	for _, key := range recordLabelKeys {
		if value := ReadableString(record[key]); value != "" {
			return value
		}
	}
	first, last := ReadableString(record["firstName"]), ReadableString(record["lastName"])

	return strings.TrimSpace(first + " " + last)
}

func stringOf(value any) string {
	s, _ := value.(string)

	return strings.TrimSpace(s)
}
