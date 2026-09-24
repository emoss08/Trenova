package vectorutils_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/shared/vectorutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackRoundTrips(t *testing.T) {
	t.Parallel()

	vector := []float32{0.25, -1.5, 3.125, 0}
	packed := vectorutils.Pack(vector)
	require.Len(t, packed, 16)

	unpacked, err := vectorutils.Unpack(packed)
	require.NoError(t, err)
	assert.Equal(t, vector, unpacked)
}

func TestUnpackRefusesTornAndNonFiniteInput(t *testing.T) {
	t.Parallel()

	_, err := vectorutils.Unpack([]byte{1, 2, 3})
	require.ErrorIs(t, err, vectorutils.ErrPackedLength)

	_, err = vectorutils.Unpack(vectorutils.Pack([]float32{float32(math.NaN())}))
	require.ErrorIs(t, err, vectorutils.ErrNotFinite)

	empty, err := vectorutils.Unpack(nil)
	require.NoError(t, err)
	assert.Nil(t, empty)
}

func TestCosine(t *testing.T) {
	t.Parallel()

	same, err := vectorutils.Cosine([]float32{1, 2}, []float32{2, 4})
	require.NoError(t, err)
	assert.InDelta(t, 1, same, 1e-9)

	orthogonal, err := vectorutils.Cosine([]float32{1, 0}, []float32{0, 3})
	require.NoError(t, err)
	assert.InDelta(t, 0, orthogonal, 1e-9)

	_, err = vectorutils.Cosine([]float32{1}, []float32{1, 2})
	require.ErrorIs(t, err, vectorutils.ErrDimensionsDiff)

	_, err = vectorutils.Cosine([]float32{0, 0}, []float32{1, 2})
	require.ErrorIs(t, err, vectorutils.ErrZeroMagnitude)
}

func TestNormalized(t *testing.T) {
	t.Parallel()

	unit, err := vectorutils.Normalized([]float32{3, 4})
	require.NoError(t, err)
	assert.InDelta(t, 1, vectorutils.Magnitude(unit), 1e-6)

	_, err = vectorutils.Normalized([]float32{0, 0})
	require.ErrorIs(t, err, vectorutils.ErrZeroMagnitude)
}
