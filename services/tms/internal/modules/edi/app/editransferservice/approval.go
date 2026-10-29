package editransferservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/edi/app/edidecision"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

func (s *Service) MappingPreview(
	ctx context.Context,
	req editenderports.GetEDITransferByIDRequest,
) (*editendermapping.MappingPreview, error) {
	transfer, err := s.transferRepo.GetTransferByID(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.MappingPreviewOf(ctx, transfer)
}

// MappingPreviewOf is how a transfer already read maps to its receiver's
// records: which of the sender's codes resolve and which still need a
// mapping.
func (s *Service) MappingPreviewOf(
	ctx context.Context,
	transfer *editransfer.EDITransfer,
) (*editendermapping.MappingPreview, error) {
	targetPartner, err := s.partnerRepo.GetByID(ctx, edipartnerports.GetEDIPartnerByIDRequest{
		ID: transfer.TargetPartnerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: transfer.TargetOrganizationID,
			BuID:  transfer.TargetBusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	return s.tenderMapping.BuildMappingPreview(ctx, targetPartner, transfer.TenderPayload)
}

//nolint:funlen // Approval coordinates validation, mapping, shipment creation, and transfer updates atomically.
func (s *Service) ApproveTransfer(
	ctx context.Context,
	req *ApproveTransferRequest,
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"approver",
			errortypes.ErrRequired,
			"Approving user is required",
		)
	}

	var original *editransfer.EDITransfer
	var updated *editransfer.EDITransfer
	var preview *editendermapping.MappingPreview
	err := s.db.Transact(
		ctx,
		kernelports.TxOptions{},
		func(txCtx context.Context) error {
			transfer, err := s.transferRepo.GetTransferForUpdate(
				txCtx,
				editenderports.GetEDITransferForUpdateRequest{
					ID:         req.TransferID,
					TenantInfo: req.TenantInfo,
					Direction:  "inbound",
				},
			)
			if err != nil {
				return err
			}
			if err = edidecision.RequireActionableTransfer(transfer, edidecision.TransferVerbApproved); err != nil {
				return err
			}
			originalCopy := *transfer
			original = &originalCopy

			if len(req.Mappings) > 0 {
				if _, err = s.setup.SaveMappingProfile(txCtx, &edipartnerports.SaveMappingItemsRequest{
					PartnerID:  transfer.TargetPartnerID,
					TenantInfo: req.TenantInfo,
					ActorID:    actor.UserID,
					Items:      req.Mappings,
				}); err != nil {
					return err
				}
			}

			targetPartner, err := s.partnerRepo.GetByID(
				txCtx,
				edipartnerports.GetEDIPartnerByIDRequest{
					ID:         transfer.TargetPartnerID,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return err
			}

			preview, err = s.tenderMapping.BuildMappingPreview(
				txCtx,
				targetPartner,
				transfer.TenderPayload,
			)
			if err != nil {
				return err
			}
			if len(preview.Unresolved) > 0 {
				return editendermapping.UnresolvedMappingsError(preview.Unresolved)
			}

			edidecision.MarkTransferApprovalStarted(transfer, &edidecision.TransferApprovalMark{
				BusinessUnitID: req.TenantInfo.BuID,
				Mapping:        preview.All,
				ApproverID:     actor.UserID,
				At:             timeutils.NowUnix(),
			})

			updated, err = s.transferRepo.UpdateTransfer(txCtx, transfer)
			return err
		},
	)
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
		editenderports.SetEDITransferApprovalWorkflowRunIDRequest{
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
	transfer *editransfer.EDITransfer,
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
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
	transfer *editransfer.EDITransfer,
	preview *editendermapping.MappingPreview,
) {
	if transfer == nil {
		return
	}

	transfer.Status = editransfer.TransferStatusPendingApproval
	if preview != nil && len(preview.Unresolved) > 0 {
		transfer.Status = editransfer.TransferStatusMappingRequired
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

//nolint:cyclop,funlen,gocognit // Temporal approval processing mirrors the workflow states explicitly.
func (s *Service) ProcessLoadTenderApproval(
	ctx context.Context,
	payload *ApproveLoadTenderTransferWorkflowPayload,
) (*ApproveLoadTenderTransferWorkflowResult, error) {
	ctx = dbscope.WithValidTenant(ctx, payload.TenantInfo.DBTenant())
	result := new(ApproveLoadTenderTransferWorkflowResult)
	var processingErr error
	var approvedTransfer *editransfer.EDITransfer
	var acceptedInternalTransfer *editransfer.EDITransfer

	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			editenderports.GetEDITransferForUpdateRequest{
				ID:         payload.TransferID,
				TenantInfo: payload.TenantInfo,
				Direction:  "inbound",
			},
		)
		if err != nil {
			return err
		}

		if transfer.Status == editransfer.TransferStatusApproved &&
			transfer.TargetShipmentID.IsNotNil() {
			processedAt := timeutils.NowUnix()
			if transfer.ProcessedAt != nil {
				processedAt = *transfer.ProcessedAt
			}
			result.TransferID = transfer.ID
			result.TargetShipmentID = transfer.TargetShipmentID
			result.ProcessedAt = processedAt
			if transfer.InboundMessageID.IsNil() {
				acceptedInternalTransfer = transfer
			}
			return nil
		}

		//nolint:exhaustive // Only terminal and processing states require special approval handling.
		switch transfer.Status {
		case editransfer.TransferStatusRejected,
			editransfer.TransferStatusExpired,
			editransfer.TransferStatusCanceled,
			editransfer.TransferStatusFailed:
			processingErr = temporal.NewNonRetryableApplicationError(
				"EDI transfer is no longer approval eligible",
				"TransferNotApprovalEligible",
				nil,
			)
			return nil
		case editransfer.TransferStatusProcessing:
		default:
			processingErr = temporal.NewNonRetryableApplicationError(
				"EDI transfer is not processing approval",
				"TransferNotProcessing",
				nil,
			)
			return nil
		}

		targetPartner, err := s.partnerRepo.GetByID(txCtx, edipartnerports.GetEDIPartnerByIDRequest{
			ID: transfer.TargetPartnerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: transfer.TargetOrganizationID,
				BuID:  transfer.TargetBusinessUnitID,
			},
		})
		if err != nil {
			return err
		}

		preview, err := s.tenderMapping.BuildMappingPreview(
			txCtx,
			targetPartner,
			transfer.TenderPayload,
		)
		if err != nil {
			return err
		}
		if len(preview.Unresolved) > 0 {
			now := timeutils.NowUnix()
			transfer.Status = editransfer.TransferStatusMappingRequired
			transfer.MappingSnapshot = preview.All
			transfer.ProcessedAt = &now
			if _, err = s.transferRepo.UpdateTransfer(txCtx, transfer); err != nil {
				return err
			}
			processingErr = temporal.NewNonRetryableApplicationError(
				"EDI mappings are no longer complete",
				"UnresolvedMappings",
				editendermapping.UnresolvedMappingsError(preview.Unresolved),
			)
			return nil
		}

		targetShipment, err := s.tenderMapping.BuildTargetShipment(
			transfer,
			payload.TenantInfo.BuID,
			preview.All,
			payload.Actor.UserID,
		)
		if err != nil {
			processingErr = err
			return s.failTransferInTransaction(txCtx, transfer, err)
		}

		var createdShipment *shipment.Shipment
		err = s.savepoints.Savepoint(txCtx, func(spCtx context.Context) error {
			created, createErr := s.shipmentSvc.Create(spCtx, targetShipment, payload.Actor)
			if createErr != nil {
				return createErr
			}
			createdShipment = created
			return s.saveTenderCustomFields(spCtx, created, transfer.TenderPayload.CustomFields)
		})
		if err != nil {
			processingErr = err
			return s.failTransferInTransaction(txCtx, transfer, err)
		}

		now := timeutils.NowUnix()
		transfer.Status = editransfer.TransferStatusApproved
		transfer.TargetShipmentID = createdShipment.ID
		transfer.MappingSnapshot = preview.All
		transfer.ProcessedAt = &now

		if transfer.InboundMessageID.IsNotNil() {
			updated, updateErr := s.transferRepo.UpdateTransfer(txCtx, transfer)
			if updateErr != nil {
				return updateErr
			}
			approvedTransfer = updated
			if commentErr := s.shipments.CreateSystemComment(
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

		updated, err := s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
			return err
		}

		link, err := s.shipmentLinkRepo.CreateShipmentLink(txCtx, &edisync.ShipmentLink{
			BusinessUnitID:       transfer.SourceBusinessUnitID,
			SourceOrganizationID: transfer.SourceOrganizationID,
			TargetOrganizationID: transfer.TargetOrganizationID,
			SourceShipmentID:     transfer.SourceShipmentID,
			TargetShipmentID:     createdShipment.ID,
			TenderTransferID:     transfer.ID,
			SyncPolicy:           edisync.ShipmentSyncPolicyAutoOperational,
			FieldOwnership:       edisync.DefaultShipmentFieldOwnership(),
			Status:               edisync.ShipmentLinkStatusActive,
		})
		if err != nil {
			return err
		}
		if err = s.recipient.UpsertInternal(
			txCtx,
			transfer,
			link,
			editenderchange.TenderRecipientBaselineStatusAccepted,
		); err != nil {
			return err
		}

		if err = s.shipments.CreateSystemComment(
			txCtx,
			createdShipment.ID,
			pagination.TenantInfo{
				OrgID: transfer.TargetOrganizationID,
				BuID:  transfer.TargetBusinessUnitID,
			},
			"Shipment created from accepted EDI load tender.",
			map[string]any{"transferId": transfer.ID, "shipmentLinkId": link.ID},
		); err != nil {
			return err
		}

		acceptedInternalTransfer = updated
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
	if acceptedInternalTransfer != nil {
		if err = s.acceptSourceTender(ctx, acceptedInternalTransfer); err != nil {
			return nil, err
		}
	}
	if approvedTransfer != nil {
		s.outbound.AttachFlowShipment(ctx, approvedTransfer)
		s.generateExternalTenderResponse(ctx, approvedTransfer, payload.Actor.UserID)
	}

	return result, nil
}

func (s *Service) acceptSourceTender(ctx context.Context, transfer *editransfer.EDITransfer) error {
	if transfer.SourceShipmentID.IsNil() {
		return nil
	}

	sourceTenant := pagination.TenantInfo{
		OrgID: transfer.SourceOrganizationID,
		BuID:  transfer.SourceBusinessUnitID,
	}

	return s.db.Transact(
		dbscope.WithTenant(ctx, sourceTenant.DBTenant()),
		kernelports.TxOptions{},
		func(txCtx context.Context) error {
			source, err := s.shipments.Lock(txCtx, transfer.SourceShipmentID, sourceTenant)
			if err != nil {
				return err
			}
			if source.TenderStatus != nil &&
				*source.TenderStatus == shipment.TenderStatusAccepted {
				return nil
			}

			if err = s.shipments.SetTenderStatus(
				txCtx,
				source.ID,
				sourceTenant,
				shipment.TenderStatusAccepted,
			); err != nil {
				return err
			}

			links, err := s.shipmentLinkRepo.GetShipmentLinksByShipmentID(
				txCtx,
				editenderports.GetEDIShipmentLinksByShipmentIDRequest{
					ShipmentID: source.ID,
					TenantInfo: sourceTenant,
				},
			)
			if err != nil {
				return err
			}
			metadata := map[string]any{"transferId": transfer.ID}
			for _, link := range links {
				if link.TenderTransferID == transfer.ID {
					metadata["shipmentLinkId"] = link.ID
				}
			}

			return s.shipments.CreateSystemComment(
				txCtx,
				source.ID,
				sourceTenant,
				"EDI load tender accepted by receiving organization.",
				metadata,
			)
		},
	)
}

func (s *Service) failTransferInTransaction(
	ctx context.Context,
	transfer *editransfer.EDITransfer,
	err error,
) error {
	now := timeutils.NowUnix()
	transfer.Status = editransfer.TransferStatusFailed
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
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	reason, err := edidecision.TransferRejectionReason(req.Reason)
	if err != nil {
		return nil, err
	}

	sourceCtx, err := s.transferSourceContext(ctx, req.TransferID, req.TenantInfo, "inbound")
	if err != nil {
		return nil, err
	}

	var original editransfer.EDITransfer
	var updated *editransfer.EDITransfer
	err = s.db.Transact(sourceCtx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			editenderports.GetEDITransferForUpdateRequest{
				ID:         req.TransferID,
				TenantInfo: req.TenantInfo,
				Direction:  "inbound",
			},
		)
		if err != nil {
			return err
		}
		if err = edidecision.RequireActionableTransfer(transfer, edidecision.TransferVerbRejected); err != nil {
			return err
		}

		original = *transfer
		edidecision.MarkTransferRejected(transfer, reason, actor.UserID, timeutils.NowUnix())

		updated, err = s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
			return err
		}

		if transfer.SourceShipmentID.IsNil() {
			return nil
		}

		if err = s.shipments.SetTenderStatus(
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

		return s.shipments.CreateSystemComment(
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
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	var original editransfer.EDITransfer
	var updated *editransfer.EDITransfer
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			editenderports.GetEDITransferForUpdateRequest{
				ID:         req.TransferID,
				TenantInfo: req.TenantInfo,
				Direction:  "outbound",
			},
		)
		if err != nil {
			return err
		}
		if err = edidecision.RequireActionableTransfer(transfer, edidecision.TransferVerbCanceled); err != nil {
			return err
		}

		original = *transfer
		edidecision.MarkTransferCanceled(transfer, actor.UserID, timeutils.NowUnix())

		updated, err = s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
			return err
		}

		if err = s.shipments.SetTenderStatus(
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

		return s.shipments.CreateSystemComment(
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
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	sourceCtx, err := s.transferSourceContext(ctx, req.TransferID, req.TenantInfo, "")
	if err != nil {
		return nil, err
	}

	var original editransfer.EDITransfer
	var updated *editransfer.EDITransfer
	err = s.db.Transact(sourceCtx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		transfer, err := s.transferRepo.GetTransferForUpdate(
			txCtx,
			editenderports.GetEDITransferForUpdateRequest{
				ID:         req.TransferID,
				TenantInfo: req.TenantInfo,
				Direction:  "",
			},
		)
		if err != nil {
			return err
		}
		if err = edidecision.RequireActionableTransfer(transfer, edidecision.TransferVerbExpired); err != nil {
			return err
		}

		original = *transfer
		edidecision.MarkTransferExpired(transfer, timeutils.NowUnix())

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
		if err = s.shipments.SetTenderStatus(
			txCtx,
			transfer.SourceShipmentID,
			sourceTenant,
			shipment.TenderStatusExpired,
		); err != nil {
			return err
		}

		return s.shipments.CreateSystemComment(
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

func (s *Service) transferSourceContext(
	ctx context.Context,
	transferID pulid.ID,
	tenantInfo pagination.TenantInfo,
	direction string,
) (context.Context, error) {
	transfer, err := s.transferRepo.GetTransferByID(ctx, editenderports.GetEDITransferByIDRequest{
		ID:         transferID,
		TenantInfo: tenantInfo,
		Direction:  direction,
	})
	if err != nil {
		return nil, err
	}

	if transfer.SourceOrganizationID == tenantInfo.OrgID &&
		transfer.SourceBusinessUnitID == tenantInfo.BuID {
		return ctx, nil
	}

	source := pagination.TenantInfo{
		OrgID: transfer.SourceOrganizationID,
		BuID:  transfer.SourceBusinessUnitID,
	}
	return dbscope.WithTenant(ctx, source.DBTenant()), nil
}

func (s *Service) saveTenderCustomFields(
	ctx context.Context,
	created *shipment.Shipment,
	values map[string]any,
) error {
	if len(values) == 0 || s.customFields == nil {
		return nil
	}
	if multiErr := s.customFields.SaveNamed(
		ctx,
		pagination.TenantInfo{OrgID: created.OrganizationID, BuID: created.BusinessUnitID},
		created.GetResourceType(),
		created.GetResourceID(),
		values,
	); multiErr != nil {
		return multiErr
	}
	return nil
}
