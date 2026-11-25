package turnhistory

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxFailedCallLines   = 8
	maxFailedReasonRunes = 200
	unansweredCallReason = "no result was recorded; the turn stopped before the call finished"
	turnAbove            = "in the turn above"
	turnOutOfView        = "earlier in the conversation, in a turn no longer in view,"
)

var failurePrefix = regexp.MustCompile(
	`^Tool "[^"]*" (?:was not run(?: or proposed)?|failed)[:.]?\s*`,
)

type failedCall struct {
	tool   string
	reason string
	count  int
}

type failedCalls struct {
	answered map[string]struct{}
	calls    []failedCall
	total    int
}

func newFailedCalls(answered map[string]struct{}) *failedCalls {
	return &failedCalls{answered: answered}
}

func (f *failedCalls) note(msg *conversation.Message) {
	if msg.Refused {
		return
	}
	switch msg.Role {
	case conversation.RoleTool:
		if msg.ToolFailed {
			f.add(msg.ToolName, failureReason(msg.Content))
		}
	case conversation.RoleAssistant:
		for _, call := range msg.ToolCalls {
			if _, ok := f.answered[call.ID]; !ok {
				f.add(call.Name, unansweredCallReason)
			}
		}
	default:
	}
}

func (f *failedCalls) add(tool, reason string) {
	f.total++
	for idx := range f.calls {
		if f.calls[idx].tool == tool && f.calls[idx].reason == reason {
			f.calls[idx].count++
			return
		}
	}
	f.calls = append(f.calls, failedCall{tool: tool, reason: reason, count: 1})
}

func (f *failedCalls) flush(messages []llmports.Message, where string) []llmports.Message {
	if f.total == 0 {
		return messages
	}
	messages = append(messages, llmports.NoticeMessage(f.notice(where)))
	f.calls, f.total = f.calls[:0], 0

	return messages
}

func (f *failedCalls) notice(where string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%d tool %s %s did not go through (failed or refused). "+
		"A later call that worked does not undo them; if the person asks what did not "+
		"go through, name each one.", f.total, pluralCalls(f.total), where)
	shown := 0
	for idx := range f.calls {
		if idx == maxFailedCallLines {
			break
		}
		call := &f.calls[idx]
		shown += call.count
		b.WriteString("\n- ")
		b.WriteString(call.tool)
		if call.count > 1 {
			fmt.Fprintf(&b, " (%d times)", call.count)
		}
		b.WriteString(": ")
		b.WriteString(call.reason)
	}
	if rest := f.total - shown; rest > 0 {
		fmt.Fprintf(&b, "\n- and %d more failed %s", rest, pluralCalls(rest))
	}
	b.WriteString("]")

	return b.String()
}

func failureReason(content string) string {
	if message, fenced := toolresult.UnfenceFailure(content); fenced {
		content = message
	}
	reason := failurePrefix.ReplaceAllString(strings.TrimSpace(content), "")
	reason = strings.Join(strings.Fields(reason), " ")
	if reason == "" {
		return "the tool reported a failure without a reason"
	}

	return stringutils.Ellipsize(reason, maxFailedReasonRunes)
}

func pluralCalls(n int) string {
	if n == 1 {
		return "call"
	}

	return "calls"
}
