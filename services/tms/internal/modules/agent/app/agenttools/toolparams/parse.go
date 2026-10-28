package toolparams

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

func ScheduleFormats(params map[string]any) []string {
	formats := StringSliceParam(params, "formats")
	normalized := make([]string, 0, len(formats))
	for _, format := range formats {
		if trimmed := strings.ToLower(strings.TrimSpace(format)); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) > 0 {
		return normalized
	}

	return []string{string(report.DefaultScheduleFormat())}
}

// stringSliceParam reads an array of strings, dropping anything that is not
// one rather than failing: a single bad entry should cost its own entry.
func StringSliceParam(params map[string]any, key string) []string {
	raw, _ := params[key].([]any)
	values := make([]string, 0, len(raw))
	for _, entry := range raw {
		if text, ok := entry.(string); ok && strings.TrimSpace(text) != "" {
			values = append(values, text)
		}
	}

	return values
}

func OptionalPulid(params map[string]any, key string) (pulid.ID, bool, error) {
	value := OptionalString(params, key)
	if value == "" {
		return pulid.Nil, false, nil
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, false, err
	}

	return id, true, nil
}

func RequireString(params map[string]any, key string) (string, error) {
	raw, ok := params[key]
	if !ok {
		return "", fmt.Errorf("missing required parameter %q", key)
	}

	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("parameter %q must be a non-empty string", key)
	}

	return value, nil
}

func OptionalString(params map[string]any, key string) string {
	if raw, ok := params[key]; ok {
		if value, ok := raw.(string); ok {
			return value
		}
	}

	return ""
}

// optionalBool reads a flag the caller may leave out. Only a literal true
// counts; a model that sends the string "true" gets the safe default.
func OptionalBool(params map[string]any, key string) bool {
	value, _ := params[key].(bool)

	return value
}

func OptionalInt64(params map[string]any, key string) int64 {
	raw, ok := params[key]
	if !ok {
		return 0
	}

	switch value := raw.(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func RequirePulid(params map[string]any, key string) (pulid.ID, error) {
	value, err := RequireString(params, key)
	if err != nil {
		return pulid.Nil, err
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, fmt.Errorf("parameter %q is not a valid id: %w", key, err)
	}

	return id, nil
}

func DecodeParam(params map[string]any, key string, out any) error {
	raw, ok := params[key]
	if !ok {
		return fmt.Errorf("missing required parameter %q", key)
	}

	if err := jsonutils.Convert(raw, out); err != nil {
		return fmt.Errorf("parameter %q: %w", key, err)
	}

	return nil
}

// requirePulidSlice reads a bounded list of ids.
//
// The cap is enforced here rather than left to the schema, because a schema's
// maxItems is a hint to the model and nothing more: the tool is what decides
// how much of a fleet, or a ledger, one call may touch.
func RequirePulidSlice(params map[string]any, key string, limit int) ([]pulid.ID, error) {
	raw, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("missing required parameter %q", key)
	}

	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return nil, fmt.Errorf("parameter %q must be a non-empty array of ids", key)
	}

	if len(values) > limit {
		return nil, fmt.Errorf(
			"parameter %q holds %d ids, which is more than the %d this tool will change "+
				"at once; split it into smaller calls", key, len(values), limit,
		)
	}

	ids := make([]pulid.ID, 0, len(values))
	for index, value := range values {
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("parameter %q[%d] must be a non-empty id", key, index)
		}

		id, err := pulid.Parse(strings.TrimSpace(text))
		if err != nil {
			return nil, fmt.Errorf("parameter %q[%d] is not a valid id: %w", key, index, err)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// requireMoney reads an amount the model wrote as a decimal string or a
// number, in major units, and returns it in minor units. It must be
// positive: a zero or negative amount is never what a payment means.
func RequireMoney(params map[string]any, key string) (int64, error) {
	minor, present, err := OptionalMoney(params, key)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, fmt.Errorf("parameter %q is required", key)
	}

	return minor, nil
}

func OptionalMoney(params map[string]any, key string) (int64, bool, error) {
	value, present, err := OptionalDecimal(params, key)
	if err != nil || !present {
		return 0, false, err
	}

	if !value.IsPositive() {
		return 0, false, fmt.Errorf("parameter %q must be greater than zero", key)
	}

	return money.MinorUnits(value), true, nil
}

func RequireSignedMoney(params map[string]any, key string) (int64, error) {
	value, present, err := OptionalDecimal(params, key)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, fmt.Errorf("parameter %q is required", key)
	}

	minor := money.MinorUnits(value)
	if minor == 0 {
		return 0, fmt.Errorf("parameter %q must not be zero", key)
	}

	return minor, nil
}

func OptionalDecimal(params map[string]any, key string) (decimal.Decimal, bool, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return decimal.Zero, false, nil
	}

	switch typed := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return decimal.Zero, false, nil
		}
		parsed, err := decimal.NewFromString(trimmed)
		if err != nil {
			return decimal.Zero, false, fmt.Errorf(
				"parameter %q must be a decimal such as 1250.00",
				key,
			)
		}
		return parsed, true, nil
	case float64:
		return decimal.NewFromFloat(typed), true, nil
	case int:
		return decimal.NewFromInt(int64(typed)), true, nil
	case int64:
		return decimal.NewFromInt(typed), true, nil
	default:
		return decimal.Zero, false, fmt.Errorf(
			"parameter %q must be a decimal such as 1250.00",
			key,
		)
	}
}

