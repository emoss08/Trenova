package agentruntime

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func (s *Service) handleCutOff(
	t *Turn,
	fx TurnEffects,
	completion *llmports.ChatCompletionResult,
) (retry, stop bool) {
	cut, isCut := loopguard.CutOffCall(completion)
	if !isCut {
		t.cutOff.Retrying = false
		return false, false
	}
	fx.Emit(kernelports.StreamEvent{
		Event: assistantports.AssistantEventRetrying,
		Data: assistantports.AssistantRetryingEvent{
			Attempt: 1,
			Reason:  loopguard.CutOffRetryReason,
			Kind:    llmports.RetryKindRestart,
		},
	})

	if !t.cutOff.Retrying {
		if next := loopguard.RaisedOutputBudget(cut.Limit); next > 0 {
			t.cutOff = loopguard.CutOffState{Budget: next, Retrying: true, First: cut.Limit}
			return true, false
		}
		t.cutOff.First = cut.Limit
	}

	return false, true
}

func (s *Service) finishCutOff(
	t *Turn,
	fx TurnEffects,
	completion *llmports.ChatCompletionResult,
) *agentruntimeports.RunResult {
	cut, _ := loopguard.CutOffCall(completion)
	reply := loopguard.CutOffNotice(cut, t.cutOff)
	if visible := strings.TrimSpace(completion.Text); visible != "" {
		reply = visible + "\n\n" + reply
	}
	fx.Emit(modelstream.DeltaEvent(reply))

	final := modelstream.CannedCompletion(completion, reply)
	final.Reasoning = completion.Reasoning
	result := s.finish(t.result, final, fx)
	result.Truncated = true

	return result
}
