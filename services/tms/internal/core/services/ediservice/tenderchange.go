package ediservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (s *Service) AfterShipmentUpdate(
	ctx context.Context,
	original *shipment.Shipment,
	updated *shipment.Shipment,
	actor *services.RequestActor,
) error {
	if original == nil || updated == nil {
		return nil
	}
	if original.ID != updated.ID || original.OrganizationID != updated.OrganizationID {
		return nil
	}

	oldPayload := buildTenderPayload(original)
	newPayload := buildTenderPayload(updated)
	newPayload.PurposeCode = edi.LoadTenderPurposeChange
	newHash := tenderPayloadHash(&newPayload)
	if tenderPayloadHash(&oldPayload) == newHash {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  updated.OrganizationID,
		BuID:   updated.BusinessUnitID,
		UserID: actorUserID(actor),
	}
	recipients, err := s.tenderRecipientRepo.ListActiveTenderRecipientsForSourceShipment(
		ctx,
		repositories.ListEDITenderRecipientsForSourceShipmentRequest{
			TenantInfo:       tenantInfo,
			SourceShipmentID: updated.ID,
		},
	)
	if err != nil {
		return err
	}

	var recipientErrs []error
	for _, recipient := range recipients {
		if recipient == nil || recipient.LatestBaselineHash == newHash {
			continue
		}
		if err = s.createTenderChangeForRecipient(
			ctx,
			recipient,
			&newPayload,
			updated.Version,
			actor,
		); err != nil {
			s.l.Error(
				"failed to record EDI tender change for recipient",
				zap.String("recipientId", recipient.ID.String()),
				zap.String("shipmentId", updated.ID.String()),
				zap.Error(err),
			)
			recipientErrs = append(recipientErrs, err)
		}
	}

	return errors.Join(recipientErrs...)
}

func (s *Service) createTenderChangeForRecipient(
	ctx context.Context,
	recipient *edi.TenderRecipient,
	newPayload *edi.LoadTenderPayload,
	sourceShipmentVersion int64,
	actor *services.RequestActor,
) error {
	newHash := tenderPayloadHash(newPayload)
	diff := diffTenderPayloads(&recipient.LatestBaselinePayload, newPayload)
	if len(diff) == 0 {
		return nil
	}

	status := edi.TenderChangeStatusPendingReview
	if recipient.RecipientKind == edi.TenderRecipientKindExternal {
		status = edi.TenderChangeStatusQueued
	}
	change := &edi.TenderChange{
		BusinessUnitID:       recipient.BusinessUnitID,
		SourceOrganizationID: recipient.SourceOrganizationID,
		SourceBusinessUnitID: recipient.SourceBusinessUnitID,
		SourceShipmentID:     recipient.SourceShipmentID,
		RecipientID:          recipient.ID,
		RecipientKind:        recipient.RecipientKind,
		Status:               status,
		ChangeType:           edi.TenderChangeTypeLoadTender,
		IdempotencyKey: fmt.Sprintf(
			"%s:%d:%s:%s",
			recipient.ID,
			sourceShipmentVersion,
			newHash,
			edi.TenderChangeTypeLoadTender,
		),
		SourceShipmentVersion:   sourceShipmentVersion,
		PreviousBaselinePayload: recipient.LatestBaselinePayload,
		NewTenderPayload:        *newPayload,
		PreviousBaselineHash:    recipient.LatestBaselineHash,
		NewPayloadHash:          newHash,
		DiffSummary:             diff,
		InternalTransferID:      recipient.OriginalTransferID,
		ShipmentLinkID:          recipient.ShipmentLinkID,
	}

	result, err := s.tenderChangeRepo.CreateTenderChangeIdempotent(ctx, change)
	if err != nil {
		return err
	}
	if !result.Created {
		return nil
	}
	if err = s.tenderChangeRepo.SupersedeActionableTenderChanges(
		ctx,
		repositories.SupersedeActionableEDITenderChangesRequest{
			RecipientID:     recipient.ID,
			ExcludeChangeID: result.TenderChange.ID,
			Statuses: []edi.TenderChangeStatus{
				edi.TenderChangeStatusPendingReview,
				edi.TenderChangeStatusQueued,
			},
		},
	); err != nil {
		return err
	}
	if recipient.RecipientKind == edi.TenderRecipientKindExternal {
		return s.generateExternalTenderChange(ctx, recipient, result.TenderChange, actor)
	}
	return nil
}

