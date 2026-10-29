package aireasoning

import (
	"slices"
)

// ReasoningEffort is how hard a model is asked to think before it answers, and
// whether it is asked at all.
//
// Off sends no reasoning parameter and is the default, because a provider
// that does not reason rejects the parameter outright: OpenAI returns 400 for
// reasoning_effort on a model without it. An operator turns this on for a
// model they know reasons. Whatever is chosen, thinking a provider volunteers
// unasked — DeepSeek-style reasoning_content — is still read and shown.
//
// Off is not "no thinking" on a model that reasons by default: the GPT-5
// family and open-weight thinking models reason at their own default effort
// when nothing is sent, out of sight and before the first token. None tells such a
// model not to reason; Minimal asks for the least reasoning it allows.
type ReasoningEffort string

const (
	ReasoningOff     = ReasoningEffort("Off")
	ReasoningNone    = ReasoningEffort("None")
	ReasoningMinimal = ReasoningEffort("Minimal")
	ReasoningLow     = ReasoningEffort("Low")
	ReasoningMedium  = ReasoningEffort("Medium")
	ReasoningHigh    = ReasoningEffort("High")
)

func AllReasoningEfforts() []ReasoningEffort {
	return []ReasoningEffort{
		ReasoningOff,
		ReasoningNone,
		ReasoningMinimal,
		ReasoningLow,
		ReasoningMedium,
		ReasoningHigh,
	}
}

func (e ReasoningEffort) IsValid() bool {
	return slices.Contains(AllReasoningEfforts(), e)
}

// ThinkingStyle is how a Claude model is asked to think, for an Anthropic
// provider whose model id does not say.
//
// Claude models from Opus 4.6 and Sonnet 4.6 on think by effort and refuse a
// token budget with a 400; older ones take only the budget. The adapter reads
// which from the model id, which works for the ids the Claude API, Bedrock and
// Vertex use. A gateway's alias says nothing, and is read as an older model,
// so an operator who knows the model behind it says which here.
//
// Auto reads the model id and is the default. Effort asks by effort whatever
// the id reads; Budget asks with a token budget. Only the Anthropic protocol
// takes either.
type ThinkingStyle string

const (
	ThinkingStyleAuto   = ThinkingStyle("Auto")
	ThinkingStyleEffort = ThinkingStyle("Effort")
	ThinkingStyleBudget = ThinkingStyle("Budget")
)

func AllThinkingStyles() []ThinkingStyle {
	return []ThinkingStyle{
		ThinkingStyleAuto,
		ThinkingStyleEffort,
		ThinkingStyleBudget,
	}
}

func (s ThinkingStyle) IsValid() bool {
	return slices.Contains(AllThinkingStyles(), s)
}

// Enabled reports whether the provider should be asked to reason.
func (e ReasoningEffort) Enabled() bool {
	return e.IsValid() && e != ReasoningOff && e != ReasoningNone
}

// Disabled reports whether the provider should be told explicitly not to
// reason, which differs from Off: Off says nothing at all.
func (e ReasoningEffort) Disabled() bool {
	return e == ReasoningNone
}

// Wire is the lower-case word the OpenAI-shaped protocols take.
func (e ReasoningEffort) Wire() string {
	switch e {
	case ReasoningNone:
		return "none"
	case ReasoningMinimal:
		return "minimal"
	case ReasoningLow:
		return "low"
	case ReasoningMedium:
		return "medium"
	case ReasoningHigh:
		return "high"
	case ReasoningOff:
		return ""
	default:
		return ""
	}
}

// ThinkingBudget is the token budget the Anthropic protocol takes for the
// effort. The floor is the API's minimum; the ceiling is well under a reply's
// own token ceiling, which the adapter raises to fit.
func (e ReasoningEffort) ThinkingBudget() int {
	switch e {
	case ReasoningMinimal, ReasoningLow:
		return 1024
	case ReasoningMedium:
		return 4096
	case ReasoningHigh:
		return 16384
	case ReasoningOff, ReasoningNone:
		return 0
	default:
		return 0
	}
}
