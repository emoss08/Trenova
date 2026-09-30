package ediservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

func (s *Service) MappingPreview(
	ctx context.Context,
	req repositories.GetEDITransferByIDRequest,
) (*MappingPreview, error) {
	transfer, err := s.transferRepo.GetTransferByID(ctx, req)
	if err != nil {
		return nil, err
	}

	targetPartner, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID: transfer.TargetPartnerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: transfer.TargetOrganizationID,
			BuID:  transfer.TargetBusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	return s.buildMappingPreview(ctx, targetPartner, transfer.TenderPayload)
}

//nolint:funlen // Approval coordinates validation, mapping, shipment creation, and transfer updates atomically.
func (s *Service) ApproveTransfer(
	ctx context.Context,
	req *ApproveTransferRequest,
	actor *services.RequestActor,
) (*edi.EDITransfer, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"approver",
			errortypes.ErrRequired,
			"Approving user is required",
		)
	}

	var original *edi.EDITransfer
	var updated *edi.EDITransfer
	var preview *MappingPreview
	err := s.db.WithTx(ctx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			repositories.GetEDITransferForUpdateRequest{
				ID:         req.TransferID,
				TenantInfo: req.TenantInfo,
				Direction:  "inbound",
			},
		)
		if err != nil {
			return err
		}
		if err = RequireActionableTransfer(transfer, TransferVerbApproved); err != nil {
			return err
		}
		originalCopy := *transfer
		original = &originalCopy

		if len(req.Mappings) > 0 {
			if _, err = s.SaveMappingProfile(txCtx, &repositories.SaveMappingItemsRequest{
				PartnerID:  transfer.TargetPartnerID,
				TenantInfo: req.TenantInfo,
				ActorID:    actor.UserID,
				Items:      req.Mappings,
			}); err != nil {
				return err
			}
		}

		targetPartner, err := s.partnerRepo.GetByID(txCtx, repositories.GetEDIPartnerByIDRequest{
			ID:         transfer.TargetPartnerID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return err
		}

		preview, err = s.buildMappingPreview(txCtx, targetPartner, transfer.TenderPayload)
		if err != nil {
			return err
		}
		if len(preview.Unresolved) > 0 {
			return unresolvedMappingsError(preview.Unresolved)
		}

		MarkTransferApprovalStarted(transfer, &TransferApprovalMark{
			BusinessUnitID: req.TenantInfo.BuID,
			Mapping:        preview.All,
			ApproverID:     actor.UserID,
			At:             timeutils.NowUnix(),
		})

		updated, err = s.transferRepo.UpdateTransfer(txCtx, transfer)
		return err
	})
	if err != nil {
		return nil, err
	}

	runID, err := s.startApprovalWorkflow(ctx, updated, req.TenantInfo, actor)
	if err != nil {
		s.restoreTransferAfterWorkflowStartFailure(ctx, updated, preview)
		return nil, err
	}

	if persisted, updateErr := s.transferRepo.SetApprovalWorkflowRunID(
		ctx,
		repositories.SetEDITransferApprovalWorkflowRunIDRequest{
			ID:         updated.ID,
			TenantInfo: req.TenantInfo,
			RunID:      runID,
		},
	); updateErr == nil {
		updated = persisted
	} else {
		return nil, updateErr
	}

	s.logAction(
		updated,
		actor,
		permission.OpUpdate,
		original,
		updated,
		"EDI load tender approval started",
	)
	return updated, nil
}

func (s *Service) startApprovalWorkflow(
	ctx context.Context,
	transfer *edi.EDITransfer,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) (string, error) {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return "", errortypes.NewBusinessError("EDI approval workflow is not configured")
	}

	payload := &ApproveLoadTenderTransferWorkflowPayload{
		TransferID: transfer.ID,
		TenantInfo: tenantInfo,
		Actor:      actor,
	}

	run, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       transfer.ApprovalWorkflowID,
			TaskQueue:                                temporaltype.EDITaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary: fmt.Sprintf(
				"Approve EDI load tender transfer %s",
				transfer.ID.String(),
			),
		},
		temporaltype.ApproveLoadTenderTransferWorkflowName,
		payload,
	)
	if err != nil {
		var alreadyStartedErr *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStartedErr) {
			return alreadyStartedErr.RunId, nil
		}

		return "", errortypes.NewBusinessError("failed to start EDI approval workflow").
			WithInternal(err)
	}

	return run.GetRunID(), nil
}

