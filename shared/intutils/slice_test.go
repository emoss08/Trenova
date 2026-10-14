package intutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/stretchr/testify/assert"
)

func TestIntsConvertsEveryItemInOrder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []int{3, -1, 1_790_000_000}, intutils.Ints([]int64{3, -1, 1_790_000_000}))
	assert.Equal(t, []int{}, intutils.Ints([]int64(nil)))
}