func (s *Service) generateExternalTenderChange(
	ctx context.Context,
	recipient *edi.TenderRecipient,
	change *edi.TenderChange,
	actor *services.RequestActor,
) error {
	payload := change.NewTenderPayload
	payload.PurposeCode = edi.LoadTenderPurposeChange
	message, err := s.GenerateDocument(ctx, &GenerateEDIDocumentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  recipient.SourceOrganizationID,
			BuID:   recipient.SourceBusinessUnitID,
			UserID: actorUserID(actor),
		},
		PartnerDocumentProfileID: recipient.PartnerDocumentProfileID,
		EDIPartnerID:             recipient.EDIPartnerID,
		ShipmentID:               recipient.SourceShipmentID,
		TransactionSet:           edi.TransactionSet204,
		Direction:                edi.DocumentDirectionOutbound,
		Payload:                  loadTenderDocumentPayload(&payload),
		GeneratedByID:            actorUserID(actor),
		DisableDeliveryQueue:     true,
	})
	if err != nil {
		change.Status = edi.TenderChangeStatusFailed
		change.FailureReason = err.Error()
		_, updateErr := s.tenderChangeRepo.UpdateTenderChange(ctx, change)
		return updateErr
	}

	change.OutboundMessageID = message.ID
	change.Status = edi.TenderChangeStatusQueued
	change.FailureReason = ""
	if _, err = s.tenderChangeRepo.UpdateTenderChange(ctx, change); err != nil {
		return err
	}
	if err = s.queueMessageForDelivery(ctx, message); err != nil {
		s.l.Warn(
			"failed to queue EDI tender change delivery",
			zap.String("changeId", change.ID.String()),
			zap.String("messageId", message.ID.String()),
			zap.Error(err),
		)
	}
	return nil
}

func (s *Service) ListTenderChanges(
	ctx context.Context,
	req *repositories.ListEDITenderChangesRequest,
) (*pagination.ListResult[*edi.TenderChange], error) {
	return s.tenderChangeRepo.ListTenderChanges(ctx, req)
}

func (s *Service) GetTenderChange(
	ctx context.Context,
	req repositories.GetEDITenderChangeByIDRequest,
) (*edi.TenderChange, error) {
	return s.tenderChangeRepo.GetTenderChangeByID(ctx, req)
}

func (s *Service) ApplyTenderChange(
	ctx context.Context,
	req *TenderChangeActionRequest,
	actor *services.RequestActor,
) (*edi.TenderChange, error) {
	return s.reviewTenderChange(ctx, req, actor, edi.TenderChangeStatusApplied)
}

func (s *Service) RejectTenderChange(
	ctx context.Context,
	req *TenderChangeActionRequest,
	actor *services.RequestActor,
) (*edi.TenderChange, error) {
	return s.reviewTenderChange(ctx, req, actor, edi.TenderChangeStatusRejected)
}

