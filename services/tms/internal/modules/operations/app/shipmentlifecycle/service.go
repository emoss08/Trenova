package shipmentlifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/shipmentjobs/shipmentcontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	l               *zap.Logger
	db              kernelports.Transactor
	repo            shipmentports.ShipmentRepository
	userRepo        tenantports.UserRepository
	continuityRepo  equipmentcontinuityports.EquipmentContinuityRepository
	billingRepo     tenantports.BillingControlRepository
	permissions     permissionports.PermissionEngine
	workflowStarter kernelports.WorkflowStarter
	coordinator     *shipmentstate.Coordinator
	invalidator     shipmentinvalidationports.ShipmentInvalidator
	tenderGuard     tenderports.TenderGuard
	quota           platformplanports.QuotaGuard
	effects         *shipmenteffects.Effects
}

type Params struct {
	fx.In

	Logger          *zap.Logger
	DB              kernelports.Transactor
	Repo            shipmentports.ShipmentRepository
	UserRepo        tenantports.UserRepository
	ContinuityRepo  equipmentcontinuityports.EquipmentContinuityRepository
	BillingRepo     tenantports.BillingControlRepository
	Permissions     permissionports.PermissionEngine
	WorkflowStarter kernelports.WorkflowStarter
	Coordinator     *shipmentstate.Coordinator
	Invalidator     shipmentinvalidationports.ShipmentInvalidator
	TenderGuard     tenderports.TenderGuard      `optional:"true"`
	Quota           platformplanports.QuotaGuard `optional:"true"`
	Effects         *shipmenteffects.Effects
}

func New(p Params) *Service {
	return &Service{
		l:               p.Logger.Named("service.shipment"),
		db:              p.DB,
		repo:            p.Repo,
		userRepo:        p.UserRepo,
		continuityRepo:  p.ContinuityRepo,
		billingRepo:     p.BillingRepo,
		permissions:     p.Permissions,
		workflowStarter: p.WorkflowStarter,
		coordinator:     p.Coordinator,
		invalidator:     p.Invalidator,
		tenderGuard:     p.TenderGuard,
		quota:           quotaservice.OrUnlimited(p.Quota),
		effects:         p.Effects,
	}
}

func (s *Service) PreviewCancel(
	ctx context.Context,
	req *shipmentports.CancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentCancelPreview, error) {
	if req == nil {
		return nil, cancelRequestRequired()
	}

	request := *req
	original, err := s.planCancel(ctx, &request, actor)
	if err != nil {
		return nil, err
	}

	after := *original
	after.ApplyCancel(request.CanceledByID, request.CanceledAt, request.CancelReason)

	return &shipmentserviceports.ShipmentCancelPreview{Before: original, After: &after}, nil
}

func (s *Service) planCancel(
	ctx context.Context,
	req *shipmentports.CancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	if req == nil {
		return nil, cancelRequestRequired()
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	if err = shipmentstate.ValidateCancel(original); err != nil {
		return nil, err
	}

	req.CanceledByID = actor.AuditActor().UserID
	req.CanceledAt = timeutils.NowUnix()
	req.ExpectedVersion = original.Version

	return original, nil
}

func (s *Service) PreviewUncancel(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
) (*shipmentserviceports.ShipmentChangePreview, error) {
	if req == nil {
		return nil, uncancelRequestRequired()
	}

	request := *req
	original, restored, err := s.PlanUncancel(ctx, &request)
	if err != nil {
		return nil, err
	}

	return &shipmentserviceports.ShipmentChangePreview{Before: original, After: restored}, nil
}

func (s *Service) PlanUncancel(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
) (original, restored *shipment.Shipment, err error) {
	if req == nil {
		return nil, nil, uncancelRequestRequired()
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, nil, multiErr
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	}

	original, err = s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:              req.ShipmentID,
		TenantInfo:      tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return nil, nil, err
	}

	if !original.IsCanceled() {
		return nil, nil, errortypes.NewBusinessError("shipment is not canceled")
	}

	control, err := s.effects.GetShipmentControl(ctx, tenantInfo)
	if err != nil {
		return nil, nil, err
	}

	restored = original.CloneMoveGraph()
	s.coordinator.PrepareForUncancel(restored, shipmenteffects.DelayThresholdMinutes(control))

	req.ExpectedVersion = original.Version
	req.RestoredStatus = restored.Status
	req.MoveStatuses, req.StopStatuses = uncancelRestores(original, restored)

	return original, restored, nil
}