// requireDay reads a YYYY-MM-DD as the start of that day in UTC, which is
// how a date-only value is stored on a payment.
func RequireDay(params map[string]any, key string) (int64, error) {
	raw, err := RequireString(params, key)
	if err != nil {
		return 0, err
	}

	day, err := time.Parse(time.DateOnly, strings.TrimSpace(raw))
	if err != nil {
		return 0, DateRefusal(key, raw, toolschema.FormatDate)
	}

	return day.Unix(), nil
}

func OptionalDay(params map[string]any, key string) (day int64, ok bool, err error) {
	if strings.TrimSpace(OptionalString(params, key)) == "" {
		return 0, false, nil
	}
	day, err = RequireDay(params, key)
	if err != nil {
		return 0, false, err
	}

	return day, true, nil
}

func OptionalPulidParam(params map[string]any, key string) (*pulid.ID, error) {
	id, ok, err := OptionalPulid(params, key)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", key, err)
	}
	if !ok {
		return nil, nil //nolint:nilnil // an absent optional id is no id and no error
	}

	return &id, nil
}

func RequireLocalTime(params map[string]any, key string, loc *time.Location) (int64, error) {
	raw, given := params[key]
	if !given || raw == nil {
		return 0, errortypes.NewValidationError(key, errortypes.ErrRequired,
			"A local date and time such as 2026-10-01T08:00 is required")
	}

	text, isText := raw.(string)
	if !isText {
		return 0, DateRefusal(key, raw, toolschema.FormatLocalDateTime)
	}

	seconds, fieldErr := ParseLocalTime(key, text, loc)
	if fieldErr != nil {
		return 0, fieldErr
	}

	return seconds, nil
}

func OptionalLocalTime(params map[string]any, key string, loc *time.Location) (*int64, error) {
	raw, given := params[key]
	if !given || raw == nil {
		return nil, nil //nolint:nilnil // an absent time is no time and no error
	}
	if text, isText := raw.(string); isText && strings.TrimSpace(text) == "" {
		return nil, nil //nolint:nilnil // an empty time is no time and no error
	}

	seconds, err := RequireLocalTime(params, key, loc)
	if err != nil {
		return nil, err
	}

	return &seconds, nil
}

func ParseLocalTime(field, value string, loc *time.Location) (int64, *errortypes.Error) {
	text := strings.TrimSpace(value)
	if _, offset := timeutils.ParseTimeRFC3339(text); offset {
		return 0, DateRefusal(field, text, toolschema.FormatLocalDateTime)
	}

	seconds, err := timeutils.ParseLocalDateTime(text, loc)
	switch {
	case err == nil:
		return seconds, nil
	case errors.Is(err, timeutils.ErrSkippedLocalTime):
		return 0, errortypes.NewValidationError(field, errortypes.ErrInvalid,
			"{0} does not exist in {1}: the clocks move forward past it", text, loc.String())
	default:
		return 0, DateRefusal(field, text, toolschema.FormatLocalDateTime)
	}
}

// dateRefusal is how a tool refuses a date its schema's format should have
// refused first, in the schema's own sentence: a call that reaches the tool
// another way, an approver's edit or a draft's field, is told what a model
// is told, and the wording lives in one place.
func DateRefusal(field string, value any, format string) *errortypes.Error {
	message, _ := toolschema.FormatMessage(value, format)

	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}

// decimalArg reads an amount written as a decimal string. It is never
// negative; a missing optional amount is zero.
func DecimalArg(params map[string]any, key string, required bool) (decimal.Decimal, error) {
	raw := strings.TrimSpace(OptionalString(params, key))
	if raw == "" {
		if required {
			return decimal.Zero, fmt.Errorf("missing required parameter %q", key)
		}

		return decimal.Zero, nil
	}

	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf(
			"parameter %q must be a decimal amount such as 1850.00, not %q", key, raw,
		)
	}
	if value.IsNegative() {
		return decimal.Zero, fmt.Errorf("parameter %q cannot be negative", key)
	}

	return value, nil
}

func RequireUTCDay(params map[string]any, key string) (time.Time, error) {
	raw, err := RequireString(params, key)
	if err != nil {
		return time.Time{}, err
	}
	day, err := time.Parse(DayLayout, raw)
	if err != nil {
		return time.Time{}, DateRefusal(key, raw, toolschema.FormatDate)
	}

	return day, nil
}

func RequireDateTime(params map[string]any, key string) (int64, error) {
	raw, err := RequireString(params, key)
	if err != nil {
		return 0, err
	}

	return ParseDateTime(key, raw)
}

