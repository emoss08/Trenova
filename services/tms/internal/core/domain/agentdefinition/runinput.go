package agentdefinition

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
)

func BackgroundRunInput(
	trigger agent.RunTrigger,
	eventKind agent.EventKind,
	subject *RuntimeSubject,
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
