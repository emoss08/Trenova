package toolcoerce

import (
	"fmt"
	"maps"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// Coercion is one value read the way the schema declares it rather than the
// way it was sent. Omitted marks an optional parameter sent empty and left
// out, which a note names together rather than one by one.
type Coercion struct {
	Path    string
	Note    string
	Omitted bool
}

type coercer struct {
	notes []Coercion
	// today is the calendar day "today" names where the call is read, when
	// the caller knows it (WithToday).
	today *time.Time
}

func clamps(property map[string]any) bool {
	clamped, _ := property[toolschema.KeyClamp].(bool)

	return clamped
}

func boundOf(property map[string]any, key string) (float64, bool) {
	switch bound := property[key].(type) {
	case int:
		return float64(bound), true
	case int64:
		return float64(bound), true
	case float64:
		return bound, true
	default:
		return 0, false
	}
}

func (c *coercer) clamp(property map[string]any, value any, path string) (any, bool) {
	number, ok := value.(float64)
	if !ok {
		if whole, isInt := value.(int); isInt {
			number, ok = float64(whole), true
		}
	}
	if !ok || !clamps(property) {
		return value, false
	}
	if most, bounded := boundOf(property, toolschema.KeyMaximum); bounded && number > most {
		c.notef(path, "%s %s is past its most, %s, and was read as %s",
			path, strconv.FormatFloat(number, 'f', -1, 64),
			strconv.FormatFloat(most, 'f', -1, 64), strconv.FormatFloat(most, 'f', -1, 64))

		return most, true
	}
	if least, bounded := boundOf(property, toolschema.KeyMinimum); bounded && number < least {
		c.notef(path, "%s %s is below its least, %s, and was read as %s",
			path, strconv.FormatFloat(number, 'f', -1, 64),
			strconv.FormatFloat(least, 'f', -1, 64), strconv.FormatFloat(least, 'f', -1, 64))

		return least, true
	}

	return value, false
}

func keepsEmpty(property map[string]any) bool {
	clears, _ := property[toolschema.KeyKeepEmpty].(bool)

	return clears
}

// placeholder is a value that says nothing: empty or blank text, or a list
// with nothing in it once its blank entries are set aside.
func placeholder(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		for _, element := range typed {
			if !placeholder(element) {
				return false
			}
		}

		return true
	default:
		return false
	}
}

var idPlaceholders = map[string]struct{}{
	"x": {}, "n/a": {}, "na": {}, "none": {}, "null": {}, "nil": {}, "-": {}, "unknown": {},
	"tbd": {}, "?": {},
}

func isRecordID(property map[string]any) bool {
	return toolschema.RecordOf(property) != "" || len(toolschema.RecordKinds(property)) > 0
}

func idPlaceholder(value any) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	_, found := idPlaceholders[strings.ToLower(strings.TrimSpace(text))]

	return found
}

func trimmedID(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimRight(strings.TrimSpace(text), ",.;")

	return trimmed, trimmed != text
}

// Apply returns a copy of the arguments with each value read as
// the schema declares it, wherever that reading is certain: a numeral sent
// as text becomes the number, a number sent for a text amount becomes its
// text, a lone value sent for a list becomes a list of one, a list sent as
// comma-separated text is split, "true" becomes true, an enum value spelled
// in another case becomes the declared spelling, a null sent for an
// optional parameter is dropped, an undeclared key whose spelling matches a
// declared parameter is renamed to it, midnight UTC sent for a day becomes
// the day, and a local time written with a space or with seconds becomes its
// minute. An optional parameter sent as "" or as a list of nothing is read
// as not sent, unless it is marked KeepEmpty; blank entries in a list of
// text are set aside.
//
// Several models fill every parameter a tool declares, sending "" or [""]
// for the ones they have no value for. A transfer that named one shipment
// also sent an empty customer, an empty search and the list's status
// filter, and the tool read the filters as given and refused the call five
// times. A required parameter sent empty is left for the schema and the
// tool to refuse by name.
//
// Before this, each of those refused the call ("limit: got string, want
// integer") and cost the model a round trip to send what it plainly meant,
// and a model that had been refused twice gave up and told the person the
// tool did not work. Anything less certain than the readings above is left
// for the schema to refuse by name. What was sent is never changed: the
// result is a copy. The runtime reads a model's call with it, and the
// proposal executor what an approver typed into a proposal.
func Apply(schema, args map[string]any, opts ...Option) (map[string]any, []Coercion) {
	if len(args) == 0 {
		return args, nil
	}
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return args, nil
	}

	c := &coercer{}
	for _, opt := range opts {
		opt(c)
	}
	coerced, changed := c.object(schema, properties, args, "")
	if !changed {
		return args, nil
	}
	sort.SliceStable(c.notes, func(i, j int) bool { return c.notes[i].Path < c.notes[j].Path })

	return coerced, c.notes
}

