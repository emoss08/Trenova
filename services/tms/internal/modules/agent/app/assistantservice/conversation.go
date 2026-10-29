package assistantservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"go.uber.org/zap"
)

// titleIfUnnamed names a thread from its first message so the list is readable
// without asking the model for a summary.
func (s *Service) titleIfUnnamed(
	ctx context.Context,
	thread *conversation.Thread,
	firstMessage string,
) {
	if strings.TrimSpace(thread.Title) != "" {
		return
	}

	thread.Title = conversation.TitleFrom(firstMessage)
	if _, err := s.conversations.UpdateThread(ctx, thread); err != nil {
		// A missing title is cosmetic, so it must not fail a turn that otherwise
		// succeeded and is already saved.
		s.logger.Warn("failed to title thread", zap.Error(err))
	}
}
