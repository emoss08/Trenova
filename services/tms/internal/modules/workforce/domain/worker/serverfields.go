package worker

const DefaultSafetyScore int16 = 100

func (wp *WorkerProfile) KeepServerFields(stored *WorkerProfile) {
	if wp == nil {
		return
	}
	if stored == nil {
		wp.ResetServerFields()
		return
	}
	wp.ComplianceStatus = stored.ComplianceStatus
	wp.NextCredentialExpiry = stored.NextCredentialExpiry
	wp.TrainingHealth = stored.TrainingHealth
	wp.NextTrainingDue = stored.NextTrainingDue
	wp.SafetyRating = stored.SafetyRating
	wp.SafetyScore = stored.SafetyScore
	wp.DrugAlcoholStatus = stored.DrugAlcoholStatus
	wp.ReturnToDutyStatus = stored.ReturnToDutyStatus
	wp.LastClearinghouseQueryAt = stored.LastClearinghouseQueryAt
	wp.NextClearinghouseQueryDue = stored.NextClearinghouseQueryDue
}

func (wp *WorkerProfile) ResetServerFields() {
	if wp == nil {
		return
	}
	wp.ComplianceStatus = ComplianceStatusPending
	wp.NextCredentialExpiry = nil
	wp.TrainingHealth = TrainingHealthCurrent
	wp.NextTrainingDue = nil
	wp.SafetyRating = SafetyRatingExcellent
	wp.SafetyScore = DefaultSafetyScore
	wp.DrugAlcoholStatus = DrugAlcoholUnknown
	wp.ReturnToDutyStatus = ReturnToDutyNotRequired
	wp.LastClearinghouseQueryAt = nil
	wp.NextClearinghouseQueryDue = nil
}
