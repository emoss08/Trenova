package agentruntime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxExecutionErrorChars = 600

// proposalLedger matches replayed tool results to what became of the
// proposals they recorded.
//
// The tool result written at proposal time says "awaiting a person's
// review", and that is what the model used to read on every later turn,
// however the person had decided since. So it re-proposed what was pending,
// and told the person a change was made when its execution had failed. The
// ledger swaps the frozen text for the proposal's current state.
//
// A proposal knows the assistant message that raised it and the tool it
// named, not the call id, so the k-th replayed call of a tool in a message
// is paired with the k-th proposal that tool raised from it. Both were
// recorded in call order.
type proposalLedger struct {
	bySource map[pulid.ID]map[string][]serviceports.ProposalOutcome
	calls    map[string]sourcedCall
}

type sourcedCall struct {
	messageID pulid.ID
	toolName  string
	ordinal   int
}

func newProposalLedger(outcomes []serviceports.ProposalOutcome) *proposalLedger {
	ledger := &proposalLedger{
		bySource: make(map[pulid.ID]map[string][]serviceports.ProposalOutcome),
		calls:    make(map[string]sourcedCall),
	}
	for _, outcome := range outcomes {
		if outcome.SourceMessageID.IsNil() {
			continue
		}
		byTool, ok := ledger.bySource[outcome.SourceMessageID]
		if !ok {
			byTool = make(map[string][]serviceports.ProposalOutcome)
			ledger.bySource[outcome.SourceMessageID] = byTool
		}
		byTool[outcome.ToolName] = append(byTool[outcome.ToolName], outcome)
	}

	return ledger
}

// noteCalls remembers which message each replayed call came from, and how
// many calls of the same tool preceded it there.
func (l *proposalLedger) noteCalls(msg conversation.Message) {
	if len(l.bySource) == 0 || msg.ID.IsNil() {
		return
	}
	ordinals := make(map[string]int, len(msg.ToolCalls))
	for _, call := range msg.ToolCalls {
		l.calls[call.ID] = sourcedCall{
			messageID: msg.ID,
			toolName:  call.Name,
			ordinal:   ordinals[call.Name],
		}
		ordinals[call.Name]++
	}
}

// currentContent is the tool result as it stands now: the proposal's
// present state when the result recorded one, the stored text otherwise.
func (l *proposalLedger) currentContent(msg conversation.Message) (string, bool) {
	call, ok := l.calls[msg.ToolCallID]
	if !ok {
		return msg.Content, false
	}
	outcomes := l.bySource[call.messageID][call.toolName]
	if call.ordinal >= len(outcomes) {
		return msg.Content, false
	}

	return proposalOutcomeText(msg.ToolName, outcomes[call.ordinal]), true
}

// proposalOutcomeText says what became of a proposal, in words the model
// can act on: what to tell the person, and whether to propose again.
// modifiedHow says what the person changed before approving, so the model
// speaks of what ran rather than of what it asked for. Keys are listed in
// order so the text is the same for the same change.
func modifiedHow(outcome serviceports.ProposalOutcome) string {
	if len(outcome.Modifications) == 0 {
		return ""
	}

	keys := make([]string, 0, len(outcome.Modifications))
	for key := range outcome.Modifications {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		encoded, err := sonic.ConfigStd.Marshal(outcome.Modifications[key])
		if err != nil {
			continue
		}
		parts = append(parts, key+" = "+string(encoded))
	}

	return " after changing " + strings.Join(parts, ", ")
}

