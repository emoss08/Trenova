package extractionshadow

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

func (s ResultStatus) IsSettled() bool { return s != ResultStatusPending }

func AllResultStatuses() []ResultStatus {
	return []ResultStatus{
		ResultStatusPending,
		ResultStatusCompleted,
		ResultStatusFailed,
		ResultStatusSkipped,
	}
}

type Verdict string

const (
	VerdictBetter = Verdict("Better")
	VerdictWorse  = Verdict("Worse")
	VerdictSame   = Verdict("Same")
)

func (v Verdict) IsValid() bool {
	switch v {
	case VerdictBetter, VerdictWorse, VerdictSame:
		return true
	default:
		return false
	}
}

func (v Verdict) String() string { return string(v) }

func AllVerdicts() []Verdict {
	return []Verdict{VerdictBetter, VerdictWorse, VerdictSame}
}
