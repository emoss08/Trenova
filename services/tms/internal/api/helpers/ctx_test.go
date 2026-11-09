package helpers_test

import (
	"testing"

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
