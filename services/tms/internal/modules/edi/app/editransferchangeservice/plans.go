package editransferchangeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) CheckTransferChangeReview(
	ctx context.Context,
	change *edisync.TransferChange,
	tenantInfo pagination.TenantInfo,
) error {
	if change == nil || change.Status != edisync.TransferChangeStatusPendingReview {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer change has already been reviewed",
		)
	}

	return s.validateTransferChangeReviewer(ctx, change, tenantInfo)
}

type TransferChangeEffect struct {
	Shipment   *shipment.Shipment
	NextStatus shipmentvocab.Status
}

func (s *Service) PlanTransferChangeEffect(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	change *edisync.TransferChange,
) (*TransferChangeEffect, error) {
	switch change.ChangeType {
	case edisync.TransferChangeTypeShipmentStatus214, edisync.TransferChangeTypeShipmentCancel214:
	default:
		return &TransferChangeEffect{}, nil
	}

	applyCtx, err := s.buildApprovedTransferChangeContext(ctx, tenantInfo, change)
	if err != nil {
		return nil, err
	}

	return &TransferChangeEffect{Shipment: applyCtx.opposite, NextStatus: applyCtx.nextStatus}, nil
}