// Option adds a reading Apply can only make with something the
// caller knows.
type Option func(*coercer)

// WithToday lets a day parameter take "today", "yesterday" and "tomorrow",
// read as the calendar day they name in the timezone now is given in. A
// person asks for "what delivered today", a model passes the word on, and
// several reads used to parse it themselves while the rest refused it; the
// runtime reads it once, for every tool, as the organization's own day.
func WithToday(now time.Time) Option {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	return func(c *coercer) { c.today = &day }
}

// relativeDays are the words a day parameter takes when the caller says
// which day today is, and how far each is from it.
var relativeDays = map[string]int{"today": 0, "yesterday": -1, "tomorrow": 1}

func (c *coercer) notef(path, format string, args ...any) {
	c.notes = append(c.notes, Coercion{Path: path, Note: fmt.Sprintf(format, args...)})
}

func (c *coercer) object(
	schema, properties, args map[string]any,
	path string,
) (map[string]any, bool) {
	out := make(map[string]any, len(args))
	maps.Copy(out, args)
	changed := false
	required := toolschema.RequiredSet(schema)

	for sent, value := range args {
		key := sent
		property, declared := properties[key].(map[string]any)
		if !declared {
			target, found := undeclaredTarget(properties, args, key)
			if !found {
				continue
			}
			property, _ = properties[target].(map[string]any)
			delete(out, key)
			out[target] = value
			key = target
			changed = true
			c.notef(
				toolschema.JoinPath(path, key),
				"%s was read as %s",
				toolschema.JoinPath(path, sent),
				key,
			)
		}
		at := toolschema.JoinPath(path, key)
		if value == nil {
			if !required[key] {
				if current, still := out[key]; still && current == nil {
					delete(out, key)
					changed = true
				}
			}
			continue
		}
		read, did := c.value(property, value, at)
		if !required[key] && !keepsEmpty(property) && placeholder(read) {
			delete(out, key)
			changed = true
			c.notes = append(c.notes, Coercion{
				Path: at, Note: at + " was empty and was left out", Omitted: true,
			})

			continue
		}
		if isRecordID(property) {
			if !required[key] && idPlaceholder(read) {
				delete(out, key)
				changed = true
				c.notes = append(c.notes, Coercion{
					Path: at,
					Note: fmt.Sprintf(
						"%s held %q, which names no record, and was left out",
						at,
						read,
					),
					Omitted: true,
				})

				continue
			}
			if trimmed, cut := trimmedID(read); cut {
				read, did = trimmed, true
				c.notef(at, "%s was read as %q", at, trimmed)
			}
		}
		if did {
			out[key] = read
			changed = true
		}
	}

	return out, changed
}

// undeclaredTarget finds the declared parameter an undeclared key plainly
// means: the same name in another spelling, when that parameter was not
// also sent under its own name.
func undeclaredTarget(properties, args map[string]any, key string) (string, bool) {
	shape := toolschema.NameShape(key)
	for name := range properties {
		if toolschema.NameShape(name) != shape {
			continue
		}
		if value, sent := args[name]; sent && value != nil {
			return "", false
		}

		return name, true
	}

	return "", false
}

