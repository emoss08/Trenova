package shipmentimportassistantservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

// maxReplayedTurns bounds how much of the conversation is replayed.
//
// The exchange used to live on the provider's side, carried by a response id,
// and it grew without limit because nothing local held it. Sending it ourselves
// makes the length our problem: an import conversation that has run for fifty
// turns would spend its whole budget on its own history, and the turns that
// matter to the model are the recent ones.
const maxReplayedTurns = 12

// buildMessages reconstructs the exchange from what we stored.
//
// It replaces OpenAI's PreviousResponseID, which kept the conversation on the
// vendor's side and made moving providers impossible: a handle issued by one
// endpoint means nothing to another, and an organization that switched provider
// mid-import would have lost the thread. Every turn is already persisted for
// the history panel, so replaying it costs a read we were making anyway.
func (s *Service) buildMessages(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	conversation *shipmentimportchat.Conversation,
) []serviceports.Message {
	messages := make([]serviceports.Message, 0, maxReplayedTurns*2+1)
	messages = append(messages, s.replayTurns(ctx, req.TenantInfo, conversation)...)

	return append(messages, serviceports.Message{
		Role:    serviceports.RoleUser,
		Content: req.UserMessage,
	})
}

func (s *Service) replayTurns(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	conversation *shipmentimportchat.Conversation,
) []serviceports.Message {
	if conversation == nil || conversation.ID.IsNil() {
		return nil
	}

	snapshot, err := s.getHistorySnapshot(ctx, conversation.DocumentID, tenantInfo)
	if err != nil || snapshot == nil {
		// A conversation that cannot be replayed is still answerable — the
		// current state travels in the system prompt either way. Losing the
		// history degrades the reply; failing the turn loses it entirely.
		if err != nil {
			s.logger.Warn("failed to replay import conversation", zap.Error(err))
		}

		return nil
	}

	history := snapshot.Messages
	if len(history) > maxReplayedTurns*2 {
		history = history[len(history)-maxReplayedTurns*2:]
	}

	messages := make([]serviceports.Message, 0, len(history))
	for i := range history {
		message := &history[i]
		if strings.TrimSpace(message.Text) == "" {
			continue
		}

		role := serviceports.RoleAssistant
		if message.Role == "user" {
			role = serviceports.RoleUser
		}
		messages = append(messages, serviceports.Message{Role: role, Content: message.Text})
	}

	return messages
}

// toolRound appends what a round of tool calls did, in the shape the next call
// needs: the assistant turn that asked, then one result per call.
func toolRound(
	calls []serviceports.ToolCall,
	results map[string]toolResult,
) []serviceports.Message {
	messages := make([]serviceports.Message, 0, len(calls)+1)
	messages = append(messages, serviceports.Message{
		Role:      serviceports.RoleAssistant,
		ToolCalls: calls,
	})

	for _, call := range calls {
		result := results[call.ID]
		messages = append(messages, serviceports.Message{
			Role:       serviceports.RoleTool,
			ToolCallID: call.ID,
			ToolName:   call.Name,
			Content:    result.output,
			IsError:    result.status == toolStatusError,
		})
	}

	return messages
}

type toolResult struct {
	output string
	status string
}

const (
	toolStatusCompleted = "completed"
	toolStatusError     = "error"
)

// conversationHandle is what the client sends back as ConversationID. It used
// to be the provider's response id; it is now our own conversation, because the
// thread is ours and a provider handle would stop meaning anything the moment
// the organization changed provider.
func conversationHandle(conversation *shipmentimportchat.Conversation) string {
	if conversation == nil || conversation.ID.IsNil() {
		return ""
	}

	return conversation.ID.String()
}