func (s *Service) restoreTransferAfterWorkflowStartFailure(
	ctx context.Context,
	transfer *edi.EDITransfer,
	preview *MappingPreview,
) {
	if transfer == nil {
		return
	}

	transfer.Status = edi.TransferStatusPendingApproval
	if preview != nil && len(preview.Unresolved) > 0 {
		transfer.Status = edi.TransferStatusMappingRequired
	}
	transfer.ApprovedByID = pulid.Nil
	transfer.ApprovedAt = nil
	transfer.ApprovalWorkflowID = ""
	transfer.ApprovalWorkflowRunID = ""
	transfer.ProcessingStartedAt = nil
	if _, err := s.transferRepo.UpdateTransfer(ctx, transfer); err != nil {
		s.l.Warn(
			"failed to restore EDI load tender transfer after workflow start failure",
			zap.Error(err),
		)
	}
}

func buildLoadTenderApprovalWorkflowID(transferID pulid.ID) string {
	return "edi-load-tender-approve-" + transferID.String()
}

//nolint:cyclop,funlen,gocognit // Temporal approval processing mirrors the workflow states explicitly.
func (s *Service) ProcessLoadTenderApproval(
	ctx context.Context,
	payload *ApproveLoadTenderTransferWorkflowPayload,
) (*ApproveLoadTenderTransferWorkflowResult, error) {
	result := new(ApproveLoadTenderTransferWorkflowResult)
	var processingErr error
	var approvedTransfer *edi.EDITransfer

	err := s.db.WithTx(ctx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			repositories.GetEDITransferForUpdateRequest{
				ID:         payload.TransferID,
				TenantInfo: payload.TenantInfo,
				Direction:  "inbound",
			},
		)
		if err != nil {
			return err
		}

		if transfer.Status == edi.TransferStatusApproved && transfer.TargetShipmentID.IsNotNil() {
			processedAt := timeutils.NowUnix()
			if transfer.ProcessedAt != nil {
				processedAt = *transfer.ProcessedAt
			}
			result.TransferID = transfer.ID
			result.TargetShipmentID = transfer.TargetShipmentID
			result.ProcessedAt = processedAt
			return nil
		}

		//nolint:exhaustive // Only terminal and processing states require special approval handling.
		switch transfer.Status {
		case edi.TransferStatusRejected,
			edi.TransferStatusExpired,
			edi.TransferStatusCanceled,
			edi.TransferStatusFailed:
			processingErr = temporal.NewNonRetryableApplicationError(
				"EDI transfer is no longer approval eligible",
				"TransferNotApprovalEligible",
				nil,
			)
			return nil
		case edi.TransferStatusProcessing:
		default:
			processingErr = temporal.NewNonRetryableApplicationError(
				"EDI transfer is not processing approval",
				"TransferNotProcessing",
				nil,
			)
			return nil
		}

		targetPartner, err := s.partnerRepo.GetByID(txCtx, repositories.GetEDIPartnerByIDRequest{
			ID: transfer.TargetPartnerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: transfer.TargetOrganizationID,
				BuID:  transfer.TargetBusinessUnitID,
			},
		})
		if err != nil {
			return err
		}

		preview, err := s.buildMappingPreview(txCtx, targetPartner, transfer.TenderPayload)
		if err != nil {
			return err
		}
		if len(preview.Unresolved) > 0 {
			now := timeutils.NowUnix()
			transfer.Status = edi.TransferStatusMappingRequired
			transfer.MappingSnapshot = preview.All
			transfer.ProcessedAt = &now
			if _, err = s.transferRepo.UpdateTransfer(txCtx, transfer); err != nil {
				return err
			}
			processingErr = temporal.NewNonRetryableApplicationError(
				"EDI mappings are no longer complete",
				"UnresolvedMappings",
				unresolvedMappingsError(preview.Unresolved),
			)
			return nil
		}

		targetShipment, err := s.buildTargetShipment(
			transfer,
			payload.TenantInfo.BuID,
			preview.All,
			payload.Actor.UserID,
		)
		if err != nil {
			processingErr = err
			return s.failTransferInTransaction(txCtx, transfer, err)
		}

		createdShipment, err := s.shipmentSvc.Create(txCtx, targetShipment, payload.Actor)
		if err != nil {
			processingErr = err
			return s.failTransferInTransaction(txCtx, transfer, err)
		}

		now := timeutils.NowUnix()
		transfer.Status = edi.TransferStatusApproved
		transfer.TargetShipmentID = createdShipment.ID
		transfer.MappingSnapshot = preview.All
		transfer.ProcessedAt = &now

		if transfer.InboundMessageID.IsNotNil() {
			updated, updateErr := s.transferRepo.UpdateTransfer(txCtx, transfer)
			if updateErr != nil {
				return updateErr
			}
			approvedTransfer = updated
			if commentErr := s.createSystemShipmentComment(
				txCtx,
				createdShipment.ID,
				pagination.TenantInfo{
					OrgID: transfer.TargetOrganizationID,
					BuID:  transfer.TargetBusinessUnitID,
				},
				"Shipment created from accepted external EDI load tender.",
				map[string]any{"transferId": transfer.ID},
			); commentErr != nil {
				return commentErr
			}
			result.TransferID = updated.ID
			result.TargetShipmentID = updated.TargetShipmentID
			result.ProcessedAt = now
			return nil
		}

		if err = s.setShipmentTenderStatus(
			txCtx,
			transfer.SourceShipmentID,
			pagination.TenantInfo{
				OrgID: transfer.SourceOrganizationID,
				BuID:  transfer.SourceBusinessUnitID,
			},
			shipment.TenderStatusAccepted,
		); err != nil {
			return err
		}

		updated, err := s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
			return err
		}

		link, err := s.shipmentLinkRepo.CreateShipmentLink(txCtx, &edi.ShipmentLink{
			BusinessUnitID:       transfer.SourceBusinessUnitID,
			SourceOrganizationID: transfer.SourceOrganizationID,
			TargetOrganizationID: transfer.TargetOrganizationID,
			SourceShipmentID:     transfer.SourceShipmentID,
			TargetShipmentID:     createdShipment.ID,
			TenderTransferID:     transfer.ID,
			SyncPolicy:           edi.ShipmentSyncPolicyAutoOperational,
			FieldOwnership:       edi.DefaultShipmentFieldOwnership(),
			Status:               edi.ShipmentLinkStatusActive,
		})
		if err != nil {
			return err
		}
		if err = s.upsertInternalTenderRecipient(
			txCtx,
			transfer,
			link,
			edi.TenderRecipientBaselineStatusAccepted,
		); err != nil {
			return err
		}

		sourceTenant := pagination.TenantInfo{
			OrgID: transfer.SourceOrganizationID,
			BuID:  transfer.SourceBusinessUnitID,
		}
		targetTenant := pagination.TenantInfo{
			OrgID: transfer.TargetOrganizationID,
			BuID:  transfer.TargetBusinessUnitID,
		}
		if err = s.createSystemShipmentComment(
			txCtx,
			transfer.SourceShipmentID,
			sourceTenant,
			"EDI load tender accepted by receiving organization.",
			map[string]any{"transferId": transfer.ID, "shipmentLinkId": link.ID},
		); err != nil {
			return err
		}
		if err = s.createSystemShipmentComment(
			txCtx,
			createdShipment.ID,
			targetTenant,
			"Shipment created from accepted EDI load tender.",
			map[string]any{"transferId": transfer.ID, "shipmentLinkId": link.ID},
		); err != nil {
			return err
		}

		result.TransferID = updated.ID
		result.TargetShipmentID = updated.TargetShipmentID
		result.ProcessedAt = now
		return nil
	})
	if err != nil {
		return nil, err
	}
	if processingErr != nil {
		return nil, processingErr
	}
	if approvedTransfer != nil {
		s.generateExternalTenderResponse(ctx, approvedTransfer, payload.Actor.UserID)
	}

	return result, nil
}

