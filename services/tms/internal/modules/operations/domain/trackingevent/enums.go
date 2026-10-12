package trackingevent

type Source string

const (
	SourceDispatcher = Source("Dispatcher")
	SourceAgent      = Source("Agent")
	SourceDriver     = Source("Driver")
	SourceTelematics = Source("Telematics")
	SourceEDI        = Source("EDI")
)

func (s Source) IsValid() bool {
	_, ok := sourcePrecedence[s]
	return ok
}

func (s Source) Precedence() int {
	return sourcePrecedence[s]
}

func (s Source) Interactive() bool {
	switch s {
	case SourceDispatcher, SourceAgent, SourceDriver:
		return true
	default:
		return false
	}
}

var sourcePrecedence = map[Source]int{
	SourceDispatcher: 50,
	SourceAgent:      40,
	SourceDriver:     30,
	SourceTelematics: 20,
	SourceEDI:        10,
}

func Sources() []Source {
	return []Source{SourceDispatcher, SourceAgent, SourceDriver, SourceTelematics, SourceEDI}
}

type MatchMethod string

const (
	MatchDirect        = MatchMethod("Direct")
	MatchStopReference = MatchMethod("StopReference")
	MatchLocation      = MatchMethod("Location")
	MatchStopRole      = MatchMethod("StopRole")
	MatchInferred      = MatchMethod("Inferred")
)

func (m MatchMethod) IsValid() bool {
	switch m {
	case MatchDirect, MatchStopReference, MatchLocation, MatchStopRole, MatchInferred:
		return true
	default:
		return false
	}
}

type Outcome string

const (
	OutcomeApplied    = Outcome("Applied")
	OutcomePending    = Outcome("Pending")
	OutcomeDuplicate  = Outcome("Duplicate")
	OutcomeSuperseded = Outcome("Superseded")
	OutcomeRefused    = Outcome("Refused")
)

func (o Outcome) IsValid() bool {
	switch o {
	case OutcomeApplied, OutcomePending, OutcomeDuplicate, OutcomeSuperseded, OutcomeRefused:
		return true
	default:
		return false
	}
}

func (o Outcome) NeedsReason() bool {
	switch o {
	case OutcomePending, OutcomeSuperseded, OutcomeRefused:
		return true
	default:
		return false
	}
}

const (
	MaxSourceKeyLength     = 255
	MaxSourceStatusLength  = 64
	MaxRawReferenceLength  = 255
	MaxOutcomeReasonLength = 500
)
