package businesscentral

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/businesscentralerrors"
)

const (
	dateLayout    = "2006-01-02"
	unsetDate     = "0001-01-01"
	dateTimeUnset = "0001-01-01T00:00:00Z"
)

func ParseDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, businesscentralerrors.ErrDateRequired
	}
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, businesscentralerrors.ErrInvalidDate
	}
	return parsed, nil
}

func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateLayout)
}

func InputDate(raw string, required bool) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		if required {
			return "", businesscentralerrors.ErrDateRequired
		}
		return "", nil
	}
	if _, err := ParseDate(value); err != nil {
		return "", err
	}
	return value, nil
}

func OutputDate(raw string) string {
	value := strings.TrimSpace(raw)
	if value == unsetDate {
		return ""
	}
	return value
}

func odataDateTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

type WireTime time.Time

func (w *WireTime) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*w = WireTime{}
		return nil
	}
	text, err := strconv.Unquote(string(trimmed))
	if err != nil {
		return businesscentralerrors.ErrInvalidDate
	}
	text = strings.TrimSpace(text)
	if text == "" || text == dateTimeUnset {
		*w = WireTime{}
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return businesscentralerrors.ErrInvalidDate
	}
	*w = WireTime(parsed.UTC())
	return nil
}

func (w WireTime) Time() time.Time {
	return time.Time(w)
}
