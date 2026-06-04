package graphql

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

func TestComplexityRoot_Trailers(t *testing.T) {
	t.Parallel()

	root := complexityRoot()
	status := domaintypes.EquipmentStatusAvailable

	tests := []struct {
		name            string
		first           *int
		childComplexity int
		expected        int
	}{
		{
			name:            "default limit",
			childComplexity: 2,
			expected:        40,
		},
		{
			name:            "requested limit",
			first:           intPtrForTest(15),
			childComplexity: 3,
			expected:        45,
		},
		{
			name:            "caps at max limit",
			first:           intPtrForTest(pagination.MaxLimit + 1),
			childComplexity: 2,
			expected:        200,
		},
		{
			name:            "common relation query stays below fixed limit",
			first:           intPtrForTest(20),
			childComplexity: 16,
			expected:        320,
		},
		{
			name:            "rich table query stays below fixed limit",
			first:           intPtrForTest(20),
			childComplexity: 41,
			expected:        820,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cost := root.Query.Trailers(
				tt.childComplexity,
				tt.first,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				&status,
				nil,
				nil,
			)

			assert.Equal(t, tt.expected, cost)
		})
	}
}

func TestCountComplexity(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 50, countComplexity(5, 10))
}

func intPtrForTest(value int) *int {
	return &value
}
