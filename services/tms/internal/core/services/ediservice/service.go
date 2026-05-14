package ediservice

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger          *zap.Logger
	PartnerRepo     repositories.EDIPartnerRepository
	TransferRepo    repositories.EDILoadTenderTransferRepository
	ShipmentSvc     services.ShipmentService
	WorkflowStarter services.WorkflowStarter
	AuditService    services.AuditService
	Validator       *Validator
	DB              coreports.DBConnection
}

type Service struct {
	l               *zap.Logger
	partnerRepo     repositories.EDIPartnerRepository
	transferRepo    repositories.EDILoadTenderTransferRepository
	shipmentSvc     services.ShipmentService
	workflowStarter services.WorkflowStarter
	auditService    services.AuditService
	validator       *Validator
	db              coreports.DBConnection
}

func New(p Params) *Service {
	return &Service{
		l:               p.Logger.Named("service.edi"),
		partnerRepo:     p.PartnerRepo,
		transferRepo:    p.TransferRepo,
		shipmentSvc:     p.ShipmentSvc,
		workflowStarter: p.WorkflowStarter,
		auditService:    p.AuditService,
		validator:       p.Validator,
		db:              p.DB,
	}
}

func (s *Service) ListPartners(
	ctx context.Context,
	req *repositories.ListEDIPartnersRequest,
) (*pagination.ListResult[*edi.EDIPartner], error) {
	return s.partnerRepo.List(ctx, req)
}

func (s *Service) SelectPartnerOptions(
	ctx context.Context,
	req *repositories.EDIPartnerSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartner], error) {
	return s.partnerRepo.SelectOptions(ctx, req)
}

func (s *Service) GetPartner(
	ctx context.Context,
	req repositories.GetEDIPartnerByIDRequest,
) (*edi.EDIPartner, error) {
	return s.partnerRepo.GetByID(ctx, req)
}

