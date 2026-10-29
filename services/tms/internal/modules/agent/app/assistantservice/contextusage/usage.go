package contextusage

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// KeepContextUsage writes the measure on the thread and on the copy in hand.
func (s *Usage) KeepContextUsage(
	ctx context.Context,
	thread *conversation.Thread,
	usage *conversation.ContextUsage,
	tenant pagination.TenantInfo,
) {
	thread.ContextUsage = usage
	if err := s.Conversations.UpdateThreadContext(
		ctx,
		&conversationports.UpdateThreadContextRequest{
			ThreadID:   thread.ID,
			TenantInfo: tenant,
			Usage:      usage,
		},
	); err != nil {
		s.Logger.Warn("could not keep how full a conversation's context is",
			zap.String("thread", thread.ID.String()),
			zap.Error(err),
		)
	}
}

// ModelHistory is the conversation as its next turn reads it: from the latest
// summary on, without another agent's steps.
func (s *Usage) ModelHistory(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
) ([]conversation.Message, error) {
	return s.Conversations.ListMessages(ctx, &conversationports.ListMessagesRequest{
		ThreadID:        threadID,
		TenantInfo:      tenant,
		Limit:           historyLimit,
		ExcludeKinds:    conversation.ModelHiddenKinds(),
		SinceCompaction: true,
	})
}

type Usage struct {
	Logger        *zap.Logger
	Conversations conversationports.ConversationRepository
}
