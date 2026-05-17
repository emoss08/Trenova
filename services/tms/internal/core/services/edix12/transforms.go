package edix12

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const transformSuggestedFix = "Check the transform operation, arguments, and base source configuration."

var transformNonDigitPattern = regexp.MustCompile(`\D+`)

var transformOperationHandlers = map[string]transformStepFunc{
	"trim":               transformTrim,
	"upper":              transformUpper,
	"lower":              transformLower,
	"concat":             transformConcat,
	"substring":          transformSubstring,
	"left_pad":           transformLeftPad,
	"right_pad":          transformRightPad,
	"truncate":           transformTruncate,
	"remove_punctuation": transformRemovePunctuation,
	"replace":            transformReplace,
	"contains":           transformContains,
	"starts_with":        transformStartsWith,
	"ends_with":          transformEndsWith,
	"coalesce":           transformCoalesce,
	"default":            transformDefault,
	"empty_if_none":      transformEmptyIfNone,
	"required":           transformRequired,
	"format_date":        transformFormatDate,
	"format_time":        transformFormatTime,
	"format_decimal":     transformFormatDecimal,
	"format_int":         transformFormatInt,
	"normalize_phone":    transformNormalizePhone,
	"normalize_state":    transformNormalizeState,
	"normalize_postal":   transformNormalizePostal,
	"qualifier":          transformQualifier,
	"conditional":        transformConditional,
}

type transformRuntime struct {
	segment *edi.EDITemplateSegment
	element *edi.TemplateElement
	env     map[string]any
}

type transformStepFunc func(*transformRuntime, any, map[string]any) (any, error)

func resolveTransformElementValue(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	env map[string]any,
) (any, []Diagnostic, error) {
	runtime := &transformRuntime{
		segment: segment,
		element: element,
		env:     env,
	}
	value, err := runtime.resolveBaseSource()
	if err != nil {
		return "", []Diagnostic{runtime.diagnostic(err.Error())}, nil
	}

	for _, step := range element.TransformPipeline {
		operation := normalizeTransformOperation(step.Operation)
		fn, ok := transformOperationHandlers[operation]
		if !ok {
			return "", []Diagnostic{runtime.diagnostic(
				fmt.Sprintf("unsupported transform operation %q", step.Operation),
			)}, nil
		}

		value, err = fn(runtime, value, step.Arguments)
		if err != nil {
			return "", []Diagnostic{runtime.diagnostic(err.Error())}, nil
		}
	}
	return value, nil, nil
}

func (r *transformRuntime) resolveBaseSource() (any, error) {
	if r.element.BaseSource == nil {
		return nil, fmt.Errorf("transform base source is required")
	}

	source := r.element.BaseSource
	switch source.Source {
	case edi.TemplateElementSourceTransform:
		return nil, fmt.Errorf("transform base source cannot be another transform")
	case edi.TemplateElementSourceStarlark:
		return nil, fmt.Errorf("transform base source cannot be starlark")
	}
	if !isDirectElementSource(source.Source) {
		return nil, fmt.Errorf("unsupported transform base source %q", source.Source)
	}

	value, _ := resolveDirectSource(baseDirectSource(source), r.env)
	if isEmptyTransformValue(value) && source.Default != "" {
		return source.Default, nil
	}
	return value, nil
}

func (r *transformRuntime) diagnostic(message string) Diagnostic {
	return Diagnostic{
		Severity:        edi.ValidationSeverityError,
		Code:            "transform_error",
		SegmentID:       r.segment.SegmentID,
		ElementPosition: r.element.Position,
		Path:            sourcePath(r.element),
		Message:         message,
		SuggestedFix:    transformSuggestedFix,
	}
}

func normalizeTransformOperation(operation string) string {
	switch strings.TrimSpace(operation) {
	case "uppercase":
		return "upper"
	case "lowercase":
		return "lower"
	case "default_value":
		return "default"
	default:
		return strings.TrimSpace(operation)
	}
}

func transformTrim(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	return strings.TrimSpace(valueToString(value)), nil
}

