package replycheck

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type SendBack struct {
	Events   []kernelports.StreamEvent
	Messages []llmports.Message
}

func SendBackDraft(
	draft *llmports.ChatCompletionResult,
	reason string,
	pass kernelports.StreamEvent,
	correction string,
) SendBack {
	return SendBack{
		Events: []kernelports.StreamEvent{
			{
				Event: assistantstreamports.AssistantEventRetrying,
				Data: assistantstreamports.AssistantRetryingEvent{
					Attempt: 1,
					Reason:  reason,
					Kind:    llmports.RetryKindRestart,
				},
			},
			pass,
		},
		Messages: []llmports.Message{
			{
				Role:          llmports.RoleAssistant,
				Content:       draft.Text,
				ReplyID:       draft.ProviderMessageID,
				ReplyProvider: draft.ProviderID,
			},
			llmports.NoticeMessage(correction),
		},
	}
}
