package aianalytics

import (
	"slices"
	"time"
)

const (
	secondsPerHour = 60 * 60
	hoursPerDay    = 24
)

func BucketStarts(window Window, location *time.Location, until int64) []int64 {
	if location == nil {
		location = time.UTC
	}
	size := window.BucketSeconds()
	count := window.Buckets()
	starts := make([]int64, 0, count)
	at := alignBucket(location, until, size)
	for len(starts) < count {
		starts = append(starts, at)
		at = alignBucket(location, at-1, size)
	}
	slices.Reverse(starts)

	return starts
}

func alignBucket(location *time.Location, at, size int64) int64 {
	local := time.Unix(at, 0).In(location)
	if size%secondsPerHour == 0 && size > secondsPerHour && hoursPerDay%(size/secondsPerHour) == 0 {
		step := int(size / secondsPerHour)
		start := time.Date(
			local.Year(), local.Month(), local.Day(),
			local.Hour()-local.Hour()%step, 0, 0, 0,
			location,
		).Unix()
		if start <= at {
			return start
		}
	}

	_, offset := local.Zone()
	rest := (at + int64(offset)) % size
	if rest < 0 {
		rest += size
	}

	return at - rest
}
