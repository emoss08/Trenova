package sim

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type fieldKind uint8

const (
	kindString fieldKind = iota
	kindBool
	kindInt
	kindNumber
	kindStringList
	kindObject
	kindExternalIDs
	kindAttributes
	kindDate
	kindTime
)

type bodyMode uint8

const (
	bodyCreate bodyMode = iota
	bodyPatch
)

const dateLayout = "2006-01-02"

type fieldRule struct {
	Name     string
	Kind     fieldKind
	Required bool
	MinLen   int
	MaxLen   int
	Enum     []string
	IntRange *intRange
	Fields   []fieldRule
	Check    func(value any) error
	Nullable bool
}

type intRange struct {
	Min int64
	Max int64
}

func sanitizeBody(body Record, rules []fieldRule, mode bodyMode) (Record, error) {
	return sanitizeObject(body, rules, mode, "")
}

func sanitizeObject(
	body map[string]any,
	rules []fieldRule,
	mode bodyMode,
	prefix string,
) (Record, error) {
	out := make(Record, len(rules))
	for idx := range rules {
		rule := &rules[idx]
		path := prefix + rule.Name
		raw, present := body[rule.Name]
		if !present {
			if rule.Required && mode == bodyCreate {
				return nil, invalidField(path, "is required")
			}
			continue
		}
		if raw == nil {
			if mode == bodyPatch && rule.Nullable {
				out[rule.Name] = nil
				continue
			}
			return nil, invalidField(path, "cannot be null")
		}
		value, err := sanitizeField(raw, rule, path)
		if err != nil {
			return nil, err
		}
		if rule.Check != nil {
			if err = rule.Check(value); err != nil {
				return nil, err
			}
		}
		out[rule.Name] = value
	}
	return out, nil
}

func sanitizeField(raw any, rule *fieldRule, path string) (any, error) {
	switch rule.Kind {
	case kindString:
		return sanitizeString(raw, rule, path)
	case kindBool:
		return sanitizeBool(raw, path)
	case kindInt:
		return sanitizeInt(raw, rule, path)
	case kindNumber:
		return sanitizeNumber(raw, path)
	case kindStringList:
		return sanitizeStringList(raw, rule, path)
	case kindObject:
		return sanitizeNestedObject(raw, rule, path)
	case kindExternalIDs:
		return validateExternalIDs(raw)
	case kindAttributes:
		return sanitizeAttributes(raw, path)
	case kindDate:
		return sanitizeDate(raw, path)
	case kindTime:
		return sanitizeTime(raw, path)
	default:
		return nil, invalidField(path, "is not supported")
	}
}

func sanitizeBool(raw any, path string) (bool, error) {
	value, ok := raw.(bool)
	if !ok {
		return false, invalidField(path, "must be a boolean")
	}
	return value, nil
}

func sanitizeNumber(raw any, path string) (float64, error) {
	value, ok := raw.(float64)
	if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, invalidField(path, "must be a number")
	}
	return value, nil
}

func sanitizeNestedObject(raw any, rule *fieldRule, path string) (map[string]any, error) {
	mapped, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField(path, "must be an object")
	}
	nested, err := sanitizeObject(mapped, rule.Fields, bodyCreate, path+".")
	if err != nil {
		return nil, err
	}
	return map[string]any(nested), nil
}

func sanitizeDate(raw any, path string) (string, error) {
	text, ok := raw.(string)
	if !ok {
		return "", invalidField(path, "must be a YYYY-MM-DD date")
	}
	if _, err := time.Parse(dateLayout, strings.TrimSpace(text)); err != nil {
		return "", invalidField(path, "must be a YYYY-MM-DD date")
	}
	return strings.TrimSpace(text), nil
}

func sanitizeTime(raw any, path string) (string, error) {
	text, ok := raw.(string)
	if !ok {
		return "", invalidField(path, "must be an RFC 3339 timestamp")
	}
	parsed, err := parseRFC3339(text)
	if err != nil {
		return "", invalidField(path, "must be an RFC 3339 timestamp")
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

func sanitizeString(raw any, rule *fieldRule, path string) (string, error) {
	text, ok := raw.(string)
	if !ok {
		return "", invalidField(path, "must be a string")
	}
	length := utf8.RuneCountInString(text)
	if rule.MinLen > 0 && length < rule.MinLen {
		return "", invalidField(path, fmt.Sprintf("must be at least %d characters", rule.MinLen))
	}
	if rule.MaxLen > 0 && length > rule.MaxLen {
		return "", invalidField(path, fmt.Sprintf("must be at most %d characters", rule.MaxLen))
	}
	if len(rule.Enum) > 0 && !slices.Contains(rule.Enum, text) {
		return "", invalidField(
			path,
			"must be one of "+strings.Join(quoteAll(rule.Enum), ", "),
		)
	}
	return text, nil
}

func sanitizeInt(raw any, rule *fieldRule, path string) (int64, error) {
	value, ok := raw.(float64)
	if !ok || math.IsNaN(value) || math.IsInf(value, 0) || value != math.Trunc(value) ||
		math.Abs(value) > 1<<53 {
		return 0, invalidField(path, "must be an integer")
	}
	integer := int64(value)
	if rule.IntRange != nil && (integer < rule.IntRange.Min || integer > rule.IntRange.Max) {
		return 0, invalidField(
			path,
			fmt.Sprintf("must be between %d and %d", rule.IntRange.Min, rule.IntRange.Max),
		)
	}
	return integer, nil
}

func sanitizeStringList(raw any, rule *fieldRule, path string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(path, "must be an array of strings")
	}
	out := make([]any, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for idx, item := range items {
		text, isString := item.(string)
		clean := strings.TrimSpace(text)
		if !isString || clean == "" {
			return nil, invalidField(fmt.Sprintf("%s[%d]", path, idx), "must be a non-empty string")
		}
		if rule.MaxLen > 0 && utf8.RuneCountInString(clean) > rule.MaxLen {
			return nil, invalidField(
				fmt.Sprintf("%s[%d]", path, idx),
				fmt.Sprintf("must be at most %d characters", rule.MaxLen),
			)
		}
		if _, dup := seen[clean]; dup {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out, nil
}

func quoteAll(values []string) []string {
	out := make([]string, len(values))
	for idx, value := range values {
		out[idx] = "`" + value + "`"
	}
	return out
}

func parseRFC3339(raw string) (time.Time, error) {
	clean := strings.TrimSpace(raw)
	parsed, err := time.Parse(time.RFC3339Nano, clean)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse RFC 3339 time %q: %w", clean, err)
	}
	return parsed.UTC(), nil
}

func stringListValues(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return stringSlice(raw)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, isString := item.(string); isString && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func int64Value(raw any) (int64, bool) {
	switch typed := raw.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		if typed != math.Trunc(typed) {
			return 0, false
		}
		return int64(typed), true
	default:
		return 0, false
	}
}

func boolValue(record Record, key string) (value, ok bool) {
	value, ok = record[key].(bool)
	return value, ok
}

func stringOf(raw any) string {
	text, ok := raw.(string)
	if !ok {
		return ""
	}
	return text
}

func listOf(raw any) []any {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	return items
}

func mapOf(raw any) map[string]any {
	mapped, ok := anyAsMap(raw)
	if !ok {
		return nil
	}
	return mapped
}