func proposalOutcomeText(toolName string, outcome serviceports.ProposalOutcome) string {
	switch outcome.Status {
	case agent.ProposalStatusPending:
		return fmt.Sprintf(
			"Recorded a proposal to run %q. It is still waiting for the person's decision "+
				"on its card in this conversation and has not run. Do not propose it again; "+
				"if asked, say it is waiting for their approval on the card, which they give "+
				"with the button rather than by replying.",
			toolName,
		)
	case agent.ProposalStatusAccepted, agent.ProposalStatusModified:
		return fmt.Sprintf(
			"The person approved the proposal to run %q%s and it is being carried out now. "+
				"Do not propose it again.",
			toolName, modifiedHow(outcome),
		)
	case agent.ProposalStatusExecuted:
		return fmt.Sprintf(
			"The person approved the proposal to run %q%s and it ran successfully%s. "+
				"The change has been made; do not propose it again.",
			toolName, modifiedHow(outcome), executedWhen(outcome),
		)
	case agent.ProposalStatusSimulated:
		return fmt.Sprintf(
			"The person approved the proposal to run %q, and because this agent is in "+
				"simulation the change was previewed rather than made. Nothing has changed.",
			toolName,
		)
	case agent.ProposalStatusExecutionFailed:
		reason := strings.TrimSpace(outcome.ExecutionError)
		if reason == "" {
			reason = "no reason was recorded"
		}

		return fmt.Sprintf(
			"The person approved the proposal to run %q but it FAILED when it ran: %s\n"+
				"Nothing was changed. Fix what the failure names before proposing again, "+
				"and tell the person the change did not go through.",
			toolName, stringutils.Ellipsize(reason, maxExecutionErrorChars),
		)
	case agent.ProposalStatusRejected:
		return fmt.Sprintf(
			"The person REJECTED the proposal to run %q. Nothing was changed. Do not propose "+
				"the same change again unless they ask for it.",
			toolName,
		)
	case agent.ProposalStatusExpired:
		return fmt.Sprintf(
			"The proposal to run %q expired without a decision and nothing was changed. "+
				"Propose it again only if the person still wants it.",
			toolName,
		)
	case agent.ProposalStatusSkipped:
		return fmt.Sprintf(
			"The proposal to run %q was skipped because an earlier step of the same plan "+
				"did not complete. Nothing was changed.",
			toolName,
		)
	case agent.ProposalStatusSuperseded:
		return fmt.Sprintf(
			"The proposal to run %q was superseded by a later one and nothing was changed.",
			toolName,
		)
	default:
		return fmt.Sprintf("The proposal to run %q is %s.", toolName, outcome.Status)
	}
}

func executedWhen(outcome serviceports.ProposalOutcome) string {
	if outcome.ExecutedAt == nil || *outcome.ExecutedAt == 0 {
		return ""
	}

	return fmt.Sprintf(" at %d", *outcome.ExecutedAt)
}

// pendingDuplicate reports whether an identical write is already waiting
// on the person, either from an earlier turn or from earlier in this one.
//
// Arguments that cannot be encoded are never a duplicate of anything. Two
// such sets both encode to nothing, and comparing those nothings dropped a
// second, different proposal as a copy of the first.
func pendingDuplicate(
	call serviceports.ToolCall,
	earlier []serviceports.ProposalOutcome,
	thisTurn []serviceports.PendingAction,
) bool {
	key := argumentsKey(call.Arguments)
	if key == "" {
		return false
	}
	for _, outcome := range earlier {
		if outcome.Pending() && outcome.ToolName == call.Name &&
			argumentsKey(outcome.ToolParams) == key {
			return true
		}
	}
	for _, action := range thisTurn {
		if !action.Executed && !action.Simulated && action.ToolName == call.Name &&
			argumentsKey(action.Arguments) == key {
			return true
		}
	}

	return false
}

func duplicateProposalText(toolName string) string {
	return fmt.Sprintf(
		"An identical proposal to run %q is already waiting for the person's decision "+
			"on its card in this conversation, so it was not recorded again. Tell the person "+
			"it is waiting for their approval there; replying \"yes\" does not approve it.",
		toolName,
	)
}

// argumentsKey identifies a set of arguments however a provider ordered
// them. Arguments that cannot be marshalled match nothing, which fails
// open: a second card is a smaller fault than a refused first one.
func argumentsKey(args map[string]any) string {
	encoded, err := sonic.ConfigStd.Marshal(args)
	if err != nil {
		return ""
	}

	return string(encoded)
}
