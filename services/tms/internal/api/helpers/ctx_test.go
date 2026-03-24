package helpers_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
)

func TestQueryString(t *testing.T) {
	t.Parallel()

	t.Run("returns value when present", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"name": "test"})
		result := helpers.QueryString(ctx.Context, "name")
		assert.Equal(t, "test", result)
	})

	t.Run("returns empty string when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryString(ctx.Context, "name")
		assert.Equal(t, "", result)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryString(ctx.Context, "name", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("returns value over default when present", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"name": "actual"})
		result := helpers.QueryString(ctx.Context, "name", "default")
		assert.Equal(t, "actual", result)
	})
}

func TestQueryStringTrimmed(t *testing.T) {
	t.Parallel()

	t.Run("trims whitespace", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"name": "  test  "})
		result := helpers.QueryStringTrimmed(ctx.Context, "name")
		assert.Equal(t, "test", result)
	})

	t.Run("returns default when empty after trim", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"name": "   "})
		result := helpers.QueryStringTrimmed(ctx.Context, "name", "default")
		assert.Equal(t, "default", result)
	})
}

func TestQueryBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"1", "1", true},
		{"0", "0", false},
		{"yes", "yes", true},
		{"no", "no", false},
		{"on", "on", true},
		{"off", "off", false},
		{"TRUE uppercase", "TRUE", true},
		{"FALSE uppercase", "FALSE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"flag": tt.value})
			result := helpers.QueryBool(ctx.Context, "flag")
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("returns false when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryBool(ctx.Context, "flag")
		assert.False(t, result)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryBool(ctx.Context, "flag", true)
		assert.True(t, result)
	})

	t.Run("returns default for invalid value", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"flag": "invalid"})
		result := helpers.QueryBool(ctx.Context, "flag", true)
		assert.True(t, result)
	})
}

func TestQueryInt(t *testing.T) {
	t.Parallel()

	t.Run("parses valid integer", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"count": "42"})
		result := helpers.QueryInt(ctx.Context, "count")
		assert.Equal(t, 42, result)
	})

	t.Run("parses negative integer", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"count": "-10"})
		result := helpers.QueryInt(ctx.Context, "count")
		assert.Equal(t, -10, result)
	})

	t.Run("returns zero when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryInt(ctx.Context, "count")
		assert.Equal(t, 0, result)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryInt(ctx.Context, "count", 100)
		assert.Equal(t, 100, result)
	})

	t.Run("returns default for invalid value", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"count": "abc"})
		result := helpers.QueryInt(ctx.Context, "count", 50)
		assert.Equal(t, 50, result)
	})
}

func TestQueryInt64(t *testing.T) {
	t.Parallel()

	t.Run("parses large integer", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().
			WithQuery(map[string]string{"id": "9223372036854775807"})
		result := helpers.QueryInt64(ctx.Context, "id")
		assert.Equal(t, int64(9223372036854775807), result)
	})

	t.Run("returns default when invalid", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"id": "invalid"})
		result := helpers.QueryInt64(ctx.Context, "id", 123)
		assert.Equal(t, int64(123), result)
	})
}

func TestQueryUint(t *testing.T) {
	t.Parallel()

	t.Run("parses valid unsigned integer", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"count": "42"})
		result := helpers.QueryUint(ctx.Context, "count")
		assert.Equal(t, uint(42), result)
	})

	t.Run("returns default for negative value", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"count": "-10"})
		result := helpers.QueryUint(ctx.Context, "count", 5)
		assert.Equal(t, uint(5), result)
	})
}

func TestQueryUint64(t *testing.T) {
	t.Parallel()

	t.Run("parses large unsigned integer", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().
			WithQuery(map[string]string{"id": "18446744073709551615"})
		result := helpers.QueryUint64(ctx.Context, "id")
		assert.Equal(t, uint64(18446744073709551615), result)
	})
}

func TestQueryFloat64(t *testing.T) {
	t.Parallel()

	t.Run("parses valid float", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"price": "19.99"})
		result := helpers.QueryFloat64(ctx.Context, "price")
		assert.Equal(t, 19.99, result)
	})

	t.Run("parses negative float", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"temp": "-5.5"})
		result := helpers.QueryFloat64(ctx.Context, "temp")
		assert.Equal(t, -5.5, result)
	})

	t.Run("returns zero when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryFloat64(ctx.Context, "price")
		assert.Equal(t, 0.0, result)
	})

	t.Run("returns default for invalid value", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"price": "expensive"})
		result := helpers.QueryFloat64(ctx.Context, "price", 9.99)
		assert.Equal(t, 9.99, result)
	})
}