func (s *Service) reviewTenderChange(
	ctx context.Context,
	req *TenderChangeActionRequest,
	actor *services.RequestActor,
	status edi.TenderChangeStatus,
) (*edi.TenderChange, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"EDI tender change review request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"userId",
			errortypes.ErrRequired,
			"Reviewing user is required",
		)
	}

	change, err := s.tenderChangeRepo.GetTenderChangeByID(
		ctx,
		repositories.GetEDITenderChangeByIDRequest{ID: req.ChangeID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if change.Status != edi.TenderChangeStatusPendingReview {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI tender change is not pending review",
		)
	}
	if err = validateTenderChangeReviewer(change, req.TenantInfo); err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	var applyErr error
	err = s.db.WithTx(ctx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if status == edi.TenderChangeStatusApplied {
			if applyErr = s.applyApprovedTenderChange(
				txCtx,
				req.TenantInfo,
				change,
				actor,
			); applyErr != nil {
				return applyErr
			}
		}
		change.Status = status
		change.ReviewedByID = actor.UserID
		change.ReviewedAt = &now
		if strings.TrimSpace(req.Reason) != "" {
			change.FailureReason = strings.TrimSpace(req.Reason)
		}
		if status == edi.TenderChangeStatusApplied {
			change.AppliedByID = actor.UserID
			change.AppliedAt = &now
		}
		_, updateErr := s.tenderChangeRepo.UpdateTenderChange(txCtx, change)
		return updateErr
	})
	if applyErr != nil {
		change.ConflictMetadata = map[string]any{"reason": applyErr.Error()}
		change.FailureReason = applyErr.Error()
		if _, updateErr := s.tenderChangeRepo.UpdateTenderChange(ctx, change); updateErr != nil {
			s.l.Warn(
				"failed to record EDI tender change apply failure",
				zap.String("changeId", change.ID.String()),
				zap.Error(updateErr),
			)
		}
		return nil, applyErr
	}
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *Service) applyApprovedTenderChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	change *edi.TenderChange,
	actor *services.RequestActor,
) error {
	recipient, err := s.tenderRecipientRepo.GetTenderRecipientByID(
		ctx,
		repositories.GetEDITenderRecipientByIDRequest{
			ID:         change.RecipientID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return err
	}
	if change.ShipmentLinkID.IsNil() {
		return s.applyPendingTransferTenderChange(ctx, tenantInfo, recipient, change)
	}
	return s.applyLinkedShipmentTenderChange(ctx, recipient, change, actor)
}

func (s *Service) applyPendingTransferTenderChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	recipient *edi.TenderRecipient,
	change *edi.TenderChange,
) error {
	transfer, err := s.transferRepo.GetTransferForUpdate(
		ctx,
		repositories.GetEDITransferForUpdateRequest{
			ID:         change.InternalTransferID,
			TenantInfo: tenantInfo,
			Direction:  "inbound",
		},
	)
	if err != nil {
		return err
	}
	targetPartner, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID:         transfer.TargetPartnerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	preview, err := s.buildMappingPreview(ctx, targetPartner, change.NewTenderPayload)
	if err != nil {
		return err
	}
	if len(preview.Unresolved) > 0 {
		return unresolvedMappingsError(preview.Unresolved)
	}
	transfer.TenderPayload = change.NewTenderPayload
	transfer.MappingSnapshot = preview.All
	if !transfer.Status.IsFinal() {
		transfer.Status = edi.TransferStatusPendingApproval
	}
	if _, err = s.transferRepo.UpdateTransfer(ctx, transfer); err != nil {
		return err
	}
	return s.advanceTenderRecipientBaseline(
		ctx,
		recipient,
		&change.NewTenderPayload,
		edi.TenderRecipientBaselineStatusSent,
	)
}

func (s *Service) applyLinkedShipmentTenderChange(
	ctx context.Context,
	recipient *edi.TenderRecipient,
	change *edi.TenderChange,
	actor *services.RequestActor,
) error {
	targetTenantInfo, err := tenderRecipientTenantInfo(recipient)
	if err != nil {
		return err
	}
	sourceTenantInfo := pagination.TenantInfo{
		OrgID: recipient.SourceOrganizationID,
		BuID:  recipient.SourceBusinessUnitID,
	}
	link, err := s.shipmentLinkRepo.GetShipmentLinkByID(
		ctx,
		repositories.GetEDIShipmentLinkByIDRequest{
			ID:         change.ShipmentLinkID,
			TenantInfo: sourceTenantInfo,
		},
	)
	if err != nil {
		return err
	}
	transfer, err := s.transferRepo.GetTransferByID(
		ctx,
		repositories.GetEDITransferByIDRequest{
			ID:         change.InternalTransferID,
			TenantInfo: targetTenantInfo,
			Direction:  "inbound",
		},
	)
	if err != nil {
		return err
	}
	targetPartner, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID:         transfer.TargetPartnerID,
		TenantInfo: targetTenantInfo,
	})
	if err != nil {
		return err
	}
	preview, err := s.buildMappingPreview(ctx, targetPartner, change.NewTenderPayload)
	if err != nil {
		return err
	}
	if len(preview.Unresolved) > 0 {
		return unresolvedMappingsError(preview.Unresolved)
	}
	targetShipment, err := s.shipmentSvc.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:              link.TargetShipmentID,
		TenantInfo:      targetTenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return err
	}
	applyTenderPayloadToShipment(targetShipment, &change.NewTenderPayload, transfer, preview.All)
	if _, err = s.shipmentSvc.Update(ctx, targetShipment, actor); err != nil {
		return err
	}
	return s.advanceTenderRecipientBaseline(
		ctx,
		recipient,
		&change.NewTenderPayload,
		edi.TenderRecipientBaselineStatusAccepted,
	)
}

