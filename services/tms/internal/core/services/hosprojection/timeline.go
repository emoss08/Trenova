package hosprojection

import (
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
)

// BuildTimeline converts persisted duty logs into a clean ascending interval list.
// Open entries are capped at now, overlapping entries are clipped to the next entry's
// start, and empty spans are dropped.
func BuildTimeline(logs []*telematics.WorkerHOSLog, now int64) []DutyInterval {
	if len(logs) == 0 {
		return nil
	}

	sorted := make([]*telematics.WorkerHOSLog, 0, len(logs))
	for _, entry := range logs {
		if entry == nil || entry.LogStartAt <= 0 || entry.LogStartAt >= now {
			continue
		}
		sorted = append(sorted, entry)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].LogStartAt < sorted[j].LogStartAt
	})

	timeline := make([]DutyInterval, 0, len(sorted))
	for i, entry := range sorted {
		end := now
		if entry.LogEndAt != nil && *entry.LogEndAt > 0 && *entry.LogEndAt < end {
			end = *entry.LogEndAt
		}
		if i+1 < len(sorted) && sorted[i+1].LogStartAt < end {
			end = sorted[i+1].LogStartAt
		}
		if end <= entry.LogStartAt {
			continue
		}
		timeline = append(timeline, DutyInterval{
			Status:  entry.DutyStatus,
			StartAt: entry.LogStartAt,
			EndAt:   end,
		})
	}
	return timeline
}

// isRestStatus reports whether the status counts toward an off-duty rest period.
// Personal conveyance is off-duty time under 49 CFR 395.8.
func isRestStatus(status telematics.DutyStatus) bool {
	switch status {
	case telematics.DutyStatusOffDuty,
		telematics.DutyStatusSleeperBed,
		telematics.DutyStatusPersonalConveyance:
		return true
	case telematics.DutyStatusDriving,
		telematics.DutyStatusOnDuty,
		telematics.DutyStatusYardMove:
		return false
	default:
		return false
	}
}

// isOnDutyStatus reports whether the status accrues on-duty (cycle and shift) time.
func isOnDutyStatus(status telematics.DutyStatus) bool {
	switch status {
	case telematics.DutyStatusOnDuty,
		telematics.DutyStatusYardMove,
		telematics.DutyStatusDriving:
		return true
	case telematics.DutyStatusOffDuty,
		telematics.DutyStatusSleeperBed,
		telematics.DutyStatusPersonalConveyance:
		return false
	default:
		return false
	}
}

// restRun is a maximal contiguous span of rest statuses, with its longest contiguous
// sleeper-berth stretch tracked separately for split-sleeper qualification.
type restRun struct {
	startAt      int64
	endAt        int64
	sleeperStart int64
	sleeperEnd   int64
	open         bool
}

// restRuns extracts every contiguous rest run from the timeline. The trailing run is
// marked open when it reaches the end of the timeline (the driver is still resting).
func restRuns(timeline []DutyInterval, now int64) []restRun {
	runs := make([]restRun, 0, len(timeline)/2+1)
	var current *restRun
	var sleeperStart, sleeperEnd int64

	flushSleeper := func() {
		if current == nil || sleeperEnd <= sleeperStart {
			return
		}
		if sleeperEnd-sleeperStart > current.sleeperEnd-current.sleeperStart {
			current.sleeperStart = sleeperStart
			current.sleeperEnd = sleeperEnd
		}
	}

	for i := range timeline {
		interval := timeline[i]
		if !isRestStatus(interval.Status) {
			if current != nil {
				flushSleeper()
				runs = append(runs, *current)
				current = nil
			}
			sleeperStart, sleeperEnd = 0, 0
			continue
		}

		if current != nil && interval.StartAt == current.endAt {
			current.endAt = interval.EndAt
		} else {
			if current != nil {
				flushSleeper()
				runs = append(runs, *current)
			}
			current = &restRun{startAt: interval.StartAt, endAt: interval.EndAt}
			sleeperStart, sleeperEnd = 0, 0
		}

		if interval.Status == telematics.DutyStatusSleeperBed {
			if sleeperEnd == interval.StartAt {
				sleeperEnd = interval.EndAt
			} else {
				flushSleeper()
				sleeperStart = interval.StartAt
				sleeperEnd = interval.EndAt
			}
		} else {
			flushSleeper()
			sleeperStart, sleeperEnd = 0, 0
		}
	}
	if current != nil {
		flushSleeper()
		current.open = current.endAt >= now
		runs = append(runs, *current)
	}
	return runs
}

// statusMsBetween sums milliseconds of intervals matching the predicate that overlap
// (from, to), both unix seconds.
func statusMsBetween(
	timeline []DutyInterval,
	from int64,
	to int64,
	match func(telematics.DutyStatus) bool,
) int64 {
	if to <= from {
		return 0
	}
	total := int64(0)
	for i := range timeline {
		interval := timeline[i]
		if interval.EndAt <= from {
			continue
		}
		if interval.StartAt >= to {
			break
		}
		if !match(interval.Status) {
			continue
		}
		start := max(interval.StartAt, from)
		end := min(interval.EndAt, to)
		if end > start {
			total += (end - start) * 1000
		}
	}
	return total
}

func drivingMsBetween(timeline []DutyInterval, from, to int64) int64 {
	return statusMsBetween(timeline, from, to, func(s telematics.DutyStatus) bool {
		return s == telematics.DutyStatusDriving
	})
}

func onDutyMsBetween(timeline []DutyInterval, from, to int64) int64 {
	return statusMsBetween(timeline, from, to, isOnDutyStatus)
}

// trailingRestStart returns the start of the open trailing rest run, or 0 when the
// driver is not currently resting according to the timeline.
func trailingRestStart(timeline []DutyInterval, now int64) int64 {
	runs := restRuns(timeline, now)
	if len(runs) == 0 {
		return 0
	}
	last := runs[len(runs)-1]
	if !last.open {
		return 0
	}
	return last.startAt
}

// lastQualifyingBreakEnd finds the end of the most recent rest span of at least 30
// minutes ending at or before the given time, which is when the 8-hour break clock
// last restarted.
func lastQualifyingBreakEnd(timeline []DutyInterval, before int64) int64 {
	runs := restRuns(timeline, before)
	for _, run := range slices.Backward(runs) {
		end := min(run.endAt, before)
		if end-run.startAt >= 30*60 {
			return end
		}
	}
	return 0
}
