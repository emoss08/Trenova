package editransferservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edibulk"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BulkApproveTransfersRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	TransferIDs []pulid.ID            `json:"transferIds"`
}

func (s *Service) BulkApproveTransfers(
	ctx context.Context,
	req *BulkApproveTransfersRequest,
	actor *permissionports.RequestActor,
) (*edibulk.Result, error) {
	if err := edibulk.ValidateIDs("transferIds", req.TransferIDs); err != nil {
		return nil, err
	}
	result := edibulk.NewResult(len(req.TransferIDs))
	for _, transferID := range req.TransferIDs {
		if _, err := s.ApproveTransfer(ctx, &ApproveTransferRequest{
			TransferID: transferID,
			TenantInfo: req.TenantInfo,
		}, actor); err != nil {
			result.Failed = append(result.Failed, edibulk.Failure{
				ID:    transferID,
				Error: err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, transferID)
	}
	return result, nil
}

type BulkRejectTransfersRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	TransferIDs []pulid.ID            `json:"transferIds"`
	Reason      string                `json:"reason"`
}

func (s *Service) BulkRejectTransfers(
	ctx context.Context,
	req *BulkRejectTransfersRequest,
	actor *permissionports.RequestActor,
) (*edibulk.Result, error) {
	if err := edibulk.ValidateIDs("transferIds", req.TransferIDs); err != nil {
		return nil, err
	}
	result := edibulk.NewResult(len(req.TransferIDs))
	for _, transferID := range req.TransferIDs {
		if _, err := s.RejectTransfer(ctx, &RejectTransferRequest{
			TransferID: transferID,
			TenantInfo: req.TenantInfo,
			Reason:     req.Reason,
		}, actor); err != nil {
			result.Failed = append(result.Failed, edibulk.Failure{
				ID:    transferID,
				Error: err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, transferID)
	}
	return result, nil
}
