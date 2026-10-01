package ediservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

func (s *Service) ListTransferChanges(
	ctx context.Context,
	req *repositories.ListEDITransferChangesRequest,
) (*pagination.ListResult[*edi.TransferChange], error) {
	return s.transferChangeRepo.ListTransferChanges(ctx, req)
}

func (s *Service) GetTransferChange(
	ctx context.Context,
	req repositories.GetEDITransferChangeByIDRequest,
) (*edi.TransferChange, error) {
	return s.transferChangeRepo.GetTransferChangeByID(ctx, req)
}

func (s *Service) ApplyTransferChange(
	ctx context.Context,
	req *TransferChangeActionRequest,
	actor *services.RequestActor,
) (*edi.TransferChange, error) {
	return s.reviewTransferChange(ctx, req, actor, edi.TransferChangeStatusApplied)
}

func (s *Service) RejectTransferChange(
	ctx context.Context,
	req *TransferChangeActionRequest,
	actor *services.RequestActor,
) (*edi.TransferChange, error) {
	return s.reviewTransferChange(ctx, req, actor, edi.TransferChangeStatusRejected)
}

func (s *Service) reviewTransferChange(
	ctx context.Context,
	req *TransferChangeActionRequest,
	actor *services.RequestActor,
	status edi.TransferChangeStatus,
) (*edi.TransferChange, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"userId",
			errortypes.ErrRequired,
			"Reviewing user is required",
		)
	}

	change, err := s.transferChangeRepo.GetTransferChangeByID(
		ctx,
		repositories.GetEDITransferChangeByIDRequest{
			ID:         req.ChangeID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if err = s.CheckTransferChangeReview(ctx, change, req.TenantInfo); err != nil {
		return nil, err
	}

	var updated *edi.TransferChange
	now := timeutils.NowUnix()
	original := *change
	reviewCtx := dbscope.WithSystem(
		ctx,
		"apply a reviewed transfer change to the linked shipments of both organizations in one transaction",
	)
	err = s.db.WithTx(reviewCtx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if status == edi.TransferChangeStatusApplied {
			if txErr := s.applyApprovedTransferChange(
				txCtx,
				req.TenantInfo,
				change,
				actor,
				now,
			); txErr != nil {
				return txErr
			}
		}

		MarkTransferChangeReviewed(change, &ChangeReviewMark{
			Applied:    status == edi.TransferChangeStatusApplied,
			ReviewerID: actor.UserID,
			Reason:     req.Reason,
			At:         now,
		})

		var txErr error
		updated, txErr = s.transferChangeRepo.UpdateTransferChange(txCtx, change)
		if txErr != nil {
			return txErr
		}

		return s.commentTransferChangeReview(txCtx, req.TenantInfo, updated)
	})
	if err != nil {
		return nil, err
	}

	s.logAction(
		updated,
		actor,
		permission.OpUpdate,
		&original,
		updated,
		"EDI transfer change reviewed",
	)
	return updated, nil
}

func (s *Service) validateTransferChangeReviewer(
	ctx context.Context,
	change *edi.TransferChange,
	tenantInfo pagination.TenantInfo,
) error {
	link, err := s.shipmentLinkRepo.GetShipmentLinkByID(
		ctx,
		repositories.GetEDIShipmentLinkByIDRequest{
			ID:         change.ShipmentLinkID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return err
	}
	reviewerOrgID := link.TargetOrganizationID
	if change.Direction == edi.TransferChangeDirectionTargetToSource {
		reviewerOrgID = link.SourceOrganizationID
	}
	if tenantInfo.OrgID != reviewerOrgID || tenantInfo.BuID != link.BusinessUnitID {
		return errortypes.NewValidationError(
			"tenantInfo",
			errortypes.ErrInvalidOperation,
			"Only the organization receiving this EDI transfer change can review it",
		)
	}
	return nil
}

func (s *Service) commentTransferChangeReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	change *edi.TransferChange,
) error {
	link, err := s.shipmentLinkRepo.GetShipmentLinkByID(
		ctx,
		repositories.GetEDIShipmentLinkByIDRequest{
			ID:         change.ShipmentLinkID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return err
	}

	comment := fmt.Sprintf("EDI transfer change %s was %s.", change.ChangeType, change.Status)
	metadata := map[string]any{
		"shipmentLinkId":   change.ShipmentLinkID,
		"transferChangeId": change.ID,
		"changeType":       change.ChangeType,
		"status":           change.Status,
	}

	sourceTenant := pagination.TenantInfo{
		OrgID: link.SourceOrganizationID,
		BuID:  link.BusinessUnitID,
	}
	if err = s.createSystemShipmentComment(
		ctx,
		link.SourceShipmentID,
		sourceTenant,
		comment,
		metadata,
	); err != nil {
		return err
	}

	targetTenant := pagination.TenantInfo{
		OrgID: link.TargetOrganizationID,
		BuID:  link.BusinessUnitID,
	}
	return s.createSystemShipmentComment(
		ctx,
		link.TargetShipmentID,
		targetTenant,
		comment,
		metadata,
	)
}