func transformUpper(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	return strings.ToUpper(valueToString(value)), nil
}

func transformLower(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	return strings.ToLower(valueToString(value)), nil
}

func transformConcat(r *transformRuntime, value any, args map[string]any) (any, error) {
	values, err := r.argumentValues(args)
	if err != nil {
		return nil, err
	}
	parts := make([]string, 0, len(values)+1)
	parts = append(parts, valueToString(value))
	for _, arg := range values {
		parts = append(parts, valueToString(arg))
	}

	separator, hasSeparator, err := r.optionalStringArg(args, "separator")
	if err != nil {
		return nil, err
	}
	if !hasSeparator {
		return strings.Join(parts, ""), nil
	}

	nonEmptyParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, separator), nil
}

func transformSubstring(r *transformRuntime, value any, args map[string]any) (any, error) {
	start, err := r.requiredIntArg(args, "start")
	if err != nil {
		return nil, err
	}
	end, hasEnd, err := r.optionalIntArg(args, "end")
	if err != nil {
		return nil, err
	}

	runes := []rune(valueToString(value))
	if start < 0 {
		start = 0
	}
	if start > len(runes) {
		return "", nil
	}
	if !hasEnd || end > len(runes) {
		end = len(runes)
	}
	if end < start {
		return "", nil
	}
	return string(runes[start:end]), nil
}

func transformLeftPad(r *transformRuntime, value any, args map[string]any) (any, error) {
	return transformPad(r, value, args, true)
}

func transformRightPad(r *transformRuntime, value any, args map[string]any) (any, error) {
	return transformPad(r, value, args, false)
}

func transformPad(r *transformRuntime, value any, args map[string]any, left bool) (any, error) {
	length, err := r.requiredIntArg(args, "length")
	if err != nil {
		return nil, err
	}
	pad, _, err := r.optionalStringArg(args, "pad")
	if err != nil {
		return nil, err
	}
	if pad == "" {
		pad = " "
	}

	text := valueToString(value)
	runes := []rune(text)
	if len(runes) >= length {
		return text, nil
	}

	padding := strings.Repeat(string([]rune(pad)[0]), length-len(runes))
	if left {
		return padding + text, nil
	}
	return text + padding, nil
}

func transformTruncate(r *transformRuntime, value any, args map[string]any) (any, error) {
	length, err := r.requiredIntArg(args, "length")
	if err != nil {
		return nil, err
	}
	if length <= 0 {
		return "", nil
	}

	runes := []rune(valueToString(value))
	if len(runes) <= length {
		return string(runes), nil
	}
	return string(runes[:length]), nil
}

func transformRemovePunctuation(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, valueToString(value))
	return cleaned, nil
}

func transformReplace(r *transformRuntime, value any, args map[string]any) (any, error) {
	oldValue, err := r.requiredStringArgAny(args, "old", "search", "from")
	if err != nil {
		return nil, err
	}
	newValue, err := r.requiredStringArgAny(args, "new", "replacement", "to")
	if err != nil {
		return nil, err
	}
	count, hasCount, err := r.optionalIntArg(args, "count")
	if err != nil {
		return nil, err
	}
	if !hasCount {
		count = -1
	}
	return strings.Replace(valueToString(value), oldValue, newValue, count), nil
}

func transformContains(r *transformRuntime, value any, args map[string]any) (any, error) {
	needle, err := r.requiredStringArgAny(args, "value", "substring", "contains")
	if err != nil {
		return nil, err
	}
	return strings.Contains(valueToString(value), needle), nil
}

func transformStartsWith(r *transformRuntime, value any, args map[string]any) (any, error) {
	prefix, err := r.requiredStringArgAny(args, "value", "prefix")
	if err != nil {
		return nil, err
	}
	return strings.HasPrefix(valueToString(value), prefix), nil
}

func transformEndsWith(r *transformRuntime, value any, args map[string]any) (any, error) {
	suffix, err := r.requiredStringArgAny(args, "value", "suffix")
	if err != nil {
		return nil, err
	}
	return strings.HasSuffix(valueToString(value), suffix), nil
}

