//nolint:gocritic // EDI service request and DI value shapes are stable application contracts.
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
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
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

	Logger              *zap.Logger
	PartnerRepo         repositories.EDIPartnerRepository
	MappingProfileRepo  repositories.EDIMappingProfileRepository
	ConnectionRepo      repositories.EDIConnectionRepository
	ProfileRepo         repositories.EDICommunicationProfileRepository
	TransferRepo        repositories.EDILoadTenderTransferRepository
	DocumentTypeRepo    repositories.EDIDocumentTypeRepository
	SourceContextRepo   repositories.EDISourceContextRepository
	PartnerSettingRepo  repositories.EDIPartnerSettingRepository
	TemplateRepo        repositories.EDITemplateRepository
	DocumentProfileRepo repositories.EDIPartnerDocumentProfileRepository
	ControlNumberRepo   repositories.EDIControlNumberRepository
	MessageRepo         repositories.EDIMessageRepository
	TestCaseRepo        repositories.EDITestCaseRepository
	InvoiceRepo         repositories.InvoiceRepository
	ShipmentEventRepo   repositories.ShipmentEventRepository
	ServiceFailureRepo  repositories.ServiceFailureRepository
	ShipmentLinkRepo    repositories.EDIShipmentLinkRepository
	TransferChangeRepo  repositories.EDITransferChangeRepository
	ShipmentCommentRepo repositories.ShipmentCommentRepository
	UserRepo            repositories.UserRepository
	ShipmentSvc         services.ShipmentService
	WorkflowStarter     services.WorkflowStarter
	AuditService        services.AuditService
	Encryption          *encryptionservice.Service
	Validator           *Validator
	DB                  coreports.DBConnection
}

type Service struct {
	l                   *zap.Logger
	partnerRepo         repositories.EDIPartnerRepository
	mappingProfileRepo  repositories.EDIMappingProfileRepository
	connectionRepo      repositories.EDIConnectionRepository
	profileRepo         repositories.EDICommunicationProfileRepository
	transferRepo        repositories.EDILoadTenderTransferRepository
	documentTypeRepo    repositories.EDIDocumentTypeRepository
	sourceContextRepo   repositories.EDISourceContextRepository
	partnerSettingRepo  repositories.EDIPartnerSettingRepository
	templateRepo        repositories.EDITemplateRepository
	documentProfileRepo repositories.EDIPartnerDocumentProfileRepository
	controlNumberRepo   repositories.EDIControlNumberRepository
	messageRepo         repositories.EDIMessageRepository
	testCaseRepo        repositories.EDITestCaseRepository
	invoiceRepo         repositories.InvoiceRepository
	shipmentEventRepo   repositories.ShipmentEventRepository
	serviceFailureRepo  repositories.ServiceFailureRepository
	shipmentLinkRepo    repositories.EDIShipmentLinkRepository
	transferChangeRepo  repositories.EDITransferChangeRepository
	shipmentCommentRepo repositories.ShipmentCommentRepository
	userRepo            repositories.UserRepository
	shipmentSvc         services.ShipmentService
	workflowStarter     services.WorkflowStarter
	auditService        services.AuditService
	encryption          *encryptionservice.Service
	validator           *Validator
	db                  coreports.DBConnection
}

func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.edi"),
		partnerRepo:         p.PartnerRepo,
		mappingProfileRepo:  p.MappingProfileRepo,
		connectionRepo:      p.ConnectionRepo,
		profileRepo:         p.ProfileRepo,
		transferRepo:        p.TransferRepo,
		documentTypeRepo:    p.DocumentTypeRepo,
		sourceContextRepo:   p.SourceContextRepo,
		partnerSettingRepo:  p.PartnerSettingRepo,
		templateRepo:        p.TemplateRepo,
		documentProfileRepo: p.DocumentProfileRepo,
		controlNumberRepo:   p.ControlNumberRepo,
		messageRepo:         p.MessageRepo,
		testCaseRepo:        p.TestCaseRepo,
		invoiceRepo:         p.InvoiceRepo,
		shipmentEventRepo:   p.ShipmentEventRepo,
		serviceFailureRepo:  p.ServiceFailureRepo,
		shipmentLinkRepo:    p.ShipmentLinkRepo,
		transferChangeRepo:  p.TransferChangeRepo,
		shipmentCommentRepo: p.ShipmentCommentRepo,
		userRepo:            p.UserRepo,
		shipmentSvc:         p.ShipmentSvc,
		workflowStarter:     p.WorkflowStarter,
		auditService:        p.AuditService,
		encryption:          p.Encryption,
		validator:           p.Validator,
		db:                  p.DB,
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

	return s.CreateInternalPartnerPairViaConnection(ctx, req, actor)
}

