package workercompliance

import (
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/shared/typeutils"
)

// ProfileRollup is the denormalised answer the roster reads. Every value in it
// is derivable from the records the HR areas own; it exists because deriving
// it needs a worker's whole history, which is affordable once on a panel and
// not fifty times on a list page.
//
// Treat it as a cache. worker_credentials, worker_training_records and
// worker_safety_events remain the ledgers of record, and RollupFrom can
// rebuild this from them at any time.
type ProfileRollup struct {
	ComplianceStatus     worker.ComplianceStatus
	TrainingHealth       worker.TrainingHealth
	SafetyRating         worker.SafetyRating
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
		typeutils.EqualPtr(r.NextCredentialExpiry, other.NextCredentialExpiry) &&
		typeutils.EqualPtr(r.NextTrainingDue, other.NextTrainingDue)
}

// RollupFrom reduces the three summaries to the row the roster reads. Any of
// them may be nil — a tenant with no courses configured has no training
// summary to speak of — and a nil section leaves its part at the value that
// means "nothing to report" rather than inventing a problem.
func RollupFrom(
	credentials *workercredential.WorkerCredentialSummary,
	training *workertraining.WorkerTrainingSummary,
	safety *workersafety.SafetyScorecard,
) ProfileRollup {
	rollup := ProfileRollup{
		ComplianceStatus: worker.ComplianceStatusPending,
		TrainingHealth:   worker.TrainingHealthCurrent,
		SafetyRating:     worker.SafetyRatingExcellent,
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
