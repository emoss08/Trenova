package agentruntime

import (
	"encoding/json"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// CompactionKeepTurns is how many of the newest turns a compaction keeps
// whole. The question being worked on and the one before it are what the
// next answer leans on word for word; everything older is summarized.
const CompactionKeepTurns = 2

// fileTools are the tools whose results are the text of a document the person
// attached, which the context meter counts as files rather than tool results.
var fileTools = map[string]struct{}{
	"get_document_summary": {},
	"get_document":         {},
}

// ContextRequest is what a conversation's next turn would send, for
// measuring.
type ContextRequest struct {
	// System and Tools are the prompt and tool definitions of the turn that
	// just ran. Empty after a compaction, which does not build a prompt;
	// Instructions then carries the figure an earlier measure took.
	System       string
	Tools        []serviceports.ToolSpec
	Instructions int
	// History is the conversation as it is read for the next turn: from its
	// latest summary on.
	History   []conversation.Message
	Proposals []serviceports.ProposalOutcome
	// Model is the model the conversation is answered by, whose window the
	// use is measured against.
	Model string
	Now   int64
}

// MeasureContext estimates how much of the model's window a conversation's
// next turn fills, by part, from the replay that turn would send.
//
// The figures are estimates: no tokenizer is shared by every provider, and
// the model a conversation is answered by can change between turns. Text is
// counted at four characters to a token and tool results, which are mostly
// JSON, at three; that errs towards compacting a little early, never late.
func MeasureContext(req ContextRequest) conversation.ContextUsage {
	usage := conversation.ContextUsage{
		Instructions: req.Instructions,
		Window:       aiprovider.ContextWindowFor(req.Model),
		Model:        req.Model,
		MeasuredAt:   req.Now,
	}
	if req.System != "" || len(req.Tools) > 0 {
		usage.Instructions = proseTokens(req.System) + specTokens(req.Tools)
	}

	messages, _ := replayHistory(req.History, req.Proposals)
	for idx := range messages {
		message := &messages[idx]
		switch message.Role {
		case serviceports.RoleTool:
			if _, file := fileTools[message.ToolName]; file {
				usage.Files += dataTokens(message.Content)
			} else {
				usage.ToolResults += dataTokens(message.Content)
			}
		default:
			usage.Messages += proseTokens(message.Content) + callTokens(message.ToolCalls)
		}
	}

	// What a compaction would summarize is everything but the newest turns,
	// measured as what replaying those turns alone would leave out.
	if older, _ := SplitForCompaction(req.History); len(older) > 0 {
		ordered := conversation.ReplayOrder(req.History)
		recent, _ := replayHistory(ordered[len(older):], req.Proposals)
		usage.Compactable = max(0, replayTokens(messages)-replayTokens(recent))
	}

	return usage
}

// SplitForCompaction cuts a conversation, in the order the model reads it,
// into the stretch a compaction summarizes and the newest turns it keeps
// whole. through is the sequence of the last message summarized. older is
// empty when there is nothing new to summarize: fewer turns than are kept
// whole, or nothing but the last summary in front of them.
func SplitForCompaction(history []conversation.Message) (older []conversation.Message, through int) {
	ordered := conversation.ReplayOrder(history)
	cut := recentTurnStart(ordered, CompactionKeepTurns)
	if cut <= 0 {
		return nil, 0
	}

	older = ordered[:cut]
	news := 0
	for idx := range older {
		if !older[idx].Compacted() {
			news++
		}
	}
	if news == 0 {
		return nil, 0
	}

	return older, older[len(older)-1].Sequence
}

func replayTokens(messages []serviceports.Message) int {
	total := 0
	for idx := range messages {
		message := &messages[idx]
		if message.Role == serviceports.RoleTool {
			total += dataTokens(message.Content)
			continue
		}
		total += proseTokens(message.Content) + callTokens(message.ToolCalls)
	}

	return total
}

// proseTokens estimates the tokens of written text.
func proseTokens(text string) int {
	return (len(text) + 3) / 4
}

// dataTokens estimates the tokens of structured data, which tokenizes more
// densely than prose: every brace, quote and key is a token of its own.
func dataTokens(text string) int {
	return (len(text) + 2) / 3
}

func callTokens(calls []serviceports.ToolCall) int {
	total := 0
	for idx := range calls {
		encoded, err := json.Marshal(calls[idx].Arguments)
		if err != nil {
			continue
		}
		total += dataTokens(calls[idx].Name) + dataTokens(string(encoded))
	}

	return total
}

func specTokens(specs []serviceports.ToolSpec) int {
	if len(specs) == 0 {
		return 0
	}
	encoded, err := json.Marshal(specs)
	if err != nil {
		return 0
	}

	return dataTokens(string(encoded))
}
