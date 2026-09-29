package agentruntime

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	changeCutOffCallRetry = "agent-loop-cut-off-call-retry"
	cutOffRetryCeiling    = 32768
	cutOffRetryReason     = "The reply ran out of room partway through a tool call. " +
		"Asking again with more room."
)

type cutOff struct {
	tool  string
	limit int
}

type cutOffState struct {
	budget   int
	retrying bool
	first    int
}

func cutOffCall(completion *serviceports.ChatCompletionResult) (cutOff, bool) {
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
		return cutOff{tool: completion.CutOffCall.Name, limit: limit}, true
	case len(completion.ToolCalls) > 0:
		return cutOff{tool: broken, limit: limit}, true
	case strings.TrimSpace(completion.Text) == "":
		return cutOff{limit: limit}, true
	default:
		return cutOff{}, false
	}
}

func raisedOutputBudget(limit int) int {
	if limit <= 0 {
		return 0
	}
	raised := min(aiprovider.ClampMaxTokens(limit*2), cutOffRetryCeiling)
	if raised <= limit {
		return 0
	}

	return raised
}

func (s *Service) handleCutOff(
	t *Turn,
	fx TurnEffects,
	completion *serviceports.ChatCompletionResult,
) (retry, stop bool) {
	cut, isCut := cutOffCall(completion)
	if !isCut {
		t.cutOff.retrying = false
		return false, false
	}
	if !fx.Supports(changeCutOffCallRetry) {
		return false, false
	}

	fx.Emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventRetrying,
		Data: serviceports.AssistantRetryingEvent{
			Attempt: 1,
			Reason:  cutOffRetryReason,
			Kind:    serviceports.RetryKindRestart,
		},
	})

	if !t.cutOff.retrying {
		if next := raisedOutputBudget(cut.limit); next > 0 {
			t.cutOff = cutOffState{budget: next, retrying: true, first: cut.limit}
			return true, false
		}
		t.cutOff.first = cut.limit
	}

	return false, true
}

func (s *Service) finishCutOff(
	t *Turn,
	fx TurnEffects,
	completion *serviceports.ChatCompletionResult,
) *serviceports.RunResult {
	cut, _ := cutOffCall(completion)
	reply := cutOffNotice(cut, t.cutOff)
	if visible := strings.TrimSpace(completion.Text); visible != "" {
		reply = visible + "\n\n" + reply
	}
	fx.Emit(deltaEvent(reply))

	final := cannedCompletion(completion, reply)
	final.Reasoning = completion.Reasoning
	result := s.finish(t.result, final, fx)
	result.Truncated = true

	return result
}

func cutOffNotice(cut cutOff, state cutOffState) string {
	first := state.first
	if first <= 0 {
		first = cut.limit
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
	if state.retrying && cut.limit > first {
		notice.WriteString(", and again at " + strconv.Itoa(cut.limit) +
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