func (s *Service) CreatePartner(
	ctx context.Context,
	entity *edi.EDIPartner,
	actor *services.RequestActor,
) (*edi.EDIPartner, error) {
	normalizePartnerForCreate(entity)
	if multiErr := s.validator.ValidatePartner(entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.partnerRepo.Create(ctx, entity)
	if err != nil {
		return nil, mapEDIPartnerConstraint(err)
	}

	s.logAction(created, actor, permission.OpCreate, nil, created, "EDI partner created")
	return created, nil
}

func (s *Service) UpdatePartner(
	ctx context.Context,
	entity *edi.EDIPartner,
	actor *services.RequestActor,
) (*edi.EDIPartner, error) {
	if multiErr := s.validator.ValidatePartner(entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.partnerRepo.Update(ctx, entity)
	if err != nil {
		return nil, mapEDIPartnerConstraint(err)
	}

	s.logAction(updated, actor, permission.OpUpdate, original, updated, "EDI partner updated")
	return updated, nil
}

func (s *Service) CreateInternalPartnerPair(
	ctx context.Context,
	req *CreateInternalPartnerPairRequest,
	actor *services.RequestActor,
) (*edi.InternalPartnerPair, error) {
	if req == nil || req.TargetOrganizationID.IsNil() {
		return nil, errortypes.NewValidationError(
			"targetOrganizationId",
			errortypes.ErrRequired,
			"Target organization is required",
		)
	}
	if req.TargetOrganizationID == req.TenantInfo.OrgID {
		return nil, errortypes.NewValidationError(
			"targetOrganizationId",
			errortypes.ErrInvalid,
			"Target organization must be different from the current organization",
		)
	}

	sourcePartner := buildInternalPairPartner(
		req,
		req.TenantInfo.OrgID,
		req.TargetOrganizationID,
		true,
	)
	targetPartner := buildInternalPairPartner(
		req,
		req.TargetOrganizationID,
		req.TenantInfo.OrgID,
		false,
	)

	if multiErr := s.validator.ValidatePartner(sourcePartner); multiErr != nil {
		return nil, multiErr
	}
	if multiErr := s.validator.ValidatePartner(targetPartner); multiErr != nil {
		return nil, multiErr
	}

	pair, err := s.partnerRepo.CreateInternalPair(
		ctx,
		&repositories.CreateInternalPartnerPairRequest{
			SourcePartner:        sourcePartner,
			TargetPartner:        targetPartner,
			SourceOrganizationID: req.TenantInfo.OrgID,
			TargetOrganizationID: req.TargetOrganizationID,
			BusinessUnitID:       req.TenantInfo.BuID,
			TenantInfo:           req.TenantInfo,
		},
	)
	if err != nil {
		return nil, mapEDIPartnerConstraint(err)
	}

	s.logAction(
		pair.SourcePartner,
		actor,
		permission.OpCreate,
		nil,
		pair.SourcePartner,
		"EDI internal partner pair source created",
	)
	s.logAction(
		pair.TargetPartner,
		actor,
		permission.OpCreate,
		nil,
		pair.TargetPartner,
		"EDI internal partner pair target created",
	)
	return pair, nil
}

func (s *Service) GetMappingProfile(
	ctx context.Context,
	req repositories.GetMappingProfileRequest,
) (*edi.EDIMappingProfile, error) {
	return s.partnerRepo.GetMappingProfile(ctx, req)
}

func (s *Service) SaveMappingProfile(
	ctx context.Context,
	req *repositories.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	if multiErr := s.validator.ValidateMappingItems(req.Items); multiErr != nil {
		return nil, multiErr
	}

	return s.partnerRepo.SaveMappingItems(ctx, req)
}

func (s *Service) DeleteMappingItem(
	ctx context.Context,
	req repositories.DeleteMappingItemRequest,
) error {
	return s.partnerRepo.DeleteMappingItem(ctx, req)
}

func (s *Service) SubmitLoadTender(
	ctx context.Context,
	req *SubmitLoadTenderRequest,
	actor *services.RequestActor,
) (*edi.EDITransfer, error) {
	if err := validateSubmitLoadTender(req); err != nil {
		return nil, err
	}

	sourcePartner, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID:         req.EDIPartnerID,
		TenantInfo: req.TenantInfo,
	})
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

	payload := buildTenderPayload(sourceShipment)
	preview, err := s.buildMappingPreview(ctx, targetPartner, payload, nil)
	if err != nil {
		return nil, err
	}

	status := edi.TransferStatusPendingApproval
	if len(preview.Unresolved) > 0 {
		status = edi.TransferStatusMappingRequired
	}

	entity := &edi.EDITransfer{
		SourceOrganizationID: req.TenantInfo.OrgID,
		SourceBusinessUnitID: req.TenantInfo.BuID,
		TargetOrganizationID: targetPartner.OrganizationID,
		TargetBusinessUnitID: req.TenantInfo.BuID,
		SourcePartnerID:      sourcePartner.ID,
		TargetPartnerID:      targetPartner.ID,
		SourceShipmentID:     sourceShipment.ID,
		Status:               status,
		TenderPayload:        payload,
		MappingSnapshot:      preview.All,
		SubmittedByID:        actor.UserID,
	}

	created, err := s.transferRepo.CreateTransfer(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAction(created, actor, permission.OpCreate, nil, created, "EDI load tender submitted")
	return created, nil
}

func (s *Service) ListInboundTransfers(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	return s.transferRepo.ListInbound(ctx, req)
}

func (s *Service) ListOutboundTransfers(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	return s.transferRepo.ListOutbound(ctx, req)
}

func (s *Service) GetTransfer(
	ctx context.Context,
	req repositories.GetEDITransferByIDRequest,
) (*edi.EDITransfer, error) {
	return s.transferRepo.GetTransferByID(ctx, req)
}

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

	return s.buildMappingPreview(ctx, targetPartner, transfer.TenderPayload, nil)
}