func TestQueryTime(t *testing.T) {
	t.Parallel()

	t.Run("parses RFC3339 format", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().
			WithQuery(map[string]string{"date": "2024-01-15T10:30:00Z"})
		result := helpers.QueryTime(ctx.Context, "date")
		expected := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.Equal(t, expected, result)
	})

	t.Run("parses RFC3339Nano format", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().
			WithQuery(map[string]string{"date": "2024-01-15T10:30:00.123456789Z"})
		result := helpers.QueryTime(ctx.Context, "date")
		assert.False(t, result.IsZero())
	})

	t.Run("parses Unix timestamp seconds", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"date": "1705315800"})
		result := helpers.QueryTime(ctx.Context, "date")
		assert.False(t, result.IsZero())
	})

	t.Run("parses Unix timestamp milliseconds", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"date": "1705315800000"})
		result := helpers.QueryTime(ctx.Context, "date")
		assert.False(t, result.IsZero())
	})

	t.Run("returns zero time when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryTime(ctx.Context, "date")
		assert.True(t, result.IsZero())
	})

	t.Run("returns default for invalid format", func(t *testing.T) {
		t.Parallel()
		defaultTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"date": "invalid"})
		result := helpers.QueryTime(ctx.Context, "date", defaultTime)
		assert.Equal(t, defaultTime, result)
	})
}

func TestQueryUnixTime(t *testing.T) {
	t.Parallel()

	t.Run("parses unix timestamp", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"ts": "1705315800"})
		result := helpers.QueryUnixTime(ctx.Context, "ts")
		assert.Equal(t, int64(1705315800), result)
	})

	t.Run("returns default for invalid value", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"ts": "invalid"})
		result := helpers.QueryUnixTime(ctx.Context, "ts", 12345)
		assert.Equal(t, int64(12345), result)
	})
}

func TestQuerySlice(t *testing.T) {
	t.Parallel()

	t.Run("parses comma-separated values", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"tags": "a,b,c"})
		result := helpers.QuerySlice(ctx.Context, "tags")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("parses repeated keys", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().
			WithQueryValues(map[string][]string{"tags": {"a", "b", "c"}})
		result := helpers.QuerySlice(ctx.Context, "tags")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("trims whitespace from values", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"tags": " a , b , c "})
		result := helpers.QuerySlice(ctx.Context, "tags")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("filters empty values", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"tags": "a,,b"})
		result := helpers.QuerySlice(ctx.Context, "tags")
		assert.Equal(t, []string{"a", "b"}, result)
	})

	t.Run("returns empty slice when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QuerySlice(ctx.Context, "tags")
		assert.Empty(t, result)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QuerySlice(ctx.Context, "tags", []string{"default"})
		assert.Equal(t, []string{"default"}, result)
	})
}

func TestQueryIntSlice(t *testing.T) {
	t.Parallel()

	t.Run("parses comma-separated integers", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"ids": "1,2,3"})
		result := helpers.QueryIntSlice(ctx.Context, "ids")
		assert.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("skips invalid integers", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"ids": "1,invalid,3"})
		result := helpers.QueryIntSlice(ctx.Context, "ids")
		assert.Equal(t, []int{1, 3}, result)
	})

	t.Run("returns default when all invalid", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"ids": "a,b,c"})
		result := helpers.QueryIntSlice(ctx.Context, "ids", []int{0})
		assert.Equal(t, []int{0}, result)
	})
}

func TestQueryEnum(t *testing.T) {
	t.Parallel()

	allowed := []string{"pending", "active", "completed"}

	t.Run("returns value when in allowed list", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"status": "active"})
		result := helpers.QueryEnum(ctx.Context, "status", allowed)
		assert.Equal(t, "active", result)
	})

	t.Run("returns empty when not in allowed list", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"status": "invalid"})
		result := helpers.QueryEnum(ctx.Context, "status", allowed)
		assert.Equal(t, "", result)
	})

	t.Run("returns default when not in allowed list", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"status": "invalid"})
		result := helpers.QueryEnum(ctx.Context, "status", allowed, "pending")
		assert.Equal(t, "pending", result)
	})

	t.Run("is case-sensitive", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"status": "ACTIVE"})
		result := helpers.QueryEnum(ctx.Context, "status", allowed, "pending")
		assert.Equal(t, "pending", result)
	})
}

