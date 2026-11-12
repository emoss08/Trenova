package recordcase

type Tone string

const (
	ToneNeutral = Tone("Neutral")
	ToneGood    = Tone("Good")
	ToneWarning = Tone("Warning")
	ToneBad     = Tone("Bad")
	ToneInfo    = Tone("Info")
)

func (t Tone) IsValid() bool {
	switch t {
	case ToneNeutral, ToneGood, ToneWarning, ToneBad, ToneInfo:
		return true
	default:
		return false
	}
}

type OptionKind string

const (
	OptionKindPlan  = OptionKind("Plan")
	OptionKindLeave = OptionKind("Leave")
)

func (k OptionKind) IsValid() bool {
	return k == OptionKindPlan || k == OptionKindLeave
}

type ReadingStatus string

const (
	ReadingStatusWriting     = ReadingStatus("Writing")
	ReadingStatusReady       = ReadingStatus("Ready")
	ReadingStatusUnavailable = ReadingStatus("Unavailable")
)

func (s ReadingStatus) IsValid() bool {
	switch s {
	case ReadingStatusWriting, ReadingStatusReady, ReadingStatusUnavailable:
		return true
	default:
		return false
	}
}

type ExhibitKind string

const (
	ExhibitRoute    = ExhibitKind("Route")
	ExhibitMeters   = ExhibitKind("Meters")
	ExhibitMessage  = ExhibitKind("Message")
	ExhibitTimeline = ExhibitKind("Timeline")
	ExhibitFacts    = ExhibitKind("Facts")
)

type PointState string

const (
	PointDone     = PointState("Done")
	PointAhead    = PointState("Ahead")
	PointProposed = PointState("Proposed")
)

type DecisionKind string

const (
	DecisionPlan     = DecisionKind("Plan")
	DecisionProposal = DecisionKind("Proposal")
)

type DecisionPhase string

const (
	PhaseAwaiting = DecisionPhase("Awaiting")
	PhaseUndoable = DecisionPhase("Undoable")
	PhaseRunning  = DecisionPhase("Running")
	PhaseDone     = DecisionPhase("Done")
	PhaseFailed   = DecisionPhase("Failed")
	PhaseDeclined = DecisionPhase("Declined")
	PhaseExpired  = DecisionPhase("Expired")
)

type TrustKey string

const (
	TrustMatters   = TrustKey("Matters")
	TrustUnchanged = TrustKey("Unchanged")
	TrustProposes  = TrustKey("Proposes")
	TrustApproval  = TrustKey("Approval")
)