func (s *Service) ApproveTransfer(
	ctx context.Context,
	req *ApproveTransferRequest,
	actor *services.RequestActor,
) (*edi.EDITransfer, error) {
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
		if !transfer.Status.IsActionable() {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"EDI transfer cannot be approved while finalized or processing",
			)
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

		preview, err = s.buildMappingPreview(txCtx, targetPartner, transfer.TenderPayload, nil)
		if err != nil {
			return err
		}
		if len(preview.Unresolved) > 0 {
			return unresolvedMappingsError(preview.Unresolved)
		}

		now := timeutils.NowUnix()
		transfer.Status = edi.TransferStatusProcessing
		transfer.TargetBusinessUnitID = req.TenantInfo.BuID
		transfer.MappingSnapshot = preview.All
		transfer.ApprovedByID = actor.UserID
		transfer.ApprovedAt = &now
		transfer.ProcessingStartedAt = &now
		transfer.ApprovalWorkflowID = buildLoadTenderApprovalWorkflowID(transfer.ID)
		transfer.ApprovalWorkflowRunID = ""
		transfer.FailureReason = ""

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

func (s *Service) ProcessLoadTenderApproval(
	ctx context.Context,
	payload *ApproveLoadTenderTransferWorkflowPayload,
) (*ApproveLoadTenderTransferWorkflowResult, error) {
	result := new(ApproveLoadTenderTransferWorkflowResult)
	var processingErr error

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

		switch transfer.Status {
		case edi.TransferStatusRejected, edi.TransferStatusCanceled, edi.TransferStatusFailed:
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

		preview, err := s.buildMappingPreview(txCtx, targetPartner, transfer.TenderPayload, nil)
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

		updated, err := s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
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
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Rejection reason is required",
		)
	}

	transfer, err := s.transferRepo.GetTransferByID(ctx, repositories.GetEDITransferByIDRequest{
		ID:         req.TransferID,
		TenantInfo: req.TenantInfo,
		Direction:  "inbound",
	})
	if err != nil {
		return nil, err
	}
	if !transfer.Status.IsActionable() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer cannot be rejected while finalized or processing",
		)
	}

	now := timeutils.NowUnix()
	original := *transfer
	transfer.Status = edi.TransferStatusRejected
	transfer.RejectionReason = reason
	transfer.RejectedByID = actor.UserID
	transfer.RejectedAt = &now

	updated, err := s.transferRepo.UpdateTransfer(ctx, transfer)
	if err != nil {
		return nil, err
	}

	s.logAction(updated, actor, permission.OpUpdate, &original, updated, "EDI load tender rejected")
	return updated, nil
}

func (s *Service) CancelTransfer(
	ctx context.Context,
	req *CancelTransferRequest,
	actor *services.RequestActor,
) (*edi.EDITransfer, error) {
	transfer, err := s.transferRepo.GetTransferByID(ctx, repositories.GetEDITransferByIDRequest{
		ID:         req.TransferID,
		TenantInfo: req.TenantInfo,
		Direction:  "outbound",
	})
	if err != nil {
		return nil, err
	}
	if !transfer.Status.IsActionable() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer cannot be canceled while finalized or processing",
		)
	}

	now := timeutils.NowUnix()
	original := *transfer
	transfer.Status = edi.TransferStatusCanceled
	transfer.CanceledByID = actor.UserID
	transfer.CanceledAt = &now

	updated, err := s.transferRepo.UpdateTransfer(ctx, transfer)
	if err != nil {
		return nil, err
	}

	s.logAction(updated, actor, permission.OpUpdate, &original, updated, "EDI load tender canceled")
	return updated, nil
}

func (s *Service) buildMappingPreview(
	ctx context.Context,
	partner *edi.EDIPartner,
	payload edi.LoadTenderPayload,
	overrides []*edi.EDIMappingProfileItem,
) (*MappingPreview, error) {
	required := payload.RequiredMappingEntityIDs
	sourceIDs := flattenRequiredIDs(required)
	items, err := s.partnerRepo.GetMappingItems(ctx, repositories.GetMappingItemsRequest{
		PartnerID: partner.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: partner.OrganizationID,
			BuID:  partner.BusinessUnitID,
		},
		EntityTypes: requiredEntityTypes(required),
		SourceIDs:   sourceIDs,
	})
	if err != nil {
		return nil, err
	}

	index := mappingIndex(items)
	for _, item := range overrides {
		if item == nil {
			continue
		}
		if _, ok := index[item.EntityType]; !ok {
			index[item.EntityType] = map[pulid.ID]*edi.EDIMappingProfileItem{}
		}
		index[item.EntityType][item.SourceID] = item
	}

	all := make([]edi.MappingResolution, 0, len(sourceIDs))
	for _, entityType := range requiredEntityTypes(required) {
		ids := append([]pulid.ID(nil), required[entityType]...)
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, sourceID := range ids {
			resolution := edi.MappingResolution{
				EntityType: entityType,
				SourceID:   sourceID,
			}
			if item := index[entityType][sourceID]; item != nil && item.TargetID.IsNotNil() {
				resolution.SourceLabel = item.SourceLabel
				resolution.TargetID = item.TargetID
				resolution.TargetLabel = item.TargetLabel
				resolution.Resolved = true
			}
			all = append(all, resolution)
		}
	}

	resolved := make([]edi.MappingResolution, 0, len(all))
	unresolved := make([]edi.MappingResolution, 0)
	for _, resolution := range all {
		if resolution.Resolved {
			resolved = append(resolved, resolution)
			continue
		}
		unresolved = append(unresolved, resolution)
	}

	return &MappingPreview{
		Resolved:   resolved,
		Unresolved: unresolved,
		All:        all,
	}, nil
}