func OptionalDateTime(params map[string]any, key string) (*int64, error) {
	raw := strings.TrimSpace(OptionalString(params, key))
	if raw == "" {
		if value, given := params[key]; given && value != nil {
			if _, isText := value.(string); !isText {
				return nil, DateRefusal(key, value, toolschema.FormatDateTime)
			}
		}

		return nil, nil //nolint:nilnil // an absent time is no change and no error
	}

	seconds, err := ParseDateTime(key, raw)
	if err != nil {
		return nil, err
	}

	return &seconds, nil
}

func ParseDateTime(key, raw string) (int64, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return 0, DateRefusal(key, raw, toolschema.FormatDateTime)
	}

	return parsed.Unix(), nil
}

func OptionalBoolPointer(params map[string]any, key string) (*bool, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, nil //nolint:nilnil // an absent flag is no change and no error
	}

	value, ok := raw.(bool)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be true or false", key)
	}

	return &value, nil
}

func OptionalBoundedText(params map[string]any, key string, limit int) (*string, error) {
	if _, ok := params[key]; !ok {
		return nil, nil //nolint:nilnil // an absent text is no change and no error
	}

	text, err := BoundedText(params, key, limit)
	if err != nil {
		return nil, err
	}

	return &text, nil
}

func RequireIntInRange(params map[string]any, key string, minimum, maximum int) (int, error) {
	raw, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("missing required parameter %q", key)
	}

	value, ok := raw.(float64)
	if !ok {
		if whole, isInt := raw.(int); isInt {
			value = float64(whole)
		} else {
			return 0, fmt.Errorf("parameter %q must be a whole number", key)
		}
	}
	if value != float64(int(value)) || int(value) < minimum || int(value) > maximum {
		return 0, fmt.Errorf("parameter %q must be a whole number from %d to %d", key, minimum,
			maximum)
	}

	return int(value), nil
}

var ErrNothingToChange = errors.New("give at least one value to change")

func ErrUnknownValue(key, raw string, names []string) error {
	return fmt.Errorf("%s %q is not one of %s", key, raw, strings.Join(names, ", "))
}

func SendsAny(params map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, given := params[key]; given {
			return true
		}
	}

	return false
}

func RequireEnum[T ~string](params map[string]any, key string, values []T) (T, error) {
	raw, err := RequireString(params, key)
	if err != nil {
		return "", err
	}

	value := T(strings.TrimSpace(raw))
	if !slices.Contains(values, value) {
		return "", ErrUnknownValue(key, raw, sliceutils.Strings(values))
	}

	return value, nil
}

func OptionalEnum[T ~string](
	params map[string]any,
	key string,
	values []T,
) (value T, given bool, err error) {
	if strings.TrimSpace(OptionalString(params, key)) == "" {
		return "", false, nil
	}
	if value, err = RequireEnum(params, key, values); err != nil {
		return "", false, err
	}

	return value, true, nil
}

func RequireAmount(params map[string]any, key string) (decimal.Decimal, error) {
	minor, err := RequireMoney(params, key)
	if err != nil {
		return decimal.Zero, err
	}

	return money.DecimalFromMinor(minor), nil
}

func BoundedText(params map[string]any, key string, limit int) (string, error) {
	text := strings.TrimSpace(OptionalString(params, key))
	if len(text) > limit {
		return "", fmt.Errorf("%s is %d characters; at most %d are kept", key, len(text), limit)
	}

	return text, nil
}

func RequireBoundedText(params map[string]any, key string, limit int) (string, error) {
	if _, err := RequireString(params, key); err != nil {
		return "", err
	}

	return BoundedText(params, key, limit)
}

func BoundedString(
	params map[string]any,
	key string,
	maxLength int,
	required bool,
) (string, error) {
	value := strings.TrimSpace(OptionalString(params, key))
	if value == "" && required {
		if _, err := RequireString(params, key); err != nil {
			return "", err
		}
	}
	if len(value) > maxLength {
		return "", fmt.Errorf(
			"%s is %d characters; it keeps at most %d",
			key,
			len(value),
			maxLength,
		)
	}

	return value, nil
}

func OptionalScheduleDay(params map[string]any, key string) (*int64, error) {
	raw, given := params[key]
	if !given || raw == nil {
		return nil, nil //nolint:nilnil // an absent day is no day and no error
	}
	if text, isText := raw.(string); !isText || strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("parameter %q must be YYYY-MM-DD", key)
	}
	day, err := RequireUTCDay(params, key)
	if err != nil {
		return nil, err
	}
	seconds := day.Unix()

	return &seconds, nil
}

func RequireScheduleDay(params map[string]any, key string) (int64, error) {
	day, err := RequireUTCDay(params, key)
	if err != nil {
		return 0, err
	}

	return day.Unix(), nil
}

func OptionalID(params map[string]any, key string) (pulid.ID, error) {
	id, err := OptionalPulidParam(params, key)
	if err != nil || id == nil {
		return pulid.Nil, err
	}

	return *id, nil
}

func OptionalIntInRange(
	params map[string]any,
	key string,
	minimum, maximum int,
) (*int, error) {
	if raw, given := params[key]; !given || raw == nil {
		return nil, nil //nolint:nilnil // an absent number is no change and no error
	}
	value, err := RequireIntInRange(params, key, minimum, maximum)
	if err != nil {
		return nil, err
	}

	return &value, nil
}