func (c *coercer) value(property map[string]any, value any, path string) (any, bool) {
	if len(property) == 0 {
		return value, false
	}
	kind := toolschema.DeclaredType(property)

	switch kind {
	case "object":
		object, ok := value.(map[string]any)
		decoded := false
		if !ok {
			object, decoded = encodedJSON[map[string]any](value, '{')
		}
		if !ok && !decoded {
			return value, false
		}
		if decoded {
			c.notef(path, "%s was sent as JSON text and read as the object it holds; "+
				"send the object itself", path)
		}
		properties, _ := property["properties"].(map[string]any)
		if len(properties) == 0 {
			return object, decoded
		}
		read, changed := c.object(property, properties, object, path)

		return read, changed || decoded
	case "array":
		return c.array(property, value, path)
	case "integer", "number":
		read, changed := c.number(kind, value, path)
		bounded, clamped := c.clamp(property, read, path)

		return bounded, changed || clamped
	case "string":
		return c.text(property, value, path)
	case "boolean":
		return c.boolean(value, path)
	default:
		return value, false
	}
}

func (c *coercer) array(property map[string]any, value any, path string) (any, bool) {
	items, _ := property["items"].(map[string]any)
	list, changed := c.asList(items, value, path)
	if list == nil {
		return value, false
	}

	out := make([]any, 0, len(list))
	blank := 0
	for _, element := range list {
		if text, isText := element.(string); isText && strings.TrimSpace(text) == "" {
			blank++

			continue
		}
		out = append(out, element)
	}
	if blank > 0 {
		changed = true
		if len(out) > 0 {
			c.notef(path, "%s held %d empty entries, which were left out", path, blank)
		}
	}
	if len(items) > 0 {
		for idx, element := range out {
			read, did := c.value(items, element, path+"["+strconv.Itoa(idx)+"]")
			if did {
				out[idx] = read
				changed = true
			}
		}
	}
	if !changed {
		return value, false
	}

	return out, true
}

// asList reads what the model sent for a list parameter as a list: the list
// itself, a lone value as a list of one, comma-separated text as its parts,
// or the list inside a one-key wrapper such as {"item": [...]}. Nil means it
// could not be read as a list and is left for the schema to refuse.
func (c *coercer) asList(items map[string]any, value any, path string) ([]any, bool) {
	if list, isList := value.([]any); isList {
		return list, false
	}
	if list, decoded := encodedJSON[[]any](value, '['); decoded {
		c.notef(path, "%s was sent as JSON text and read as the list it holds; "+
			"send the array itself", path)

		return list, true
	}
	if isScalar(value) {
		if text, ok := value.(string); ok && toolschema.DeclaredType(items) == "string" &&
			strings.Contains(text, ",") {
			parts := strings.Split(text, ",")
			list := make([]any, 0, len(parts))
			for _, part := range parts {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					list = append(list, trimmed)
				}
			}
			c.notef(
				path,
				"%s %q was read as a list of %d; send a JSON array",
				path,
				text,
				len(list),
			)

			return list, true
		}
		c.notef(path, "%s was read as a list of one; send a JSON array", path)

		return []any{value}, true
	}
	wrapped, ok := value.(map[string]any)
	if !ok || len(wrapped) != 1 {
		return nil, false
	}
	for key, inner := range wrapped {
		list, isList := inner.([]any)
		if !isList {
			return nil, false
		}
		c.notef(path, "%s was read as the list inside %q; send the array itself", path, key)

		return list, true
	}

	return nil, false
}

func (c *coercer) number(kind string, value any, path string) (any, bool) {
	text, ok := value.(string)
	if !ok {
		return value, false
	}
	cleaned := strings.ReplaceAll(strings.TrimSpace(text), ",", "")
	if cleaned == "" {
		return value, false
	}
	parsed, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return value, false
	}
	if kind == "integer" && parsed != math.Trunc(parsed) {
		return value, false
	}
	// A JSON number decodes as float64, so that is what a tool reads when
	// the model sends one unquoted; the re-read value takes the same shape.
	c.notef(path, "%s %q was read as the number %s; send numbers unquoted",
		path, text, strconv.FormatFloat(parsed, 'f', -1, 64))

	return parsed, true
}

