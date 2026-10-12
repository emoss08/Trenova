package businesscentral

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

const (
	dateLayout    = "2006-01-02"
	unsetDate     = "0001-01-01"
	dateTimeUnset = "0001-01-01T00:00:00Z"
)

func ParseDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, ErrDateRequired
	}
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return parsed, nil
}

func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateLayout)
}

func inputDate(raw string, required bool) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		if required {
			return "", ErrDateRequired
		}
		return "", nil
	}
	if _, err := ParseDate(value); err != nil {
		return "", err
	}
	return value, nil
}

func outputDate(raw string) string {
	value := strings.TrimSpace(raw)
	if value == unsetDate {
		return ""
	}
	return value
}

func odataDateTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

type wireTime time.Time

func (w *wireTime) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*w = wireTime{}
		return nil
	}
	text, err := strconv.Unquote(string(trimmed))
	if err != nil {
		return ErrInvalidDate
	}
	text = strings.TrimSpace(text)
	if text == "" || text == dateTimeUnset {
		*w = wireTime{}
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return ErrInvalidDate
	}
	*w = wireTime(parsed.UTC())
	return nil
}

func (w wireTime) time() time.Time {
	return time.Time(w)
}