func (s *Service) PreviewTransferOwnership(
	ctx context.Context,
	req *shipmentports.TransferOwnershipRequest,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentChangePreview, error) {
	original, err := s.planTransferOwnership(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	after := *original
	after.OwnerID = req.OwnerID

	return &shipmentserviceports.ShipmentChangePreview{Before: original, After: &after}, nil
}

func (s *Service) planTransferOwnership(
	ctx context.Context,
	req *shipmentports.TransferOwnershipRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	if req == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("request", errortypes.ErrRequired, "Transfer ownership request is required")
		return nil, multiErr
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	auditActor := actor.AuditActor()
	if actor.PersonUserID().IsNil() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrInvalidOperation,
			"Shipment ownership transfer requires a user actor",
		)
	}

	original, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	if original.OwnerID == req.OwnerID {
		return nil, errortypes.NewValidationError(
			"ownerId",
			errortypes.ErrInvalid,
			"Shipment already belongs to this owner",
		)
	}

	if err = s.validateTransferActor(ctx, auditActor, original, req.TenantInfo.OrgID); err != nil {
		return nil, err
	}

	if err = s.validateTransferTarget(ctx, req); err != nil {
		return nil, err
	}

	return original, nil
}

func (s *Service) PreviewDuplicate(
	ctx context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) (*shipmentserviceports.ShipmentDuplicatePreview, error) {
	plan, err := s.planDuplicate(ctx, req)
	if err != nil {
		return nil, err
	}

	return &shipmentserviceports.ShipmentDuplicatePreview{
		Source: plan.Source,
		Copies: plan.Copies,
	}, nil
}

func (s *Service) planDuplicate(
	ctx context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) (*shipmentports.ShipmentDuplicatePlan, error) {
	if err := s.guardDuplicate(req); err != nil {
		return nil, err
	}

	plan, err := s.repo.PlanDuplicate(ctx, req)
	if err != nil {
		return nil, err
	}

	if err = (shipmentvalidator.RateCoverage{Billing: s.billingRepo}).RefuseUnratedCopies(ctx, plan); err != nil {
		return nil, err
	}
	if err = s.effects.RefuseCreditHold(ctx, plan.Source, nil); err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *Service) guardDuplicate(req *shipmentports.BulkDuplicateShipmentRequest) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	if !s.workflowStarter.Enabled() {
		return errortypes.NewBusinessError("shipment duplication is not configured")
	}

	return nil
}

func (s *Service) TransferOwnership(
	ctx context.Context,
	req *shipmentports.TransferOwnershipRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	original, err := s.planTransferOwnership(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "TransferOwnership"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("shipmentID", req.ShipmentID.String()),
		zap.String("ownerID", req.OwnerID.String()),
	)

	updatedEntity, err := s.repo.TransferOwnership(ctx, req)
	if err != nil {
		log.Error("failed to transfer shipment ownership", zap.Error(err))
		return nil, err
	}

	if err = s.effects.LogShipmentAction(
		updatedEntity,
		auditActor,
		permission.OpUpdate,
		original,
		updatedEntity,
		auditservice.WithComment("Shipment ownership transferred"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"ownership_transferred",
			updatedEntity,
		),
	)

	s.effects.RecordShipmentEvent(ctx, shipmenteventservice.BuildOwnershipTransferred(
		shipmenteffects.TenantRefForShipment(updatedEntity),
		updatedEntity,
		original.OwnerID,
		req.OwnerID,
		auditActor,
	))

	return updatedEntity, nil
}

func (s *Service) GetDelayedShipments(
	ctx context.Context,
	req *shipmentports.GetDelayedShipmentsRequest,
) ([]*shipment.Shipment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	control, err := s.effects.GetShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if !control.AutoDelayShipments {
		return []*shipment.Shipment{}, nil
	}

	return s.repo.GetDelayedShipments(ctx, req, shipmenteffects.DelayThresholdMinutes(control))
}

func (s *Service) DelayShipments(
	ctx context.Context,
	req *shipmentports.DelayShipmentsRequest,
	actor *permissionports.RequestActor,
) ([]*shipment.Shipment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	control, err := s.effects.GetShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if !control.AutoDelayShipments {
		return []*shipment.Shipment{}, nil
	}

	delayedShipments, err := s.repo.DelayShipments(
		ctx,
		req,
		shipmenteffects.DelayThresholdMinutes(control),
	)
	if err != nil {
		return nil, err
	}

	if len(delayedShipments) == 0 {
		return delayedShipments, nil
	}

	if err = s.effects.RecomputeOrdersForShipments(ctx, req.TenantInfo, delayedShipments); err != nil {
		return nil, err
	}

	var auditActor kernelports.AuditActor
	if actor != nil {
		auditActor = actor.AuditActor()
	}

	for _, entity := range delayedShipments {
		shipmentinvalidationports.InvalidateShipments(
			ctx,
			s.invalidator,
			shipmentinvalidationports.ShipmentInvalidationForRecord(
				entity,
				auditActor,
				"delayed",
				entity,
			),
		)
	}

	return delayedShipments, nil
}

