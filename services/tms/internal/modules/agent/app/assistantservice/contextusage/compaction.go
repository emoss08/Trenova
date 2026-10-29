package contextusage

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/timeutils"
)

// LastInstructions is the prompt's measure from the conversation's last
// measure, for a measure taken without building a prompt.
func LastInstructions(thread *conversation.Thread) int {
	if thread.ContextUsage == nil {
		return 0
	}

	return thread.ContextUsage.Instructions
}

// AnsweringModel is the model that wrote a turn's last reply.
func AnsweringModel(saved []conversation.Message) string {
	for idx := len(saved) - 1; idx >= 0; idx-- {
		if saved[idx].Role == conversation.RoleAssistant && saved[idx].Model != "" {
			return saved[idx].Model
		}
	}

	return ""
}

// UsageModel is the model a measure is taken against: the one that just
// answered, or failing that the one the conversation was last measured on.
func UsageModel(thread *conversation.Thread, model string) string {
	if model != "" {
		return model
	}
	if thread.ContextUsage != nil {
		return thread.ContextUsage.Model
	}

	return ""
}

// UsageWindow is the configured window to measure against, taken with the
// model usageModel picks: the answering provider's when a model answered,
// and otherwise the window the conversation was last measured against, so a
// window an operator configured is not lost to a measure that had no reply.
func UsageWindow(thread *conversation.Thread, model string, configured int) int {
	if model != "" {
		return configured
	}
	if thread.ContextUsage != nil {
		return thread.ContextUsage.Window
	}

	return 0
}

// currentUsage is how full the conversation is, taken fresh from its history
// with the instructions the last turn measured.
func CurrentUsage(
	thread *conversation.Thread,
	history []conversation.Message,
	outcomes []agentruntimeports.ProposalOutcome,
) conversation.ContextUsage {
	return turnhistory.MeasureContext(turnhistory.ContextRequest{
		Instructions: LastInstructions(thread),
		History:      history,
		Proposals:    outcomes,
		Model:        UsageModel(thread, ""),
		Window:       UsageWindow(thread, "", 0),
		Now:          timeutils.NowUnix(),
	})
}
