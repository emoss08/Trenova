package edijobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	EDIService *ediservice.Service
	Logger     *zap.Logger
}

type Activities struct {
	ediService *ediservice.Service
	logger     *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ediService: p.EDIService,
		logger:     p.Logger.Named("edi-activities"),
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
