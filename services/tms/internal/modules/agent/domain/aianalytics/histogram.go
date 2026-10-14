package aianalytics

import "math"

const (
	HistogramBuckets       = 64
	HistogramBaseMs        = 16
	HistogramStepsPerOctet = 4
)

func BucketOf(ms int64) int {
	if ms < HistogramBaseMs {
		return 0
	}
	index := int(math.Floor(HistogramStepsPerOctet*math.Log2(float64(ms)/HistogramBaseMs))) + 1

	return min(index, HistogramBuckets-1)
}

func bucketBounds(index int) (float64, float64) {
	if index <= 0 {
		return 0, HistogramBaseMs
	}
	lower := HistogramBaseMs * math.Pow(2, float64(index-1)/HistogramStepsPerOctet)
	upper := HistogramBaseMs * math.Pow(2, float64(index)/HistogramStepsPerOctet)

	return lower, upper
}

type Histogram struct {
	counts [HistogramBuckets]int64
	total  int64
}

func (h *Histogram) Add(counts []int32) {
	for index, count := range counts {
		if index >= HistogramBuckets || count <= 0 {
			continue
		}
		h.counts[index] += int64(count)
		h.total += int64(count)
	}
}

func (h *Histogram) AddBucket(index int, count int64) {
	if index < 0 || index >= HistogramBuckets || count <= 0 {
		return
	}
	h.counts[index] += count
	h.total += count
}

func (h *Histogram) Observe(ms int64) {
	h.counts[BucketOf(ms)]++
	h.total++
}

func (h *Histogram) Total() int64 {
	return h.total
}

func (h *Histogram) Percentile(p float64) (int64, bool) {
	if h.total == 0 {
		return 0, false
	}
	p = min(max(p, 0), 1)
	rank := p * float64(h.total)
	var seen int64
	for index, count := range h.counts {
		if count == 0 {
			continue
		}
		if float64(seen+count) >= rank {
			lower, upper := bucketBounds(index)
			within := (rank - float64(seen)) / float64(count)
			if index == 0 {
				return int64(math.Round(upper * within)), true
			}

			return int64(math.Round(lower * math.Pow(upper/lower, within))), true
		}
		seen += count
	}
	_, upper := bucketBounds(HistogramBuckets - 1)

	return int64(math.Round(upper)), true
}

func (h *Histogram) Buckets() []int32 {
	if h.total == 0 {
		return []int32{}
	}
	out := make([]int32, HistogramBuckets)
	for index, count := range h.counts {
		out[index] = int32(min(count, math.MaxInt32))
	}

	return out
}
