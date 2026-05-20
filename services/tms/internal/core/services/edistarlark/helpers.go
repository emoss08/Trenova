package edistarlark

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"go.starlark.net/starlark"
)

var nonDigitPattern = regexp.MustCompile(`\D+`)

const maxDecimalPlaces = 1<<31 - 1

func approvedHelpers() starlark.StringDict {
	return starlark.StringDict{
		"trim":               starlark.NewBuiltin("trim", trimHelper),
		"upper":              starlark.NewBuiltin("upper", upperHelper),
		"lower":              starlark.NewBuiltin("lower", lowerHelper),
		"concat":             starlark.NewBuiltin("concat", concatHelper),
		"substring":          starlark.NewBuiltin("substring", substringHelper),
		"left_pad":           starlark.NewBuiltin("left_pad", leftPadHelper),
		"right_pad":          starlark.NewBuiltin("right_pad", rightPadHelper),
		"truncate":           starlark.NewBuiltin("truncate", truncateHelper),
		"remove_punctuation": starlark.NewBuiltin("remove_punctuation", removePunctuationHelper),
		"coalesce":           starlark.NewBuiltin("coalesce", coalesceHelper),
		"default":            starlark.NewBuiltin("default", defaultHelper),
		"exists":             starlark.NewBuiltin("exists", existsHelper),
		"required":           starlark.NewBuiltin("required", requiredHelper),
		"format_date":        starlark.NewBuiltin("format_date", formatDateHelper),
		"format_time":        starlark.NewBuiltin("format_time", formatTimeHelper),
		"format_decimal":     starlark.NewBuiltin("format_decimal", formatDecimalHelper),
		"format_int":         starlark.NewBuiltin("format_int", formatIntHelper),
		"normalize_phone":    starlark.NewBuiltin("normalize_phone", normalizePhoneHelper),
		"normalize_state":    starlark.NewBuiltin("normalize_state", normalizeStateHelper),
		"normalize_postal":   starlark.NewBuiltin("normalize_postal", normalizePostalHelper),
		"qualifier":          starlark.NewBuiltin("qualifier", qualifierHelper),
		"empty_if_none":      starlark.NewBuiltin("empty_if_none", emptyIfNoneHelper),
	}
}

func trimHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("trim", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	return starlark.String(strings.TrimSpace(stringify(value))), nil
}

func upperHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("upper", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	return starlark.String(strings.ToUpper(stringify(value))), nil
}

func lowerHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("lower", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	return starlark.String(strings.ToLower(stringify(value))), nil
}

func concatHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	if len(kwargs) > 0 {
		return nil, errors.New("concat: got unexpected keyword arguments")
	}

	var builder strings.Builder
	for _, arg := range args {
		builder.WriteString(stringify(arg))
	}
	return starlark.String(builder.String()), nil
}

func substringHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	var start int
	end := -1
	if err := starlark.UnpackArgs(
		"substring",
		args,
		kwargs,
		"value",
		&value,
		"start",
		&start,
		"end?",
		&end,
	); err != nil {
		return nil, err
	}

	runes := []rune(stringify(value))
	if start < 0 {
		start = 0
	}
	if start > len(runes) {
		return starlark.String(""), nil
	}
	if end < 0 || end > len(runes) {
		end = len(runes)
	}
	if end < start {
		return starlark.String(""), nil
	}
	return starlark.String(string(runes[start:end])), nil
}

func leftPadHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	params, err := unpackPadArgs("left_pad", args, kwargs)
	if err != nil {
		return nil, err
	}
	if len([]rune(params.value)) >= params.length {
		return starlark.String(params.value), nil
	}
	return starlark.String(
		strings.Repeat(params.pad, params.length-len([]rune(params.value))) + params.value,
	), nil
}

func rightPadHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	params, err := unpackPadArgs("right_pad", args, kwargs)
	if err != nil {
		return nil, err
	}
	if len([]rune(params.value)) >= params.length {
		return starlark.String(params.value), nil
	}
	return starlark.String(
		params.value + strings.Repeat(params.pad, params.length-len([]rune(params.value))),
	), nil
}

func truncateHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	var length int
	if err := starlark.UnpackArgs(
		"truncate",
		args,
		kwargs,
		"value",
		&value,
		"length",
		&length,
	); err != nil {
		return nil, err
	}
	if length <= 0 {
		return starlark.String(""), nil
	}

	runes := []rune(stringify(value))
	if len(runes) <= length {
		return starlark.String(string(runes)), nil
	}
	return starlark.String(string(runes[:length])), nil
}

func removePunctuationHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("remove_punctuation", args, kwargs, "value", &value); err != nil {
		return nil, err
	}

	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, stringify(value))
	return starlark.String(cleaned), nil
}

func coalesceHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	if len(kwargs) > 0 {
		return nil, errors.New("coalesce: got unexpected keyword arguments")
	}
	for _, arg := range args {
		if !isEmptyValue(arg) {
			return arg, nil
		}
	}
	return starlark.None, nil
}

func defaultHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	var fallback starlark.Value
	if err := starlark.UnpackArgs(
		"default",
		args,
		kwargs,
		"value",
		&value,
		"fallback",
		&fallback,
	); err != nil {
		return nil, err
	}
	if isEmptyValue(value) {
		return fallback, nil
	}
	return value, nil
}

func existsHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("exists", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	return starlark.Bool(!isEmptyValue(value)), nil
}

func requiredHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	message := "required value is missing"
	if err := starlark.UnpackArgs(
		"required",
		args,
		kwargs,
		"value",
		&value,
		"message?",
		&message,
	); err != nil {
		return nil, err
	}
	if isEmptyValue(value) {
		return nil, fmt.Errorf("%s", message)
	}
	return value, nil
}

func formatDateHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("format_date", args, kwargs, "value", &value); err != nil {
		return nil, err
	}

	timestamp, ok := parseTime(value)
	if !ok {
		return starlark.String(""), nil
	}
	return starlark.String(timestamp.UTC().Format("20060102")), nil
}

func formatTimeHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("format_time", args, kwargs, "value", &value); err != nil {
		return nil, err
	}

	timestamp, ok := parseTime(value)
	if !ok {
		return starlark.String(""), nil
	}
	return starlark.String(timestamp.UTC().Format("1504")), nil
}

func formatDecimalHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	places := 2
	if err := starlark.UnpackArgs(
		"format_decimal",
		args,
		kwargs,
		"value",
		&value,
		"places?",
		&places,
	); err != nil {
		return nil, err
	}
	number, ok := decimalFromValue(value)
	if !ok {
		return starlark.String(""), nil
	}
	if places < 0 || places > maxDecimalPlaces {
		return nil, errors.New("format_decimal places must fit in int32")
	}
	return starlark.String(number.StringFixed(int32(places))), nil
}

func formatIntHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("format_int", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	number, ok := decimalFromValue(value)
	if !ok {
		return starlark.String(""), nil
	}
	return starlark.String(number.Round(0).StringFixed(0)), nil
}

func normalizePhoneHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("normalize_phone", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	digits := nonDigitPattern.ReplaceAllString(stringify(value), "")
	if len(digits) == 11 && strings.HasPrefix(digits, "1") {
		digits = digits[1:]
	}
	return starlark.String(digits), nil
}

func normalizeStateHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("normalize_state", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	normalized := alnumUpper(stringify(value))
	if len(normalized) > 2 {
		normalized = normalized[:2]
	}
	return starlark.String(normalized), nil
}

func normalizePostalHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("normalize_postal", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	return starlark.String(alnumUpper(stringify(value))), nil
}

func qualifierHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	var mapping *starlark.Dict
	fallback := ""
	if err := starlark.UnpackArgs(
		"qualifier",
		args,
		kwargs,
		"value",
		&value,
		"mapping",
		&mapping,
		"fallback?",
		&fallback,
	); err != nil {
		return nil, err
	}

	if mapped, ok, err := mapping.Get(value); err != nil {
		return nil, err
	} else if ok {
		return mapped, nil
	}
	if mapped, ok, err := mapping.Get(starlark.String(stringify(value))); err != nil {
		return nil, err
	} else if ok {
		return mapped, nil
	}
	return starlark.String(fallback), nil
}

func emptyIfNoneHelper(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var value starlark.Value
	if err := starlark.UnpackArgs("empty_if_none", args, kwargs, "value", &value); err != nil {
		return nil, err
	}
	if value == starlark.None {
		return starlark.String(""), nil
	}
	return value, nil
}

type padParams struct {
	value  string
	length int
	pad    string
}

func unpackPadArgs(name string, args starlark.Tuple, kwargs []starlark.Tuple) (padParams, error) {
	var value starlark.Value
	length := 0
	pad := " "
	if err := starlark.UnpackArgs(
		name,
		args,
		kwargs,
		"value",
		&value,
		"length",
		&length,
		"pad?",
		&pad,
	); err != nil {
		return padParams{}, err
	}
	if pad == "" {
		pad = " "
	}
	r, _ := utf8.DecodeRuneInString(pad)
	return padParams{
		value:  stringify(value),
		length: length,
		pad:    string(r),
	}, nil
}

func parseTime(value starlark.Value) (time.Time, bool) {
	switch typed := value.(type) {
	case starlark.Int:
		seconds, ok := typed.Int64()
		if !ok || seconds <= 0 {
			return time.Time{}, false
		}
		return time.Unix(seconds, 0), true
	case starlark.Float:
		seconds := int64(typed)
		if seconds <= 0 {
			return time.Time{}, false
		}
		return time.Unix(seconds, 0), true
	}

	raw := strings.TrimSpace(stringify(value))
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

func decimalFromValue(value starlark.Value) (decimal.Decimal, bool) {
	switch typed := value.(type) {
	case starlark.Int:
		if intValue, ok := typed.Int64(); ok {
			return decimal.NewFromInt(intValue), true
		}
		parsed, err := decimal.NewFromString(typed.BigInt().String())
		return parsed, err == nil
	case starlark.Float:
		return decimal.NewFromFloat(float64(typed)), true
	default:
		parsed, err := decimal.NewFromString(stringify(value))
		return parsed, err == nil
	}
}

func alnumUpper(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToUpper(r))
		}
	}
	return builder.String()
}
