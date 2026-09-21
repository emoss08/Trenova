package agentruntime

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxRationaleChars = 600

func toToolCallRecords(calls []serviceports.ToolCall) []conversation.ToolCallRecord {
	if len(calls) == 0 {
		return nil
	}

	records := make([]conversation.ToolCallRecord, 0, len(calls))
	for _, call := range calls {
		records = append(records, conversation.ToolCallRecord{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	return records
}

func fromToolCallRecords(records []conversation.ToolCallRecord) []serviceports.ToolCall {
	if len(records) == 0 {
		return nil
	}

	calls := make([]serviceports.ToolCall, 0, len(records))
	for _, record := range records {
		calls = append(calls, serviceports.ToolCall{
			ID:        record.ID,
			Name:      record.Name,
			Arguments: record.Arguments,
		})
	}

	return calls
}

func toAdapterMessages(
	history []conversation.Message,
	outcomes []serviceports.ProposalOutcome,
) []serviceports.Message {
	messages := make([]serviceports.Message, 0, len(history)+1)
	ledger := newProposalLedger(outcomes)

	// The history is the newest N messages of the thread, and that cut lands
	// wherever it lands — including between an assistant's tool calls and the
	// results that answer them. Every provider rejects a conversation that
	// opens on an orphaned tool result or on tool calls with nothing answering
	// them, so replay starts at the first whole turn.
	start := 0
	for start < len(history) && history[start].Role != conversation.RoleUser {
		start++
	}

	for _, msg := range history[start:] {
		switch msg.Role {
		case conversation.RoleAssistant:
			if msg.Refused {
				continue
			}
			ledger.noteCalls(msg)
			messages = append(messages, serviceports.Message{
				Role:      serviceports.RoleAssistant,
				Content:   msg.Content,
				ToolCalls: fromToolCallRecords(msg.ToolCalls),
				Reasoning: msg.Reasoning,
			})
		case conversation.RoleTool:
			messages = append(messages, serviceports.Message{
				Role:       serviceports.RoleTool,
				Content:    ledger.currentContent(msg),
				ToolCallID: msg.ToolCallID,
				ToolName:   msg.ToolName,
				IsError:    msg.ToolFailed,
			})
		default:
			if msg.Refused {
				continue
			}
			messages = append(messages, serviceports.Message{
				Role:    serviceports.RoleUser,
				Content: msg.Content,
			})
		}
	}

	return messages
}

// rationaleInput is everything the runtime knows about why a tool was
// called, in the order it is trusted.
type rationaleInput struct {
	// Narration is what the model said in the same turn as the call.
	Narration string
	ToolName  string
	Arguments map[string]any
	// Input is what the person asked, for a call with no account of itself.
	Input string
}

// rationaleArguments are the argument names a tool's own explanation lives
// under, most specific first. raise_exception carries attemptSummary; the
// writes carry a reason or a note.
var rationaleArguments = [...]string{
	"attemptSummary",
	"reason",
	"rationale",
	"summary",
	"justification",
	"note",
	"notes",
	"description",
}

// proposalRationale is the account of a proposal the approver reads.
//
// The model's own narration comes first. A model that calls a tool without
// narrating has usually put its reasoning in the call itself, so the
// arguments are read next; failing that, the person's own request is the
// best account of why, because they asked. Only a call with none of those
// says the agent gave no reason — which used to be the line under every
// proposal from a model that explains in arguments rather than prose.
func proposalRationale(in rationaleInput) string {
	if trimmed := strings.TrimSpace(in.Narration); trimmed != "" {
		return stringutils.Ellipsize(trimmed, maxRationaleChars)
	}

	for _, key := range rationaleArguments {
		value, ok := in.Arguments[key].(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return stringutils.Ellipsize(trimmed, maxRationaleChars)
		}
	}

	if trimmed := strings.TrimSpace(in.Input); trimmed != "" {
		prefix := fmt.Sprintf("Asked to %s in reply to: \u201c", stringutils.HumanizeSnakeCase(in.ToolName))
		room := maxRationaleChars - len([]rune(prefix)) - 1

		return prefix + stringutils.Ellipsize(trimmed, room) + "\u201d"
	}

	return fmt.Sprintf(
		"The agent asked to run %s without explaining why.",
		in.ToolName,
	)
}
