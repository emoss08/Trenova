package inboundmessageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/zap"
)

const (
	readNotStartedCode = "read_not_started"
	readNotStartedText = "Nova could not start reading this message, so it is waiting on a person."
)

func (s *Service) ReadUnread(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error) {
	if s.workflows == nil || !s.workflows.Enabled() {
		return 0, errortypes.NewBusinessError(
			"Background work is not running, so Nova cannot read messages right now",
		)
	}

	started := 0
	for started < inboundmessageports.MaxBulkSelection {
		ids, err := s.queueRepo.UnreadInReview(ctx, inboundmessageports.UnreadInReviewRequest{
			TenantInfo: tenantInfo,
			Limit: min(
				inboundmessageports.ReadUnreadBatchSize,
				inboundmessageports.MaxBulkSelection-started,
			),
		})
		if err != nil {
			return started, err
		}
		if len(ids) == 0 {
			break
		}

		result, err := s.runTransition(ctx, &transition{
			tenant: tenantInfo,
			ids:    ids,
			eligible: func(state *inboundmessageports.QueueState) bool {
				return state.Status == inboundmessage.StatusInReview && !state.NovaRead
			},
			patch: inboundmessageports.QueuePatch{
				Status: inboundmessage.StatusProcessing,
				Unpark: true,
				Due:    inboundmessageports.DueClear,
			},
		})
		if err != nil {
			return started, err
		}
		if len(result.changed) == 0 {
			break
		}

		for _, state := range result.changed {
			if s.beginReading(ctx, tenantInfo, state) {
				started++
			}
		}
		if len(ids) < inboundmessageports.ReadUnreadBatchSize {
			break
		}
	}

	return started, nil
}

func (s *Service) beginReading(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	state *inboundmessageports.QueueState,
) bool {
	err := s.startReading(ctx, messageFromState(state), readingWorkflow{
		id:    "inbound-message-read/" + state.ID.String(),
		reuse: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	})
	if err == nil {
		return true
	}

	s.l.Warn("could not start Nova reading an inbound message",
		zap.String("messageId", state.ID.String()), zap.Error(err))
	if markErr := s.MarkFailed(
		ctx, state.ID, tenantInfo, readNotStartedCode, readNotStartedText,
	); markErr != nil {
		s.l.Error("could not put an unread inbound message back in the queue",
			zap.String("messageId", state.ID.String()), zap.Error(markErr))
	}

	return false
}
