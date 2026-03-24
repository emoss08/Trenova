package formulatypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetNestedValue(t *testing.T) {
	t.Parallel()

	t.Run("sets simple top-level key", func(t *testing.T) {
		t.Parallel()
		m := make(map[string]any)
		SetNestedValue(m, "name", "test")
		assert.Equal(t, "test", m["name"])
	})

	t.Run("sets nested value with dot path", func(t *testing.T) {
		t.Parallel()
		m := make(map[string]any)
		SetNestedValue(m, "user.address.city", "NYC")
		user := m["user"].(map[string]any)
		address := user["address"].(map[string]any)
		assert.Equal(t, "NYC", address["city"])
	})

	t.Run("strips array bracket suffix", func(t *testing.T) {
		t.Parallel()
		m := make(map[string]any)
		SetNestedValue(m, "items[].name", "item1")
		items := m["items"].(map[string]any)
		assert.Equal(t, "item1", items["name"])
	})

	t.Run("uses existing intermediate map", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{
			"user": map[string]any{
				"existing": "value",
			},
		}
		SetNestedValue(m, "user.name", "test")
		user := m["user"].(map[string]any)
		assert.Equal(t, "test", user["name"])
		assert.Equal(t, "value", user["existing"])
	})

	t.Run("stops when intermediate is not a map", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{
			"user": "not a map",
		}
		SetNestedValue(m, "user.name", "test")
		assert.Equal(t, "not a map", m["user"])
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{
			"key": "old",
		}
		SetNestedValue(m, "key", "new")
		assert.Equal(t, "new", m["key"])
	})

	t.Run("handles deeply nested path", func(t *testing.T) {
		t.Parallel()
		m := make(map[string]any)
		SetNestedValue(m, "a.b.c.d.e", 42)
		a := m["a"].(map[string]any)
		b := a["b"].(map[string]any)
		c := b["c"].(map[string]any)
		d := c["d"].(map[string]any)
		assert.Equal(t, 42, d["e"])
	})

	t.Run("handles array bracket in middle of path", func(t *testing.T) {
		t.Parallel()
		m := make(map[string]any)
		SetNestedValue(m, "items[].price", 9.99)
		items := m["items"].(map[string]any)
		assert.Equal(t, 9.99, items["price"])
	})
}
