package editransferservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) generateExternalTenderResponse(
	ctx context.Context,
	transfer *editransfer.EDITransfer,
	actorID pulid.ID,
) {
	if transfer == nil || transfer.InboundMessageID.IsNil() {
		return
	}
	tenantInfo := pagination.TenantInfo{
		OrgID:  transfer.TargetOrganizationID,
		BuID:   transfer.TargetBusinessUnitID,
		UserID: actorID,
	}
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	taken, _, err := s.outbound.DecideTender(ctx, &edioutbound.DecideTenderRequest{
		TenantInfo: tenantInfo,
		TransferID: transfer.ID,
		Accept:     transfer.Status == editransfer.TransferStatusApproved,
		Reason:     transfer.RejectionReason,
		ActorID:    actorID,
	})
	if err != nil {
		s.l.Warn(
			"the partner's flow failed to answer the EDI tender",
			zap.String(fieldTransferID, transfer.ID.String()),
			zap.Error(err),
		)
	}
	if taken {
		return
	}
	if _, err = s.outbound.GenerateDocument(
		ctx,
		tenderResponseRequest(tenantInfo, transfer, actorID),
	); err != nil {
		s.l.Warn(
			"failed to generate outbound 990 EDI tender response",
			zap.String(fieldTransferID, transfer.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) ListInboundTransfersCursor(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*editransfer.EDITransfer], error) {
	return s.transferRepo.ListInboundCursor(ctx, req)
}

func (s *Service) ListOutboundTransfersCursor(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*editransfer.EDITransfer], error) {
	return s.transferRepo.ListOutboundCursor(ctx, req)
}
