package aianalytics

import "strings"

const (
	VerdictInvalid    = "invalid"
	VerdictDuplicate  = "duplicate"
	VerdictDenied     = "denied"
	VerdictOverBudget = "over_budget"
	VerdictFailed     = "failed"
)

const (
	ReasonNotHeld      = "the agent does not hold the tool"
	ReasonUnregistered = "no such tool is registered"
)

var timeoutMarkers = []string{
	"deadline exceeded",
	"timed out",
	"timeout",
}

func ClassifyToolFailure(verdict, reason, content string) Cause {
	switch verdict {
	case VerdictInvalid:
		if reason == ReasonUnregistered {
			return CauseToolNotHeld
		}

		return CauseWrongArguments
	case VerdictDuplicate:
		return CauseRepeatedCall
	case VerdictDenied:
		if reason == ReasonNotHeld {
			return CauseToolNotHeld
		}

		return CauseBlockedByRule
	case VerdictOverBudget:
		return CauseBlockedByRule
	default:
		if timedOut(reason) || timedOut(content) {
			return CauseToolTimedOut
		}

		return CauseToolError
	}
}

func timedOut(text string) bool {
	if text == "" {
		return false
	}
	lowered := strings.ToLower(text)
	for _, marker := range timeoutMarkers {
		if strings.Contains(lowered, marker) {
			return true
		}
	}

	return false
}
