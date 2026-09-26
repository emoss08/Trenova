package extractioneval

type CaseStatus string

const (
	CaseStatusCandidate = CaseStatus("Candidate")
	CaseStatusActive    = CaseStatus("Active")
	CaseStatusRetired   = CaseStatus("Retired")
)

func (s CaseStatus) IsValid() bool {
	switch s {
	case CaseStatusCandidate, CaseStatusActive, CaseStatusRetired:
		return true
	default:
		return false
	}
}

func (s CaseStatus) String() string { return string(s) }

func (s CaseStatus) CanMoveTo(next CaseStatus) bool {
	switch s {
	case CaseStatusCandidate:
		return next == CaseStatusActive || next == CaseStatusRetired
	case CaseStatusActive:
		return next == CaseStatusRetired
	case CaseStatusRetired:
		return next == CaseStatusActive
	default:
		return false
	}
}

func AllCaseStatuses() []CaseStatus {
	return []CaseStatus{CaseStatusCandidate, CaseStatusActive, CaseStatusRetired}
}

type RunStatus string

const (
	RunStatusQueued        = RunStatus("Queued")
	RunStatusRunning       = RunStatus("Running")
	RunStatusCompleted     = RunStatus("Completed")
	RunStatusBudgetStopped = RunStatus("BudgetStopped")
	RunStatusCanceled      = RunStatus("Canceled")
	RunStatusFailed        = RunStatus("Failed")
)

func (s RunStatus) IsValid() bool {
	switch s {
	case RunStatusQueued,
		RunStatusRunning,
		RunStatusCompleted,
		RunStatusBudgetStopped,
		RunStatusCanceled,
		RunStatusFailed:
		return true
	default:
		return false
	}
}

func (s RunStatus) String() string { return string(s) }

func (s RunStatus) IsActive() bool {
	return s == RunStatusQueued || s == RunStatusRunning
}

func AllRunStatuses() []RunStatus {
	return []RunStatus{
		RunStatusQueued,
		RunStatusRunning,
		RunStatusCompleted,
		RunStatusBudgetStopped,
		RunStatusCanceled,
		RunStatusFailed,
	}
}

type ResultStatus string

const (
	ResultStatusPending   = ResultStatus("Pending")
	ResultStatusCompleted = ResultStatus("Completed")
	ResultStatusFailed    = ResultStatus("Failed")
	ResultStatusSkipped   = ResultStatus("Skipped")
)

func (s ResultStatus) IsValid() bool {
	switch s {
	case ResultStatusPending, ResultStatusCompleted, ResultStatusFailed, ResultStatusSkipped:
		return true
	default:
		return false
	}
}

func (s ResultStatus) String() string { return string(s) }

func AllResultStatuses() []ResultStatus {
	return []ResultStatus{
		ResultStatusPending,
		ResultStatusCompleted,
		ResultStatusFailed,
		ResultStatusSkipped,
	}
}
