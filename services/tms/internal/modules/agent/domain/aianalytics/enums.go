package aianalytics

type Source string

const (
	SourceAgentRun      = Source("AgentRun")
	SourceAssistantTurn = Source("AssistantTurn")
	SourceCall          = Source("Call")
)

func AllSources() []Source {
	return []Source{SourceAgentRun, SourceAssistantTurn, SourceCall}
}

type Outcome string

const (
	OutcomeCompleted = Outcome("Completed")
	OutcomeHandedOff = Outcome("HandedOff")
	OutcomeStopped   = Outcome("Stopped")
	OutcomeFailed    = Outcome("Failed")
)

func AllOutcomes() []Outcome {
	return []Outcome{OutcomeCompleted, OutcomeHandedOff, OutcomeStopped, OutcomeFailed}
}

func (o Outcome) IsValid() bool {
	switch o {
	case OutcomeCompleted, OutcomeHandedOff, OutcomeStopped, OutcomeFailed:
		return true
	default:
		return false
	}
}

type Cause string

const (
	CauseWrongArguments = Cause("WrongArguments")
	CauseToolTimedOut   = Cause("ToolTimedOut")
	CauseRepeatedCall   = Cause("RepeatedCall")
	CauseBlockedByRule  = Cause("BlockedByRule")
	CauseToolError      = Cause("ToolError")
	CauseToolNotHeld    = Cause("ToolNotHeld")
)

func AllCauses() []Cause {
	return []Cause{
		CauseWrongArguments,
		CauseToolTimedOut,
		CauseRepeatedCall,
		CauseBlockedByRule,
		CauseToolError,
		CauseToolNotHeld,
	}
}

func (c Cause) IsValid() bool {
	switch c {
	case CauseWrongArguments,
		CauseToolTimedOut,
		CauseRepeatedCall,
		CauseBlockedByRule,
		CauseToolError,
		CauseToolNotHeld:
		return true
	default:
		return false
	}
}

type Blame string

const (
	BlameModel = Blame("Model")
	BlameTool  = Blame("Tool")
)

func (c Cause) Blame() Blame {
	switch c {
	case CauseWrongArguments, CauseRepeatedCall, CauseToolNotHeld:
		return BlameModel
	case CauseToolTimedOut, CauseBlockedByRule, CauseToolError:
		return BlameTool
	default:
		return BlameTool
	}
}

func ModelCauses() []Cause {
	out := make([]Cause, 0, 3)
	for _, cause := range AllCauses() {
		if cause.Blame() == BlameModel {
			out = append(out, cause)
		}
	}

	return out
}

type Window string

const (
	WindowDay   = Window("Day")
	WindowWeek  = Window("Week")
	WindowMonth = Window("Month")
)

func (w Window) IsValid() bool {
	switch w {
	case WindowDay, WindowWeek, WindowMonth:
		return true
	default:
		return false
	}
}

func (w Window) Seconds() int64 {
	switch w {
	case WindowDay:
		return 24 * 60 * 60
	case WindowWeek:
		return 7 * 24 * 60 * 60
	case WindowMonth:
		return 30 * 24 * 60 * 60
	default:
		return 30 * 24 * 60 * 60
	}
}

func (w Window) BucketSeconds() int64 {
	switch w {
	case WindowDay:
		return 60 * 60
	case WindowWeek:
		return 6 * 60 * 60
	default:
		return 24 * 60 * 60
	}
}

type CompareBy string

const (
	CompareByModel    = CompareBy("Model")
	CompareByProvider = CompareBy("Provider")
)

func (c CompareBy) IsValid() bool {
	switch c {
	case CompareByModel, CompareByProvider:
		return true
	default:
		return false
	}
}

func (c CompareBy) Grouping() Grouping {
	if c == CompareByProvider {
		return GroupingProvider
	}

	return GroupingModel
}

type Grouping string

const (
	GroupingModel      = Grouping("Model")
	GroupingProvider   = Grouping("Provider")
	GroupingDeployment = Grouping("Deployment")
)

type ProviderError string

const (
	ProviderErrorNone            = ProviderError("")
	ProviderErrorRateLimited     = ProviderError("RateLimited")
	ProviderErrorTimedOut        = ProviderError("TimedOut")
	ProviderErrorCouldNotConnect = ProviderError("CouldNotConnect")
	ProviderErrorAccessDenied    = ProviderError("AccessDenied")
	ProviderErrorServer          = ProviderError("ServerError")
	ProviderErrorDeclined        = ProviderError("Declined")
	ProviderErrorCancelled       = ProviderError("Cancelled")
)

func AllProviderErrors() []ProviderError {
	return []ProviderError{
		ProviderErrorRateLimited,
		ProviderErrorTimedOut,
		ProviderErrorCouldNotConnect,
		ProviderErrorAccessDenied,
		ProviderErrorServer,
	}
}

func StoredProviderErrors() []ProviderError {
	return append(
		AllProviderErrors(),
		ProviderErrorNone,
		ProviderErrorDeclined,
		ProviderErrorCancelled,
	)
}

func (e ProviderError) BeforeAnswer() bool {
	switch e {
	case ProviderErrorRateLimited,
		ProviderErrorTimedOut,
		ProviderErrorCouldNotConnect,
		ProviderErrorAccessDenied,
		ProviderErrorServer:
		return true
	default:
		return false
	}
}

type EventKind string

const (
	EventTurnedOff        = EventKind("TurnedOff")
	EventTurnedOn         = EventKind("TurnedOn")
	EventModelChanged     = EventKind("ModelChanged")
	EventStoppedAnswering = EventKind("StoppedAnswering")
	EventLostAccess       = EventKind("LostAccess")
	EventRecovered        = EventKind("Recovered")
)
