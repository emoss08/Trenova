package sliceutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/stretchr/testify/assert"
)

func TestKeepEnds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		weights  []int
		budget   int
		wantHead int
		wantTail int
	}{
		{name: "nothing to keep", weights: nil, budget: 10},
		{name: "everything fits", weights: []int{1, 2, 3}, budget: 6, wantHead: 2, wantTail: 1},
		{name: "drops the middle", weights: []int{2, 2, 2, 2, 2}, budget: 6, wantHead: 2, wantTail: 1},
		{
			name:     "a heavy first item keeps the tail going",
			weights:  []int{9, 1, 1, 1},
			budget:   3,
			wantHead: 0,
			wantTail: 3,
		},
		{
			name:     "a heavy last item keeps the head going",
			weights:  []int{1, 1, 1, 9},
			budget:   3,
			wantHead: 3,
			wantTail: 0,
		},
		{name: "nothing fits", weights: []int{5, 5}, budget: 4},
		{name: "a zero budget keeps weightless items", weights: []int{0, 0}, budget: 0, wantHead: 1, wantTail: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			head, tail := sliceutils.KeepEnds(tt.weights, tt.budget)

			assert.Equal(t, tt.wantHead, head)
			assert.Equal(t, tt.wantTail, tail)
			assert.LessOrEqual(t, head+tail, len(tt.weights))

			kept := 0
			for _, weight := range tt.weights[:head] {
				kept += weight
			}
			for _, weight := range tt.weights[len(tt.weights)-tail:] {
				kept += weight
			}
			assert.LessOrEqual(t, kept, max(tt.budget, 0))
		})
	}
}
