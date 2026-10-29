package editenderchangeservice

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidecision"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) AfterShipmentUpdate(
	ctx context.Context,
	original *shipment.Shipment,
	updated *shipment.Shipment,
	actor *permissionports.RequestActor,
) error {
	plan, err := s.planTenderChanges(ctx, original, updated, actor)
	if err != nil || plan == nil {
		return err
	}

	var recipientErrs []error
	for _, recipient := range plan.recipients {
		if err = s.createTenderChangeForRecipient(
			ctx,
			recipient,
			plan.payload,
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
	recipient *editenderchange.TenderRecipient,
	newPayload *editransfer.LoadTenderPayload,
	sourceShipmentVersion int64,
	actor *permissionports.RequestActor,
) error {
	newHash := editenderrecipient.PayloadHash(newPayload)
	diff := diffTenderPayloads(&recipient.LatestBaselinePayload, newPayload)
	if len(diff) == 0 {
		return nil
	}

	status := editenderchange.TenderChangeStatusPendingReview
	if recipient.RecipientKind == editenderchange.TenderRecipientKindExternal {
		status = editenderchange.TenderChangeStatusQueued
	}
	change := &editenderchange.TenderChange{
		BusinessUnitID:       recipient.BusinessUnitID,
		SourceOrganizationID: recipient.SourceOrganizationID,
		SourceShipmentID:     recipient.SourceShipmentID,
		RecipientID:          recipient.ID,
		RecipientKind:        recipient.RecipientKind,
		Status:               status,
		ChangeType:           editenderchange.TenderChangeTypeLoadTender,
		IdempotencyKey: fmt.Sprintf(
			"%s:%d:%s:%s",
			recipient.ID,
			sourceShipmentVersion,
			newHash,
			editenderchange.TenderChangeTypeLoadTender,
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
		editenderports.SupersedeActionableEDITenderChangesRequest{
			RecipientID:     recipient.ID,
			ExcludeChangeID: result.TenderChange.ID,
			Statuses: []editenderchange.TenderChangeStatus{
				editenderchange.TenderChangeStatusPendingReview,
				editenderchange.TenderChangeStatusQueued,
			},
		},
	); err != nil {
		return err
	}
	if recipient.RecipientKind == editenderchange.TenderRecipientKindExternal {
		return s.generateExternalTenderChange(ctx, recipient, result.TenderChange, actor)
	}
	return nil
}

func (s *Service) generateExternalTenderChange(
	ctx context.Context,
	recipient *editenderchange.TenderRecipient,
	change *editenderchange.TenderChange,
	actor *permissionports.RequestActor,
) error {
	payload := change.NewTenderPayload
	payload.PurposeCode = editransfer.LoadTenderPurposeChange
	message, err := s.outbound.GenerateDocument(ctx, &edioutbound.GenerateEDIDocumentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  recipient.SourceOrganizationID,
			BuID:   recipient.BusinessUnitID,
			UserID: actorUserID(actor),
		},
		EDIPartnerID:         recipient.EDIPartnerID,
		ShipmentID:           recipient.SourceShipmentID,
		TransactionSet:       edidocument.TransactionSet204,
		Direction:            edidocument.DocumentDirectionOutbound,
		Payload:              loadTenderDocumentPayload(&payload),
		GeneratedByID:        actorUserID(actor),
		DisableDeliveryQueue: true,
	})
	if err != nil {
		change.Status = editenderchange.TenderChangeStatusFailed
		change.FailureReason = err.Error()
		_, updateErr := s.tenderChangeRepo.UpdateTenderChange(ctx, change)
		return updateErr
	}

	change.OutboundMessageID = message.ID
	change.Status = editenderchange.TenderChangeStatusQueued
	change.FailureReason = ""
	if _, err = s.tenderChangeRepo.UpdateTenderChange(ctx, change); err != nil {
		return err
	}
	if err = s.delivery.QueueMessage(ctx, message); err != nil {
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
	req *editenderports.ListEDITenderChangesRequest,
) (*pagination.ListResult[*editenderchange.TenderChange], error) {
	return s.tenderChangeRepo.ListTenderChanges(ctx, req)
}

func (s *Service) GetTenderChange(
	ctx context.Context,
	req editenderports.GetEDITenderChangeByIDRequest,
) (*editenderchange.TenderChange, error) {
	return s.tenderChangeRepo.GetTenderChangeByID(ctx, req)
}

func (s *Service) ApplyTenderChange(
	ctx context.Context,
	req *TenderChangeActionRequest,
	actor *permissionports.RequestActor,
) (*editenderchange.TenderChange, error) {
	return s.reviewTenderChange(ctx, req, actor, editenderchange.TenderChangeStatusApplied)
}

func (s *Service) RejectTenderChange(
	ctx context.Context,
	req *TenderChangeActionRequest,
	actor *permissionports.RequestActor,
) (*editenderchange.TenderChange, error) {
	return s.reviewTenderChange(ctx, req, actor, editenderchange.TenderChangeStatusRejected)
}

func (s *Service) reviewTenderChange(
	ctx context.Context,
	req *TenderChangeActionRequest,
	actor *permissionports.RequestActor,
	status editenderchange.TenderChangeStatus,
) (*editenderchange.TenderChange, error) {
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
		editenderports.GetEDITenderChangeByIDRequest{ID: req.ChangeID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if err = edidecision.CheckTenderChangeReview(change, req.TenantInfo); err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	var applyErr error
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if status == editenderchange.TenderChangeStatusApplied {
			if applyErr = s.applyApprovedTenderChange(
				txCtx,
				req.TenantInfo,
				change,
				actor,
			); applyErr != nil {
				return applyErr
			}
		}
		edidecision.MarkTenderChangeReviewed(change, &edidecision.ChangeReviewMark{
			Applied:    status == editenderchange.TenderChangeStatusApplied,
			ReviewerID: actor.UserID,
			Reason:     req.Reason,
			At:         now,
		})
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
	change *editenderchange.TenderChange,
	actor *permissionports.RequestActor,
) error {
	recipient, err := s.tenderRecipientRepo.GetTenderRecipientByID(
		ctx,
		editenderports.GetEDITenderRecipientByIDRequest{
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
	recipient *editenderchange.TenderRecipient,
	change *editenderchange.TenderChange,
) error {
	transfer, err := s.transferRepo.GetTransferForUpdate(
		ctx,
		editenderports.GetEDITransferForUpdateRequest{
			ID:         change.InternalTransferID,
			TenantInfo: tenantInfo,
			Direction:  "inbound",
		},
	)
	if err != nil {
		return err
	}
	targetPartner, err := s.partnerRepo.GetByID(ctx, edipartnerports.GetEDIPartnerByIDRequest{
		ID:         transfer.TargetPartnerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	preview, err := s.tenderMapping.BuildMappingPreview(
		ctx,
		targetPartner,
		&change.NewTenderPayload,
	)
	if err != nil {
		return err
	}
	if len(preview.Unresolved) > 0 {
		return editendermapping.UnresolvedMappingsError(preview.Unresolved)
	}
	transfer.TenderPayload = change.NewTenderPayload
	transfer.MappingSnapshot = preview.All
	if !transfer.Status.IsFinal() {
		transfer.Status = editransfer.TransferStatusPendingApproval
	}
	if _, err = s.transferRepo.UpdateTransfer(ctx, transfer); err != nil {
		return err
	}
	return s.recipient.AdvanceBaseline(
		ctx,
		recipient,
		&change.NewTenderPayload,
		editenderchange.TenderRecipientBaselineStatusSent,
	)
}

func (s *Service) applyLinkedShipmentTenderChange(
	ctx context.Context,
	recipient *editenderchange.TenderRecipient,
	change *editenderchange.TenderChange,
	actor *permissionports.RequestActor,
) error {
	targetTenantInfo, err := tenderRecipientTenantInfo(recipient)
	if err != nil {
		return err
	}
	sourceTenantInfo := pagination.TenantInfo{
		OrgID: recipient.SourceOrganizationID,
		BuID:  recipient.BusinessUnitID,
	}
	link, err := s.shipmentLinkRepo.GetShipmentLinkByID(
		ctx,
		editenderports.GetEDIShipmentLinkByIDRequest{
			ID:         change.ShipmentLinkID,
			TenantInfo: sourceTenantInfo,
		},
	)
	if err != nil {
		return err
	}
	transfer, err := s.transferRepo.GetTransferByID(
		ctx,
		editenderports.GetEDITransferByIDRequest{
			ID:         change.InternalTransferID,
			TenantInfo: targetTenantInfo,
			Direction:  "inbound",
		},
	)
	if err != nil {
		return err
	}
	targetPartner, err := s.partnerRepo.GetByID(ctx, edipartnerports.GetEDIPartnerByIDRequest{
		ID:         transfer.TargetPartnerID,
		TenantInfo: targetTenantInfo,
	})
	if err != nil {
		return err
	}
	preview, err := s.tenderMapping.BuildMappingPreview(
		ctx,
		targetPartner,
		&change.NewTenderPayload,
	)
	if err != nil {
		return err
	}
	if len(preview.Unresolved) > 0 {
		return editendermapping.UnresolvedMappingsError(preview.Unresolved)
	}
	targetShipment, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:              link.TargetShipmentID,
		TenantInfo:      targetTenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return err
	}
	applyTenderPayloadToShipment(targetShipment, &change.NewTenderPayload, transfer, preview.All)
	if _, err = s.shipmentWriter.Update(ctx, targetShipment, actor); err != nil {
		return err
	}
	return s.recipient.AdvanceBaseline(
		ctx,
		recipient,
		&change.NewTenderPayload,
		editenderchange.TenderRecipientBaselineStatusAccepted,
	)
}

func tenderRecipientTenantInfo(
	recipient *editenderchange.TenderRecipient,
) (pagination.TenantInfo, error) {
	if recipient == nil || recipient.RecipientOrganizationID.IsNil() {
		return pagination.TenantInfo{}, errortypes.NewValidationError(
			"recipientId",
			errortypes.ErrRequired,
			"Internal tender recipient tenant is required",
		)
	}
	return pagination.TenantInfo{
		OrgID: recipient.RecipientOrganizationID,
		BuID:  recipient.BusinessUnitID,
	}, nil
}

func applyTenderPayloadToShipment(
	target *shipment.Shipment,
	payload *editransfer.LoadTenderPayload,
	transfer *editransfer.EDITransfer,
	resolutions []editransfer.MappingResolution,
) {
	mappings := editendermapping.ResolutionIndex(resolutions)
	target.ServiceTypeID = mustMappedID(
		mappings,
		edi.MappingEntityTypeServiceType,
		payload.ServiceTypeID,
	)
	target.ShipmentTypeID = editendermapping.OptionalMappedID(
		mappings,
		edi.MappingEntityTypeShipmentType,
		payload.ShipmentTypeID,
	)
	target.CustomerID = mustMappedID(mappings, edi.MappingEntityTypeCustomer, payload.CustomerID)
	target.FormulaTemplateID = editendermapping.TenderedTemplate(payload, mustMappedID(
		mappings,
		edi.MappingEntityTypeFormulaTemplate,
		payload.FormulaTemplateID,
	))
	target.TenderStatus = new(shipmentvocab.TenderStatusAccepted)
	target.BOL = payload.BOL
	target.Pieces = payload.Pieces
	target.Weight = payload.Weight
	target.TemperatureMin = payload.TemperatureMin
	target.TemperatureMax = payload.TemperatureMax
	target.FreightChargeAmount = payload.FreightChargeAmount
	target.OtherChargeAmount = payload.OtherChargeAmount
	target.TotalChargeAmount = payload.TotalChargeAmount
	target.Moves = mappedTenderMoves(target, payload, transfer, mappings)
	target.Commodities = mappedTenderCommodities(target, payload, transfer, mappings)
	target.AdditionalCharges = mappedTenderCharges(target, payload, transfer, mappings)
}

func mappedTenderMoves(
	target *shipment.Shipment,
	payload *editransfer.LoadTenderPayload,
	transfer *editransfer.EDITransfer,
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
			Status:         shipmentvocab.MoveStatusNew,
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
	move editransfer.LoadTenderMove,
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
			Status:               shipmentvocab.StopStatusNew,
			Type:                 shipmentvocab.StopType(stop.Type),
			ScheduleType:         shipmentvocab.StopScheduleType(stop.ScheduleType),
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
	payload *editransfer.LoadTenderPayload,
	transfer *editransfer.EDITransfer,
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
	payload *editransfer.LoadTenderPayload,
	transfer *editransfer.EDITransfer,
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
	targetID, ok := editendermapping.MappedID(mappings, entityType, sourceID)
	if !ok {
		return pulid.Nil
	}
	return targetID
}

func diffTenderPayloads(
	previous *editransfer.LoadTenderPayload,
	next *editransfer.LoadTenderPayload,
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

func tenderPayloadMap(payload *editransfer.LoadTenderPayload) map[string]any {
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

func loadTenderDocumentPayload(payload *editransfer.LoadTenderPayload) *edipayload.DocumentPayload {
	documentPayload := edipayload.NewLoadTenderDocumentPayload(*payload)
	documentPayload.PurposeCode = editransfer.LoadTenderPurposeChange
	return &documentPayload
}

func actorUserID(actor *permissionports.RequestActor) pulid.ID {
	if actor == nil {
		return pulid.Nil
	}
	return actor.UserID
}
