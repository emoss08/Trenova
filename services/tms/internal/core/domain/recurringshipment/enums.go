package recurringshipment

type Status string

const (
	StatusActive  = Status("Active")
	StatusPaused  = Status("Paused")
	StatusExpired = Status("Expired")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusPaused, StatusExpired:
		return true
	default:
		return false
	}
}

type ExceptionPolicy string

const (
	ExceptionPolicySkip                = ExceptionPolicy("Skip")
	ExceptionPolicyPreviousBusinessDay = ExceptionPolicy("PreviousBusinessDay")
	ExceptionPolicyNextBusinessDay     = ExceptionPolicy("NextBusinessDay")
)

func (p ExceptionPolicy) IsValid() bool {
	switch p {
	case ExceptionPolicySkip, ExceptionPolicyPreviousBusinessDay, ExceptionPolicyNextBusinessDay:
		return true
	default:
		return false
	}
}

type RunStatus string

const (
	RunStatusGenerated = RunStatus("Generated")
	RunStatusSkipped   = RunStatus("Skipped")
	RunStatusFailed    = RunStatus("Failed")
)

type RunTrigger string

const (
	RunTriggerAuto   = RunTrigger("Auto")
	RunTriggerManual = RunTrigger("Manual")
)
