package deskbench

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatOf_MeanAndStudentInterval(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Stat{}, statOf(nil))

	single := statOf([]float64{7})
	assert.Equal(t, Stat{N: 1, Mean: 7, Low: 7, High: 7}, single)

	stat := statOf([]float64{2, 4, 4, 6})
	assert.Equal(t, 4, stat.N)
	assert.InDelta(t, 4.0, stat.Mean, 1e-9)
	assert.InDelta(t, 4.0-3.182*0.816497, stat.Low, 1e-3)
	assert.InDelta(t, 4.0+3.182*0.816497, stat.High, 1e-3)
}

func TestRateOf_WilsonInterval(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Rate{}, rateOf(0, 0))

	all := rateOf(4, 4)
	assert.InDelta(t, 1.0, all.Value, 1e-9)
	assert.InDelta(t, 0.5101, all.Low, 1e-3)
	assert.InDelta(t, 1.0, all.High, 1e-9)

	half := rateOf(2, 4)
	assert.InDelta(t, 0.5, half.Value, 1e-9)
	assert.InDelta(t, 0.1500, half.Low, 1e-3)
	assert.InDelta(t, 0.8500, half.High, 1e-3)
}

func TestPassHatK_IsTheChanceEveryOneOfKTrialsPasses(t *testing.T) {
	t.Parallel()

	assert.Nil(t, passHatK(3, 3, 4), "three trials cannot estimate four")

	every := passHatK(4, 4, 4)
	require.NotNil(t, every)
	assert.InDelta(t, 1.0, *every, 1e-9)

	threeOfFour := passHatK(3, 4, 4)
	require.NotNil(t, threeOfFour)
	assert.InDelta(t, 0.0, *threeOfFour, 1e-9)

	sixOfEight := passHatK(6, 8, 4)
	require.NotNil(t, sixOfEight)
	assert.InDelta(t, (6.0*5*4*3)/(8.0*7*6*5), *sixOfEight, 1e-9)

	one := passHatK(2, 4, 1)
	require.NotNil(t, one)
	assert.InDelta(t, 0.5, *one, 1e-9)
}
