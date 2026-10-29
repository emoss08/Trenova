package loopguard

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const (
	CutOffRetryCeiling = 32768
	CutOffRetryReason  = "The reply ran out of room partway through a tool call. " +
		"Asking again with more room."
)

type cutOff struct {
	tool  string
	Limit int
}

type CutOffState struct {
	Budget   int
	Retrying bool
	First    int
}

func CutOffCall(completion *llmports.ChatCompletionResult) (cutOff, bool) {
	if completion == nil || !completion.Truncated {
		return cutOff{}, false
	}

	broken := ""
	for idx := range completion.ToolCalls {
		call := &completion.ToolCalls[idx]
		if call.ArgumentsError == "" {
			return cutOff{}, false
		}
		if broken == "" {
			broken = call.Name
		}
	}

	limit := completion.OutputLimit
	if limit <= 0 {
		limit = completion.OutputTokens
	}

	switch {
	case completion.CutOffCall != nil:
		return cutOff{tool: completion.CutOffCall.Name, Limit: limit}, true
	case len(completion.ToolCalls) > 0:
		return cutOff{tool: broken, Limit: limit}, true
	case strings.TrimSpace(completion.Text) == "":
		return cutOff{Limit: limit}, true
	default:
		return cutOff{}, false
	}
}

func RaisedOutputBudget(limit int) int {
	if limit <= 0 {
		return 0
	}
	raised := min(aiprovider.ClampMaxTokens(limit*2), CutOffRetryCeiling)
	if raised <= limit {
		return 0
	}

	return raised
}

func CutOffNotice(cut cutOff, state CutOffState) string {
	first := state.First
	if first <= 0 {
		first = cut.Limit
	}

	var notice strings.Builder
	if cut.tool != "" {
		notice.WriteString("_I ran out of room before I could file `" + cut.tool +
			"`: the model's output limit cut off the call")
	} else {
		notice.WriteString("_I ran out of room before I could answer or file anything: " +
			"the model's output limit was reached")
	}
	if first > 0 {
		notice.WriteString(" at " + strconv.Itoa(first) + " tokens")
	}
	if state.Retrying && cut.Limit > first {
		notice.WriteString(", and again at " + strconv.Itoa(cut.Limit) +
			" when I asked with more room")
	}
	if cut.tool != "" {
		notice.WriteString(". `" + cut.tool + "` was not filed.")
	} else {
		notice.WriteString(". Nothing was filed.")
	}
	notice.WriteString(" Raise **Max tokens** on this model's provider in AI Control, " +
		"or ask for fewer records at once._")

	return notice.String()
}
