package toolschema

import (
	"fmt"
	"maps"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Coercion is one value read the way the schema declares it rather than the
// way it was sent.
type Coercion struct {
	Path string
	Note string
}

// Coerce returns a copy of the arguments with each value read as
// the schema declares it, wherever that reading is certain: a numeral sent
// as text becomes the number, a number sent for a text amount becomes its
// text, a lone value sent for a list becomes a list of one, a list sent as
// comma-separated text is split, "true" becomes true, an enum value spelled
// in another case becomes the declared spelling, a null sent for an
// optional parameter is dropped, an undeclared key whose spelling matches a
// declared parameter is renamed to it, midnight UTC sent for a day becomes
// the day, and a local time written with a space or with seconds becomes its
// minute.
//
// Before this, each of those refused the call ("limit: got string, want
// integer") and cost the model a round trip to send what it plainly meant,
// and a model that had been refused twice gave up and told the person the
// tool did not work. Anything less certain than the readings above is left
// for the schema to refuse by name. What was sent is never changed: the
// result is a copy. The runtime reads a model's call with it, and the
// proposal executor what an approver typed into a proposal.
func Coerce(schema, args map[string]any) (map[string]any, []Coercion) {
	if len(args) == 0 {
		return args, nil
	}
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return args, nil
	}

	c := &coercer{}
	coerced, changed := c.object(schema, properties, args, "")
	if !changed {
		return args, nil
	}
	sort.SliceStable(c.notes, func(i, j int) bool { return c.notes[i].Path < c.notes[j].Path })

	return coerced, c.notes
}

type coercer struct {
	notes []Coercion
}

func (c *coercer) note(path, format string, args ...any) {
	c.notes = append(c.notes, Coercion{Path: path, Note: fmt.Sprintf(format, args...)})
}

func (c *coercer) object(
	schema, properties, args map[string]any,
	path string,
) (map[string]any, bool) {
	out := make(map[string]any, len(args))
	maps.Copy(out, args)
	changed := false
	required := requiredSet(schema)

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
			c.note(JoinPath(path, key), "%s was read as %s", JoinPath(path, sent), key)
		}
		at := JoinPath(path, key)
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
		if did {
			out[key] = read
			changed = true
		}
	}

	return out, changed
}

// JoinPath writes a nested field's path the way the validator writes one.
func JoinPath(path, name string) string {
	if path == "" {
		return name
	}

	return path + "." + name
}

// NameShape is a parameter name with case and separators taken out, so
// shipment_id, shipmentID and ShipmentId read as the same name.
func NameShape(name string) string {
	return strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(name))
}

// undeclaredTarget finds the declared parameter an undeclared key plainly
// means: the same name in another spelling, when that parameter was not
// also sent under its own name.
func undeclaredTarget(properties, args map[string]any, key string) (string, bool) {
	shape := NameShape(key)
	for name := range properties {
		if NameShape(name) != shape {
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
	kind := DeclaredType(property)

	switch kind {
	case "object":
		object, ok := value.(map[string]any)
		properties, _ := property["properties"].(map[string]any)
		if !ok || len(properties) == 0 {
			return value, false
		}

		return c.object(property, properties, object, path)
	case "array":
		return c.array(property, value, path)
	case "integer", "number":
		return c.number(kind, value, path)
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

	out := make([]any, len(list))
	copy(out, list)
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
	if isScalar(value) {
		if text, ok := value.(string); ok && DeclaredType(items) == "string" &&
			strings.Contains(text, ",") {
			parts := strings.Split(text, ",")
			list := make([]any, 0, len(parts))
			for _, part := range parts {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					list = append(list, trimmed)
				}
			}
			c.note(path, "%s %q was read as a list of %d; send a JSON array", path, text, len(list))

			return list, true
		}
		c.note(path, "%s was read as a list of one; send a JSON array", path)

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
		c.note(path, "%s was read as the list inside %q; send the array itself", path, key)

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
	c.note(path, "%s %q was read as the number %s; send numbers unquoted",
		path, text, strconv.FormatFloat(parsed, 'f', -1, 64))

	return parsed, true
}

func (c *coercer) text(property map[string]any, value any, path string) (any, bool) {
	if format, dated := DateFormatNamed(FormatOf(property)); dated {
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
	c.note(path, "%s %v was read as the text %q; this parameter takes text", path, value, text)

	return c.enum(property, text, path)
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
func (c *coercer) date(format DateFormat, value any, path string) (any, bool) {
	text, isText := value.(string)
	if !isText {
		if digits, numeric := numberText(value); numeric {
			return digits, true
		}

		return value, false
	}
	trimmed := strings.TrimSpace(text)

	switch format.Name {
	case FormatDate:
		if match := midnightDay.FindStringSubmatch(trimmed); match != nil {
			c.note(path, "%s %q was read as the day %s; send a day as %s",
				path, text, match[1], format.Shape)

			return match[1], true
		}
	case FormatLocalDateTime:
		if match := localMinute.FindStringSubmatch(trimmed); match != nil {
			read := match[1] + "T" + match[2]
			if read != text {
				c.note(path, "%s %q was read as %s; send %s", path, text, read, format.Shape)

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
func (c *coercer) enum(property map[string]any, text string, path string) (any, bool) {
	allowed := EnumValues(property)
	if len(allowed) == 0 {
		return text, false
	}
	for _, option := range allowed {
		if option == text {
			return text, false
		}
	}
	shape := NameShape(strings.TrimSpace(text))
	for _, option := range allowed {
		if NameShape(option) == shape {
			c.note(path, "%s %q was read as %q", path, text, option)

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
		c.note(path, "%s %q was read as true; send a JSON boolean", path, text)

		return true, true
	case "false", "no":
		c.note(path, "%s %q was read as false; send a JSON boolean", path, text)

		return false, true
	default:
		return value, false
	}
}

// DeclaredType is the one type a property declares, the first non-null one
// when it declares several.
func DeclaredType(property map[string]any) string {
	switch typed := property["type"].(type) {
	case string:
		return typed
	case []string:
		for _, kind := range typed {
			if kind != "null" {
				return kind
			}
		}
	case []any:
		for _, kind := range typed {
			if name, ok := kind.(string); ok && name != "null" {
				return name
			}
		}
	}

	return ""
}

// EnumValues is the values a property allows, whichever way the schema lists them.
func EnumValues(property map[string]any) []string {
	switch typed := property["enum"].(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, value := range typed {
			if text, ok := value.(string); ok {
				values = append(values, text)
			}
		}

		return values
	default:
		return nil
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
