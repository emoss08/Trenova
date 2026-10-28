package worker

// DrugAlcoholStatus is the worker's testing record in one word, denormalised
// onto the profile so the roster and the dispatch check can read it without
// replaying every test. Unknown means nothing is on file, which is not the same
// as clear: a driver nobody has ever tested has not passed anything.
type DrugAlcoholStatus string

const (
	DrugAlcoholUnknown    = DrugAlcoholStatus("Unknown")
	DrugAlcoholClear      = DrugAlcoholStatus("Clear")
	DrugAlcoholPending    = DrugAlcoholStatus("Pending")
	DrugAlcoholProhibited = DrugAlcoholStatus("Prohibited")
)

func (s DrugAlcoholStatus) String() string { return string(s) }

func (s DrugAlcoholStatus) IsValid() bool {
	switch s {
	case DrugAlcoholUnknown, DrugAlcoholClear, DrugAlcoholPending, DrugAlcoholProhibited:
		return true
	default:
		return false
	}
}

// Normalized treats an unset value as Unknown, so a profile assembled in memory
// reads the same as one loaded from the database, where the column defaults to
// Unknown and can never be blank.
func (s DrugAlcoholStatus) Normalized() DrugAlcoholStatus {
	if s == "" {
		return DrugAlcoholUnknown
	}
	return s
}

// Blocks reports whether the status bars a driver from safety-sensitive duty.
// Only a standing prohibition does. Pending and Unknown are warnings: the
// office has work to do, but nothing has been found against the driver, and
// grounding a fleet over missing paperwork is not what the rule asks for.
func (s DrugAlcoholStatus) Blocks() bool { return s == DrugAlcoholProhibited }

func (s DrugAlcoholStatus) Label() string {
	switch s {
	case DrugAlcoholUnknown:
		return "Not on file"
	case DrugAlcoholClear:
		return "Clear"
	case DrugAlcoholPending:
		return "Awaiting result"
	case DrugAlcoholProhibited:
		return "Prohibited"
	default:
		return string(s)
	}
}

// ReturnToDutyStatus is the worker-level projection of the process, kept on the
// profile so the roster can be filtered without reading every violation.
type ReturnToDutyStatus string

const (
	ReturnToDutyNotRequired     = ReturnToDutyStatus("NotRequired")
	ReturnToDutySAPEvaluation   = ReturnToDutyStatus("SAPEvaluation")
	ReturnToDutyRTDTestRequired = ReturnToDutyStatus("RTDTestRequired")
	ReturnToDutyFollowUpTesting = ReturnToDutyStatus("FollowUpTesting")
	ReturnToDutyComplete        = ReturnToDutyStatus("Complete")
)

func (s ReturnToDutyStatus) String() string { return string(s) }

func (s ReturnToDutyStatus) IsValid() bool {
	switch s {
	case ReturnToDutyNotRequired, ReturnToDutySAPEvaluation, ReturnToDutyRTDTestRequired,
		ReturnToDutyFollowUpTesting, ReturnToDutyComplete:
		return true
	default:
		return false
	}
}

// SafetyRating buckets a scorecard for chips and gauges.
type SafetyRating string

const (
	SafetyRatingExcellent = SafetyRating("Excellent")
	SafetyRatingGood      = SafetyRating("Good")
	SafetyRatingWatch     = SafetyRating("Watch")
	SafetyRatingAtRisk    = SafetyRating("AtRisk")
)

func (r SafetyRating) String() string { return string(r) }

func (r SafetyRating) IsValid() bool {
	switch r {
	case SafetyRatingExcellent, SafetyRatingGood, SafetyRatingWatch, SafetyRatingAtRisk:
		return true
	default:
		return false
	}
}

// TrainingHealth is the evaluated state of one course slot for a worker.
type TrainingHealth string

const (
	TrainingHealthCurrent      = TrainingHealth("Current")
	TrainingHealthScheduled    = TrainingHealth("Scheduled")
	TrainingHealthDueSoon      = TrainingHealth("DueSoon")
	TrainingHealthOverdue      = TrainingHealth("Overdue")
	TrainingHealthExpiringSoon = TrainingHealth("ExpiringSoon")
	TrainingHealthExpired      = TrainingHealth("Expired")
	TrainingHealthFailed       = TrainingHealth("Failed")
	TrainingHealthMissing      = TrainingHealth("Missing")
)

func (h TrainingHealth) String() string { return string(h) }

func (h TrainingHealth) IsValid() bool {
	switch h {
	case TrainingHealthCurrent, TrainingHealthScheduled, TrainingHealthDueSoon,
		TrainingHealthOverdue, TrainingHealthExpiringSoon, TrainingHealthExpired,
		TrainingHealthFailed, TrainingHealthMissing:
		return true
	default:
		return false
	}
}

// Blocks reports whether this health leaves a required course unsatisfied.
func (h TrainingHealth) Blocks() bool {
	switch h {
	case TrainingHealthOverdue, TrainingHealthExpired, TrainingHealthFailed, TrainingHealthMissing:
		return true
	default:
		return false
	}
}

// Satisfied reports whether the course counts as done for the matrix.
func (h TrainingHealth) Satisfied() bool {
	return h == TrainingHealthCurrent || h == TrainingHealthExpiringSoon
}
