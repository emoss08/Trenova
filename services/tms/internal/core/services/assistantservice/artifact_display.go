package assistantservice

import (
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

/*
A tool's result is written for the model, and the model needs what a person
does not: every record's id so the next call can name it, the tenant, the
version a write checks, which way a metric counts as worse. Handed to a person
as a table, that was a column of "inst_01M37R101VKZTB7TSKR30FJ0AT", a cell of
`[{"direction":"HigherIsWorse","label":"Workers affected",…}]` and a detected
date of 1790187600.

So an artifact keeps a projection of the result rather than the result: each
column a person can reason with, with the type that says how to draw it, and
nothing else. The model's copy is untouched. The one identifier a row keeps is
its own record's, under a key no column names, and only when that kind of
record has a page to open — the link is built from it and it is never shown.

The client holds the same rules for artifacts stored before this, so an old
table reads the same as a new one.
*/

// recordIDKey is where a projected row keeps the id its link is built from.
const recordIDKey = "id"

// longTextRunes is the length past which a string is prose rather than a
// value, and is read in the row's detail instead of a cell.
const longTextRunes = 120

// maxIdentifierRunes bounds an enum or status value; anything longer is a
// sentence that happens to have no spaces.
const maxIdentifierRunes = 40

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

// leadKeys name a record, most specific first; a table leads with them.
var leadKeys = []string{
	"proNumber",
	"invoiceNumber",
	"number",
	"referenceNumber",
	"name",
	"fullName",
	"displayName",
	"title",
	"headline",
	"subject",
	"label",
	"code",
	"unitNumber",
	"licenseNumber",
}

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

// tableProjection is a list result as a person reads it.
type tableProjection struct {
	columns      []assistantartifact.DisplayColumn
	rows         []any
	recordEntity string
}

// projectTable reads rows against the columns the tool declared, keeping the
// columns that have something readable in them and the values that fill them.
func projectTable(entity string, declared []string, rows []any) tableProjection {
	records := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if record, ok := row.(map[string]any); ok {
			records = append(records, record)
		}
	}

	amountIsMoney := !slices.Contains(declared, "method")
	columns := make([]assistantartifact.DisplayColumn, 0, len(declared))
	values := make([]any, len(records))
	for _, key := range leadFirst(declared) {
		for i, record := range records {
			values[i] = record[key]
		}
		displayType, shown := classifyValues(key, values, amountIsMoney)
		if !shown {
			continue
		}
		columns = append(columns, assistantartifact.DisplayColumn{
			Key:   key,
			Label: displayLabel(key, displayType),
			Type:  displayType,
		})
	}

	recordEntity := recordEntityOf(entity)
	projected := make([]any, 0, len(records))
	for _, record := range records {
		row := make(map[string]any, len(columns)+1)
		for _, column := range columns {
			if value, ok := projectValue(column.Type, record[column.Key]); ok {
				row[column.Key] = value
			}
		}
		if recordEntity != "" {
			if id := stringOf(record[recordIDKey]); id != "" {
				row[recordIDKey] = id
			}
		}
		projected = append(projected, row)
	}

	return tableProjection{columns: columns, rows: projected, recordEntity: recordEntity}
}

// projectRecord reads one record as labelled values, in the order the record
// declares its fields with the ones that name it first. A nested record is
// read by its name; a nested set of plain sentences — a rule's wording — is
// read field by field; anything else nested is structure, not a fact.
func projectRecord(record map[string]any, order []string) []assistantartifact.DisplayField {
	amountIsMoney := record["method"] == nil
	fields := make([]assistantartifact.DisplayField, 0, len(order))
	single := make([]any, 1)

	appendField := func(key string, value any) {
		single[0] = value
		displayType, shown := classifyValues(key, single, amountIsMoney)
		if !shown {
			return
		}
		projected, ok := projectValue(displayType, value)
		if !ok {
			return
		}
		fields = append(fields, assistantartifact.DisplayField{
			DisplayColumn: assistantartifact.DisplayColumn{
				Key:   key,
				Label: displayLabel(key, displayType),
				Type:  displayType,
			},
			Value: projected,
		})
	}

	for _, key := range leadFirst(order) {
		value := record[key]
		nested, isRecord := value.(map[string]any)
		if isRecord && !hiddenKey(key) && recordLabel(nested) == "" {
			for _, child := range slices.Sorted(maps.Keys(nested)) {
				if text, isText := nested[child].(string); isText {
					appendField(key+stringutils.CapitalizeFirst(child), text)
				}
			}
			continue
		}
		appendField(key, value)
	}

	return fields
}

