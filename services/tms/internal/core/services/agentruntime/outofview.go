package agentruntime

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// maxOutOfViewDecisions bounds how many decided proposals outside the replay
// the model is told about, newest last. Enough for the cards a delegate raised
// on one task; a thread's whole history of decisions is the Decision entries'.
const maxOutOfViewDecisions = 5

// outOfViewDecisions says what became of decided proposals whose tool call the
// model cannot see in its replay, so the ledger cannot swap their frozen text.
//
// A delegate's proposals are the case this is for. Its steps are never
// replayed, so the only account the model read was the delegate_task result
// saying the card was waiting for approval, and it kept saying so after the
// person had approved it. A proposal from before the replayed history falls
// out of view the same way.
func outOfViewDecisions(
	history []conversation.Message,
	outcomes []serviceports.ProposalOutcome,
) string {
	if len(outcomes) == 0 {
		return ""
	}

	inView := make(map[pulid.ID]struct{}, len(history))
	for idx := range history {
		if history[idx].ID.IsNotNil() {
			inView[history[idx].ID] = struct{}{}
		}
	}

	lines := make([]string, 0, maxOutOfViewDecisions)
	for idx := len(outcomes) - 1; idx >= 0 && len(lines) < maxOutOfViewDecisions; idx-- {
		outcome := outcomes[idx]
		if outcome.SourceMessageID.IsNil() || outcome.Pending() ||
			outcome.AutonomyTier == agent.TierAutoExecute {
			continue
		}
		if _, seen := inView[outcome.SourceMessageID]; seen {
			continue
		}
		lines = append(lines, "- "+proposalOutcomeText(outcome.ToolName, outcome))
	}
	if len(lines) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("[What became of proposals raised earlier in this conversation whose ")
	b.WriteString("calls are not above, such as another agent's on a task you handed it. ")
	b.WriteString("This is their current state; it replaces anything earlier that said ")
	b.WriteString("they were waiting:\n")
	for idx := len(lines) - 1; idx >= 0; idx-- {
		b.WriteString(lines[idx])
		b.WriteByte('\n')
	}
	b.WriteString("]\n\n")

	return b.String()
}
