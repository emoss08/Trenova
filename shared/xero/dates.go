package xero

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	dateLayout       = "2006-01-02"
	msDatePrefix     = "/Date("
	msDateSuffix     = ")/"
	offsetDigits     = 4
	millisecondsBase = 10
)

var isoLayouts = [...]string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	dateLayout,
}

func ParseDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, ErrInvalidDate
	}
	if strings.HasPrefix(value, msDatePrefix) && strings.HasSuffix(value, msDateSuffix) {
		return parseMSDate(value[len(msDatePrefix) : len(value)-len(msDateSuffix)])
	}
	for _, layout := range isoLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, ErrInvalidDate
}

func parseMSDate(inner string) (time.Time, error) {
	digits := inner
	if cut := strings.LastIndexAny(inner, "+-"); cut > 0 {
		offset := inner[cut+1:]
		if len(offset) != offsetDigits || !allDigits(offset) {
			return time.Time{}, ErrInvalidDate
		}
		digits = inner[:cut]
	}
	millis, err := strconv.ParseInt(digits, millisecondsBase, 64)
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return time.UnixMilli(millis).UTC(), nil
}

func allDigits(value string) bool {
	for idx := range len(value) {
		if value[idx] < '0' || value[idx] > '9' {
			return false
		}
	}
	return value != ""
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateLayout)
}

func formatHTTPDate(t time.Time) string {
	return t.UTC().Format(http.TimeFormat)
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
	if strings.TrimSpace(text) == "" {
		*w = wireTime{}
		return nil
	}
	parsed, err := ParseDate(text)
	if err != nil {
		return err
	}
	*w = wireTime(parsed)
	return nil
}

func (w wireTime) time() time.Time {
	return time.Time(w)
}

func (w wireTime) ptr() *time.Time {
	t := time.Time(w)
	if t.IsZero() {
		return nil
	}
	return &t
}
