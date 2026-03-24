package httpx

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

func SetInt(values url.Values, key string, value int) {
	if value <= 0 {
		return
	}
	values.Set(key, strconv.Itoa(value))
}

func SetInt64(values url.Values, key string, value int64) {
	if value <= 0 {
		return
	}
	values.Set(key, strconv.FormatInt(value, 10))
}

func SetString(values url.Values, key, value string) {
	if value == "" {
		return
	}
	values.Set(key, value)
}

func SetBool(values url.Values, key string, value bool) {
	if !value {
		return
	}
	values.Set(key, strconv.FormatBool(value))
}

func SetStringsCSV(values url.Values, key string, items []string) {
	if len(items) == 0 {
		return
	}
	values.Set(key, strings.Join(items, ","))
}

func SetTime(values url.Values, key string, t *time.Time) {
	if t == nil {
		return
	}
	values.Set(key, t.Format(time.RFC3339))
}
