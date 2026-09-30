package sliceutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/stretchr/testify/assert"
)

func TestDedupeSorted(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"alpha", "beta", "gamma"},
		sliceutils.DedupeSorted([]string{"gamma", "alpha", "beta", "alpha"}),
	)
	assert.Equal(
		t,
		[]string{"alpha", "beta"},
		sliceutils.DedupeSorted([]string{"  beta  ", "", "alpha", "beta"}),
		"blanks go and members are trimmed before they are compared",
	)
	assert.Empty(t, sliceutils.DedupeSorted(nil))
}