func (s *Service) buildTargetShipment(
	transfer *edi.EDITransfer,
	businessUnitID pulid.ID,
	resolutions []edi.MappingResolution,
) (*shipment.Shipment, error) {
	mappings := resolutionIndex(resolutions)
	payload := transfer.TenderPayload

	customerID, ok := mappedID(mappings, edi.MappingEntityTypeCustomer, payload.CustomerID)
	if !ok {
		return nil, fmt.Errorf("customer mapping is missing for %s", payload.CustomerID)
	}
	serviceTypeID, ok := mappedID(mappings, edi.MappingEntityTypeServiceType, payload.ServiceTypeID)
	if !ok {
		return nil, fmt.Errorf("service type mapping is missing for %s", payload.ServiceTypeID)
	}
	formulaTemplateID, ok := mappedID(
		mappings,
		edi.MappingEntityTypeFormulaTemplate,
		payload.FormulaTemplateID,
	)
	if !ok {
		return nil, fmt.Errorf(
			"formula template mapping is missing for %s",
			payload.FormulaTemplateID,
		)
	}

	target := &shipment.Shipment{
		BusinessUnitID: businessUnitID,
		OrganizationID: transfer.TargetOrganizationID,
		ServiceTypeID:  serviceTypeID,
		ShipmentTypeID: optionalMappedID(
			mappings,
			edi.MappingEntityTypeShipmentType,
			payload.ShipmentTypeID,
		),
		CustomerID:          customerID,
		FormulaTemplateID:   formulaTemplateID,
		Status:              shipment.StatusNew,
		BOL:                 payload.BOL,
		Pieces:              payload.Pieces,
		Weight:              payload.Weight,
		TemperatureMin:      payload.TemperatureMin,
		TemperatureMax:      payload.TemperatureMax,
		FreightChargeAmount: payload.FreightChargeAmount,
		OtherChargeAmount:   payload.OtherChargeAmount,
		BaseRate:            payload.BaseRate,
		TotalChargeAmount:   payload.TotalChargeAmount,
		RatingUnit:          payload.RatingUnit,
		Moves:               make([]*shipment.ShipmentMove, 0, len(payload.Moves)),
		Commodities:         make([]*shipment.ShipmentCommodity, 0, len(payload.Commodities)),
		AdditionalCharges:   make([]*shipment.AdditionalCharge, 0, len(payload.AdditionalCharges)),
	}

	for _, move := range payload.Moves {
		targetMove := &shipment.ShipmentMove{
			BusinessUnitID: businessUnitID,
			OrganizationID: transfer.TargetOrganizationID,
			Status:         shipment.MoveStatusNew,
			Loaded:         move.Loaded,
			Sequence:       move.Sequence,
			Distance:       move.Distance,
			Stops:          make([]*shipment.Stop, 0, len(move.Stops)),
		}
		for _, stop := range move.Stops {
			locationID, ok := mappedID(mappings, edi.MappingEntityTypeLocation, stop.LocationID)
			if !ok {
				return nil, fmt.Errorf("location mapping is missing for %s", stop.LocationID)
			}
			targetMove.Stops = append(targetMove.Stops, &shipment.Stop{
				BusinessUnitID:       businessUnitID,
				OrganizationID:       transfer.TargetOrganizationID,
				LocationID:           locationID,
				Status:               shipment.StopStatusNew,
				Type:                 shipment.StopType(stop.Type),
				ScheduleType:         shipment.StopScheduleType(stop.ScheduleType),
				Sequence:             stop.Sequence,
				Pieces:               stop.Pieces,
				Weight:               stop.Weight,
				ScheduledWindowStart: stop.ScheduledWindowStart,
				ScheduledWindowEnd:   stop.ScheduledWindowEnd,
				AddressLine:          stop.AddressLine,
			})
		}
		target.Moves = append(target.Moves, targetMove)
	}

	for _, commodity := range payload.Commodities {
		commodityID, ok := mappedID(mappings, edi.MappingEntityTypeCommodity, commodity.CommodityID)
		if !ok {
			return nil, fmt.Errorf("commodity mapping is missing for %s", commodity.CommodityID)
		}
		target.Commodities = append(target.Commodities, &shipment.ShipmentCommodity{
			BusinessUnitID: businessUnitID,
			OrganizationID: transfer.TargetOrganizationID,
			CommodityID:    commodityID,
			Weight:         commodity.Weight,
			Pieces:         commodity.Pieces,
		})
	}

	for _, charge := range payload.AdditionalCharges {
		chargeID, ok := mappedID(
			mappings,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
		)
		if !ok {
			return nil, fmt.Errorf(
				"accessorial charge mapping is missing for %s",
				charge.AccessorialChargeID,
			)
		}
		target.AdditionalCharges = append(target.AdditionalCharges, &shipment.AdditionalCharge{
			BusinessUnitID:      businessUnitID,
			OrganizationID:      transfer.TargetOrganizationID,
			AccessorialChargeID: chargeID,
			Method:              accessorialcharge.Method(charge.Method),
			Amount:              charge.Amount,
			Unit:                charge.Unit,
		})
	}

	return target, nil
}

