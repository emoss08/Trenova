package agentsuggestion

type MatchOutcome string

const (
	MatchBetterAgent  = MatchOutcome("BetterAgent")
	MatchCurrentAgent = MatchOutcome("CurrentAgent")
	MatchNoAgent      = MatchOutcome("NoAgent")
	MatchNone         = MatchOutcome("NoMatch")
)

func (o MatchOutcome) IsValid() bool {
	switch o {
	case MatchBetterAgent, MatchCurrentAgent, MatchNoAgent, MatchNone:
		return true
	default:
		return false
	}
}

func (o MatchOutcome) String() string { return string(o) }

func AllMatchOutcomes() []MatchOutcome {
	return []MatchOutcome{MatchBetterAgent, MatchCurrentAgent, MatchNoAgent, MatchNone}
}

type Surface string

const (
	SurfaceHome         = Surface("Home")
	SurfaceConversation = Surface("Conversation")
)

func (s Surface) IsValid() bool {
	return s == SurfaceHome || s == SurfaceConversation
}

func (s Surface) String() string { return string(s) }

func AllSurfaces() []Surface {
	return []Surface{SurfaceHome, SurfaceConversation}
}

type Outcome string

const (
	OutcomeShown        = Outcome("Shown")
	OutcomeSwitched     = Outcome("Switched")
	OutcomeSentThrough  = Outcome("SentThrough")
	OutcomeAutoSwitched = Outcome("AutoSwitched")
	OutcomeKept         = Outcome("Kept")
	OutcomePassedOver   = Outcome("PassedOver")
)

func (o Outcome) IsValid() bool {
	switch o {
	case OutcomeShown,
		OutcomeSwitched,
		OutcomeSentThrough,
		OutcomeAutoSwitched,
		OutcomeKept,
		OutcomePassedOver:
		return true
	default:
		return false
	}
}

func (o Outcome) String() string { return string(o) }

func (o Outcome) Settles() bool {
	switch o {
	case OutcomeSwitched, OutcomeSentThrough, OutcomeKept, OutcomePassedOver:
		return true
	default:
		return false
	}
}

func (o Outcome) Opens() bool {
	return o == OutcomeShown || o == OutcomeAutoSwitched
}

func (o Outcome) Taken() bool {
	return o == OutcomeSwitched || o == OutcomeSentThrough || o == OutcomeAutoSwitched
}

func AllOutcomes() []Outcome {
	return []Outcome{
		OutcomeShown,
		OutcomeSwitched,
		OutcomeSentThrough,
		OutcomeAutoSwitched,
		OutcomeKept,
		OutcomePassedOver,
	}
}