func (s *Service) failTransferInTransaction(
	ctx context.Context,
	transfer *edi.EDITransfer,
	err error,
) error {
	now := timeutils.NowUnix()
	transfer.Status = edi.TransferStatusFailed
	transfer.FailureReason = err.Error()
	transfer.ProcessedAt = &now
	if _, updateErr := s.transferRepo.UpdateTransfer(ctx, transfer); updateErr != nil {
		return updateErr
	}
	return nil
}

func (s *Service) RejectTransfer(
	ctx context.Context,
	req *RejectTransferRequest,
	actor *services.RequestActor,
) (*edi.EDITransfer, error) {
	reason, err := TransferRejectionReason(req.Reason)
	if err != nil {
		return nil, err
	}

	var original edi.EDITransfer
	var updated *edi.EDITransfer
	err = s.db.WithTx(ctx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			repositories.GetEDITransferForUpdateRequest{
				ID:         req.TransferID,
				TenantInfo: req.TenantInfo,
				Direction:  "inbound",
			},
		)
		if err != nil {
			return err
		}
		if err = RequireActionableTransfer(transfer, TransferVerbRejected); err != nil {
			return err
		}

		original = *transfer
		MarkTransferRejected(transfer, reason, actor.UserID, timeutils.NowUnix())

		updated, err = s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
			return err
		}

		if transfer.SourceShipmentID.IsNil() {
			return nil
		}

		if err = s.setShipmentTenderStatus(
			txCtx,
			transfer.SourceShipmentID,
			pagination.TenantInfo{
				OrgID: transfer.SourceOrganizationID,
				BuID:  transfer.SourceBusinessUnitID,
			},
			shipment.TenderStatusRejected,
		); err != nil {
			return err
		}

		return s.createSystemShipmentComment(
			txCtx,
			transfer.SourceShipmentID,
			pagination.TenantInfo{
				OrgID: transfer.SourceOrganizationID,
				BuID:  transfer.SourceBusinessUnitID,
			},
			"EDI load tender rejected: "+reason,
			map[string]any{"transferId": transfer.ID},
		)
	})
	if err != nil {
		return nil, err
	}

	if updated.InboundMessageID.IsNotNil() {
		s.generateExternalTenderResponse(ctx, updated, actor.UserID)
	}
	s.logAction(updated, actor, permission.OpUpdate, &original, updated, "EDI load tender rejected")
	return updated, nil
}

