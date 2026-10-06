package agentruntime

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

/*
A turn can outgrow its window on its own.

Compaction runs between turns: it keeps a conversation small enough that the
next question starts with room. Inside a turn nothing did. An agent that reads
twenty long listings in a row sends every one of them again with each call
after it, and the request the provider finally refuses is one the person never
saw coming, halfway through an answer they were waiting for.

So before each call the request is measured as the context meter measures it,
and when it is past the share a conversation compacts itself at, the oldest
large tool results are cut to their opening until it fits. The model read each
of them whole on the call after it ran, and has said what it made of them
since; what it is still working from is the newest results, which stay whole.

Only what is sent is cut. The turn keeps every result as it came back, so the
saved conversation, the grounding guard and the next turn's replay all see the
whole of it.
*/

// turnWindowShare is how full a turn's request may get before its older tool
// results are shortened. It is the share a conversation compacts itself at,
// for the same reason: what is left is room for the reply, and for the
// estimate being off.
const turnWindowShare = conversation.AutoCompactShare

// squeezedToolResultChars is how much of an older result survives being
// squeezed. More than the replay keeps of an earlier turn's result: the model
// read this one moments ago and may still be leaning on its first rows.
const squeezedToolResultChars = 1_500

// squeezeFloorTokens is the smallest result worth squeezing. Below it the cut
// and its note free next to nothing.
const squeezeFloorTokens = 1_000

// window is the context window of the model answering the turn, or zero
// before any has answered. The first call carries no results of the turn's
// own, so there is nothing to squeeze before it.
func (t *Turn) window() int {
	if t.result == nil || t.result.Model == "" {
		return 0
	}

	return aiprovider.WindowOr(t.result.ContextWindow, t.result.Model)
}

// fitWindow shortens the request's older tool results, oldest first, while it
// is over turnWindowShare of window. The results after the latest assistant
// message, which the model has not answered yet, are never cut. The request's
// messages are copied before the first cut, so the turn's own are untouched.
func fitWindow(req *serviceports.ChatCompletionRequest, window int) {
	if req == nil || window <= 0 {
		return
	}

	budget := int(float64(window) * turnWindowShare)
	total := requestTokens(req)
	if total <= budget {
		return
	}

	latest := latestResults(req.Messages)
	var messages []serviceports.Message
	for idx := 0; idx < latest && total > budget; idx++ {
		message := &req.Messages[idx]
		if message.Role != serviceports.RoleTool ||
			dataTokens(message.Content) <= squeezeFloorTokens {
			continue
		}
		squeezed := squeezeToolResult(message.Content)
		if len(squeezed) >= len(message.Content) {
			continue
		}
		if messages == nil {
			messages = slices.Clone(req.Messages)
		}
		total -= dataTokens(message.Content) - dataTokens(squeezed)
		messages[idx].Content = squeezed
	}

	if messages != nil {
		req.Messages = messages
	}
}

// requestTokens estimates a request the way the context meter estimates a
// conversation: the prompt and tool definitions, then the messages.
func requestTokens(req *serviceports.ChatCompletionRequest) int {
	return proseTokens(req.System) + specTokens(req.Tools) + replayTokens(req.Messages)
}

// latestResults is where the results the model has yet to answer begin: just
// after the latest assistant message, or the end of the request when it has
// none.
func latestResults(messages []serviceports.Message) int {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		if messages[idx].Role == serviceports.RoleAssistant {
			return idx + 1
		}
	}

	return len(messages)
}

// squeezeToolResult cuts a result the model already read to its opening and
// says so. A fenced result is cut inside a fence of its own, with the note
// after it, so the note is never read as part of the data.
func squeezeToolResult(content string) string {
	tool, payload, fenced := UnfenceToolResult(content)
	if !fenced {
		if len(content) <= squeezedToolResultChars {
			return content
		}
		cut := runeCut(content, squeezedToolResultChars)

		return content[:cut] + squeezeNote(len(content)-cut, "the tool")
	}
	if len(payload) <= squeezedToolResultChars {
		return content
	}

	head := payload[:runeCut(payload, squeezedToolResultChars)]
	trailer := ""
	if closeAt := strings.Index(content, "\n"+untrustedCloseTag); closeAt >= 0 {
		trailer = content[closeAt+len("\n"+untrustedCloseTag):]
	}

	return FenceToolResult(tool, head) + trailer + squeezeNote(len(payload)-len(head), tool)
}

func squeezeNote(left int, tool string) string {
	return fmt.Sprintf("\n[This result, from earlier in this turn, was shortened to keep the "+
		"request inside the model's context window: %d more characters were left out. "+
		"You read it whole when it arrived; rely on what you concluded from it then, "+
		"do not quote or count anything from the part left out, and call %s again "+
		"if you need those details now.]", left, tool)
}