func validateTenderChangeReviewer(
	change *edi.TenderChange,
	tenantInfo pagination.TenantInfo,
) error {
	if change == nil || change.Recipient == nil {
		return errortypes.NewValidationError(
			"recipientId",
			errortypes.ErrRequired,
			"EDI tender change recipient is required for review",
		)
	}
	if change.Recipient.RecipientKind != edi.TenderRecipientKindInternal {
		return errortypes.NewValidationError(
			"recipientKind",
			errortypes.ErrInvalidOperation,
			"Only internal tender changes can be manually reviewed",
		)
	}
	isRecipientOrg := change.Recipient.RecipientOrganizationID == tenantInfo.OrgID
	isRecipientBU := change.Recipient.RecipientBusinessUnitID.IsNil() ||
		change.Recipient.RecipientBusinessUnitID == tenantInfo.BuID
	if !isRecipientOrg || !isRecipientBU {
		return errortypes.NewValidationError(
			"tenantInfo",
			errortypes.ErrInvalidOperation,
			"Only the original internal tender recipient can review this tender change",
		)
	}
	return nil
}

func tenderRecipientTenantInfo(
	recipient *edi.TenderRecipient,
) (pagination.TenantInfo, error) {
	if recipient == nil ||
		recipient.RecipientOrganizationID.IsNil() ||
		recipient.RecipientBusinessUnitID.IsNil() {
		return pagination.TenantInfo{}, errortypes.NewValidationError(
			"recipientId",
			errortypes.ErrRequired,
			"Internal tender recipient tenant is required",
		)
	}
	return pagination.TenantInfo{
		OrgID: recipient.RecipientOrganizationID,
		BuID:  recipient.RecipientBusinessUnitID,
	}, nil
}

func (s *Service) advanceTenderRecipientBaseline(
	ctx context.Context,
	recipient *edi.TenderRecipient,
	payload *edi.LoadTenderPayload,
	status edi.TenderRecipientBaselineStatus,
) error {
	baseline := *payload
	baseline.PurposeCode = edi.LoadTenderPurposeOriginal
	recipient.LatestBaselinePayload = baseline
	recipient.LatestBaselineHash = tenderPayloadHash(&baseline)
	recipient.BaselineRecordedAt = timeutils.NowUnix()
	recipient.BaselineStatus = status
	_, err := s.tenderRecipientRepo.UpdateTenderRecipient(ctx, recipient)
	return err
}

func (s *Service) upsertInternalTenderRecipient(
	ctx context.Context,
	transfer *edi.EDITransfer,
	link *edi.ShipmentLink,
	status edi.TenderRecipientBaselineStatus,
) error {
	if s.tenderRecipientRepo == nil || transfer == nil {
		return nil
	}
	payload := transfer.TenderPayload
	payload.PurposeCode = edi.LoadTenderPurposeOriginal
	recipient := &edi.TenderRecipient{
		BusinessUnitID:          transfer.SourceBusinessUnitID,
		SourceOrganizationID:    transfer.SourceOrganizationID,
		SourceBusinessUnitID:    transfer.SourceBusinessUnitID,
		SourceShipmentID:        transfer.SourceShipmentID,
		RecipientKind:           edi.TenderRecipientKindInternal,
		RecipientOrganizationID: transfer.TargetOrganizationID,
		RecipientBusinessUnitID: transfer.TargetBusinessUnitID,
		EDIPartnerID:            transfer.TargetPartnerID,
		OriginalTransferID:      transfer.ID,
		LatestBaselinePayload:   payload,
		LatestBaselineHash:      tenderPayloadHash(&payload),
		BaselineRecordedAt:      timeutils.NowUnix(),
		BaselineStatus:          status,
		Status:                  edi.TenderRecipientStatusActive,
	}
	if link != nil {
		recipient.ShipmentLinkID = link.ID
	}
	_, err := s.tenderRecipientRepo.UpsertTenderRecipient(
		ctx,
		repositories.UpsertEDITenderRecipientRequest{Recipient: recipient},
	)
	return err
}

