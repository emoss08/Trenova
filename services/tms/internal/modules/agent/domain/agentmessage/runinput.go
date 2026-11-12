package agentmessage

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
)

func BackgroundRunInput(
	trigger agent.RunTrigger,
	eventKind agenttrigger.EventKind,
	subject *agentturn.RuntimeSubject,
) string {
	var builder strings.Builder
	switch trigger {
	case agent.RunTriggerEvent:
		builder.WriteString("An event started this run")
		if eventKind != "" {
			builder.WriteString(": ")
			builder.WriteString(string(eventKind))
		}
		builder.WriteString(".")
	case agent.RunTriggerScheduled:
		builder.WriteString("This is a scheduled run.")
	case agent.RunTriggerContinuous:
		builder.WriteString("This is one pass of a continuous run.")
	case agent.RunTriggerWait:
		builder.WriteString("An earlier run parked this work on a wait, and the wait has ended.")
	default:
		builder.WriteString("A person started this run.")
	}
	if subject != nil {
		builder.WriteString(" It concerns ")
		builder.WriteString(subject.Label)
		builder.WriteString(" (")
		builder.WriteString(string(subject.Type))
		builder.WriteString(" ")
		builder.WriteString(subject.ID)
		builder.WriteString("), described in the runtime context.")
	}
	builder.WriteString(
		" Follow your instructions: look up what you need, act through your tools " +
			"where you are allowed to, propose what needs a person, and finish with a " +
			"short report of what you found and did.",
	)

	return builder.String()
}

// HandedNote tells a run what it was handed. A run a person starts from the
// Watchtower is about one thing that was raised on the record, and an agent
// told only which record it concerns works its routine and reports on
// something else.
func HandedNote(handed string) string {
	return "A person handed you this from the Watchtower, and it is the thing to work: " +
		strings.TrimSpace(handed) +
		"\nWork this first. If you cannot settle it with the tools you hold, say plainly " +
		"what a person has to do to settle it, and keep anything else you notice about " +
		"the record to a line at the end."
}
