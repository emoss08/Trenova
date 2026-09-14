package billingtransfer

type RunStatus string

const (
	RunStatusQueued    = RunStatus("Queued")
	RunStatusRunning   = RunStatus("Running")
	RunStatusCompleted = RunStatus("Completed")
	RunStatusCanceled  = RunStatus("Canceled")
	RunStatusFailed    = RunStatus("Failed")
)

func (v RunStatus) IsValid() bool {
	switch v {
	case RunStatusQueued, RunStatusRunning, RunStatusCompleted, RunStatusCanceled, RunStatusFailed:
		return true
	default:
		return false
	}
}

func (v RunStatus) IsTerminal() bool {
	return v == RunStatusCompleted || v == RunStatusCanceled || v == RunStatusFailed
}

type RunScope string

const (
	RunScopeSelected    = RunScope("Selected")
	RunScopeAllMatching = RunScope("AllMatching")
	RunScopeRetry       = RunScope("Retry")
)

func (v RunScope) IsValid() bool {
	switch v {
	case RunScopeSelected, RunScopeAllMatching, RunScopeRetry:
		return true
	default:
		return false
	}
}

type ItemStatus string

const (
	ItemStatusPending        = ItemStatus("Pending")
	ItemStatusTransferred    = ItemStatus("Transferred")
	ItemStatusNotTransferred = ItemStatus("NotTransferred")
	ItemStatusSkipped        = ItemStatus("Skipped")
)

func (v ItemStatus) IsValid() bool {
	switch v {
	case ItemStatusPending, ItemStatusTransferred, ItemStatusNotTransferred, ItemStatusSkipped:
		return true
	default:
		return false
	}
}

type FailureCode string

const (
	FailureNotFound           = FailureCode("NotFound")
	FailureInvalidStatus      = FailureCode("InvalidStatus")
	FailureAlreadyTransferred = FailureCode("AlreadyTransferred")
	FailureRequirementsUnmet  = FailureCode("RequirementsUnmet")
	FailureRateValidation     = FailureCode("RateValidation")
	FailureReturnToOperations = FailureCode("ReturnToOperations")
	FailureUnexpected         = FailureCode("Unexpected")
)

func (v FailureCode) IsValid() bool {
	switch v {
	case FailureNotFound,
		FailureInvalidStatus,
		FailureAlreadyTransferred,
		FailureRequirementsUnmet,
		FailureRateValidation,
		FailureReturnToOperations,
		FailureUnexpected:
		return true
	default:
		return false
	}
}

func (v FailureCode) IsRetryable() bool {
	return v.IsValid() && v != FailureNotFound && v != FailureAlreadyTransferred
}
