package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/contextusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// measureAfterTurn measures the conversation's context once a turn is saved,
// keeps the measure on the thread, and says so on emit. It is best effort: a
// meter that missed a turn is corrected by the next, and a reply already
// saved is not failed for it.
func (s *Service) measureAfterTurn(
	ctx context.Context,
	thread *conversation.Thread,
	plan *TurnPlan,
	saved []conversation.Message,
	window int,
	tenant pagination.TenantInfo,
	emit kernelports.AssistantStreamEmitter,
) {
	history, err := s.usage().ModelHistory(ctx, thread.ID, tenant)
	if err != nil {
		s.logger.Warn("could not read the conversation to measure its context",
			zap.String("thread", thread.ID.String()),
			zap.Error(err),
		)
		return
	}

	answering := contextusage.AnsweringModel(saved)
	usage := turnhistory.MeasureContext(turnhistory.ContextRequest{
		System:       plan.Turn.System,
		Tools:        plan.Turn.Tools.Specs,
		Instructions: contextusage.LastInstructions(thread),
		History:      history,
		Proposals:    plan.Proposals,
		Model:        contextusage.UsageModel(thread, answering),
		Window:       contextusage.UsageWindow(thread, answering, window),
		Now:          timeutils.NowUnix(),
	})
	s.usage().KeepContextUsage(ctx, thread, &usage, tenant)

	if emit != nil {
		emit(kernelports.StreamEvent{
			Event: assistantstreamports.AssistantEventContext,
			Data: assistantstreamports.AssistantContextEvent{
				ThreadID:       thread.ID,
				Usage:          &usage,
				AutoCompactOff: thread.AutoCompactOff,
			},
		})
	}
}