func (s *Service) GetMappingProfile(
	ctx context.Context,
	req repositories.GetMappingProfileRequest,
) (*edi.EDIMappingProfile, error) {
	return s.mappingProfileRepo.GetMappingProfile(ctx, req)
}

func (s *Service) ListMappingProfiles(
	ctx context.Context,
	req *repositories.ListEDIMappingProfilesRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.ListMappingProfiles(ctx, req)
}

func (s *Service) SelectMappingProfileOptions(
	ctx context.Context,
	req *repositories.EDIMappingProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.SelectMappingProfileOptions(ctx, req)
}

func (s *Service) GetMappingProfileByID(
	ctx context.Context,
	req repositories.GetMappingProfileByIDRequest,
) (*edi.EDIMappingProfile, error) {
	return s.mappingProfileRepo.GetMappingProfileByID(ctx, req)
}

func (s *Service) SaveMappingProfile(
	ctx context.Context,
	req *repositories.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	if multiErr := s.validator.ValidateMappingItems(req.Items); multiErr != nil {
		return nil, multiErr
	}

	return s.mappingProfileRepo.SaveMappingItems(ctx, req)
}

func (s *Service) SaveMappingProfileItems(
	ctx context.Context,
	req *repositories.SaveMappingProfileItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	if multiErr := s.validator.ValidateMappingItems(req.Items); multiErr != nil {
		return nil, multiErr
	}

	return s.mappingProfileRepo.SaveMappingProfileItems(ctx, req)
}

func (s *Service) DeleteMappingItem(
	ctx context.Context,
	req repositories.DeleteMappingItemRequest,
) error {
	return s.mappingProfileRepo.DeleteMappingItem(ctx, req)
}

func (s *Service) DeleteMappingProfileItem(
	ctx context.Context,
	req repositories.DeleteMappingProfileItemRequest,
) error {
	return s.mappingProfileRepo.DeleteMappingProfileItem(ctx, req)
}