func transformCoalesce(r *transformRuntime, value any, args map[string]any) (any, error) {
	if !isEmptyTransformValue(value) {
		return value, nil
	}
	values, err := r.argumentValues(args)
	if err != nil {
		return nil, err
	}
	for _, arg := range values {
		if !isEmptyTransformValue(arg) {
			return arg, nil
		}
	}
	return "", nil
}

func transformDefault(r *transformRuntime, value any, args map[string]any) (any, error) {
	if !isEmptyTransformValue(value) {
		return value, nil
	}
	fallback, ok, err := r.optionalArgAny(args, "value", "fallback")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("default requires value or fallback")
	}
	return fallback, nil
}

func transformEmptyIfNone(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	if value == nil {
		return "", nil
	}
	return value, nil
}

func transformRequired(r *transformRuntime, value any, args map[string]any) (any, error) {
	if !isEmptyTransformValue(value) {
		return value, nil
	}
	message, _, err := r.optionalStringArg(args, "message")
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%s", stringutils.FirstNonEmpty(message, "required value is missing"))
}

func transformFormatDate(r *transformRuntime, value any, args map[string]any) (any, error) {
	return r.formatTimeValue("format_date", value, args, "20060102")
}

func transformFormatTime(r *transformRuntime, value any, args map[string]any) (any, error) {
	return r.formatTimeValue("format_time", value, args, "1504")
}

func transformFormatDecimal(r *transformRuntime, value any, args map[string]any) (any, error) {
	if isEmptyTransformValue(value) {
		return "", nil
	}
	places := 2
	if argPlaces, ok, err := r.optionalIntArg(args, "places"); err != nil {
		return nil, err
	} else if ok {
		places = argPlaces
	}
	if places < 0 {
		return nil, fmt.Errorf("format_decimal places must be greater than or equal to 0")
	}

	number, ok := decimalFromTransformValue(value)
	if !ok {
		return nil, fmt.Errorf("format_decimal input %q is not a valid decimal", valueToString(value))
	}
	return number.StringFixed(int32(places)), nil
}

func transformFormatInt(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	if isEmptyTransformValue(value) {
		return "", nil
	}
	number, ok := decimalFromTransformValue(value)
	if !ok {
		return nil, fmt.Errorf("format_int input %q is not a valid number", valueToString(value))
	}
	return number.Round(0).StringFixed(0), nil
}

func transformNormalizePhone(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	digits := transformNonDigitPattern.ReplaceAllString(valueToString(value), "")
	if len(digits) == 11 && strings.HasPrefix(digits, "1") {
		digits = digits[1:]
	}
	return digits, nil
}

func transformNormalizeState(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	normalized := alnumUpperTransform(valueToString(value))
	if len(normalized) > 2 {
		return normalized[:2], nil
	}
	return normalized, nil
}

func transformNormalizePostal(_ *transformRuntime, value any, _ map[string]any) (any, error) {
	return alnumUpperTransform(valueToString(value)), nil
}

func transformQualifier(r *transformRuntime, value any, args map[string]any) (any, error) {
	rawMapping, ok := args["mapping"]
	if !ok {
		return nil, fmt.Errorf("qualifier requires mapping")
	}
	key := valueToString(value)

	switch mapping := rawMapping.(type) {
	case map[string]any:
		if mapped, ok := mapping[key]; ok {
			return r.resolveArgument(mapped), nil
		}
	case map[string]string:
		if mapped, ok := mapping[key]; ok {
			return r.resolveArgument(mapped), nil
		}
	default:
		return nil, fmt.Errorf("qualifier mapping must be an object")
	}

	fallback, ok, err := r.optionalArg(args, "fallback")
	if err != nil {
		return nil, err
	}
	if ok {
		return fallback, nil
	}
	return "", nil
}

