package edijobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	EDIService     *ediservice.Service
	InboundService *ediinboundservice.Service
	Logger         *zap.Logger
}

type Activities struct {
	ediService     *ediservice.Service
	inboundService *ediinboundservice.Service
	logger         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ediService:     p.EDIService,
		inboundService: p.InboundService,
		logger:         p.Logger.Named("edi-activities"),
	}
}

func (a *Activities) ApproveLoadTenderTransferActivity(
	ctx context.Context,
	payload *ApproveLoadTenderTransferWorkflowPayload,
) (*ApproveLoadTenderTransferWorkflowResult, error) {
	result, err := a.ediService.ProcessLoadTenderApproval(ctx, payload)
	if err != nil {
		a.logger.Error("EDI load tender approval activity failed", zap.Error(err))
		return nil, err
	}

	a.logger.Info(
		"EDI load tender approval activity completed",
		zap.String("transferId", result.TransferID.String()),
		zap.String("targetShipmentId", result.TargetShipmentID.String()),
	)
	return result, nil
}

func (a *Activities) DeliverEDIMessageActivity(
	ctx context.Context,
	payload *DeliverEDIMessageWorkflowPayload,
) (*DeliverEDIMessageWorkflowResult, error) {
	result, err := a.ediService.DeliverMessage(ctx, payload)
	if err != nil {
		a.logger.Error("EDI message delivery activity failed", zap.Error(err))
		return nil, err
	}

	a.logger.Info(
		"EDI message delivery activity completed",
		zap.String("messageId", result.MessageID.String()),
		zap.String("remotePath", result.RemotePath),
	)
	return result, nil
}

func (a *Activities) MarkEDIMessageDeadLetteredActivity(
	ctx context.Context,
	payload *MarkEDIMessageDeadLetteredPayload,
) error {
	if err := a.ediService.MarkMessageDeadLettered(ctx, payload); err != nil {
		a.logger.Error("EDI message dead-letter activity failed", zap.Error(err))
		return err
	}

	a.logger.Warn(
		"EDI message dead-lettered after exhausted delivery retries",
		zap.String("messageId", payload.MessageID.String()),
	)
	return nil
}