func (s *Service) upsertExternalTenderRecipient(
	ctx context.Context,
	message *edi.EDIMessage,
	profile *edi.EDIPartnerDocumentProfile,
) error {
	if s.tenderRecipientRepo == nil || message == nil || profile == nil ||
		message.TransactionSet != edi.TransactionSet204 ||
		message.Direction != edi.DocumentDirectionOutbound ||
		message.PayloadSnapshot.LoadTender == nil ||
		message.PayloadSnapshot.LoadTender.PurposeCode == edi.LoadTenderPurposeChange {
		return nil
	}
	if profile.Partner != nil && profile.Partner.Kind != edi.PartnerKindExternal {
		return nil
	}
	commProfile, err := s.profileRepo.GetActiveProfileByPartner(
		ctx,
		repositories.GetActiveEDICommunicationProfileByPartnerRequest{
			PartnerID: message.EDIPartnerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: message.OrganizationID,
				BuID:  message.BusinessUnitID,
			},
			Method: edi.ConnectionMethodSFTP,
		},
	)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			return err
		}
		commProfile = nil
	}
	payload := *message.PayloadSnapshot.LoadTender
	payload.PurposeCode = edi.LoadTenderPurposeOriginal
	recipient := &edi.TenderRecipient{
		BusinessUnitID:           message.BusinessUnitID,
		SourceOrganizationID:     message.OrganizationID,
		SourceBusinessUnitID:     message.BusinessUnitID,
		SourceShipmentID:         payload.ShipmentID,
		RecipientKind:            edi.TenderRecipientKindExternal,
		EDIPartnerID:             message.EDIPartnerID,
		PartnerDocumentProfileID: message.PartnerDocumentProfileID,
		OriginalMessageID:        message.ID,
		LatestBaselinePayload:    payload,
		LatestBaselineHash:       tenderPayloadHash(&payload),
		BaselineRecordedAt:       timeutils.NowUnix(),
		BaselineStatus:           edi.TenderRecipientBaselineStatusSent,
		Status:                   edi.TenderRecipientStatusActive,
	}
	if commProfile != nil {
		recipient.CommunicationProfileID = commProfile.ID
	}
	_, err = s.tenderRecipientRepo.UpsertTenderRecipient(
		ctx,
		repositories.UpsertEDITenderRecipientRequest{Recipient: recipient},
	)
	return err
}

func applyTenderPayloadToShipment(
	target *shipment.Shipment,
	payload *edi.LoadTenderPayload,
	transfer *edi.EDITransfer,
	resolutions []edi.MappingResolution,
) {
	mappings := resolutionIndex(resolutions)
	target.ServiceTypeID = mustMappedID(
		mappings,
		edi.MappingEntityTypeServiceType,
		payload.ServiceTypeID,
	)
	target.ShipmentTypeID = optionalMappedID(
		mappings,
		edi.MappingEntityTypeShipmentType,
		payload.ShipmentTypeID,
	)
	target.CustomerID = mustMappedID(mappings, edi.MappingEntityTypeCustomer, payload.CustomerID)
	target.FormulaTemplateID = mustMappedID(
		mappings,
		edi.MappingEntityTypeFormulaTemplate,
		payload.FormulaTemplateID,
	)
	target.TenderStatus = tenderStatusPtr(shipment.TenderStatusAccepted)
	target.BOL = payload.BOL
	target.Pieces = payload.Pieces
	target.Weight = payload.Weight
	target.TemperatureMin = payload.TemperatureMin
	target.TemperatureMax = payload.TemperatureMax
	target.FreightChargeAmount = payload.FreightChargeAmount
	target.OtherChargeAmount = payload.OtherChargeAmount
	target.BaseRate = payload.BaseRate
	target.TotalChargeAmount = payload.TotalChargeAmount
	target.RatingUnit = payload.RatingUnit
	target.Moves = mappedTenderMoves(target, payload, transfer, mappings)
	target.Commodities = mappedTenderCommodities(target, payload, transfer, mappings)
	target.AdditionalCharges = mappedTenderCharges(target, payload, transfer, mappings)
}