func TestQueryEnumIgnoreCase(t *testing.T) {
	t.Parallel()

	allowed := []string{"Pending", "Active", "Completed"}

	t.Run("matches case-insensitively", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"status": "active"})
		result := helpers.QueryEnumIgnoreCase(ctx.Context, "status", allowed)
		assert.Equal(t, "Active", result)
	})

	t.Run("returns original case from allowed list", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"status": "PENDING"})
		result := helpers.QueryEnumIgnoreCase(ctx.Context, "status", allowed)
		assert.Equal(t, "Pending", result)
	})

	t.Run("returns default when not found", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"status": "invalid"})
		result := helpers.QueryEnumIgnoreCase(ctx.Context, "status", allowed, "default")
		assert.Equal(t, "default", result)
	})
}

func TestQueryBounded(t *testing.T) {
	t.Parallel()

	t.Run("returns value within bounds", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"page": "5"})
		result := helpers.QueryBounded(ctx.Context, "page", 1, 10)
		assert.Equal(t, 5, result)
	})

	t.Run("clamps to min", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"page": "0"})
		result := helpers.QueryBounded(ctx.Context, "page", 1, 10)
		assert.Equal(t, 1, result)
	})

	t.Run("clamps to max", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"page": "100"})
		result := helpers.QueryBounded(ctx.Context, "page", 1, 10)
		assert.Equal(t, 10, result)
	})

	t.Run("uses default when missing then clamps", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryBounded(ctx.Context, "page", 1, 10, 5)
		assert.Equal(t, 5, result)
	})
}

func TestQueryBounded64(t *testing.T) {
	t.Parallel()

	t.Run("clamps large values", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"limit": "9999999999999"})
		result := helpers.QueryBounded64(ctx.Context, "limit", 0, 1000)
		assert.Equal(t, int64(1000), result)
	})
}

func TestHasQuery(t *testing.T) {
	t.Parallel()

	t.Run("returns true when key exists", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"flag": ""})
		result := helpers.HasQuery(ctx.Context, "flag")
		assert.True(t, result)
	})

	t.Run("returns true when key has value", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"flag": "value"})
		result := helpers.HasQuery(ctx.Context, "flag")
		assert.True(t, result)
	})

	t.Run("returns false when key missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.HasQuery(ctx.Context, "flag")
		assert.False(t, result)
	})
}

func TestQueryOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("returns value when present", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"name": "test"})
		result := helpers.QueryOrDefault(ctx.Context, "name", "default")
		assert.Equal(t, "test", result)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryOrDefault(ctx.Context, "name", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("returns default when empty", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"name": ""})
		result := helpers.QueryOrDefault(ctx.Context, "name", "default")
		assert.Equal(t, "default", result)
	})
}

func TestQuerySlice_AllEmptyCommaSeparated(t *testing.T) {
	t.Parallel()

	t.Run("returns default when multiple repeated values are all empty", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQueryValues(map[string][]string{"tags": {"", ""}})
		result := helpers.QuerySlice(ctx.Context, "tags", []string{"fallback"})
		assert.Equal(t, []string{"fallback"}, result)
	})

	t.Run("returns empty when multiple repeated values are all whitespace", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().
			WithQueryValues(map[string][]string{"tags": {"  ", " "}})
		result := helpers.QuerySlice(ctx.Context, "tags")
		assert.Empty(t, result)
	})
}

func TestQuerySlice_MultipleEmptyValues(t *testing.T) {
	t.Parallel()

	ctx := testutil.NewGinTestContext().WithQueryValues(map[string][]string{"tags": {"", "", ""}})
	result := helpers.QuerySlice(ctx.Context, "tags", []string{"default"})

	assert.Equal(t, []string{"default"}, result)
}

func TestQueryIntSlice_EmptyStrValues(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no query present and no default", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext()
		result := helpers.QueryIntSlice(ctx.Context, "ids")
		assert.Empty(t, result)
	})

	t.Run("returns default when no valid ints and default provided", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"ids": "abc,xyz"})
		result := helpers.QueryIntSlice(ctx.Context, "ids", []int{99})
		assert.Equal(t, []int{99}, result)
	})

	t.Run("returns empty when no valid ints and no default", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.NewGinTestContext().WithQuery(map[string]string{"ids": "abc,xyz"})
		result := helpers.QueryIntSlice(ctx.Context, "ids")
		assert.Empty(t, result)
	})
}
