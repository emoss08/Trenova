package setutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func set(items ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}

	return out
}

func TestJaccard(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.0, Jaccard(set(), set()), 1e-9)
	assert.InDelta(t, 1.0, Jaccard(set("a", "b"), set("b", "a")), 1e-9)
	assert.InDelta(t, 0.5, Jaccard(set("a", "b", "c"), set("b", "c", "d", "a", "e", "f")), 1e-9)
	assert.InDelta(t, 0.0, Jaccard(set("a"), set("b")), 1e-9)
	assert.InDelta(t, 1.0/3.0, Jaccard(set("a", "b"), set("b", "c")), 1e-9)
}

func TestUnion(t *testing.T) {
	t.Parallel()

	merged := Union(nil, set("a", "b"))
	merged = Union(merged, set("b", "c"))
	assert.Equal(t, set("a", "b", "c"), merged)
}
