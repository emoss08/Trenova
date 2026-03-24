package jsonutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONDiff(t *testing.T) {
	t.Parallel()

	t.Run("detects updated field", func(t *testing.T) {
		t.Parallel()
		before := map[string]any{"name": "Alice"}
		after := map[string]any{"name": "Bob"}

		diff, err := jsonutils.JSONDiff(before, after, nil)

		require.NoError(t, err)
		require.Contains(t, diff, "name")
		assert.Equal(t, jsonutils.ChangeTypeUpdated, diff["name"].Type)
		assert.Equal(t, "Alice", diff["name"].From)
		assert.Equal(t, "Bob", diff["name"].To)
	})

	t.Run("detects created field", func(t *testing.T) {
		t.Parallel()
		before := map[string]any{"name": "Alice"}
		after := map[string]any{"name": "Alice", "age": float64(30)}

		diff, err := jsonutils.JSONDiff(before, after, nil)

		require.NoError(t, err)
		require.Contains(t, diff, "age")
		assert.Equal(t, jsonutils.ChangeTypeCreated, diff["age"].Type)
		assert.Equal(t, float64(30), diff["age"].To)
	})

	t.Run("detects deleted field", func(t *testing.T) {
		t.Parallel()
		before := map[string]any{"name": "Alice", "age": float64(30)}
		after := map[string]any{"name": "Alice"}

		diff, err := jsonutils.JSONDiff(before, after, nil)

		require.NoError(t, err)
		require.Contains(t, diff, "age")
		assert.Equal(t, jsonutils.ChangeTypeDeleted, diff["age"].Type)
		assert.Equal(t, float64(30), diff["age"].From)
	})

	t.Run("no diff for identical objects", func(t *testing.T) {
		t.Parallel()
		before := map[string]any{"name": "Alice", "age": float64(30)}
		after := map[string]any{"name": "Alice", "age": float64(30)}

		diff, err := jsonutils.JSONDiff(before, after, nil)

		require.NoError(t, err)
		assert.Empty(t, diff)
	})

	t.Run("detects nested field changes", func(t *testing.T) {
		t.Parallel()
		before := map[string]any{
			"address": map[string]any{"city": "NYC"},
		}
		after := map[string]any{
			"address": map[string]any{"city": "LA"},
		}

		diff, err := jsonutils.JSONDiff(before, after, nil)

		require.NoError(t, err)
		require.Contains(t, diff, "address.city")
		assert.Equal(t, jsonutils.ChangeTypeUpdated, diff["address.city"].Type)
	})

	t.Run("respects ignore fields option", func(t *testing.T) {
		t.Parallel()
		before := map[string]any{"name": "Alice", "updated_at": "old"}
		after := map[string]any{"name": "Bob", "updated_at": "new"}

		opts := jsonutils.DefaultOptions()
		opts.IgnoreFields = []string{"updated_at"}

		diff, err := jsonutils.JSONDiff(before, after, opts)

		require.NoError(t, err)
		assert.Contains(t, diff, "name")
		assert.NotContains(t, diff, "updated_at")
	})

	t.Run("detects array changes", func(t *testing.T) {
		t.Parallel()
		before := map[string]any{"tags": []any{"a", "b"}}
		after := map[string]any{"tags": []any{"a", "c"}}

		diff, err := jsonutils.JSONDiff(before, after, nil)

		require.NoError(t, err)
		require.Contains(t, diff, "tags")
		assert.Equal(t, jsonutils.FieldTypeArray, diff["tags"].FieldType)
	})

	t.Run("returns error exceeding max depth", func(t *testing.T) {
		t.Parallel()
		opts := jsonutils.DefaultOptions()
		opts.MaxDepth = 1

		before := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"value": "old",
				},
			},
		}
		after := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"value": "new",
				},
			},
		}

		_, err := jsonutils.JSONDiff(before, after, opts)

		assert.Error(t, err)
	})

	t.Run("determines correct field types", func(t *testing.T) {
		t.Parallel()
		before := map[string]any{}
		after := map[string]any{
			"str":  "hello",
			"num":  float64(42),
			"bool": true,
			"arr":  []any{1, 2},
			"obj":  map[string]any{"k": "v"},
		}

		diff, err := jsonutils.JSONDiff(before, after, nil)

		require.NoError(t, err)
		assert.Equal(t, jsonutils.FieldTypeString, diff["str"].FieldType)
		assert.Equal(t, jsonutils.FieldTypeNumber, diff["num"].FieldType)
		assert.Equal(t, jsonutils.FieldTypeBoolean, diff["bool"].FieldType)
		assert.Equal(t, jsonutils.FieldTypeArray, diff["arr"].FieldType)
		assert.Equal(t, jsonutils.FieldTypeObject, diff["obj"].FieldType)
	})

	t.Run("works with structs", func(t *testing.T) {
		t.Parallel()
		type Item struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}

		before := Item{Name: "widget", Count: 5}
		after := Item{Name: "widget", Count: 10}

		diff, err := jsonutils.JSONDiff(before, after, nil)

		require.NoError(t, err)
		require.Contains(t, diff, "count")
		assert.Equal(t, jsonutils.ChangeTypeUpdated, diff["count"].Type)
	})
}

func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil options", func(t *testing.T) {
		t.Parallel()
		opts := jsonutils.DefaultOptions()

		require.NotNil(t, opts)
		assert.Equal(t, 10, opts.MaxDepth)
		assert.False(t, opts.IgnoreCase)
		assert.Empty(t, opts.IgnoreFields)
		assert.Contains(t, opts.CustomComparors, "time")
	})
}