func (s *Service) GetAutoCancelableShipments(
	ctx context.Context,
	req *shipmentports.GetAutoCancelableShipmentsRequest,
) ([]*shipment.Shipment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	control, err := s.effects.GetShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if !control.AutoCancelShipments {
		return []*shipment.Shipment{}, nil
	}

	return s.repo.GetAutoCancelableShipments(ctx, req, autoCancelThresholdDays(control))
}

func (s *Service) AutoCancelShipments(
	ctx context.Context,
	req *shipmentports.AutoCancelShipmentsRequest,
	actor *permissionports.RequestActor,
) ([]*shipment.Shipment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	control, err := s.effects.GetShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if !control.AutoCancelShipments {
		return []*shipment.Shipment{}, nil
	}

	canceledShipments, err := s.repo.AutoCancelShipments(ctx, req, autoCancelThresholdDays(control))
	if err != nil {
		return nil, err
	}
	if s.continuityRepo != nil {
		for _, canceledShipment := range canceledShipments {
			if canceledShipment == nil {
				continue
			}
			if err = s.continuityRepo.RollbackCurrentByShipment(ctx,
				equipmentcontinuityports.RollbackEquipmentContinuityByShipmentRequest{
					TenantInfo: req.TenantInfo,
					ShipmentID: canceledShipment.ID,
				}); err != nil {
				return nil, err
			}
		}
	}

	if len(canceledShipments) == 0 {
		return canceledShipments, nil
	}

	if err = s.effects.RecomputeOrdersForShipments(ctx, req.TenantInfo, canceledShipments); err != nil {
		return nil, err
	}

	var auditActor kernelports.AuditActor
	if actor != nil {
		auditActor = actor.AuditActor()
	}

	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationByActor(
			req.TenantInfo,
			auditActor,
			pulid.Nil,
			"bulk_canceled",
		),
	)

	return canceledShipments, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	req *shipmentports.CancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	original, err := s.planCancel(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Cancel"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	var updatedEntity *shipment.Shipment
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		canceled, txErr := s.repo.Cancel(txCtx, req)
		if txErr != nil {
			return txErr
		}

		if s.continuityRepo != nil {
			if txErr = s.continuityRepo.RollbackCurrentByShipment(txCtx,
				equipmentcontinuityports.RollbackEquipmentContinuityByShipmentRequest{
					TenantInfo: req.TenantInfo,
					ShipmentID: req.ShipmentID,
				}); txErr != nil {
				return txErr
			}
		}

		updatedEntity = canceled
		return nil
	})
	if err != nil {
		log.Error("failed to cancel shipment", zap.Error(err))
		return nil, err
	}

	if err = s.effects.LogShipmentAction(
		updatedEntity,
		auditActor,
		permission.OpCancel,
		original,
		updatedEntity,
		auditservice.WithComment("Shipment canceled"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"canceled",
			updatedEntity,
		),
	)

	s.effects.RecordShipmentEvent(ctx, shipmenteventservice.BuildShipmentCanceled(
		shipmenteffects.TenantRefForShipment(updatedEntity),
		updatedEntity,
		req.CancelReason,
		auditActor,
	))

	if s.tenderGuard != nil {
		if err = s.tenderGuard.CancelLiveTendersForShipment(
			ctx, req.TenantInfo, req.ShipmentID, "Shipment was canceled",
		); err != nil {
			log.Error("failed to withdraw live tenders for canceled shipment", zap.Error(err))
		}
	}

	return updatedEntity, nil
}

func (s *Service) Uncancel(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	original, _, err := s.PlanUncancel(ctx, req)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Uncancel"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	updatedEntity, err := s.repo.Uncancel(ctx, req)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) &&
			dberror.ExtractConstraintName(err) == shipmenteffects.ExternalReferenceConstraint {
			multiErr := errortypes.NewMultiError()
			multiErr.Add(
				"externalReference",
				errortypes.ErrDuplicate,
				"Another live shipment for this customer now uses this shipment's customer reference; change one of them before restoring this shipment",
			)
			return nil, multiErr
		}
		log.Error("failed to uncancel shipment", zap.Error(err))
		return nil, err
	}

	if err = s.effects.LogShipmentAction(
		updatedEntity,
		auditActor,
		permission.OpUpdate,
		original,
		updatedEntity,
		auditservice.WithComment("Shipment uncanceled"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"uncanceled",
			updatedEntity,
		),
	)

	s.effects.RecordShipmentEvent(ctx, shipmenteventservice.BuildShipmentUncanceled(
		shipmenteffects.TenantRefForShipment(updatedEntity),
		updatedEntity,
		auditActor,
	))

	return updatedEntity, nil
}