func buildInternalPairPartner(
	req *CreateInternalPartnerPairRequest,
	organizationID pulid.ID,
	internalOrganizationID pulid.ID,
	sourceFacing bool,
) *edi.EDIPartner {
	code := req.TargetCode
	name := req.TargetName
	description := req.TargetDescription
	contactName := req.TargetContactName
	contactEmail := req.TargetContactEmail
	contactPhone := req.TargetContactPhone
	enabledInbound := req.TargetEnabledInbound
	enabledOutbound := req.TargetEnabledOutbound
	settings := req.TargetSettings
	if sourceFacing {
		code = req.SourceCode
		name = req.SourceName
		description = req.SourceDescription
		contactName = req.SourceContactName
		contactEmail = req.SourceContactEmail
		contactPhone = req.SourceContactPhone
		enabledInbound = req.SourceEnabledInbound
		enabledOutbound = req.SourceEnabledOutbound
		settings = req.SourceSettings
	}
	if settings == nil {
		settings = map[string]any{}
	}

	return &edi.EDIPartner{
		BusinessUnitID:         req.TenantInfo.BuID,
		OrganizationID:         organizationID,
		Kind:                   edi.PartnerKindInternal,
		Status:                 domaintypes.StatusActive,
		Code:                   strings.TrimSpace(code),
		Name:                   strings.TrimSpace(name),
		Description:            strings.TrimSpace(description),
		InternalOrganizationID: internalOrganizationID,
		Country:                "US",
		ContactName:            strings.TrimSpace(contactName),
		ContactEmail:           strings.TrimSpace(contactEmail),
		ContactPhone:           strings.TrimSpace(contactPhone),
		EnabledForInbound:      enabledInbound,
		EnabledForOutbound:     enabledOutbound,
		Settings:               settings,
	}
}

func mapEDIPartnerConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case "idx_edi_partners_code_org":
		multiErr.Add("code", errortypes.ErrDuplicate, "EDI partner with this code already exists")
	case "idx_edi_partners_name_org":
		multiErr.Add("name", errortypes.ErrDuplicate, "EDI partner with this name already exists")
	case "idx_edi_partners_internal_relationship_org_bu":
		multiErr.Add(
			"internalOrganizationId",
			errortypes.ErrDuplicate,
			"An internal EDI partner already exists for this target organization",
		)
	default:
		return err
	}

	return multiErr
}

func (s *Service) logAction(
	entity interface {
		GetID() pulid.ID
		GetOrganizationID() pulid.ID
		GetBusinessUnitID() pulid.ID
	},
	actor *services.RequestActor,
	operation permission.Operation,
	previous any,
	current any,
	comment string,
) {
	if s.auditService == nil || actor == nil || entity == nil {
		return
	}

	auditActor := actor.AuditActor()
	params := &services.LogActionParams{
		Resource:       permission.ResourceEDI,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		CurrentState:   jsonutils.MustToJSON(current),
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	if err := s.auditService.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Warn("failed to log EDI audit action", zap.Error(err))
	}
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
	if req.EDIPartnerID.IsNil() {
		multiErr.Add("ediPartnerId", errortypes.ErrRequired, "EDI partner ID is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func normalizePartnerForCreate(entity *edi.EDIPartner) {
	if entity == nil {
		return
	}
	if entity.Kind == "" {
		entity.Kind = edi.PartnerKindExternal
	}
	if entity.Status == "" {
		entity.Status = domaintypes.StatusActive
	}
	if entity.Country == "" {
		entity.Country = "US"
	}
	entity.EnabledForInbound = true
	entity.EnabledForOutbound = true
	if entity.Settings == nil {
		entity.Settings = map[string]any{}
	}
}