func (s *Service) CancelTransfer(
	ctx context.Context,
	req *CancelTransferRequest,
	actor *services.RequestActor,
) (*edi.EDITransfer, error) {
	var original edi.EDITransfer
	var updated *edi.EDITransfer
	err := s.db.WithTx(ctx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			repositories.GetEDITransferForUpdateRequest{
				ID:         req.TransferID,
				TenantInfo: req.TenantInfo,
				Direction:  "outbound",
			},
		)
		if err != nil {
			return err
		}
		if err = RequireActionableTransfer(transfer, TransferVerbCanceled); err != nil {
			return err
		}

		original = *transfer
		MarkTransferCanceled(transfer, actor.UserID, timeutils.NowUnix())

		updated, err = s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
			return err
		}

		if err = s.setShipmentTenderStatus(
			txCtx,
			transfer.SourceShipmentID,
			pagination.TenantInfo{
				OrgID: transfer.SourceOrganizationID,
				BuID:  transfer.SourceBusinessUnitID,
			},
			shipment.TenderStatusCanceled,
		); err != nil {
			return err
		}

		return s.createSystemShipmentComment(
			txCtx,
			transfer.SourceShipmentID,
			pagination.TenantInfo{
				OrgID: transfer.SourceOrganizationID,
				BuID:  transfer.SourceBusinessUnitID,
			},
			"EDI load tender canceled.",
			map[string]any{"transferId": transfer.ID},
		)
	})
	if err != nil {
		return nil, err
	}

	s.logAction(updated, actor, permission.OpUpdate, &original, updated, "EDI load tender canceled")
	return updated, nil
}

func (s *Service) ExpireTransfer(
	ctx context.Context,
	req *ExpireTransferRequest,
	actor *services.RequestActor,
) (*edi.EDITransfer, error) {
	var original edi.EDITransfer
	var updated *edi.EDITransfer
	err := s.db.WithTx(ctx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			repositories.GetEDITransferForUpdateRequest{
				ID:         req.TransferID,
				TenantInfo: req.TenantInfo,
				Direction:  "",
			},
		)
		if err != nil {
			return err
		}
		if err = RequireActionableTransfer(transfer, TransferVerbExpired); err != nil {
			return err
		}

		original = *transfer
		MarkTransferExpired(transfer, timeutils.NowUnix())

		updated, err = s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
			return err
		}

		if transfer.SourceShipmentID.IsNil() {
			return nil
		}

		sourceTenant := pagination.TenantInfo{
			OrgID: transfer.SourceOrganizationID,
			BuID:  transfer.SourceBusinessUnitID,
		}
		if err = s.setShipmentTenderStatus(
			txCtx,
			transfer.SourceShipmentID,
			sourceTenant,
			shipment.TenderStatusExpired,
		); err != nil {
			return err
		}

		return s.createSystemShipmentComment(
			txCtx,
			transfer.SourceShipmentID,
			sourceTenant,
			"EDI load tender expired.",
			map[string]any{"transferId": transfer.ID},
		)
	})
	if err != nil {
		return nil, err
	}

	s.logAction(updated, actor, permission.OpUpdate, &original, updated, "EDI load tender expired")
	return updated, nil
}
