package agentprompt

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
)

const (
	frameApproval = "What your tools record as a proposal waits on a person: file it and say " +
		"so in the report rather than waiting for the decision."
	frameInferredCaveat = "This is how the request read before you started. When the message " +
		"says otherwise, the message wins."
)

func buildTaskFrameSection(frame *agentturn.TaskFrame) string {
	if frame == nil || strings.TrimSpace(frame.Goal) == "" {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## This task\n")
	if frame.Inferred {
		builder.WriteString("What the person wants: ")
		builder.WriteString(frame.Goal)
		builder.WriteString("\n")
		builder.WriteString(frameInferredCaveat)

		return builder.String()
	}

	builder.WriteString("Goal: ")
	builder.WriteString(frame.Goal)
	if len(frame.Steps) > 0 {
		builder.WriteString("\nHow it is done (playbook ")
		builder.WriteString(frame.Playbook)
		builder.WriteString("), skipping a step whose answer you already have:")
		for idx, step := range frame.Steps {
			builder.WriteString("\n")
			builder.WriteString(strconv.Itoa(idx + 1))
			builder.WriteString(". ")
			builder.WriteString(step)
		}
	}
	if frame.Done != "" {
		builder.WriteString("\nDone when: ")
		builder.WriteString(frame.Done)
	}
	builder.WriteString("\n")
	builder.WriteString(frameApproval)

	return builder.String()
}