func transformConditional(r *transformRuntime, _ any, args map[string]any) (any, error) {
	when, ok, err := r.optionalArg(args, "when")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("conditional requires when")
	}

	matches, err := r.evaluateConditional(when, args)
	if err != nil {
		return nil, err
	}
	if matches {
		thenValue, _, err := r.optionalArg(args, "then")
		return thenValue, err
	}
	elseValue, _, err := r.optionalArg(args, "else")
	return elseValue, err
}

func (r *transformRuntime) formatTimeValue(
	operation string,
	value any,
	args map[string]any,
	defaultLayout string,
) (any, error) {
	if isEmptyTransformValue(value) {
		return "", nil
	}
	layout, _, err := r.optionalStringArg(args, "layout")
	if err != nil {
		return nil, err
	}
	if layout == "" {
		layout, _, err = r.optionalStringArg(args, "format")
		if err != nil {
			return nil, err
		}
	}
	if layout == "" {
		layout = defaultLayout
	}

	timestamp, ok := parseTransformTime(value)
	if !ok {
		return nil, fmt.Errorf("%s input %q is not a valid time", operation, valueToString(value))
	}
	return timestamp.UTC().Format(layout), nil
}

func (r *transformRuntime) evaluateConditional(when any, args map[string]any) (bool, error) {
	rule, _, err := r.optionalStringArgAny(args, "rule", "operator", "condition")
	if err != nil {
		return false, err
	}
	switch strings.TrimSpace(rule) {
	case "", "truthy", "exists", "not_empty":
		return !isEmptyTransformValue(when), nil
	case "empty":
		return isEmptyTransformValue(when), nil
	case "equals", "eq":
		target, err := r.requiredStringArgAny(args, "value", "equals", "target")
		if err != nil {
			return false, err
		}
		return valueToString(when) == target, nil
	case "not_equals", "ne":
		target, err := r.requiredStringArgAny(args, "value", "equals", "target")
		if err != nil {
			return false, err
		}
		return valueToString(when) != target, nil
	case "contains":
		target, err := r.requiredStringArgAny(args, "value", "substring", "contains")
		if err != nil {
			return false, err
		}
		return strings.Contains(valueToString(when), target), nil
	case "starts_with":
		target, err := r.requiredStringArgAny(args, "value", "prefix")
		if err != nil {
			return false, err
		}
		return strings.HasPrefix(valueToString(when), target), nil
	case "ends_with":
		target, err := r.requiredStringArgAny(args, "value", "suffix")
		if err != nil {
			return false, err
		}
		return strings.HasSuffix(valueToString(when), target), nil
	default:
		return false, fmt.Errorf("unsupported conditional rule %q", rule)
	}
}

func (r *transformRuntime) argumentValues(args map[string]any) ([]any, error) {
	rawValues, ok := args["values"]
	if !ok {
		return []any{}, nil
	}

	switch values := rawValues.(type) {
	case []any:
		resolved := make([]any, 0, len(values))
		for _, value := range values {
			resolved = append(resolved, r.resolveArgument(value))
		}
		return resolved, nil
	case []string:
		resolved := make([]any, 0, len(values))
		for _, value := range values {
			resolved = append(resolved, r.resolveArgument(value))
		}
		return resolved, nil
	default:
		return nil, fmt.Errorf("values must be an array")
	}
}

func (r *transformRuntime) optionalArg(args map[string]any, key string) (any, bool, error) {
	value, ok := args[key]
	if !ok {
		return nil, false, nil
	}
	return r.resolveArgument(value), true, nil
}

func (r *transformRuntime) optionalArgAny(args map[string]any, keys ...string) (any, bool, error) {
	for _, key := range keys {
		value, ok, err := r.optionalArg(args, key)
		if err != nil || ok {
			return value, ok, err
		}
	}
	return nil, false, nil
}

func (r *transformRuntime) optionalStringArg(args map[string]any, key string) (string, bool, error) {
	value, ok, err := r.optionalArg(args, key)
	if err != nil || !ok {
		return "", ok, err
	}
	return valueToString(value), true, nil
}