func (s *Service) Duplicate(
	ctx context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) (*shipmentports.ShipmentDuplicateWorkflowResponse, error) {
	if _, err := s.planDuplicate(ctx, req); err != nil {
		return nil, err
	}
	if err := quotaservice.Preflight(ctx, s.quota, &platformplanports.QuotaRequest{
		TenantInfo: req.TenantInfo,
		Meter:      platformcatalog.MeterShipmentsTotal,
		Quantity:   int64(max(req.Count, 0)),
	}); err != nil {
		return nil, err
	}

	payload := &shipmentcontract.BulkDuplicateShipmentsPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			UserID:         req.TenantInfo.UserID,
			Timestamp:      timeutils.NowUnix(),
			Metadata: map[string]any{
				"trigger":    "api",
				"shipmentId": req.ShipmentID.String(),
				"count":      req.Count,
			},
		},
		ShipmentID:    req.ShipmentID,
		Count:         req.Count,
		OverrideDates: req.OverrideDates,
		FirstPickupAt: req.FirstPickupAt,
		RequestedBy:   req.TenantInfo.UserID,
	}

	workflowID := fmt.Sprintf(
		"shipment-duplicate-%s-%s-%s-%d",
		req.TenantInfo.OrgID.String(),
		req.TenantInfo.BuID.String(),
		req.ShipmentID.String(),
		time.Now().UnixNano(),
	)

	run, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: temporaltype.TaskQueueSystem.String(),
			StaticSummary: fmt.Sprintf(
				"Duplicate shipment %s (%d copies)",
				req.ShipmentID.String(),
				req.Count,
			),
		},
		shipmentcontract.BulkDuplicateShipmentsWorkflowName,
		payload,
	)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to start shipment duplication workflow").
			WithInternal(err)
	}

	return &shipmentports.ShipmentDuplicateWorkflowResponse{
		WorkflowID:  run.GetID(),
		RunID:       run.GetRunID(),
		TaskQueue:   temporaltype.TaskQueueSystem.String(),
		Status:      enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING.String(),
		SubmittedAt: timeutils.NowUnix(),
	}, nil
}

func (s *Service) validateTransferActor(
	ctx context.Context,
	auditActor kernelports.AuditActor,
	original *shipment.Shipment,
	orgID pulid.ID,
) error {
	if original != nil && original.OwnerID == auditActor.UserID {
		return nil
	}

	if original == nil {
		return errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalidOperation,
			"Shipment is required to validate ownership transfer permissions",
		)
	}

	if s.permissions == nil {
		return errortypes.NewBusinessError("Permission engine is not configured")
	}

	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		BusinessUnitID: original.BusinessUnitID,
		OrganizationID: orgID,
		Resource:       permission.ResourceShipment.String(),
		Operation:      permission.OpUpdate,
		ResourceID:     &original.ID,
		ResourceAttributes: permissionports.ResourceAttributes{
			OrganizationID: original.OrganizationID,
			BusinessUnitID: original.BusinessUnitID,
			OwnerID:        original.OwnerID,
		},
	})
	if err != nil {
		return err
	}

	if result.Allowed {
		return nil
	}

	return errortypes.NewValidationError(
		"ownerId",
		errortypes.ErrInvalidOperation,
		"You do not have permission to transfer ownership of this shipment",
	)
}

func (s *Service) validateTransferTarget(
	ctx context.Context,
	req *shipmentports.TransferOwnershipRequest,
) error {
	if s.userRepo == nil {
		return errortypes.NewBusinessError("User repository is not configured")
	}

	_, err := s.userRepo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  req.TenantInfo.OrgID,
			BuID:   req.TenantInfo.BuID,
			UserID: req.OwnerID,
		},
	})
	if err != nil {
		return errortypes.NewValidationError(
			"ownerId",
			errortypes.ErrInvalid,
			"Owner does not exist in your organization",
		)
	}

	return nil
}

var _ shipmentserviceports.ShipmentLifecycleService = (*Service)(nil)
