package sliceutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/stretchr/testify/assert"
)

type sequenced struct {
	name     string
	sequence int
}

func TestSortedNonNil(t *testing.T) {
	t.Parallel()

	third := &sequenced{name: "third", sequence: 3}
	first := &sequenced{name: "first", sequence: 1}
	tied := &sequenced{name: "tied", sequence: 1}
	items := []*sequenced{third, nil, first, tied}

	sorted := sliceutils.SortedNonNil(items, func(item *sequenced) int { return item.sequence })

	assert.Equal(t, []*sequenced{first, tied, third}, sorted)
	assert.Equal(t, []*sequenced{third, nil, first, tied}, items, "the input keeps its order")
	assert.Empty(t, sliceutils.SortedNonNil[sequenced, int](nil, nil))
}
