package jsonflex

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

var ErrUnsupported = errors.New("jsonflex: unsupported JSON value")

const millisecondThreshold = 100_000_000_000

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999-0700",
	"2006-01-02T15:04:05-0700",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"20060102",
	"01/02/2006",
	"1/2/2006",
	"01/02/2006 15:04:05",
	"Jan 2, 2006",
	"02-Jan-2006",
	"02-Jan-06",
}

var numericCleaner = strings.NewReplacer(",", "", "$", "", "%", "", " ", "", "_", "")

type String string

func ParseString(raw []byte) (*String, bool) {
	text, _, ok := scalarText(raw)
	if !ok {
		return nil, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	value := String(text)
	return &value, true
}

func (s *String) UnmarshalJSON(data []byte) error {
	if KindOf(data) == KindNull {
		return nil
	}
	text, _, ok := scalarText(data)
	if !ok {
		return fmt.Errorf("%w: expected string-compatible scalar", ErrUnsupported)
	}
	*s = String(strings.TrimSpace(text))
	return nil
}

func (s *String) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return sonic.Marshal(string(*s))
}

func (s *String) Value() string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func (s *String) Ptr() *string {
	if s == nil {
		return nil
	}
	value := string(*s)
	return &value
}

type Int int64

func ParseInt(raw []byte) (*Int, bool) {
	text, kind, ok := scalarText(raw)
	if !ok {
		return nil, false
	}
	parsed, ok := parseIntText(text, kind)
	if !ok {
		return nil, false
	}
	value := Int(parsed)
	return &value, true
}

func (i *Int) UnmarshalJSON(data []byte) error {
	if KindOf(data) == KindNull {
		return nil
	}
	text, kind, ok := scalarText(data)
	if !ok {
		return fmt.Errorf("%w: expected integer-compatible scalar", ErrUnsupported)
	}
	if kind == KindString && strings.TrimSpace(text) == "" {
		*i = 0
		return nil
	}
	parsed, ok := parseIntText(text, kind)
	if !ok {
		return fmt.Errorf("%w: %q is not an integer", ErrUnsupported, text)
	}
	*i = Int(parsed)
	return nil
}

func (i *Int) MarshalJSON() ([]byte, error) {
	if i == nil {
		return []byte("null"), nil
	}
	return strconv.AppendInt(nil, int64(*i), 10), nil
}

func (i *Int) Value() int64 {
	if i == nil {
		return 0
	}
	return int64(*i)
}

func (i *Int) Ptr() *int64 {
	if i == nil {
		return nil
	}
	value := int64(*i)
	return &value
}

type Float float64

func ParseFloat(raw []byte) (*Float, bool) {
	text, kind, ok := scalarText(raw)
	if !ok {
		return nil, false
	}
	parsed, ok := parseFloatText(text, kind)
	if !ok {
		return nil, false
	}
	value := Float(parsed)
	return &value, true
}

func (f *Float) UnmarshalJSON(data []byte) error {
	if KindOf(data) == KindNull {
		return nil
	}
	text, kind, ok := scalarText(data)
	if !ok {
		return fmt.Errorf("%w: expected number-compatible scalar", ErrUnsupported)
	}
	if kind == KindString && strings.TrimSpace(text) == "" {
		*f = 0
		return nil
	}
	parsed, ok := parseFloatText(text, kind)
	if !ok {
		return fmt.Errorf("%w: %q is not a number", ErrUnsupported, text)
	}
	*f = Float(parsed)
	return nil
}

func (f *Float) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("null"), nil
	}
	return strconv.AppendFloat(nil, float64(*f), 'f', -1, 64), nil
}

func (f *Float) Value() float64 {
	if f == nil {
		return 0
	}
	return float64(*f)
}

func (f *Float) Ptr() *float64 {
	if f == nil {
		return nil
	}
	value := float64(*f)
	return &value
}

type Bool bool

func ParseBool(raw []byte) (*Bool, bool) {
	text, kind, ok := scalarText(raw)
	if !ok {
		return nil, false
	}
	parsed, ok := parseBoolText(text, kind)
	if !ok {
		return nil, false
	}
	value := Bool(parsed)
	return &value, true
}

func (b *Bool) UnmarshalJSON(data []byte) error {
	if KindOf(data) == KindNull {
		return nil
	}
	text, kind, ok := scalarText(data)
	if !ok {
		return fmt.Errorf("%w: expected boolean-compatible scalar", ErrUnsupported)
	}
	parsed, ok := parseBoolText(text, kind)
	if !ok {
		return fmt.Errorf("%w: %q is not a boolean", ErrUnsupported, text)
	}
	*b = Bool(parsed)
	return nil
}

