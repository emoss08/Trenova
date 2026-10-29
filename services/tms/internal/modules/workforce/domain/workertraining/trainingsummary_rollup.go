package workertraining

import (
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
)

// WorstHealth is the worker's training in one word: the worst state any
// required course is in. A worker with nine current courses and one expired
// certification is not "current", and the roster has one column to say so.
func (s *WorkerTrainingSummary) WorstHealth() worker.TrainingHealth {
	if s == nil {
		return worker.TrainingHealthCurrent
	}
	worst := worker.TrainingHealthCurrent
	for _, item := range s.Items {
		if item == nil || !item.Required {
			continue
		}
		if trainingHealthRank(item.Health) < trainingHealthRank(worst) {
			worst = item.Health
		}
	}
	return worst
}

// NextDue is the soonest a required course needs attention, whichever comes
// first: the due date on something open, or the expiry on something current.
func (s *WorkerTrainingSummary) NextDue() *int64 {
	if s == nil {
		return nil
	}
	var earliest *int64
	for _, item := range s.Items {
		if item == nil || !item.Required || item.Record == nil {
			continue
		}
		for _, candidate := range []*int64{item.Record.DueAt, item.Record.ExpiresAt} {
			if candidate == nil || *candidate <= 0 {
				continue
			}
			if earliest == nil || *candidate < *earliest {
				earliest = candidate
			}
		}
	}
	if earliest == nil {
		return nil
	}
	value := *earliest
	return &value
}
