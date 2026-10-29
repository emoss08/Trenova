package edijobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundpoll"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferservice"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	TransferService   *editransferservice.Service
	DeliveryService   *edidelivery.Service
	OutboundService   *edioutbound.Service
	Replayer          *edilading.Replayer
	InboundService    *ediinboundservice.Service
	PollService       *ediinboundpoll.Service
	DataRetentionRepo tenantports.DataRetentionRepository
	InboundFileRepo   edimessageports.EDIInboundFileRepository
	MessageRepo       edimessageports.EDIMessageRepository
	Logger            *zap.Logger
}

type Activities struct {
	transferService   *editransferservice.Service
	deliveryService   *edidelivery.Service
	outboundService   *edioutbound.Service
	replayer          *edilading.Replayer
	inboundService    *ediinboundservice.Service
	pollService       *ediinboundpoll.Service
	dataRetentionRepo tenantports.DataRetentionRepository
	inboundFileRepo   edimessageports.EDIInboundFileRepository
	messageRepo       edimessageports.EDIMessageRepository
	logger            *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		transferService:   p.TransferService,
		deliveryService:   p.DeliveryService,
		outboundService:   p.OutboundService,
		replayer:          p.Replayer,
		inboundService:    p.InboundService,
		pollService:       p.PollService,
		dataRetentionRepo: p.DataRetentionRepo,
		inboundFileRepo:   p.InboundFileRepo,
		messageRepo:       p.MessageRepo,
		logger:            p.Logger.Named("edi-activities"),
	}
}

func (a *Activities) ApproveLoadTenderTransferActivity(
	ctx context.Context,
	payload *ApproveLoadTenderTransferWorkflowPayload,
) (*ApproveLoadTenderTransferWorkflowResult, error) {
	result, err := a.transferService.ProcessLoadTenderApproval(ctx, payload)
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
	result, err := a.deliveryService.DeliverMessage(ctx, payload)
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
	if err := a.deliveryService.MarkMessageDeadLettered(ctx, payload); err != nil {
		a.logger.Error("EDI message dead-letter activity failed", zap.Error(err))
		return err
	}

	a.logger.Warn(
		"EDI message dead-lettered after exhausted delivery retries",
		zap.String("messageId", payload.MessageID.String()),
	)
	return nil
}