func (b *Bool) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("null"), nil
	}
	return strconv.AppendBool(nil, bool(*b)), nil
}

func (b *Bool) Value() bool {
	if b == nil {
		return false
	}
	return bool(*b)
}

func (b *Bool) Ptr() *bool {
	if b == nil {
		return nil
	}
	value := bool(*b)
	return &value
}

type Time time.Time

func ParseTime(raw []byte) (*Time, bool) {
	text, kind, ok := scalarText(raw)
	if !ok {
		return nil, false
	}
	parsed, ok := parseTimeText(text, kind)
	if !ok {
		return nil, false
	}
	value := Time(parsed)
	return &value, true
}

func (t *Time) UnmarshalJSON(data []byte) error {
	if KindOf(data) == KindNull {
		return nil
	}
	text, kind, ok := scalarText(data)
	if !ok {
		return fmt.Errorf("%w: expected time-compatible scalar", ErrUnsupported)
	}
	if kind == KindString && strings.TrimSpace(text) == "" {
		*t = Time(time.Time{})
		return nil
	}
	parsed, ok := parseTimeText(text, kind)
	if !ok {
		return fmt.Errorf("%w: %q is not a recognised time", ErrUnsupported, text)
	}
	*t = Time(parsed)
	return nil
}

func (t *Time) MarshalJSON() ([]byte, error) {
	if t == nil || time.Time(*t).IsZero() {
		return []byte("null"), nil
	}
	return sonic.Marshal(time.Time(*t).UTC().Format(time.RFC3339))
}

func (t *Time) Time() time.Time {
	if t == nil {
		return time.Time{}
	}
	return time.Time(*t)
}

func (t *Time) Unix() *int64 {
	if t == nil || time.Time(*t).IsZero() {
		return nil
	}
	value := time.Time(*t).Unix()
	return &value
}

func (t *Time) Ptr() *time.Time {
	if t == nil || time.Time(*t).IsZero() {
		return nil
	}
	value := time.Time(*t)
	return &value
}

func parseIntText(text string, kind Kind) (int64, bool) {
	if kind == KindBool {
		return 0, false
	}
	cleaned := numericCleaner.Replace(strings.TrimSpace(text))
	if cleaned == "" {
		return 0, false
	}
	if parsed, err := strconv.ParseInt(cleaned, 10, 64); err == nil {
		return parsed, true
	}
	parsed, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, false
	}
	truncated := math.Trunc(parsed)
	if truncated >= math.MaxInt64 || truncated < math.MinInt64 {
		return 0, false
	}
	return int64(truncated), true
}

func parseFloatText(text string, kind Kind) (float64, bool) {
	if kind == KindBool {
		return 0, false
	}
	cleaned := numericCleaner.Replace(strings.TrimSpace(text))
	if cleaned == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, false
	}
	return parsed, true
}

func parseBoolText(text string, kind Kind) (bool, bool) {
	switch kind {
	case KindBool:
		return text == "true", true
	case KindNumber:
		parsed, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return false, false
		}
		switch parsed {
		case 1:
			return true, true
		case 0:
			return false, true
		default:
			return false, false
		}
	case KindString:
		switch strings.ToUpper(strings.TrimSpace(text)) {
		case "Y", "YES", "TRUE", "T", "1":
			return true, true
		case "N", "NO", "FALSE", "F", "0":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func parseTimeText(text string, kind Kind) (time.Time, bool) {
	switch kind {
	case KindNumber:
		return parseEpoch(text)
	case KindString:
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return time.Time{}, false
		}
		if isDigits(trimmed) {
			if len(trimmed) == 8 {
				if parsed, err := time.ParseInLocation("20060102", trimmed, time.UTC); err == nil {
					return parsed, true
				}
			}
			return parseEpoch(trimmed)
		}
		for _, layout := range timeLayouts {
			if parsed, err := time.ParseInLocation(layout, trimmed, time.UTC); err == nil {
				return parsed.UTC(), true
			}
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

func parseEpoch(text string) (time.Time, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed <= 0 {
		return time.Time{}, false
	}
	if parsed >= millisecondThreshold {
		return time.UnixMilli(int64(parsed)).UTC(), true
	}
	seconds, fraction := math.Modf(parsed)
	return time.Unix(int64(seconds), int64(fraction*float64(time.Second))).UTC(), true
}

func isDigits(value string) bool {
	for i := range len(value) {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return value != ""
}
