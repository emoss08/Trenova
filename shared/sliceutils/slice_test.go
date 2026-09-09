package sliceutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/stretchr/testify/assert"
)

func TestFirstOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("returns fallback for empty slice", func(t *testing.T) {
		t.Parallel()
		result := sliceutils.FirstOrDefault([]int{}, 42)
		assert.Equal(t, 42, result)
	})

	t.Run("returns fallback for nil slice", func(t *testing.T) {
		t.Parallel()
		var nilSlice []string
		result := sliceutils.FirstOrDefault(nilSlice, "default")
		assert.Equal(t, "default", result)
	})

	t.Run("returns first element for single element slice", func(t *testing.T) {
		t.Parallel()
		result := sliceutils.FirstOrDefault([]int{100}, 42)
		assert.Equal(t, 100, result)
	})

	t.Run("returns first element for multi element slice", func(t *testing.T) {
		t.Parallel()
		result := sliceutils.FirstOrDefault([]int{1, 2, 3, 4, 5}, 42)
		assert.Equal(t, 1, result)
	})

	t.Run("works with string type", func(t *testing.T) {
		t.Parallel()
		result := sliceutils.FirstOrDefault([]string{"first", "second"}, "default")
		assert.Equal(t, "first", result)
	})

	t.Run("works with struct type", func(t *testing.T) {
		t.Parallel()
		type item struct {
			id   int
			name string
		}
		items := []item{{id: 1, name: "one"}, {id: 2, name: "two"}}
		fallback := item{id: 0, name: "fallback"}

		result := sliceutils.FirstOrDefault(items, fallback)
		assert.Equal(t, item{id: 1, name: "one"}, result)
	})

	t.Run("returns fallback for empty struct slice", func(t *testing.T) {
		t.Parallel()
		type item struct {
			id   int
			name string
		}
		fallback := item{id: 0, name: "fallback"}

		result := sliceutils.FirstOrDefault([]item{}, fallback)
		assert.Equal(t, fallback, result)
	})
}

func TestPage(t *testing.T) {
	t.Parallel()

	items := []string{"a", "b", "c", "d", "e"}

	tests := []struct {
		name   string
		offset int
		limit  int
		want   []string
	}{
		{name: "first page", offset: 0, limit: 2, want: []string{"a", "b"}},
		{name: "middle page", offset: 2, limit: 2, want: []string{"c", "d"}},
		{name: "short last page", offset: 4, limit: 2, want: []string{"e"}},
		{name: "offset past the end", offset: 5, limit: 2, want: []string{}},
		{name: "negative offset starts at the top", offset: -3, limit: 2, want: []string{"a", "b"}},
		{name: "limit beyond the end", offset: 1, limit: 99, want: []string{"b", "c", "d", "e"}},
		{name: "zero limit keeps the tail", offset: 3, limit: 0, want: []string{"d", "e"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, sliceutils.Page(items, tt.offset, tt.limit))
		})
	}

	t.Run("returns a copy", func(t *testing.T) {
		t.Parallel()
		page := sliceutils.Page(items, 0, 2)
		page[0] = "mutated"
		assert.Equal(t, "a", items[0])
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{}, sliceutils.Page([]string{}, 0, 5))
	})
}
