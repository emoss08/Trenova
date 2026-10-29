package editenderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

//nolint:funlen // Tender planning checks each partner, connection, and profile explicitly.
func (s *Service) PlanLoadTender(
	ctx context.Context,
	req *SubmitLoadTenderRequest,
) (*LoadTenderPlan, error) {
	if err := validateSubmitLoadTender(req); err != nil {
		return nil, err
	}

	sourceShipment, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         req.SourceShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
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
		edipartnerports.GetActiveEDIConnectionForPartnerRequest{
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
		edipartnerports.GetReciprocalInternalPartnerRequest{
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
		edipartnerports.GetActiveEDICommunicationProfileByPartnerRequest{
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
	targetTenant := pagination.TenantInfo{
		OrgID: targetPartner.OrganizationID,
		BuID:  targetPartner.BusinessUnitID,
	}
	if _, err = s.profileRepo.GetActiveProfileByPartner(
		dbscope.WithTenant(ctx, targetTenant.DBTenant()),
		edipartnerports.GetActiveEDICommunicationProfileByPartnerRequest{
			PartnerID:  targetPartner.ID,
			TenantInfo: targetTenant,
			Method:     edi.ConnectionMethodInternal,
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

	payload := edipayloadbuild.BuildTenderPayload(sourceShipment)
	preview, err := s.tenderMapping.BuildMappingPreview(ctx, targetPartner, payload)
	if err != nil {
		return nil, err
	}

	status := editransfer.TransferStatusPendingApproval
	if len(preview.Unresolved) > 0 {
		status = editransfer.TransferStatusMappingRequired
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
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	plan, err := s.PlanLoadTender(ctx, req)
	if err != nil {
		return nil, err
	}

	entity := &editransfer.EDITransfer{
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

	var created *editransfer.EDITransfer
	if s.db == nil {
		created, err = s.transferRepo.CreateTransfer(ctx, entity)
		if err != nil {
			return nil, err
		}
		if err = s.recipient.UpsertInternal(
			ctx,
			created,
			nil,
			editenderchange.TenderRecipientBaselineStatusSent,
		); err != nil {
			return nil, err
		}
	} else {
		err = s.db.Transact(
			ctx,
			kernelports.TxOptions{},
			func(txCtx context.Context) error {
				lockedShipment, lockErr := s.shipments.Lock(
					txCtx,
					req.SourceShipmentID,
					req.TenantInfo,
				)
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
				if err = s.recipient.UpsertInternal(
					txCtx,
					created,
					nil,
					editenderchange.TenderRecipientBaselineStatusSent,
				); err != nil {
					return err
				}

				if err = s.shipments.SetTenderStatus(
					txCtx,
					req.SourceShipmentID,
					req.TenantInfo,
					shipment.TenderStatusTendered,
				); err != nil {
					return err
				}

				return s.shipments.CreateSystemComment(
					txCtx,
					req.SourceShipmentID,
					req.TenantInfo,
					"EDI load tender submitted.",
					map[string]any{"transferId": created.ID},
				)
			},
		)
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
		return s.partnerRepo.GetByID(ctx, edipartnerports.GetEDIPartnerByIDRequest{
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
		edipartnerports.ListEDIPartnersByCustomerIDsRequest{
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
