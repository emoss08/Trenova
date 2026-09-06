package worker

// ProfileRollup is the denormalised answer the roster reads. Every value in it
// is derivable from the records the HR areas own; it exists because deriving
// it needs a worker's whole history, which is affordable once on a panel and
// not fifty times on a list page.
//
// Treat it as a cache. worker_credentials, worker_training_records and
// worker_safety_events remain the ledgers of record, and RollupFrom can
// rebuild this from them at any time.
type ProfileRollup struct {
	ComplianceStatus     ComplianceStatus
	TrainingHealth       TrainingHealth
	SafetyRating         SafetyRating
	SafetyScore          int16
	NextCredentialExpiry *int64
	NextTrainingDue      *int64
}

// Equal reports whether two roll-ups say the same thing, so a refresh that
// changes nothing can skip the write and leave updated_at alone.
func (r ProfileRollup) Equal(other ProfileRollup) bool {
	return r.ComplianceStatus == other.ComplianceStatus &&
		r.TrainingHealth == other.TrainingHealth &&
		r.SafetyRating == other.SafetyRating &&
		r.SafetyScore == other.SafetyScore &&
		equalOptionalUnix(r.NextCredentialExpiry, other.NextCredentialExpiry) &&
		equalOptionalUnix(r.NextTrainingDue, other.NextTrainingDue)
}

func equalOptionalUnix(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// RollupFrom reduces the three summaries to the row the roster reads. Any of
// them may be nil — a tenant with no courses configured has no training
// summary to speak of — and a nil section leaves its part at the value that
// means "nothing to report" rather than inventing a problem.
func RollupFrom(
	credentials *WorkerCredentialSummary,
	training *WorkerTrainingSummary,
	safety *SafetyScorecard,
) ProfileRollup {
	rollup := ProfileRollup{
		ComplianceStatus: ComplianceStatusPending,
		TrainingHealth:   TrainingHealthCurrent,
		SafetyRating:     SafetyRatingExcellent,
		SafetyScore:      100,
	}
	if credentials != nil {
		rollup.ComplianceStatus = credentials.ComplianceStatus
		rollup.NextCredentialExpiry = credentials.NextExpiry()
	}
	if training != nil {
		rollup.TrainingHealth = training.WorstHealth()
		rollup.NextTrainingDue = training.NextDue()
	}
	if safety != nil {
		rollup.SafetyRating = safety.Rating
		rollup.SafetyScore = int16(safety.Score)
	}
	return rollup
}

// NextExpiry is the earliest expiry among the worker's required credentials —
// what the roster sorts by and the expiry forecast counts down to. Optional
// credentials are excluded: nobody should be chased over a certificate the
// role does not need.
func (s *WorkerCredentialSummary) NextExpiry() *int64 {
	if s == nil {
		return nil
	}
	var earliest *int64
	for _, item := range s.Items {
		if item == nil || !item.Required || item.Credential == nil {
			continue
		}
		expiry := item.Credential.ExpiresAt
		if expiry == nil || *expiry <= 0 {
			continue
		}
		if earliest == nil || *expiry < *earliest {
			earliest = expiry
		}
	}
	if earliest == nil {
		return nil
	}
	value := *earliest
	return &value
}

// WorstHealth is the worker's training in one word: the worst state any
// required course is in. A worker with nine current courses and one expired
// certification is not "current", and the roster has one column to say so.
func (s *WorkerTrainingSummary) WorstHealth() TrainingHealth {
	if s == nil {
		return TrainingHealthCurrent
	}
	worst := TrainingHealthCurrent
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