func (r *transformRuntime) optionalStringArgAny(
	args map[string]any,
	keys ...string,
) (string, bool, error) {
	value, ok, err := r.optionalArgAny(args, keys...)
	if err != nil || !ok {
		return "", ok, err
	}
	return valueToString(value), true, nil
}

func (r *transformRuntime) requiredStringArgAny(args map[string]any, keys ...string) (string, error) {
	value, ok, err := r.optionalStringArgAny(args, keys...)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%s is required", strings.Join(keys, " or "))
	}
	return value, nil
}

func (r *transformRuntime) optionalIntArg(args map[string]any, key string) (int, bool, error) {
	value, ok, err := r.optionalArg(args, key)
	if err != nil || !ok {
		return 0, ok, err
	}

	integer, ok := intFromTransformValue(value)
	if !ok {
		return 0, true, fmt.Errorf("%s must be an integer", key)
	}
	return integer, true, nil
}

func (r *transformRuntime) requiredIntArg(args map[string]any, key string) (int, error) {
	value, ok, err := r.optionalIntArg(args, key)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func (r *transformRuntime) resolveArgument(value any) any {
	text, ok := value.(string)
	if !ok || !strings.HasPrefix(text, "$") {
		return value
	}
	return maputils.Path(r.env, strings.TrimPrefix(text, "$"))
}

func isEmptyTransformValue(value any) bool {
	return strings.TrimSpace(valueToString(value)) == ""
}

func intFromTransformValue(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		return int(typed), true
	case float32:
		floatValue := float64(typed)
		if floatValue == float64(int(floatValue)) {
			return int(floatValue), true
		}
	case float64:
		if typed == float64(int(typed)) {
			return int(typed), true
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	}
	return 0, false
}

func parseTransformTime(value any) (time.Time, bool) {
	if timestamp, ok := unixTimestamp(value); ok {
		if timestamp <= 0 {
			return time.Time{}, false
		}
		return time.Unix(timestamp, 0), true
	}

	raw := strings.TrimSpace(valueToString(value))
	if raw == "" {
		return time.Time{}, false
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil && seconds > 0 {
		return time.Unix(seconds, 0), true
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"20060102",
		"01/02/2006 15:04:05",
		"01/02/2006 15:04",
		"01/02/2006",
		"1/2/2006",
		"15:04:05",
		"15:04",
		"3:04 PM",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, raw, time.UTC)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func decimalFromTransformValue(value any) (decimal.Decimal, bool) {
	switch typed := value.(type) {
	case decimal.NullDecimal:
		if !typed.Valid {
			return decimal.Zero, false
		}
		return typed.Decimal, true
	case decimal.Decimal:
		return typed, true
	case int:
		return decimal.NewFromInt(int64(typed)), true
	case int8:
		return decimal.NewFromInt(int64(typed)), true
	case int16:
		return decimal.NewFromInt(int64(typed)), true
	case int32:
		return decimal.NewFromInt(int64(typed)), true
	case int64:
		return decimal.NewFromInt(typed), true
	case uint:
		return decimal.NewFromInt(int64(typed)), true
	case uint8:
		return decimal.NewFromInt(int64(typed)), true
	case uint16:
		return decimal.NewFromInt(int64(typed)), true
	case uint32:
		return decimal.NewFromInt(int64(typed)), true
	case uint64:
		return decimal.NewFromInt(int64(typed)), true
	case float32:
		return decimal.NewFromFloat(float64(typed)), true
	case float64:
		return decimal.NewFromFloat(typed), true
	case string:
		parsed, err := decimal.NewFromString(strings.TrimSpace(typed))
		return parsed, err == nil
	case map[string]any:
		if valid, ok := typed["Valid"].(bool); ok && !valid {
			return decimal.Zero, false
		}
		if decimalValue, ok := typed["Decimal"]; ok {
			return decimalFromTransformValue(decimalValue)
		}
	}
	parsed, err := decimal.NewFromString(valueToString(value))
	return parsed, err == nil
}

func alnumUpperTransform(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToUpper(r))
		}
	}
	return builder.String()
}