//nolint:cyclop,funlen,nestif // Tender submission keeps the transaction and non-transaction paths explicit.
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

	var created *edi.EDITransfer
	if s.db == nil {
		created, err = s.transferRepo.CreateTransfer(ctx, entity)
		if err != nil {
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

		preview, err = s.buildMappingPreview(txCtx, targetPartner, transfer.TenderPayload)
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

//nolint:cyclop,funlen,gocognit // Temporal approval processing mirrors the workflow states explicitly.
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

	var original edi.EDITransfer
	var updated *edi.EDITransfer
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
				"EDI transfer cannot be rejected while finalized or processing",
			)
		}

		now := timeutils.NowUnix()
		original = *transfer
		transfer.Status = edi.TransferStatusRejected
		transfer.RejectionReason = reason
		transfer.RejectedByID = actor.UserID
		transfer.RejectedAt = &now

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
		if !transfer.Status.IsActionable() {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"EDI transfer cannot be canceled while finalized or processing",
			)
		}

		now := timeutils.NowUnix()
		original = *transfer
		transfer.Status = edi.TransferStatusCanceled
		transfer.CanceledByID = actor.UserID
		transfer.CanceledAt = &now

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
		if !transfer.Status.IsActionable() {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"EDI transfer cannot be expired while finalized or processing",
			)
		}

		now := timeutils.NowUnix()
		original = *transfer
		transfer.Status = edi.TransferStatusExpired
		transfer.ProcessedAt = &now

		updated, err = s.transferRepo.UpdateTransfer(txCtx, transfer)
		if err != nil {
			return err
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

func (s *Service) ListShipmentLinks(
	ctx context.Context,
	req *repositories.ListEDIShipmentLinksRequest,
) (*pagination.ListResult[*edi.ShipmentLink], error) {
	return s.shipmentLinkRepo.ListShipmentLinks(ctx, req)
}

func (s *Service) GetShipmentLink(
	ctx context.Context,
	req repositories.GetEDIShipmentLinkByIDRequest,
) (*edi.ShipmentLink, error) {
	return s.shipmentLinkRepo.GetShipmentLinkByID(ctx, req)
}

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
	if change.Status != edi.TransferChangeStatusPendingReview {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"EDI transfer change has already been reviewed",
		)
	}

	now := timeutils.NowUnix()
	original := *change
	change.Status = status
	change.ReviewedByID = actor.UserID
	change.ReviewedAt = &now
	if status == edi.TransferChangeStatusApplied {
		change.AppliedByID = actor.UserID
		change.AppliedAt = &now
	}
	if strings.TrimSpace(req.Reason) != "" {
		change.FailureReason = strings.TrimSpace(req.Reason)
	}

	updated, err := s.transferChangeRepo.UpdateTransferChange(ctx, change)
	if err != nil {
		return nil, err
	}

	if err = s.commentTransferChangeReview(ctx, req.TenantInfo, updated); err != nil {
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

func (s *Service) buildMappingPreview(
	ctx context.Context,
	partner *edi.EDIPartner,
	payload edi.LoadTenderPayload,
) (*MappingPreview, error) {
	required := payload.RequiredMappingEntityIDs
	sourceIDs := flattenRequiredIDs(required)
	items, err := s.mappingProfileRepo.GetMappingItems(ctx, repositories.GetMappingItemsRequest{
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
	sourceLabels := sourceLabelIndex(&payload)

	all := make([]edi.MappingResolution, 0, len(sourceIDs))
	for _, entityType := range requiredEntityTypes(required) {
		ids := append([]pulid.ID(nil), required[entityType]...)
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, sourceID := range ids {
			resolution := edi.MappingResolution{
				EntityType:  entityType,
				SourceID:    sourceID,
				SourceLabel: sourceLabels[entityType][sourceID],
			}
			if item := index[entityType][sourceID]; item != nil && item.TargetID.IsNotNil() {
				resolution.SourceLabel = stringutils.FirstNonEmpty(
					item.SourceLabel,
					resolution.SourceLabel,
				)
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

//nolint:funlen // Shipment reconstruction maps each payload section directly into the shipment aggregate.
func (s *Service) buildTargetShipment(
	transfer *edi.EDITransfer,
	businessUnitID pulid.ID,
	resolutions []edi.MappingResolution,
	approverID pulid.ID,
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
		EnteredByID:         approverID,
		FormulaTemplateID:   formulaTemplateID,
		Status:              shipment.StatusNew,
		TenderStatus:        tenderStatusPtr(shipment.TenderStatusAccepted),
		EntryMethod:         shipment.EntryMethodEDI,
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
		for idx := range move.Stops {
			stop := move.Stops[idx]
			locationID, locationOK := mappedID(
				mappings,
				edi.MappingEntityTypeLocation,
				stop.LocationID,
			)
			if !locationOK {
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
		commodityID, commodityOK := mappedID(
			mappings,
			edi.MappingEntityTypeCommodity,
			commodity.CommodityID,
		)
		if !commodityOK {
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
		chargeID, chargeOK := mappedID(
			mappings,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
		)
		if !chargeOK {
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

func mapEDIConnectionConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case "idx_edi_connections_internal_open":
		multiErr.Add(
			"targetOrganizationId",
			errortypes.ErrDuplicate,
			"An open internal EDI connection already exists for these organizations",
		)
	default:
		return err
	}

	return multiErr
}

func mapEDICommunicationProfileConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case "idx_edi_communication_profiles_name_org":
		multiErr.Add(
			"name",
			errortypes.ErrDuplicate,
			"EDI communication profile with this name already exists",
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
}

func (s *Service) setShipmentTenderStatus(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	status shipment.TenderStatus,
) error {
	results, err := s.db.DBForContext(ctx).
		NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("tender_status = ?", status).
		Set("version = version + 1").
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Where("id = ?", shipmentID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(results, "Shipment", shipmentID.String())
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
	metadata["source"] = "edi"

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

func tenderStatusPtr(status shipment.TenderStatus) *shipment.TenderStatus {
	return &status
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
	if entity.Settings == nil {
		entity.Settings = map[string]any{}
	}
}
