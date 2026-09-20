package agentguard

import (
	"strings"

	"github.com/emoss08/trenova/pkg/pagination"
)

// Turn is one earlier message, as the classifier needs to read it.
type Turn struct {
	// Role is "user" or "assistant". Tool traffic is deliberately absent: a
	// tool result is a payload, and what it establishes about the subject is
	// already in the prose the assistant wrote from it.
	Role    string
	Content string
}

// EvaluateRequest is one scope decision.
type EvaluateRequest struct {
	TenantInfo pagination.TenantInfo
	// Input is the message being judged. Only this is classified; Recent exists
	// to say what it is about.
	Input string
	// Recent is the tail of the conversation, oldest first.
	Recent []Turn
}

const (
	// maxContextTurns is how far back the classifier is shown. A follow-up
	// refers to something said a turn or two ago; more than this is history,
	// and history dilutes what the message is actually about.
	maxContextTurns = 6
	// maxContextChars bounds one turn. The subject of a message is established
	// in its first sentence, and a full answer — a table of drivers, say —
	// would otherwise dominate the prompt and cost more than the call it saves.
	maxContextChars = 400
)

// conversationContext renders the tail of a thread for the classifier.
//
// Empty when there is nothing to show, so a first message is classified exactly
// as it was before this existed.
func (r EvaluateRequest) conversationContext() string {
	turns := r.Recent
	if len(turns) > maxContextTurns {
		turns = turns[len(turns)-maxContextTurns:]
	}

	var builder strings.Builder
	for _, turn := range turns {
		content := strings.TrimSpace(turn.Content)
		if content == "" {
			continue
		}
		if len(content) > maxContextChars {
			content = content[:truncationBoundary(content, maxContextChars)] + "…"
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(turn.Role)
		builder.WriteString(": ")
		builder.WriteString(content)
	}

	return builder.String()
}

// truncationBoundary backs a cut off a multi-byte rune, so a trimmed turn is
// still valid text rather than a broken one the provider may reject.
func truncationBoundary(content string, limit int) int {
	for limit > 0 && !isRuneStart(content[limit]) {
		limit--
	}

	return limit
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }
