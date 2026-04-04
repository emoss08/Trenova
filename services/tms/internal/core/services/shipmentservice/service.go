package shipmentservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
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

type shipmentTenantResource interface {
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
	GetID() pulid.ID
}

type Params struct {
	fx.In

	Logger          *zap.Logger
	Repo            repositories.ShipmentRepository
	CacheRepo       repositories.ShipmentCacheRepository
	AssignmentRepo  repositories.AssignmentRepository
	UserRepo        repositories.UserRepository
	ControlRepo     repositories.ShipmentControlRepository
	ContinuityRepo  repositories.EquipmentContinuityRepository
	CommodityRepo   repositories.CommodityRepository
	HazmatRuleRepo  repositories.HazmatSegregationRuleRepository
	AccessorialRepo repositories.AccessorialChargeRepository
	Permissions     services.PermissionEngine
	Validator       *Validator
	AuditService    services.AuditService
	Realtime        services.RealtimeService
	WorkflowStarter services.WorkflowStarter
	Coordinator     *shipmentstate.Coordinator
	Commercial      *shipmentcommercial.Calculator
}

type service struct {
	l               *zap.Logger
	repo            repositories.ShipmentRepository
	cacheRepo       repositories.ShipmentCacheRepository
	assignmentRepo  repositories.AssignmentRepository
	userRepo        repositories.UserRepository
	controlRepo     repositories.ShipmentControlRepository
	continuityRepo  repositories.EquipmentContinuityRepository
	commodityRepo   repositories.CommodityRepository
	hazmatRuleRepo  repositories.HazmatSegregationRuleRepository
	accessorialRepo repositories.AccessorialChargeRepository
	permissions     services.PermissionEngine
	validator       *Validator
	auditService    services.AuditService
	realtime        services.RealtimeService
	workflowStarter services.WorkflowStarter
	coordinator     *shipmentstate.Coordinator
	commercial      *shipmentcommercial.Calculator
}

//nolint:gocritic // service constructor
func New(p Params) services.ShipmentService {
	return &service{
		l:               p.Logger.Named("service.shipment"),
		repo:            p.Repo,
		cacheRepo:       p.CacheRepo,
		assignmentRepo:  p.AssignmentRepo,
		userRepo:        p.UserRepo,
		controlRepo:     p.ControlRepo,
		continuityRepo:  p.ContinuityRepo,
		commodityRepo:   p.CommodityRepo,
		hazmatRuleRepo:  p.HazmatRuleRepo,
		accessorialRepo: p.AccessorialRepo,
		permissions:     p.Permissions,
		validator:       p.Validator,
		auditService:    p.AuditService,
		realtime:        p.Realtime,
		workflowStarter: p.WorkflowStarter,
		coordinator:     p.Coordinator,
		commercial:      p.Commercial,
	}
}

func (s *service) List(
	ctx context.Context,
	req *repositories.ListShipmentsRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	return s.repo.List(ctx, req)
}

func (s *service) Get(
	ctx context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	entity, err := s.cacheRepo.GetByID(ctx, req)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, repositories.ErrCacheMiss) {
		s.l.Warn(
			"failed to load shipment from cache",
			zap.Error(err),
			zap.String("shipmentID", req.ID.String()),
		)
	}

	return s.repo.GetByID(ctx, req)
}

func (s *service) GetUIPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.ShipmentUIPolicy, error) {
	control, err := s.getShipmentControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return &services.ShipmentUIPolicy{
		AllowMoveRemovals:      control.AllowMoveRemovals,
		CheckForDuplicateBOLs:  control.CheckForDuplicateBOLs,
		CheckHazmatSegregation: control.CheckHazmatSegregation,
		MaxShipmentWeightLimit: control.MaxShipmentWeightLimit,
	}, nil
}

func (s *service) GetPreviousRates(
	ctx context.Context,
	req *repositories.GetPreviousRatesRequest,
) (*pagination.ListResult[*repositories.PreviousRateSummary], error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	return s.repo.GetPreviousRates(ctx, req)
}

