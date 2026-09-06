package worker

import (
	"sort"

	"github.com/emoss08/trenova/shared/pulid"
)

// TrainingSummaryItem is one course slot in a worker's training picture: every
// required course appears (possibly Missing), and every other course the
// worker has a record for.
type TrainingSummaryItem struct {
	Course          *TrainingCourse
	Record          *WorkerTrainingRecord
	Health          TrainingHealth
	DaysUntilDue    *int64
	DaysUntilExpiry *int64
	Required        bool
}

type WorkerTrainingSummary struct {
	WorkerID      pulid.ID
	Compliant     bool
	RequiredCount int
	CurrentCount  int
	DueCount      int
	OverdueCount  int
	ExpiringCount int
	ExpiredCount  int
	MissingCount  int
	Items         []*TrainingSummaryItem
}

// CurrentTrainingRecord picks the record that speaks for a course: an open
// assignment wins; otherwise the strongest closed record — a completion or
// waiver over a lapsed one over a failure — with the most recent winning a
// tie, so a failed retake never hides a certification that is still good.
// Cancelled records never count.
func CurrentTrainingRecord(records []*WorkerTrainingRecord) *WorkerTrainingRecord {
	var best *WorkerTrainingRecord
	for _, record := range records {
		if record == nil || record.Status == TrainingStatusCancelled {
			continue
		}
		if record.IsOpen() {
			return record
		}
		if best == nil || closedRank(record) < closedRank(best) ||
			closedRank(record) == closedRank(best) && recordMoment(record) > recordMoment(best) {
			best = record
		}
	}
	return best
}

func closedRank(record *WorkerTrainingRecord) int {
	switch record.Status {
	case TrainingStatusCompleted:
		return 0
	case TrainingStatusWaived:
		return 1
	case TrainingStatusExpired:
		return 2
	case TrainingStatusFailed:
		return 3
	default:
		return 4
	}
}

func recordMoment(record *WorkerTrainingRecord) int64 {
	if record.CompletedAt != nil && *record.CompletedAt > 0 {
		return *record.CompletedAt
	}
	return record.UpdatedAt
}

// BuildTrainingSummary is the pure roll-up. A worker is compliant when no
// required course is Overdue, Expired, Failed or Missing; a course that is
// merely scheduled or due soon does not block.
func BuildTrainingSummary(
	wrk *Worker,
	courses []*TrainingCourse,
	records []*WorkerTrainingRecord,
	now int64,
) *WorkerTrainingSummary {
	summary := &WorkerTrainingSummary{
		Compliant: true,
		Items:     make([]*TrainingSummaryItem, 0, len(courses)),
	}
	if wrk != nil {
		summary.WorkerID = wrk.ID
	}

	byCourse := make(map[pulid.ID][]*WorkerTrainingRecord, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		byCourse[record.CourseID] = append(byCourse[record.CourseID], record)
	}

	seen := make(map[pulid.ID]struct{}, len(courses))
	for _, course := range courses {
		if course == nil {
			continue
		}
		seen[course.ID] = struct{}{}
		required := course.AppliesTo(wrk)
		record := CurrentTrainingRecord(byCourse[course.ID])
		if !required && record == nil {
			continue
		}
		summary.add(itemFor(course, record, required, now))
	}

	for courseID, list := range byCourse {
		if _, ok := seen[courseID]; ok {
			continue
		}
		record := CurrentTrainingRecord(list)
		if record == nil || record.Course == nil {
			continue
		}
		summary.add(itemFor(record.Course, record, false, now))
	}

	sort.SliceStable(summary.Items, func(i, j int) bool {
		a, b := summary.Items[i], summary.Items[j]
		if a.Required != b.Required {
			return a.Required
		}
		if a.Course.SortOrder != b.Course.SortOrder {
			return a.Course.SortOrder < b.Course.SortOrder
		}
		return a.Course.Name < b.Course.Name
	})

	return summary
}

func itemFor(
	course *TrainingCourse,
	record *WorkerTrainingRecord,
	required bool,
	now int64,
) *TrainingSummaryItem {
	item := &TrainingSummaryItem{
		Course:   course,
		Record:   record,
		Required: required,
		Health:   TrainingHealthMissing,
	}
	if record != nil {
		record.Course = course
		item.Health = EvaluateTrainingHealth(record, course.RenewalWindowDays, now)
		item.DaysUntilDue = record.DaysUntilDue(now)
		item.DaysUntilExpiry = record.DaysUntilExpiry(now)
	}
	return item
}

func (s *WorkerTrainingSummary) add(item *TrainingSummaryItem) {
	s.Items = append(s.Items, item)
	if item.Required {
		s.RequiredCount++
	}
	switch item.Health {
	case TrainingHealthCurrent:
		s.CurrentCount++
	case TrainingHealthScheduled, TrainingHealthDueSoon:
		s.DueCount++
	case TrainingHealthOverdue:
		s.OverdueCount++
	case TrainingHealthExpiringSoon:
		s.CurrentCount++
		s.ExpiringCount++
	case TrainingHealthExpired:
		s.ExpiredCount++
	case TrainingHealthFailed, TrainingHealthMissing:
		s.MissingCount++
	}
	if item.Required && item.Health.Blocks() {
		s.Compliant = false
	}
}

// Attention returns the slots that need action, worst first.
func (s *WorkerTrainingSummary) Attention() []*TrainingSummaryItem {
	out := make([]*TrainingSummaryItem, 0, len(s.Items))
	for _, item := range s.Items {
		if item.Health != TrainingHealthCurrent {
			out = append(out, item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return trainingHealthRank(out[i].Health) < trainingHealthRank(out[j].Health)
	})
	return out
}

func trainingHealthRank(h TrainingHealth) int {
	switch h {
	case TrainingHealthMissing:
		return 0
	case TrainingHealthFailed:
		return 1
	case TrainingHealthExpired:
		return 2
	case TrainingHealthOverdue:
		return 3
	case TrainingHealthDueSoon:
		return 4
	case TrainingHealthExpiringSoon:
		return 5
	case TrainingHealthScheduled:
		return 6
	case TrainingHealthCurrent:
		return 7
	default:
		return 8
	}
}

// RequiredGaps lists the required courses with no usable record — what an
// automatic assignment should open.
func (s *WorkerTrainingSummary) RequiredGaps() []*TrainingCourse {
	out := make([]*TrainingCourse, 0, len(s.Items))
	for _, item := range s.Items {
		if !item.Required || item.Record != nil && item.Record.IsOpen() {
			continue
		}
		if item.Health.Blocks() {
			out = append(out, item.Course)
		}
	}
	return out
}
