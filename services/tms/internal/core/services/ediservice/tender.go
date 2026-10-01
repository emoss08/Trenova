package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

//nolint:funlen // Tender planning checks each partner, connection, and profile explicitly.
func (s *Service) PlanLoadTender(
	ctx context.Context,
	req *SubmitLoadTenderRequest,
) (*LoadTenderPlan, error) {
	if err := validateSubmitLoadTender(req); err != nil {
		return nil, err
	}

	sourceShipment, err := s.shipmentSvc.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.SourceShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	sourcePartner, err := s.resolveTenderSourcePartner(ctx, req, sourceShipment)
	if err != nil {
		return nil, err
	}
	if sourcePartner.Kind != edi.PartnerKindInternal ||
		sourcePartner.InternalOrganizationID.IsNil() {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalid,
			"Load tender transfers require an internal EDI partner",
		)
	}
	if !sourcePartner.EnabledForOutbound {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"EDI partner is not enabled for outbound transfers",
		)
	}

	connection, err := s.connectionRepo.GetActiveConnectionForPartner(
		ctx,
		repositories.GetActiveEDIConnectionForPartnerRequest{
			PartnerID:  sourcePartner.ID,
			TenantInfo: req.TenantInfo,
			Method:     edi.ConnectionMethodInternal,
		},
	)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"EDI partner does not have an active internal EDI connection",
		)
	}
	if !connection.Capabilities.LoadTenderOutbound {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"EDI connection is not enabled for outbound load tenders",
		)
	}

	targetPartner, err := s.partnerRepo.GetReciprocalInternalPartner(
		ctx,
		repositories.GetReciprocalInternalPartnerRequest{
			SourceOrganizationID: req.TenantInfo.OrgID,
			TargetOrganizationID: sourcePartner.InternalOrganizationID,
			BusinessUnitID:       req.TenantInfo.BuID,
		},
	)
	if err != nil {
		return nil, err
	}
	if !targetPartner.EnabledForInbound {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"Target EDI partner is not enabled for inbound transfers",
		)
	}
	if !connection.Capabilities.LoadTenderInbound {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"EDI connection is not enabled for inbound load tenders",
		)
	}
	if _, err = s.profileRepo.GetActiveProfileByPartner(
		ctx,
		repositories.GetActiveEDICommunicationProfileByPartnerRequest{
			PartnerID:  sourcePartner.ID,
			TenantInfo: req.TenantInfo,
			Method:     edi.ConnectionMethodInternal,
		},
	); err != nil {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"EDI partner does not have an active internal communication profile",
		)
	}
	if _, err = s.profileRepo.GetActiveProfileByPartner(
		ctx,
		repositories.GetActiveEDICommunicationProfileByPartnerRequest{
			PartnerID: targetPartner.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: targetPartner.OrganizationID,
				BuID:  targetPartner.BusinessUnitID,
			},
			Method: edi.ConnectionMethodInternal,
		},
	); err != nil {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"Target EDI partner does not have an active internal communication profile",
		)
	}

	if err = validateTenderEligibility(sourceShipment); err != nil {
		return nil, err
	}

	payload := buildTenderPayload(sourceShipment)
	preview, err := s.buildMappingPreview(ctx, targetPartner, payload)
	if err != nil {
		return nil, err
	}

	status := edi.TransferStatusPendingApproval
	if len(preview.Unresolved) > 0 {
		status = edi.TransferStatusMappingRequired
	}

	return &LoadTenderPlan{
		SourceShipment: sourceShipment,
		SourcePartner:  sourcePartner,
		TargetPartner:  targetPartner,
		Payload:        payload,
		Mapping:        preview,
		Status:         status,
	}, nil
}

//nolint:nestif // Tender submission keeps the transaction and non-transaction paths explicit.
func (s *Service) SubmitLoadTender(
	ctx context.Context,
	req *SubmitLoadTenderRequest,
	actor *services.RequestActor,
) (*edi.EDITransfer, error) {
	plan, err := s.PlanLoadTender(ctx, req)
	if err != nil {
		return nil, err
	}

	entity := &edi.EDITransfer{
		SourceOrganizationID: req.TenantInfo.OrgID,
		SourceBusinessUnitID: req.TenantInfo.BuID,
		TargetOrganizationID: plan.TargetPartner.OrganizationID,
		TargetBusinessUnitID: req.TenantInfo.BuID,
		SourcePartnerID:      plan.SourcePartner.ID,
		TargetPartnerID:      plan.TargetPartner.ID,
		SourceShipmentID:     plan.SourceShipment.ID,
		Status:               plan.Status,
		TenderPayload:        plan.Payload,
		MappingSnapshot:      plan.Mapping.All,
		SubmittedByID:        actor.UserID,
	}

	var created *edi.EDITransfer
	if s.db == nil {
		created, err = s.transferRepo.CreateTransfer(ctx, entity)
		if err != nil {
			return nil, err
		}
		if err = s.upsertInternalTenderRecipient(
			ctx,
			created,
			nil,
			edi.TenderRecipientBaselineStatusSent,
		); err != nil {
			return nil, err
		}
	} else {
		err = s.db.WithTx(ctx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
			lockedShipment, lockErr := s.lockShipment(txCtx, req.SourceShipmentID, req.TenantInfo)
			if lockErr != nil {
				return lockErr
			}
			if validateErr := validateTenderEligibility(lockedShipment); validateErr != nil {
				return validateErr
			}

			created, err = s.transferRepo.CreateTransfer(txCtx, entity)
			if err != nil {
				return err
			}
			if err = s.upsertInternalTenderRecipient(
				txCtx,
				created,
				nil,
				edi.TenderRecipientBaselineStatusSent,
			); err != nil {
				return err
			}

			if err = s.setShipmentTenderStatus(
				txCtx,
				req.SourceShipmentID,
				req.TenantInfo,
				shipment.TenderStatusTendered,
			); err != nil {
				return err
			}

			return s.createSystemShipmentComment(
				txCtx,
				req.SourceShipmentID,
				req.TenantInfo,
				"EDI load tender submitted.",
				map[string]any{"transferId": created.ID},
			)
		})
	}
	if err != nil {
		return nil, err
	}

	s.logAction(created, actor, permission.OpCreate, nil, created, "EDI load tender submitted")
	return created, nil
}