// classifyValues decides how a column reads from its key and every value in
// it, or that it is not shown at all.
func classifyValues(
	key string,
	values []any,
	amountIsMoney bool,
) (assistantartifact.DisplayType, bool) {
	if hiddenKey(key) {
		return "", false
	}

	dated, datedType := datedKey(key)
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
		return assistantartifact.DisplayFlag, anyTrue(present)
	}

	switch shapeOf(present) {
	case shapeBool:
		return assistantartifact.DisplayBoolean, true
	case shapeList:
		return listType(present)
	case shapeObject:
		for _, value := range present {
			if nested, _ := value.(map[string]any); recordLabel(nested) == "" {
				return "", false
			}
		}
		return assistantartifact.DisplayText, true
	case shapeScalar:
	}

	numeric := allNumeric(present)
	switch {
	case dated && allInstants(present):
		return datedType, true
	case numeric && isMoneyKey(key, present, amountIsMoney):
		return assistantartifact.DisplayMoney, true
	case numeric && endsWithWord(key, percentWords):
		return assistantartifact.DisplayPercent, true
	case endsWithWord(key, statusWords) && allCodes(present):
		return assistantartifact.DisplayStatus, true
	case endsWithWord(key, enumWords) && allCodes(present):
		return assistantartifact.DisplayEnum, true
	case allNumbers(present) && !endsWithWord(key, labelNumberWords):
		return assistantartifact.DisplayNumber, true
	case isProse(key, present):
		return assistantartifact.DisplayLongText, true
	default:
		return assistantartifact.DisplayText, true
	}
}

// projectValue is one value as its column draws it, or false when it has
// nothing to draw.
func projectValue(displayType assistantartifact.DisplayType, value any) (any, bool) {
	switch displayType {
	case assistantartifact.DisplayText:
		text := textOf(value)
		return text, text != ""
	case assistantartifact.DisplayLongText,
		assistantartifact.DisplayEnum,
		assistantartifact.DisplayStatus:
		text := readableString(value)
		return text, text != ""
	case assistantartifact.DisplayDate, assistantartifact.DisplayDateTime:
		return instantOf(value)
	case assistantartifact.DisplayMoney,
		assistantartifact.DisplayPercent,
		assistantartifact.DisplayNumber:
		return numericOf(value)
	case assistantartifact.DisplayBoolean:
		flag, ok := value.(bool)
		return flag, ok
	case assistantartifact.DisplayFlag:
		flag, _ := value.(bool)
		return true, flag
	case assistantartifact.DisplayMetrics:
		metrics := metricsOf(value)
		return metrics, len(metrics) > 0
	case assistantartifact.DisplayLinks:
		links := linksOf(value)
		return links, len(links) > 0
	default:
		return nil, false
	}
}

// displayLabel is a key as a column head: "windowStart" reads "Window start".
// A date says what happened rather than when, so "detectedOn" reads
// "Detected" and "createdAt" reads "Created".
func displayLabel(key string, displayType assistantartifact.DisplayType) string {
	if displayType == assistantartifact.DisplayDate ||
		displayType == assistantartifact.DisplayDateTime {
		for _, word := range []string{"At", "On"} {
			if trimmed, ok := strings.CutSuffix(key, word); ok && trimmed != "" &&
				endsWithWord(key, []string{word}) {
				key = trimmed
				break
			}
		}
	}

	return stringutils.HumanizeCamelCaseSentence(key)
}

// recordEntityOf is the record-link registry's name for a list's entity, when
// its records have a page: "shipments" opens as "shipment".
func recordEntityOf(entity string) string {
	for _, candidate := range []string{entity, singular(entity)} {
		if candidate == "" {
			continue
		}
		if _, ok := productguide.Default.Record(candidate); ok {
			return candidate
		}
	}

	return ""
}

