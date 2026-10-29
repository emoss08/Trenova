package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func (t *Turn) honoured(spec agentextension.CommandSpec) bool {
	for idx := range t.result.Messages {
		for _, call := range t.result.Messages[idx].ToolCalls {
			if spec.HonouredByCall(call.Name) {
				return true
			}
		}
	}

	return false
}

// holdToCommand sends a reply that would end a command's turn without calling
// any tool that honours it back once, with the command's correction. A second
// such reply stands, so a request the extension cannot serve still ends in an
// answer.
func (s *Service) holdToCommand(
	t *Turn,
	fx TurnEffects,
	completion *llmports.ChatCompletionResult,
	held *bool,
) bool {
	if t.command == "" || *held || completion == nil {
		return false
	}
	spec, specified := agentextension.CommandSpecFor(t.command)
	hold, holds := toolaccess.CommandHolds[t.command]
	if !specified || !holds || t.honoured(spec) {
		return false
	}

	*held = true
	fx.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventRetrying,
		Data: assistantstreamports.AssistantRetryingEvent{
			Attempt: 1,
			Reason:  hold.RetryReason,
			Kind:    llmports.RetryKindRestart,
		},
	})
	t.messages = append(t.messages,
		llmports.Message{
			Role:          llmports.RoleAssistant,
			Content:       completion.Text,
			ReplyID:       completion.ProviderMessageID,
			ReplyProvider: completion.ProviderID,
		},
		llmports.NoticeMessage(hold.Correction),
	)

	return true
}
