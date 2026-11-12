package watchtowersources

import (
	"strings"

	"github.com/emoss08/trenova/shared/recordids"
)

const (
	maxRunFailureReason = 240

	// runFailedWithoutReason stands in for a run that recorded no error. A
	// run's own summary says what it did, not why it stopped, so it is never
	// offered as the reason.
	runFailedWithoutReason = "The run stopped without recording why. Open the run to see " +
		"how far it got."
)

type runFailure struct {
	marks  []string
	reason string
}

var runFailures = []runFailure{
	{
		marks:  []string{"context deadline exceeded", "timeout", "timed out"},
		reason: "The AI provider did not answer in time.",
	},
	{
		marks:  []string{"no provider configured", "no ai provider"},
		reason: "No AI provider is set up for this agent.",
	},
	{
		marks:  []string{"status 401", "status 403", "invalid api key", "unauthorized"},
		reason: "The AI provider refused the sign-in. Check its key in AI control.",
	},
	{
		marks:  []string{"status 429", "rate limit", "quota", "overloaded"},
		reason: "The AI provider is over its limit and turned the request away.",
	},
	{
		marks:  []string{"every configured chat provider failed", "status 400", "status 5"},
		reason: "The AI provider could not take the request.",
	},
	{
		marks:  []string{"budget", "spend limit"},
		reason: "The agent reached its spending limit.",
	},
}

/*
RunFailureReason is why a run did not finish, as a person reads it. What a
run records is the error of whatever broke (a workflow activity's name and
event numbers, a provider's HTTP reply), which tells an engineer where to
look and tells the person relying on the agent nothing. The kinds of failure
a person can act on are named in their own words; any other is said plainly,
and the run itself, one click away, keeps the error as it was recorded.
*/
func RunFailureReason(recorded string) string {
	recorded = strings.TrimSpace(recorded)
	if recorded == "" {
		return ""
	}

	lowered := strings.ToLower(recorded)
	for _, failure := range runFailures {
		for _, mark := range failure.marks {
			if strings.Contains(lowered, mark) {
				return failure.reason
			}
		}
	}
	if technical(lowered) {
		return "The run stopped on an error. Open the run to see what it recorded."
	}

	return ellipsize(recordids.Strip(recorded))
}

func technical(lowered string) bool {
	for _, mark := range []string{
		"activity error", "workflow", "scheduledeventid", "panic", "goroutine", "sqlstate",
		"rpc error", "nil pointer", "stack trace", "eof",
	} {
		if strings.Contains(lowered, mark) {
			return true
		}
	}

	return false
}

func ellipsize(text string) string {
	runes := []rune(text)
	if len(runes) <= maxRunFailureReason {
		return text
	}

	return strings.TrimSpace(string(runes[:maxRunFailureReason-1])) + "…"
}