func (s *service) Create(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *services.RequestActor,
) (*shipment.Shipment, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	control, err := s.getShipmentControl(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	if multiErr := s.coordinator.PrepareForCreateWithDelayThreshold(
		entity,
		delayThresholdMinutes(control),
	); multiErr != nil {
		return nil, multiErr
	}

	s.normalizeAdditionalChargeSystemGenerationForCreate(entity)

	if err = s.hydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}

	if err = s.commercial.Recalculate(ctx, entity, control, auditActor.UserID); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	req := duplicateBOLCheckRequest(entity)
	if err = s.checkDuplicateBOLsWithControl(ctx, control, req); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create shipment", zap.Error(err))
		return nil, err
	}

	if err = s.logShipmentAction(
		createdEntity,
		auditActor,
		permission.OpCreate,
		nil,
		createdEntity,
		auditservice.WithComment("Shipment created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	if err = s.publishShipmentInvalidation(
		ctx,
		createdEntity,
		auditActor,
		"created",
		createdEntity,
	); err != nil {
		log.Warn("failed to publish realtime invalidation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *service) Update(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *services.RequestActor,
) (*shipment.Shipment, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	original, err := s.repo.GetByID(
		ctx,
		&repositories.GetShipmentByIDRequest{
			ID: entity.GetID(),
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.GetOrganizationID(),
				BuID:  entity.GetBusinessUnitID(),
			},
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		},
	)
	if err != nil {
		s.l.Error("failed to get original shipment", zap.Error(err))
		return nil, err
	}

	control, err := s.getShipmentControl(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	s.restoreAssignmentsForExistingMoves(original, entity)
	s.restoreAdditionalChargeSystemGeneration(original, entity)

	if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
		original,
		entity,
		delayThresholdMinutes(control),
	); multiErr != nil {
		return nil, multiErr
	}

	if err = s.hydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}

	if err = s.commercial.Recalculate(ctx, entity, control, auditActor.UserID); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdateWithOriginal(ctx, original, entity); multiErr != nil {
		return nil, multiErr
	}
	if err = s.ensureEquipmentAvailableForShipmentUpdate(ctx, original, entity); err != nil {
		return nil, err
	}

	req := duplicateBOLCheckRequest(entity)
	if err = s.checkDuplicateBOLsWithControl(ctx, control, req); err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to update shipment", zap.Error(err))
		return nil, err
	}
	if err = s.advanceContinuityForCompletedMoves(ctx, original, updatedEntity); err != nil {
		return nil, err
	}

	if err = s.logShipmentAction(
		updatedEntity,
		auditActor,
		permission.OpUpdate,
		original,
		updatedEntity,
		auditservice.WithComment("Shipment updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	if err = s.publishShipmentInvalidation(
		ctx,
		updatedEntity,
		auditActor,
		"updated",
		updatedEntity,
	); err != nil {
		log.Warn("failed to publish realtime invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *service) TransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
	actor *services.RequestActor,
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
	if auditActor.PrincipalType == services.PrincipalTypeAPIKey || auditActor.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrInvalidOperation,
			"Shipment ownership transfer requires a user actor",
		)
	}

	log := s.l.With(
		zap.String("operation", "TransferOwnership"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("shipmentID", req.ShipmentID.String()),
		zap.String("ownerID", req.OwnerID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		log.Error("failed to get original shipment", zap.Error(err))
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

	updatedEntity, err := s.repo.TransferOwnership(ctx, req)
	if err != nil {
		log.Error("failed to transfer shipment ownership", zap.Error(err))
		return nil, err
	}

	if err = s.logShipmentAction(
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

	if err = s.publishShipmentInvalidation(
		ctx,
		updatedEntity,
		auditActor,
		"ownership_transferred",
		updatedEntity,
	); err != nil {
		log.Warn("failed to publish realtime invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *service) GetDelayedShipments(
	ctx context.Context,
	req *repositories.GetDelayedShipmentsRequest,
) ([]*shipment.Shipment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	control, err := s.getShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if !control.AutoDelayShipments {
		return []*shipment.Shipment{}, nil
	}

	return s.repo.GetDelayedShipments(ctx, req, delayThresholdMinutes(control))
}

func (s *service) DelayShipments(
	ctx context.Context,
	req *repositories.DelayShipmentsRequest,
	actor *services.RequestActor,
) ([]*shipment.Shipment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	control, err := s.getShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if !control.AutoDelayShipments {
		return []*shipment.Shipment{}, nil
	}

	delayedShipments, err := s.repo.DelayShipments(ctx, req, delayThresholdMinutes(control))
	if err != nil {
		return nil, err
	}

	if len(delayedShipments) == 0 {
		return delayedShipments, nil
	}

	var auditActor services.AuditActor
	if actor != nil {
		auditActor = actor.AuditActor()
	}

	for _, entity := range delayedShipments {
		if err = s.publishShipmentInvalidation(
			ctx, entity, auditActor, "delayed", entity,
		); err != nil {
			s.l.Warn("failed to publish realtime invalidation", zap.Error(err))
		}
	}

	return delayedShipments, nil
}

func (s *service) GetAutoCancelableShipments(
	ctx context.Context,
	req *repositories.GetAutoCancelableShipmentsRequest,
) ([]*shipment.Shipment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	control, err := s.getShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if !control.AutoCancelShipments {
		return []*shipment.Shipment{}, nil
	}

	return s.repo.GetAutoCancelableShipments(ctx, req, autoCancelThresholdDays(control))
}

func (s *service) AutoCancelShipments(
	ctx context.Context,
	req *repositories.AutoCancelShipmentsRequest,
	actor *services.RequestActor,
) ([]*shipment.Shipment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	control, err := s.getShipmentControl(ctx, req.TenantInfo)
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
				repositories.RollbackEquipmentContinuityByShipmentRequest{
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

	var auditActor services.AuditActor
	if actor != nil {
		auditActor = actor.AuditActor()
	}

	if err = s.publishBulkShipmentInvalidation(
		ctx, req.TenantInfo, auditActor, "bulk_canceled",
	); err != nil {
		s.l.Warn("failed to publish realtime invalidation", zap.Error(err))
	}

	return canceledShipments, nil
}

func (s *service) CheckForDuplicateBOLs(
	ctx context.Context,
	req *repositories.DuplicateBOLCheckRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	control, err := s.getShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return err
	}

	return s.checkDuplicateBOLsWithControl(ctx, control, req)
}

func (s *service) CheckHazmatSegregation(
	ctx context.Context,
	req *repositories.CheckHazmatSegregationRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	control, err := s.getShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return err
	}

	conflicts, err := s.evaluateHazmatSegregationRequest(ctx, control, req)
	if err != nil {
		return err
	}

	multiErr := errortypes.NewMultiError()
	addHazmatConflictsToMultiError(multiErr, conflicts)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *service) Cancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
	actor *services.RequestActor,
) (*shipment.Shipment, error) {
	if req == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("request", errortypes.ErrRequired, "Cancel request is required")
		return nil, multiErr
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Cancel"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		log.Error("failed to get original shipment", zap.Error(err))
		return nil, err
	}

	if original.IsCanceled() {
		return nil, errortypes.NewBusinessError("shipment is already canceled")
	}

	req.CanceledByID = auditActor.UserID
	req.CanceledAt = timeutils.NowUnix()

	updatedEntity, err := s.repo.Cancel(ctx, req)
	if err != nil {
		log.Error("failed to cancel shipment", zap.Error(err))
		return nil, err
	}
	if s.continuityRepo != nil {
		if err = s.continuityRepo.RollbackCurrentByShipment(ctx,
			repositories.RollbackEquipmentContinuityByShipmentRequest{
				TenantInfo: req.TenantInfo,
				ShipmentID: req.ShipmentID,
			}); err != nil {
			return nil, err
		}
	}

	if err = s.logShipmentAction(
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

	if err = s.publishShipmentInvalidation(
		ctx, updatedEntity, auditActor, "canceled", updatedEntity,
	); err != nil {
		log.Warn("failed to publish realtime invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *service) Uncancel(
	ctx context.Context,
	req *repositories.UncancelShipmentRequest,
	actor *services.RequestActor,
) (*shipment.Shipment, error) {
	if req == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("request", errortypes.ErrRequired, "Uncancel request is required")
		return nil, multiErr
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Uncancel"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		log.Error("failed to get original shipment", zap.Error(err))
		return nil, err
	}

	if !original.IsCanceled() {
		return nil, errortypes.NewBusinessError("shipment is not canceled")
	}

	updatedEntity, err := s.repo.Uncancel(ctx, req)
	if err != nil {
		log.Error("failed to uncancel shipment", zap.Error(err))
		return nil, err
	}

	if err = s.logShipmentAction(
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

	if err = s.publishShipmentInvalidation(
		ctx, updatedEntity, auditActor, "uncanceled", updatedEntity,
	); err != nil {
		log.Warn("failed to publish realtime invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *service) Duplicate(
	ctx context.Context,
	req *repositories.BulkDuplicateShipmentRequest,
) (*repositories.ShipmentDuplicateWorkflowResponse, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	payload := &shipmentjobs.BulkDuplicateShipmentsPayload{
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
		RequestedBy:   req.TenantInfo.UserID,
	}

	workflowID := fmt.Sprintf(
		"shipment-duplicate-%s-%s-%s-%d",
		req.TenantInfo.OrgID.String(),
		req.TenantInfo.BuID.String(),
		req.ShipmentID.String(),
		time.Now().UnixNano(),
	)

	if !s.workflowStarter.Enabled() {
		return nil, errortypes.NewBusinessError("shipment duplication is not configured")
	}

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
		shipmentjobs.BulkDuplicateShipmentsWorkflowName,
		payload,
	)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to start shipment duplication workflow").
			WithInternal(err)
	}

	return &repositories.ShipmentDuplicateWorkflowResponse{
		WorkflowID:  run.GetID(),
		RunID:       run.GetRunID(),
		TaskQueue:   temporaltype.TaskQueueSystem.String(),
		Status:      enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING.String(),
		SubmittedAt: timeutils.NowUnix(),
	}, nil
}

func (s *service) CalculateTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*repositories.ShipmentTotalsResponse, error) {
	// Shipment totals are derived from the formula template plus nested additional charges.
	if entity == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("shipment", errortypes.ErrRequired, "Shipment is required")
		return nil, multiErr
	}

	if entity.FormulaTemplateID.IsNil() {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("formulaTemplateId", errortypes.ErrRequired, "Formula template is required")
		return nil, multiErr
	}

	control, err := s.getShipmentControl(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	if err = s.hydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}

	resp, err := s.commercial.CalculateTotals(ctx, entity, control, userID)
	if err != nil {
		s.l.Error("failed to calculate shipment totals", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

func (s *service) validateTransferActor(
	ctx context.Context,
	auditActor services.AuditActor,
	original *shipment.Shipment,
	orgID pulid.ID,
) error {
	if original != nil && original.OwnerID == auditActor.UserID {
		return nil
	}

	if s.permissions == nil {
		return errortypes.NewBusinessError("Permission engine is not configured")
	}

	manifest, err := s.permissions.GetLightManifest(ctx, auditActor.UserID, orgID)
	if err != nil {
		return err
	}

	if manifest.IsOrgAdmin || manifest.IsBusinessUnitAdmin {
		return nil
	}

	return errortypes.NewValidationError(
		"ownerId",
		errortypes.ErrInvalidOperation,
		"You do not have permission to transfer ownership of this shipment",
	)
}

func (s *service) validateTransferTarget(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
) error {
	if s.userRepo == nil {
		return errortypes.NewBusinessError("User repository is not configured")
	}

	_, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
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
