package aianalytics_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketOfPlacesLatenciesOnTheLogScale(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, aianalytics.BucketOf(0))
	assert.Equal(t, 0, aianalytics.BucketOf(15))
	assert.Equal(t, 1, aianalytics.BucketOf(16))
	assert.Equal(t, 5, aianalytics.BucketOf(32))
	assert.Equal(t, 9, aianalytics.BucketOf(64))
	assert.Equal(t, aianalytics.HistogramBuckets-1, aianalytics.BucketOf(10_000_000_000))
}

func TestHistogramPercentileStaysWithinABucketOfTheTrueValue(t *testing.T) {
	t.Parallel()

	var histogram aianalytics.Histogram
	for ms := int64(1); ms <= 10_000; ms++ {
		histogram.Observe(ms)
	}

	median, ok := histogram.Percentile(0.5)
	require.True(t, ok)
	assert.InDelta(t, 5_000, median, 5_000*0.19)

	p95, ok := histogram.Percentile(0.95)
	require.True(t, ok)
	assert.InDelta(t, 9_500, p95, 9_500*0.19)
}

func TestHistogramMergesStoredBucketsAndIgnoresOverflowAndNegatives(t *testing.T) {
	t.Parallel()

	var histogram aianalytics.Histogram
	counts := make([]int32, aianalytics.HistogramBuckets+3)
	counts[aianalytics.BucketOf(700)] = 4
	counts[3] = -2
	counts[aianalytics.HistogramBuckets+1] = 9
	histogram.Add(counts)
	histogram.Add(counts)

	assert.Equal(t, int64(8), histogram.Total())
	median, ok := histogram.Percentile(0.5)
	require.True(t, ok)
	assert.InDelta(t, 700, median, 700*0.19)
}

func TestHistogramWithNothingObservedHasNoPercentile(t *testing.T) {
	t.Parallel()

	var histogram aianalytics.Histogram
	_, ok := histogram.Percentile(0.5)
	assert.False(t, ok)
}

func TestWindowBucketsMatchTheTrendPoints(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		int64(24),
		aianalytics.WindowDay.Seconds()/aianalytics.WindowDay.BucketSeconds(),
	)
	assert.Equal(
		t,
		int64(28),
		aianalytics.WindowWeek.Seconds()/aianalytics.WindowWeek.BucketSeconds(),
	)
	assert.Equal(
		t,
		int64(30),
		aianalytics.WindowMonth.Seconds()/aianalytics.WindowMonth.BucketSeconds(),
	)
}

func TestHistogramBucketsRoundTripThroughAdd(t *testing.T) {
	t.Parallel()

	var empty aianalytics.Histogram
	assert.Empty(t, empty.Buckets(), "an answered call with no first token stores no histogram")

	var first aianalytics.Histogram
	first.Observe(310)
	first.Observe(820)
	first.Observe(820)
	buckets := first.Buckets()
	assert.Len(t, buckets, aianalytics.HistogramBuckets)

	var merged aianalytics.Histogram
	merged.Add(buckets)
	assert.Equal(t, int64(3), merged.Total())
	firstMedian, _ := first.Percentile(0.5)
	mergedMedian, _ := merged.Percentile(0.5)
	assert.Equal(t, firstMedian, mergedMedian)
}
