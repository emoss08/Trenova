package extractionrollout

type Arm string

const (
	ArmCandidate = Arm("Candidate")
	ArmControl   = Arm("Control")
)

func (a Arm) IsValid() bool {
	switch a {
	case ArmCandidate, ArmControl:
		return true
	default:
		return false
	}
}

func (a Arm) String() string { return string(a) }

func AllArms() []Arm {
	return []Arm{ArmCandidate, ArmControl}
}

type Outcome string

const (
	OutcomePending    = Outcome("Pending")
	OutcomeAccepted   = Outcome("Accepted")
	OutcomeRejected   = Outcome("Rejected")
	OutcomeFailed     = Outcome("Failed")
	OutcomeSuperseded = Outcome("Superseded")
)

func (o Outcome) IsValid() bool {
	switch o {
	case OutcomePending, OutcomeAccepted, OutcomeRejected, OutcomeFailed, OutcomeSuperseded:
		return true
	default:
		return false
	}
}

func (o Outcome) String() string { return string(o) }

func (o Outcome) IsSettled() bool { return o != OutcomePending }

func AllOutcomes() []Outcome {
	return []Outcome{
		OutcomePending,
		OutcomeAccepted,
		OutcomeRejected,
		OutcomeFailed,
		OutcomeSuperseded,
	}
}

type HaltReason string

const (
	HaltReasonAccuracyDrop = HaltReason("AccuracyDrop")
	HaltReasonRejections   = HaltReason("Rejections")
)

func (r HaltReason) IsValid() bool {
	switch r {
	case HaltReasonAccuracyDrop, HaltReasonRejections:
		return true
	default:
		return false
	}
}

func (r HaltReason) String() string { return string(r) }

func AllHaltReasons() []HaltReason {
	return []HaltReason{HaltReasonAccuracyDrop, HaltReasonRejections}
}

type ServedBy string

const (
	ServedByCandidate = ServedBy("Candidate")
	ServedByOther     = ServedBy("Other")
	ServedByNone      = ServedBy("None")
)