func (c *coercer) text(property map[string]any, value any, path string) (any, bool) {
	if format, dated := toolschema.DateFormatNamed(toolschema.FormatOf(property)); dated {
		return c.date(format, value, path)
	}

	var text string
	switch typed := value.(type) {
	case string:
		return c.enum(property, typed, path)
	case float64:
		text = strconv.FormatFloat(typed, 'f', -1, 64)
	case int64:
		text = strconv.FormatInt(typed, 10)
	case int:
		text = strconv.Itoa(typed)
	case bool:
		text = strconv.FormatBool(typed)
	default:
		return value, false
	}
	c.notef(path, "%s %v was read as the text %q; this parameter takes text", path, value, text)
	read, _ := c.enum(property, text, path)

	return read, true
}

var (
	midnightDay = regexp.MustCompile(
		`^(\d{4}-\d{2}-\d{2})T00:00(?::00(?:\.0+)?)?(?:Z|[+-]00:?00)?$`,
	)
	localMinute = regexp.MustCompile(
		`^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2})(?::\d{2}(?:\.\d+)?)?$`,
	)
)

// date reads a date parameter's value where the reading is certain: midnight
// UTC sent for a day is that day, and a local date-time written with a space
// or with seconds is the minute it names. Anything else is left for the
// schema's format to refuse with the shape it wants.
//
// A number is never read as a date. It is almost always Unix seconds, and
// converting one lands on a day in UTC that may not be the person's; it is
// passed on as its digits, which the format refuses as the Unix time it is.
func (c *coercer) date(format toolschema.DateFormat, value any, path string) (any, bool) {
	text, isText := value.(string)
	if !isText {
		if digits, numeric := numberText(value); numeric {
			return digits, true
		}

		return value, false
	}
	trimmed := strings.TrimSpace(text)

	switch format.Name {
	case toolschema.FormatDate:
		if offset, relative := relativeDays[strings.ToLower(trimmed)]; relative && c.today != nil {
			day := c.today.AddDate(0, 0, offset).Format(time.DateOnly)
			c.notef(path, "%s %q was read as the day %s", path, text, day)

			return day, true
		}
		if match := midnightDay.FindStringSubmatch(trimmed); match != nil {
			c.notef(path, "%s %q was read as the day %s; send a day as %s",
				path, text, match[1], format.Shape)

			return match[1], true
		}
	case toolschema.FormatLocalDateTime:
		if match := localMinute.FindStringSubmatch(trimmed); match != nil {
			read := match[1] + "T" + match[2]
			if read != text {
				c.notef(path, "%s %q was read as %s; send %s", path, text, read, format.Shape)

				return read, true
			}
		}
	}

	return value, false
}

func numberText(value any) (string, bool) {
	switch typed := value.(type) {
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case int:
		return strconv.Itoa(typed), true
	default:
		return "", false
	}
}

// enum reads a value spelled in another case or with other separators as
// the declared value it plainly means.
func (c *coercer) enum(property map[string]any, text, path string) (any, bool) {
	allowed := toolschema.EnumValues(property)
	if len(allowed) == 0 {
		return text, false
	}
	for _, option := range allowed {
		if option == text {
			return text, false
		}
	}
	shape := toolschema.NameShape(strings.TrimSpace(text))
	for _, option := range allowed {
		if toolschema.NameShape(option) == shape {
			c.notef(path, "%s %q was read as %q", path, text, option)

			return option, true
		}
	}

	return text, false
}

func (c *coercer) boolean(value any, path string) (any, bool) {
	text, ok := value.(string)
	if !ok {
		return value, false
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "true", "yes":
		c.notef(path, "%s %q was read as true; send a JSON boolean", path, text)

		return true, true
	case "false", "no":
		c.notef(path, "%s %q was read as false; send a JSON boolean", path, text)

		return false, true
	default:
		return value, false
	}
}

func isScalar(value any) bool {
	switch value.(type) {
	case string, float64, int64, int, bool:
		return true
	default:
		return false
	}
}

func encodedJSON[T map[string]any | []any](value any, opener byte) (T, bool) {
	var decoded T
	text, ok := value.(string)
	if !ok {
		return decoded, false
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || trimmed[0] != opener {
		return decoded, false
	}
	if err := sonic.UnmarshalString(trimmed, &decoded); err != nil {
		return decoded, false
	}

	return decoded, decoded != nil
}
