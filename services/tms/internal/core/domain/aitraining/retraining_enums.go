package aitraining

type RetrainingStatus string

const (
	RetrainingStatusSkipped   = RetrainingStatus("Skipped")
	RetrainingStatusExporting = RetrainingStatus("Exporting")
	RetrainingStatusReady     = RetrainingStatus("Ready")
	RetrainingStatusTraining  = RetrainingStatus("Training")
	RetrainingStatusPassed    = RetrainingStatus("Passed")
	RetrainingStatusRejected  = RetrainingStatus("Rejected")
	RetrainingStatusFailed    = RetrainingStatus("Failed")
	RetrainingStatusCanceled  = RetrainingStatus("Canceled")
)

func (s RetrainingStatus) IsValid() bool {
	switch s {
	case RetrainingStatusSkipped,
		RetrainingStatusExporting,
		RetrainingStatusReady,
		RetrainingStatusTraining,
		RetrainingStatusPassed,
		RetrainingStatusRejected,
		RetrainingStatusFailed,
		RetrainingStatusCanceled:
		return true
	default:
		return false
	}
}

func (s RetrainingStatus) String() string { return string(s) }

func (s RetrainingStatus) IsOpen() bool {
	return s == RetrainingStatusExporting ||
		s == RetrainingStatusReady ||
		s == RetrainingStatusTraining
}

func (s RetrainingStatus) Trained() bool {
	return s == RetrainingStatusPassed || s == RetrainingStatusRejected
}

func AllRetrainingStatuses() []RetrainingStatus {
	return []RetrainingStatus{
		RetrainingStatusSkipped,
		RetrainingStatusExporting,
		RetrainingStatusReady,
		RetrainingStatusTraining,
		RetrainingStatusPassed,
		RetrainingStatusRejected,
		RetrainingStatusFailed,
		RetrainingStatusCanceled,
	}
}

func OpenRetrainingStatuses() []RetrainingStatus {
	return []RetrainingStatus{
		RetrainingStatusExporting,
		RetrainingStatusReady,
		RetrainingStatusTraining,
	}
}

func BaselineRetrainingStatuses() []RetrainingStatus {
	return []RetrainingStatus{
		RetrainingStatusExporting,
		RetrainingStatusReady,
		RetrainingStatusTraining,
		RetrainingStatusPassed,
		RetrainingStatusRejected,
	}
}

type RetrainingTrigger string

const (
	RetrainingTriggerScheduled = RetrainingTrigger("Scheduled")
	RetrainingTriggerDrift     = RetrainingTrigger("Drift")
	RetrainingTriggerManual    = RetrainingTrigger("Manual")
)

func (t RetrainingTrigger) IsValid() bool {
	switch t {
	case RetrainingTriggerScheduled, RetrainingTriggerDrift, RetrainingTriggerManual:
		return true
	default:
		return false
	}
}

func (t RetrainingTrigger) String() string { return string(t) }

func AllRetrainingTriggers() []RetrainingTrigger {
	return []RetrainingTrigger{
		RetrainingTriggerScheduled,
		RetrainingTriggerDrift,
		RetrainingTriggerManual,
	}
}

type RetrainingSkipReason string

const (
	RetrainingSkipCycleOpen         = RetrainingSkipReason("CycleOpen")
	RetrainingSkipExportActive      = RetrainingSkipReason("ExportActive")
	RetrainingSkipTooSoon           = RetrainingSkipReason("TooSoon")
	RetrainingSkipNotEnoughExamples = RetrainingSkipReason("NotEnoughExamples")
)

func (r RetrainingSkipReason) IsValid() bool {
	switch r {
	case RetrainingSkipCycleOpen,
		RetrainingSkipExportActive,
		RetrainingSkipTooSoon,
		RetrainingSkipNotEnoughExamples:
		return true
	default:
		return false
	}
}

func (r RetrainingSkipReason) String() string { return string(r) }

func (r RetrainingSkipReason) Message() string {
	switch r {
	case RetrainingSkipCycleOpen:
		return "A retraining cycle is already exporting, waiting for a trainer, or training"
	case RetrainingSkipExportActive:
		return "Another training export is running"
	case RetrainingSkipTooSoon:
		return "The last retraining started too recently"
	case RetrainingSkipNotEnoughExamples:
		return "Too few new corrections have been confirmed since the last retraining"
	default:
		return string(r)
	}
}

func AllRetrainingSkipReasons() []RetrainingSkipReason {
	return []RetrainingSkipReason{
		RetrainingSkipCycleOpen,
		RetrainingSkipExportActive,
		RetrainingSkipTooSoon,
		RetrainingSkipNotEnoughExamples,
	}
}
