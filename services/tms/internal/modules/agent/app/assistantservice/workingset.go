package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/workingset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationworkingset"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (s *Service) keepWorkingSet(
	ctx context.Context,
	turn *workingset.Turn,
	tenant pagination.TenantInfo,
) {
	touched := workingset.TouchedRecords(turn)
	if len(touched) == 0 {
		return
	}

	next := conversationworkingset.Touch(turn.Thread.WorkingSet, touched...)
	if err := s.conversations.UpdateThreadContext(ctx, conversationports.UpdateThreadContextRequest{
		ThreadID:   turn.Thread.ID,
		TenantInfo: tenant,
		WorkingSet: &next,
	}); err != nil {
		s.logger.Warn("could not keep the records a conversation is about",
			zap.String("thread", turn.Thread.ID.String()),
			zap.Error(err),
		)
		return
	}
	turn.Thread.WorkingSet = next
}

func (s *Service) anchorRecords(
	ctx context.Context,
	scope *workingset.AnchorScope,
) []agentturn.RuntimeAnchor {
	if s.anchors == nil {
		return nil
	}
	refs := workingset.AnchorRefs(scope)
	if len(refs) == 0 {
		return nil
	}

	return s.anchors.Anchor(ctx, &agentports.RecordAnchorRequest{
		TenantInfo: scope.Tenant,
		Actor:      scope.Actor,
		Records:    refs,
	})
}