func (s *Service) resolveTenderSourcePartner(
	ctx context.Context,
	req *SubmitLoadTenderRequest,
	sourceShipment *shipment.Shipment,
) (*edi.EDIPartner, error) {
	if req.EDIPartnerID.IsNotNil() {
		return s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
			ID:         req.EDIPartnerID,
			TenantInfo: req.TenantInfo,
		})
	}

	if sourceShipment.CustomerID.IsNil() {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"Shipment has no customer to resolve an EDI partner from",
		)
	}

	partners, err := s.partnerRepo.ListInternalOutboundPartnersByCustomerIDs(
		ctx,
		repositories.ListEDIPartnersByCustomerIDsRequest{
			CustomerIDs: []pulid.ID{sourceShipment.CustomerID},
			TenantInfo:  req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(partners) == 0 {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrInvalidOperation,
			"Customer is not linked to an internal EDI partner enabled for outbound load tenders",
		)
	}

	return partners[0], nil
}

func validateTenderEligibility(entity *shipment.Shipment) error {
	if entity == nil {
		return errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment is required",
		)
	}

	eligibleTenderStatus := entity.TenderStatus == nil ||
		*entity.TenderStatus == shipment.TenderStatusRejected ||
		*entity.TenderStatus == shipment.TenderStatusExpired ||
		*entity.TenderStatus == shipment.TenderStatusCanceled
	if entity.Status == shipment.StatusNew && eligibleTenderStatus {
		return nil
	}

	return errortypes.NewValidationError(
		"shipmentId",
		errortypes.ErrInvalidOperation,
		"Only New shipments without an active or accepted tender can be tendered.",
	)
}

func (s *Service) lockShipment(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, s.db, func(ctx context.Context) (*shipment.Shipment, error) {
		entity := new(shipment.Shipment)
		err := s.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where("sp.id = ?", shipmentID).
			Where("sp.organization_id = ?", tenantInfo.OrgID).
			Where("sp.business_unit_id = ?", tenantInfo.BuID).
			For("UPDATE").
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Shipment")
		}

		return entity, nil
	})
}

func (s *Service) setShipmentTenderStatus(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	status shipment.TenderStatus,
) error {
	return dbtx.WriteErr(ctx, s.db, func(ctx context.Context) error {
		db := s.db.DBForContext(ctx)

		results, err := db.
			NewUpdate().
			Model((*shipment.Shipment)(nil)).
			Set("tender_status = ?", status).
			Set("version = version + 1").
			Set("updated_at = "+dbdialect.NowEpochFromBun(db)).
			Where("id = ?", shipmentID).
			Where("organization_id = ?", tenantInfo.OrgID).
			Where("business_unit_id = ?", tenantInfo.BuID).
			Exec(ctx)
		if err != nil {
			return err
		}

		return dberror.CheckRowsAffected(results, "Shipment", shipmentID.String())
	})
}

func (s *Service) createSystemShipmentComment(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	comment string,
	metadata map[string]any,
) error {
	if s.shipmentCommentRepo == nil || s.userRepo == nil {
		return nil
	}

	systemUser, err := s.userRepo.GetSystemUser(ctx, "id")
	if err != nil {
		return err
	}

	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata[shipment.CommentMetadataOrigin] = shipment.CommentOriginEDI

	_, err = s.shipmentCommentRepo.Create(ctx, &shipment.ShipmentComment{
		ShipmentID:       shipmentID,
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		UserID:           systemUser.ID,
		Comment:          comment,
		Type:             shipment.CommentTypeStatusUpdate,
		Visibility:       shipment.CommentVisibilityOperations,
		Priority:         shipment.CommentPriorityNormal,
		Source:           shipment.CommentSourceSystem,
		Metadata:         metadata,
		MentionedUserIDs: []pulid.ID{},
	})
	return err
}

func validateSubmitLoadTender(req *SubmitLoadTenderRequest) error {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("", errortypes.ErrRequired, "Load tender request is required")
		return multiErr
	}
	if req.SourceShipmentID.IsNil() {
		multiErr.Add("sourceShipmentId", errortypes.ErrRequired, "Source shipment ID is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