func mappedTenderMoves(
	target *shipment.Shipment,
	payload *edi.LoadTenderPayload,
	transfer *edi.EDITransfer,
	mappings map[edi.MappingEntityType]map[pulid.ID]pulid.ID,
) []*shipment.ShipmentMove {
	existingMoves := make(map[int64]*shipment.ShipmentMove, len(target.Moves))
	for _, move := range target.Moves {
		if move != nil {
			existingMoves[move.Sequence] = move
		}
	}
	moves := make([]*shipment.ShipmentMove, 0, len(payload.Moves))
	for _, move := range payload.Moves {
		existing := existingMoves[move.Sequence]
		targetMove := &shipment.ShipmentMove{
			BusinessUnitID: target.BusinessUnitID,
			OrganizationID: target.OrganizationID,
			ShipmentID:     target.ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         move.Loaded,
			Sequence:       move.Sequence,
			Distance:       move.Distance,
			Stops:          mappedTenderStops(existing, target, move, mappings),
		}
		if existing != nil {
			targetMove.ID = existing.ID
			targetMove.Status = existing.Status
			targetMove.Version = existing.Version
		}
		if transfer != nil {
			targetMove.BusinessUnitID = transfer.TargetBusinessUnitID
			targetMove.OrganizationID = transfer.TargetOrganizationID
		}
		moves = append(moves, targetMove)
	}
	return moves
}

func mappedTenderStops(
	existingMove *shipment.ShipmentMove,
	target *shipment.Shipment,
	move edi.LoadTenderMove,
	mappings map[edi.MappingEntityType]map[pulid.ID]pulid.ID,
) []*shipment.Stop {
	existingStops := map[int64]*shipment.Stop{}
	if existingMove != nil {
		existingStops = make(map[int64]*shipment.Stop, len(existingMove.Stops))
		for _, stop := range existingMove.Stops {
			if stop != nil {
				existingStops[stop.Sequence] = stop
			}
		}
	}
	stops := make([]*shipment.Stop, 0, len(move.Stops))
	for i := range move.Stops {
		stop := &move.Stops[i]
		existing := existingStops[stop.Sequence]
		targetStop := &shipment.Stop{
			BusinessUnitID: target.BusinessUnitID,
			OrganizationID: target.OrganizationID,
			LocationID: mustMappedID(
				mappings,
				edi.MappingEntityTypeLocation,
				stop.LocationID,
			),
			Status:               shipment.StopStatusNew,
			Type:                 shipment.StopType(stop.Type),
			ScheduleType:         shipment.StopScheduleType(stop.ScheduleType),
			Sequence:             stop.Sequence,
			Pieces:               stop.Pieces,
			Weight:               stop.Weight,
			ScheduledWindowStart: stop.ScheduledWindowStart,
			ScheduledWindowEnd:   stop.ScheduledWindowEnd,
			AddressLine:          stop.AddressLine,
		}
		if existing != nil {
			targetStop.ID = existing.ID
			targetStop.ShipmentMoveID = existing.ShipmentMoveID
			targetStop.Status = existing.Status
			targetStop.ActualArrival = existing.ActualArrival
			targetStop.ActualDeparture = existing.ActualDeparture
			targetStop.CountLateOverride = existing.CountLateOverride
			targetStop.CountDetentionOverride = existing.CountDetentionOverride
			targetStop.Version = existing.Version
		}
		stops = append(stops, targetStop)
	}
	return stops
}

func mappedTenderCommodities(
	target *shipment.Shipment,
	payload *edi.LoadTenderPayload,
	transfer *edi.EDITransfer,
	mappings map[edi.MappingEntityType]map[pulid.ID]pulid.ID,
) []*shipment.ShipmentCommodity {
	commodities := make([]*shipment.ShipmentCommodity, 0, len(payload.Commodities))
	existing := make(map[pulid.ID]*shipment.ShipmentCommodity, len(target.Commodities))
	for _, commodity := range target.Commodities {
		if commodity != nil {
			existing[commodity.CommodityID] = commodity
		}
	}
	for _, commodity := range payload.Commodities {
		targetID := mustMappedID(mappings, edi.MappingEntityTypeCommodity, commodity.CommodityID)
		item := &shipment.ShipmentCommodity{
			BusinessUnitID: target.BusinessUnitID,
			OrganizationID: target.OrganizationID,
			ShipmentID:     target.ID,
			CommodityID:    targetID,
			Weight:         commodity.Weight,
			Pieces:         commodity.Pieces,
		}
		if transfer != nil {
			item.BusinessUnitID = transfer.TargetBusinessUnitID
			item.OrganizationID = transfer.TargetOrganizationID
		}
		if current := existing[targetID]; current != nil {
			item.ID = current.ID
			item.Version = current.Version
		}
		commodities = append(commodities, item)
	}
	return commodities
}

