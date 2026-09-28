package hashutils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPercentBucketIsStableAndInRange(t *testing.T) {
	t.Parallel()

	for i := range 1000 {
		key := "doc_" + strconv.Itoa(i)
		bucket := PercentBucket(key)
		assert.GreaterOrEqual(t, bucket, 0)
		assert.Less(t, bucket, 100)
		assert.Equal(t, bucket, PercentBucket(key))
	}
}

func TestInPercentSampleHonoursTheBounds(t *testing.T) {
	t.Parallel()

	assert.False(t, InPercentSample("anything", 0))
	assert.False(t, InPercentSample("anything", -5))
	assert.True(t, InPercentSample("anything", 100))
	assert.True(t, InPercentSample("anything", 250))
}

func TestInPercentSampleTakesRoughlyThePercentAsked(t *testing.T) {
	t.Parallel()

	const keys = 20000
	for _, percent := range []int{1, 10, 50} {
		sampled := 0
		for i := range keys {
			if InPercentSample("doc_"+strconv.Itoa(i), percent) {
				sampled++
			}
		}
		share := float64(sampled) / keys * 100
		assert.InDelta(t, float64(percent), share, 1.5, "percent %d", percent)
	}
}

func TestInPercentSampleNestsSmallerSamplesInLargerOnes(t *testing.T) {
	t.Parallel()

	for i := range 2000 {
		key := "doc_" + strconv.Itoa(i)
		if InPercentSample(key, 10) {
			assert.True(t, InPercentSample(key, 25), key)
		}
	}
}
