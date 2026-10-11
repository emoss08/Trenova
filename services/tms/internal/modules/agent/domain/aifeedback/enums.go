package aifeedback

type TargetType string

const (
	TargetAssistantMessage = TargetType("AssistantMessage")
	TargetDelegatedAnswer  = TargetType("DelegatedAnswer")
	TargetBriefing         = TargetType("Briefing")
	TargetBriefingSection  = TargetType("BriefingSection")
	TargetInsight          = TargetType("Insight")
	TargetWatchtowerItem   = TargetType("WatchtowerItem")
)

func (t TargetType) IsValid() bool {
	switch t {
	case TargetAssistantMessage,
		TargetDelegatedAnswer,
		TargetBriefing,
		TargetBriefingSection,
		TargetInsight,
		TargetWatchtowerItem:
		return true
	default:
		return false
	}
}

func (t TargetType) String() string { return string(t) }

func (t TargetType) RequiresPart() bool {
	return t == TargetBriefingSection
}

func AllTargetTypes() []TargetType {
	return []TargetType{
		TargetAssistantMessage,
		TargetDelegatedAnswer,
		TargetBriefing,
		TargetBriefingSection,
		TargetInsight,
		TargetWatchtowerItem,
	}
}

type FingerprintSource string

const (
	FingerprintAtTurn   = FingerprintSource("AtTurn")
	FingerprintAtRating = FingerprintSource("AtRating")
	FingerprintNone     = FingerprintSource("None")
)

func (s FingerprintSource) IsValid() bool {
	switch s {
	case FingerprintAtTurn, FingerprintAtRating, FingerprintNone:
		return true
	default:
		return false
	}
}

func AllFingerprintSources() []FingerprintSource {
	return []FingerprintSource{FingerprintAtTurn, FingerprintAtRating, FingerprintNone}
}

type Rating int16

const (
	RatingNegative = Rating(-1)
	RatingPositive = Rating(1)
)

func (r Rating) IsValid() bool {
	return r == RatingNegative || r == RatingPositive
}

type Reason string

const (
	ReasonInaccurate          = Reason("Inaccurate")
	ReasonMadeUpNumbers       = Reason("MadeUpNumbers")
	ReasonIncomplete          = Reason("Incomplete")
	ReasonWrongAction         = Reason("WrongAction")
	ReasonIgnoredInstructions = Reason("IgnoredInstructions")
	ReasonNotRelevant         = Reason("NotRelevant")
	ReasonHardToRead          = Reason("HardToRead")
	ReasonUnsafe              = Reason("Unsafe")
	ReasonOther               = Reason("Other")

	ReasonAccurate  = Reason("Accurate")
	ReasonHelpful   = Reason("Helpful")
	ReasonSavedTime = Reason("SavedTime")
)

func (r Reason) IsValid() bool {
	return r.Negative() || r.Positive()
}

func (r Reason) Negative() bool {
	switch r {
	case ReasonInaccurate,
		ReasonMadeUpNumbers,
		ReasonIncomplete,
		ReasonWrongAction,
		ReasonIgnoredInstructions,
		ReasonNotRelevant,
		ReasonHardToRead,
		ReasonUnsafe,
		ReasonOther:
		return true
	default:
		return false
	}
}

func (r Reason) Positive() bool {
	switch r {
	case ReasonAccurate, ReasonHelpful, ReasonSavedTime:
		return true
	default:
		return false
	}
}

func (r Reason) MatchesRating(rating Rating) bool {
	switch rating {
	case RatingNegative:
		return r.Negative()
	case RatingPositive:
		return r.Positive()
	default:
		return false
	}
}

func (r Reason) Phrase() string {
	switch r {
	case ReasonInaccurate:
		return "inaccurate"
	case ReasonMadeUpNumbers:
		return "made-up numbers"
	case ReasonIncomplete:
		return "incomplete"
	case ReasonWrongAction:
		return "the wrong action"
	case ReasonIgnoredInstructions:
		return "ignored instructions"
	case ReasonNotRelevant:
		return "not relevant"
	case ReasonHardToRead:
		return "hard to read"
	case ReasonUnsafe:
		return "unsafe"
	case ReasonAccurate:
		return "accurate"
	case ReasonHelpful:
		return "helpful"
	case ReasonSavedTime:
		return "saved time"
	default:
		return "another reason"
	}
}

func NegativeReasons() []Reason {
	return []Reason{
		ReasonInaccurate,
		ReasonMadeUpNumbers,
		ReasonIncomplete,
		ReasonWrongAction,
		ReasonIgnoredInstructions,
		ReasonNotRelevant,
		ReasonHardToRead,
		ReasonUnsafe,
		ReasonOther,
	}
}

func PositiveReasons() []Reason {
	return []Reason{ReasonAccurate, ReasonHelpful, ReasonSavedTime}
}

func AllReasons() []Reason {
	return append(NegativeReasons(), PositiveReasons()...)
}