func mappedTenderCharges(
	target *shipment.Shipment,
	payload *edi.LoadTenderPayload,
	transfer *edi.EDITransfer,
	mappings map[edi.MappingEntityType]map[pulid.ID]pulid.ID,
) []*shipment.AdditionalCharge {
	charges := make([]*shipment.AdditionalCharge, 0, len(payload.AdditionalCharges))
	existing := make(map[pulid.ID]*shipment.AdditionalCharge, len(target.AdditionalCharges))
	for _, charge := range target.AdditionalCharges {
		if charge != nil {
			existing[charge.AccessorialChargeID] = charge
		}
	}
	for _, charge := range payload.AdditionalCharges {
		targetID := mustMappedID(
			mappings,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
		)
		item := &shipment.AdditionalCharge{
			BusinessUnitID:      target.BusinessUnitID,
			OrganizationID:      target.OrganizationID,
			ShipmentID:          target.ID,
			AccessorialChargeID: targetID,
			Method:              accessorialcharge.Method(charge.Method),
			Amount:              charge.Amount,
			Unit:                charge.Unit,
		}
		if transfer != nil {
			item.BusinessUnitID = transfer.TargetBusinessUnitID
			item.OrganizationID = transfer.TargetOrganizationID
		}
		if current := existing[targetID]; current != nil {
			item.ID = current.ID
			item.IsSystemGenerated = current.IsSystemGenerated
			item.Version = current.Version
		}
		charges = append(charges, item)
	}
	return charges
}

func mustMappedID(
	mappings map[edi.MappingEntityType]map[pulid.ID]pulid.ID,
	entityType edi.MappingEntityType,
	sourceID pulid.ID,
) pulid.ID {
	targetID, ok := mappedID(mappings, entityType, sourceID)
	if !ok {
		return pulid.Nil
	}
	return targetID
}

func tenderPayloadHash(payload *edi.LoadTenderPayload) string {
	if payload == nil {
		return ""
	}
	hashPayload := *payload
	hashPayload.PurposeCode = ""
	data, err := sonic.ConfigStd.Marshal(hashPayload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func diffTenderPayloads(
	previous *edi.LoadTenderPayload,
	next *edi.LoadTenderPayload,
) map[string]any {
	prevMap := tenderPayloadMap(previous)
	nextMap := tenderPayloadMap(next)
	diff := make(map[string]any, len(nextMap))
	for key, nextValue := range nextMap {
		if !reflect.DeepEqual(prevMap[key], nextValue) {
			diff[key] = map[string]any{
				"previous": prevMap[key],
				"next":     nextValue,
			}
		}
	}
	for key, prevValue := range prevMap {
		if _, ok := nextMap[key]; !ok {
			diff[key] = map[string]any{"previous": prevValue}
		}
	}
	return diff
}

func tenderPayloadMap(payload *edi.LoadTenderPayload) map[string]any {
	result := map[string]any{}
	if payload == nil {
		return result
	}
	mapPayload := *payload
	mapPayload.PurposeCode = ""
	data, err := sonic.ConfigStd.Marshal(mapPayload)
	if err != nil {
		return result
	}
	if err = sonic.Unmarshal(data, &result); err != nil {
		return map[string]any{}
	}
	return result
}

func loadTenderDocumentPayload(payload *edi.LoadTenderPayload) *edi.DocumentPayload {
	documentPayload := edi.NewLoadTenderDocumentPayload(*payload)
	documentPayload.PurposeCode = edi.LoadTenderPurposeChange
	return &documentPayload
}

func (s *Service) decryptProfileSecret(
	profile *edi.EDICommunicationProfile,
	key string,
) (string, error) {
	if profile == nil || len(profile.EncryptedSecrets) == 0 {
		return "", nil
	}
	value := strings.TrimSpace(profile.EncryptedSecrets[key])
	if value == "" {
		return "", nil
	}
	if s.encryption == nil {
		if encryptionservice.IsEnvelope(value) {
			return "", errors.New(
				"encrypted EDI communication profile secret cannot be decrypted without encryption service",
			)
		}
		return value, nil
	}
	if !encryptionservice.IsEnvelope(value) {
		return value, nil
	}
	return s.encryption.DecryptStringWithAAD(value, encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeEDICommunicationProfileItem,
		OrganizationID: profile.OrganizationID,
		BusinessUnitID: profile.BusinessUnitID,
		ResourceID:     profile.ID.String() + ":" + key,
	})
}

func actorUserID(actor *services.RequestActor) pulid.ID {
	if actor == nil {
		return pulid.Nil
	}
	return actor.UserID
}
