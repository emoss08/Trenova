package floatutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeanPtr(t *testing.T) {
	t.Parallel()

	mean := MeanPtr([]float64{0.5, 0.7, 0.9})
	require.NotNil(t, mean)
	assert.InDelta(t, 0.7, *mean, 1e-9)
	assert.Nil(t, MeanPtr(nil))
	assert.Nil(t, MeanPtr([]float64{}))
}