func singular(plural string) string {
	switch {
	case strings.HasSuffix(plural, "ices"):
		return strings.TrimSuffix(plural, "ices") + "ix"
	case strings.HasSuffix(plural, "ies"):
		return strings.TrimSuffix(plural, "ies") + "y"
	case strings.HasSuffix(plural, "sses"):
		return strings.TrimSuffix(plural, "es")
	case strings.HasSuffix(plural, "s"):
		return strings.TrimSuffix(plural, "s")
	default:
		return plural
	}
}

// leadFirst moves the keys that name a record to the front and keeps the rest
// where the record declared them.
func leadFirst(keys []string) []string {
	ordered := make([]string, 0, len(keys))
	for _, lead := range leadKeys {
		if slices.Contains(keys, lead) {
			ordered = append(ordered, lead)
		}
	}
	for _, key := range keys {
		if !slices.Contains(leadKeys, key) {
			ordered = append(ordered, key)
		}
	}

	return ordered
}

// objectKeyOrder reads an encoded object's keys in the order they were
// written, which for a tool's typed result is the order its fields are
// declared in. A decoded map has no order of its own.
func objectKeyOrder(encoded []byte) []string {
	root, err := sonic.Get(encoded)
	if err != nil {
		return nil
	}

	keys := make([]string, 0, 32)
	if err = root.ForEach(func(path ast.Sequence, _ *ast.Node) bool {
		if path.Key != nil {
			keys = append(keys, *path.Key)
		}

		return true
	}); err != nil {
		return nil
	}

	return keys
}

func hiddenKey(key string) bool {
	lowered := strings.ToLower(key)
	if _, hidden := hiddenKeys[lowered]; hidden {
		return true
	}

	return endsWithWord(key, idWords) ||
		strings.HasSuffix(lowered, "_id") ||
		strings.HasSuffix(lowered, "_ids")
}

// endsWithWord reports whether key is one of words, or ends in one of them as
// a camel-case word of its own.
func endsWithWord(key string, words []string) bool {
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

// datedKey reports whether a key names an instant, and whether its hour
// matters.
func datedKey(key string) (bool, assistantartifact.DisplayType) {
	switch {
	case endsWithWord(key, dateTimeWords):
		return true, assistantartifact.DisplayDateTime
	case endsWithWord(key, dateWords):
		return true, assistantartifact.DisplayDate
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
func listType(values []any) (assistantartifact.DisplayType, bool) {
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
		return assistantartifact.DisplayMetrics, true
	case links:
		return assistantartifact.DisplayLinks, true
	case words:
		return assistantartifact.DisplayText, true
	default:
		return "", false
	}
}

func isMetric(object map[string]any) bool {
	_, hasValue := object["value"]

	return stringOf(object["label"]) != "" && hasValue
}

func isLink(object map[string]any) bool {
	return stringOf(object["label"]) != "" && stringOf(object["path"]) != ""
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
		figure, ok := numericOf(object["value"])
		if !ok {
			continue
		}
		metric := map[string]any{"label": stringOf(object["label"]), "value": figure}
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
		if !ok || !isLink(object) || !isAppPath(stringOf(object["path"])) {
			continue
		}
		link := map[string]any{"label": stringOf(object["label"]), "path": stringOf(object["path"])}
		if count, isCount := object["count"].(float64); isCount && count > 0 {
			link["count"] = count
		}
		out = append(out, link)
	}

	return out
}

// isAppPath is a path inside the app: one leading slash, and never the
// protocol-relative or backslashed forms that browsers read as another host.
func isAppPath(path string) bool {
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
		if _, ok := numericOf(value); !ok {
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

// numericOf keeps a figure as it came: a decimal string stays a string, so a
// cent is never lost on its way to the reader.
func numericOf(value any) (any, bool) {
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
	if endsWithWord(key, decimalMoneyWord) {
		for _, value := range values {
			if _, isText := value.(string); !isText {
				return false
			}
		}

		return true
	}
	if !endsWithWord(key, moneyWords) {
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

// readableString is a string a person can read: never a record's id.
func readableString(value any) string {
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
		return readableString(typed)
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
		return recordLabel(typed)
	default:
		return ""
	}
}
