package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func (s *Service) reground(
	t *Turn,
	fx TurnEffects,
	completion *llmports.ChatCompletionResult,
	state *replycheck.GroundingState,
) bool {
	if completion == nil {
		return false
	}

	finding, found := replycheck.Drift(t, s.access.SpecFor, completion.Text)
	if !found {
		return false
	}

	if !state.Rewritten {
		state.Rewritten = true
		reason := finding.RetryReason()
		fx.Emit(kernelports.StreamEvent{
			Event: assistantstreamports.AssistantEventRetrying,
			Data: assistantstreamports.AssistantRetryingEvent{
				Attempt: 1,
				Reason:  reason,
				Kind:    llmports.RetryKindRestart,
			},
		})
		fx.Emit(replycheck.RegroundedEvent(assistantstreamports.RegroundRewrite, finding, reason))
		t.messages = append(t.messages,
			llmports.Message{
				Role:          llmports.RoleAssistant,
				Content:       completion.Text,
				ReplyID:       completion.ProviderMessageID,
				ReplyProvider: completion.ProviderID,
			},
			llmports.NoticeMessage(finding.Correction()),
		)

		return true
	}

	note := finding.Note()
	fx.Emit(
		replycheck.RegroundedEvent(
			assistantstreamports.RegroundNote,
			finding,
			finding.NoteReason(),
		),
	)
	completion.Text += note
	fx.Emit(modelstream.DeltaEvent(note))

	return false
}
