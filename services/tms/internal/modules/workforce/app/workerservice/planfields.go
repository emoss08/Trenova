package workerservice

import "github.com/emoss08/trenova/internal/modules/workforce/domain/worker"

// workerDefaults is what a new worker holds where the plan withholds a field:
// the profile's starting compliance, safety and training standing, and zero
// values elsewhere.
func workerDefaults(entity *worker.Worker) *worker.Worker {
	if entity == nil || entity.Profile == nil {
		return nil
	}

	return &worker.Worker{
		Profile: &worker.WorkerProfile{
			ComplianceStatus:   worker.ComplianceStatusPending,
			TrainingHealth:     worker.TrainingHealthCurrent,
			SafetyRating:       worker.SafetyRatingExcellent,
			SafetyScore:        100,
			DrugAlcoholStatus:  worker.DrugAlcoholUnknown,
			ReturnToDutyStatus: worker.ReturnToDutyNotRequired,
		},
	}
}
