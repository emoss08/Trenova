package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxBulkEDIActionItems = 500

type BulkEDIActionFailure struct {
	ID    pulid.ID `json:"id"`
	Error string   `json:"error"`
}

type BulkEDIActionResult struct {
	Succeeded []pulid.ID             `json:"succeeded"`
	Failed    []BulkEDIActionFailure `json:"failed"`
}

func newBulkEDIActionResult(capacity int) *BulkEDIActionResult {
	return &BulkEDIActionResult{
		Succeeded: make([]pulid.ID, 0, capacity),
		Failed:    make([]BulkEDIActionFailure, 0),
	}
}

func ValidateBulkEDIActionIDs(field string, ids []pulid.ID) error {
	if len(ids) == 0 {
		return errortypes.NewValidationError(
			field,
			errortypes.ErrRequired,
			"At least one record must be selected",
		)
	}
	if len(ids) > MaxBulkEDIActionItems {
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"A bulk action may target at most 500 records",
		)
	}
	return nil
}

type BulkRetryMessageDeliveryRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MessageIDs []pulid.ID            `json:"messageIds"`
}

func (s *Service) BulkRetryMessageDelivery(
	ctx context.Context,
	req *BulkRetryMessageDeliveryRequest,
) (*BulkEDIActionResult, error) {
	if err := ValidateBulkEDIActionIDs("messageIds", req.MessageIDs); err != nil {
		return nil, err
	}
	result := newBulkEDIActionResult(len(req.MessageIDs))
	for _, messageID := range req.MessageIDs {
		if _, err := s.RetryMessageDelivery(ctx, &RetryMessageDeliveryRequest{
			MessageID:  messageID,
			TenantInfo: req.TenantInfo,
		}); err != nil {
			result.Failed = append(result.Failed, BulkEDIActionFailure{
				ID:    messageID,
				Error: err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, messageID)
	}
	return result, nil
}

type BulkApproveTransfersRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	TransferIDs []pulid.ID            `json:"transferIds"`
}

func (s *Service) BulkApproveTransfers(
	ctx context.Context,
	req *BulkApproveTransfersRequest,
	actor *services.RequestActor,
) (*BulkEDIActionResult, error) {
	if err := ValidateBulkEDIActionIDs("transferIds", req.TransferIDs); err != nil {
		return nil, err
	}
	result := newBulkEDIActionResult(len(req.TransferIDs))
	for _, transferID := range req.TransferIDs {
		if _, err := s.ApproveTransfer(ctx, &ApproveTransferRequest{
			TransferID: transferID,
			TenantInfo: req.TenantInfo,
		}, actor); err != nil {
			result.Failed = append(result.Failed, BulkEDIActionFailure{
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
	actor *services.RequestActor,
) (*BulkEDIActionResult, error) {
	if err := ValidateBulkEDIActionIDs("transferIds", req.TransferIDs); err != nil {
		return nil, err
	}
	result := newBulkEDIActionResult(len(req.TransferIDs))
	for _, transferID := range req.TransferIDs {
		if _, err := s.RejectTransfer(ctx, &RejectTransferRequest{
			TransferID: transferID,
			TenantInfo: req.TenantInfo,
			Reason:     req.Reason,
		}, actor); err != nil {
			result.Failed = append(result.Failed, BulkEDIActionFailure{
				ID:    transferID,
				Error: err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, transferID)
	}
	return result, nil
}
