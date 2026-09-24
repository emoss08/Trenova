package floatutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMedian(t *testing.T) {
	t.Parallel()

	values := []float64{0.9, 0.7, 0.8}
	assert.InDelta(t, 0.8, Median(values), 1e-9)
	assert.Equal(t, []float64{0.9, 0.7, 0.8}, values, "the input is left in its order")
	assert.InDelta(t, 0.85, Median([]float64{0.9, 0.7, 0.8, 1.0}), 1e-9)
	assert.InDelta(t, 0.4, Median([]float64{0.4}), 1e-9)
	assert.Zero(t, Median(nil))
}
