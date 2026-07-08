package ediservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) syncActionableTransferMappings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
	excludeIDs ...pulid.ID,
) error {
	partner, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID:         partnerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	transfers, err := s.transferRepo.ListActionableInboundTransfersByPartner(
		ctx,
		repositories.ListActionableInboundEDITransfersByPartnerRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  partner.OrganizationID,
				BuID:   partner.BusinessUnitID,
				UserID: tenantInfo.UserID,
			},
			PartnerID: partnerID,
			Statuses: []edi.TransferStatus{
				edi.TransferStatusMappingRequired,
				edi.TransferStatusPendingApproval,
			},
			ExcludeIDs: excludeIDs,
		},
	)
	if err != nil {
		return err
	}

	for _, transfer := range transfers {
		preview, previewErr := s.buildMappingPreview(ctx, partner, transfer.TenderPayload)
		if previewErr != nil {
			return previewErr
		}

		status := edi.TransferStatusPendingApproval
		if len(preview.Unresolved) > 0 {
			status = edi.TransferStatusMappingRequired
		}
		if status == transfer.Status && slices.Equal(preview.All, transfer.MappingSnapshot) {
			continue
		}

		previousStatus := transfer.Status
		transfer.Status = status
		transfer.MappingSnapshot = preview.All
		if _, err = s.transferRepo.UpdateTransfer(ctx, transfer); err != nil {
			return err
		}

		s.l.Info("reconciled inbound EDI transfer after mapping change",
			zap.String("transferId", transfer.ID.String()),
			zap.String("partnerId", partnerID.String()),
			zap.String("previousStatus", string(previousStatus)),
			zap.String("status", string(status)),
			zap.Int("unresolved", len(preview.Unresolved)),
		)
	}

	return nil
}
