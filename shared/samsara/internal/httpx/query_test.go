package httpx

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQueryHelpers(t *testing.T) {
	t.Parallel()

	values := url.Values{}

	SetInt(values, "limit", 0)
	assert.Equal(t, "", values.Get("limit"))
	SetInt(values, "limit", 10)
	assert.Equal(t, "10", values.Get("limit"))

	SetInt64(values, "endMs", -1)
	assert.Equal(t, "", values.Get("endMs"))
	SetInt64(values, "endMs", 42)
	assert.Equal(t, "42", values.Get("endMs"))

	SetString(values, "after", "")
	assert.Equal(t, "", values.Get("after"))
	SetString(values, "after", "cursor")
	assert.Equal(t, "cursor", values.Get("after"))

	SetBool(values, "includeTags", false)
	assert.Equal(t, "", values.Get("includeTags"))
	SetBool(values, "includeTags", true)
	assert.Equal(t, "true", values.Get("includeTags"))

	SetStringsCSV(values, "ids", nil)
	assert.Equal(t, "", values.Get("ids"))
	SetStringsCSV(values, "ids", []string{"a", "b"})
	assert.Equal(t, "a,b", values.Get("ids"))

	SetTime(values, "startTime", nil)
	assert.Equal(t, "", values.Get("startTime"))
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	SetTime(values, "startTime", &now)
	assert.Equal(t, now.Format(time.RFC3339), values.Get("startTime"))
}
