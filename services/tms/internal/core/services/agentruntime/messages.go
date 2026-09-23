package agentruntime

import (
	"fmt"
	"strings"
	"unicode/utf8"

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
			ID:           call.ID,
			Name:         call.Name,
			Arguments:    call.Arguments,
			ProviderData: call.ProviderData,
			ProviderID:   call.ProviderID,
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
			ID:           record.ID,
			Name:         record.Name,
			Arguments:    record.Arguments,
			ProviderData: record.ProviderData,
			ProviderID:   record.ProviderID,
		})
	}

	return calls
}

// recentToolTurns is how many of the newest user turns keep their tool
// results whole in the replay. Older results are cut to their opening: they
// are the bulk of a long thread, they are stale by the time the thread is
// long, and the model can call the tool again for the current answer.
const recentToolTurns = 3

// compactedToolResultChars is how much of an older tool result survives.
// Enough to see what the call returned; not enough to carry a listing.
const compactedToolResultChars = 320

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
	history = history[start:]

	recentFrom := recentTurnStart(history, recentToolTurns)

	// A call and its result are replayed together or not at all. A refused
	// assistant turn is left out, so a result answering one of its calls
	// would be an orphan; a turn that died between the call and its result
	// leaves a call nothing answers. A provider rejects either.
	answered := answeredCalls(history)
	open := make(map[string]struct{})

	for idx, msg := range history {
		switch msg.Role {
		case conversation.RoleAssistant:
			if msg.Refused {
				continue
			}
			calls := make([]conversation.ToolCallRecord, 0, len(msg.ToolCalls))
			for _, call := range msg.ToolCalls {
				if _, ok := answered[call.ID]; !ok {
					continue
				}
				open[call.ID] = struct{}{}
				calls = append(calls, call)
			}
			if len(calls) == 0 && strings.TrimSpace(msg.Content) == "" {
				continue
			}
			// The ledger counts the calls as they were made, so a proposal's
			// ordinal is not shifted by a call that was left out.
			ledger.noteCalls(msg)
			messages = append(messages, serviceports.Message{
				Role:      serviceports.RoleAssistant,
				Content:   msg.Content,
				ToolCalls: fromToolCallRecords(calls),
				Reasoning: msg.Reasoning,
			})
		case conversation.RoleTool:
			if _, answers := open[msg.ToolCallID]; !answers {
				continue
			}
			delete(open, msg.ToolCallID)
			content, current := ledger.currentContent(msg)
			if !current && idx < recentFrom {
				content = compactToolResult(content)
			}
			messages = append(messages, serviceports.Message{
				Role:       serviceports.RoleTool,
				Content:    content,
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

// answeredCalls is the set of tool call ids a recorded result answers.
func answeredCalls(history []conversation.Message) map[string]struct{} {
	answered := make(map[string]struct{})
	for _, msg := range history {
		if msg.Role == conversation.RoleTool && msg.ToolCallID != "" {
			answered[msg.ToolCallID] = struct{}{}
		}
	}

	return answered
}

// recentTurnStart is the index of the user message that opens the newest
// turns-many turns, or zero when the history holds fewer.
func recentTurnStart(history []conversation.Message, turns int) int {
	seen := 0
	for idx := len(history) - 1; idx >= 0; idx-- {
		if history[idx].Role != conversation.RoleUser || history[idx].Refused {
			continue
		}
		seen++
		if seen == turns {
			return idx
		}
	}

	return 0
}

// compactToolResult cuts an older result to its opening and says what was
// left out. A result already short enough is returned as it is: the note
// would be longer than what it replaced.
func compactToolResult(content string) string {
	if len(content) <= compactedToolResultChars {
		return content
	}

	cut := compactedToolResultChars
	for cut > 0 && !utf8.RuneStart(content[cut]) {
		cut--
	}

	return fmt.Sprintf(
		"%s\n[%d more characters from earlier in the conversation elided; "+
			"call the tool again if the current details matter]",
		content[:cut], len(content)-cut,
	)
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
		prefix := fmt.Sprintf(
			"Asked to %s in reply to: \u201c",
			stringutils.HumanizeSnakeCase(in.ToolName),
		)
		room := maxRationaleChars - len([]rune(prefix)) - 1

		return prefix + stringutils.Ellipsize(trimmed, room) + "\u201d"
	}

	return fmt.Sprintf(
		"The agent asked to run %s without explaining why.",
		in.ToolName,
	)
}
