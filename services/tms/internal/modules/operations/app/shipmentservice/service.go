//nolint:funlen // existing legacy workflow/API shape is intentionally kept stable
package shipmentservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/servicefailuretrigger"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/shipmentjobs/shipmentcontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazmatsegregationruleports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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

type shipmentTenantResource interface {
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
	GetID() pulid.ID
}

type MutationObserverSetter interface {
	SetShipmentMutationObservers(observers []shipmentports.ShipmentMutationObserver)
}

type Params struct {
	fx.In

	Logger               *zap.Logger
	DB                   kernelports.Transactor
	Repo                 shipmentports.ShipmentRepository
	OrderRepo            orderports.OrderRepository
	CacheRepo            customerports.ShipmentCacheRepository
	AssignmentRepo       shipmentports.AssignmentRepository
	UserRepo             tenantports.UserRepository
	ControlRepo          tenantports.ShipmentControlRepository
	ModeProfileService   modeprofileports.ModeProfileService
	PermitService        permitports.PermitService
	ContinuityRepo       equipmentcontinuityports.EquipmentContinuityRepository
	CommodityRepo        commodityports.CommodityRepository
	HazmatRuleRepo       hazmatsegregationruleports.HazmatSegregationRuleRepository
	EquipmentTypeRepo    equipmenttypeports.EquipmentTypeRepository
	AccessorialRepo      accessorialchargeports.AccessorialChargeRepository
	CustomerRepo         customerports.CustomerRepository
	ChargeAllocationRepo shipmentports.ChargeAllocationRepository
	DocumentRepo         documentports.DocumentRepository
	BillingRepo          tenantports.BillingControlRepository
	NotificationService  *notificationservice.Service
	BillingQueueService  billingqueueports.BillingQueueService        `optional:"true"`
	ServiceFailureRepo   servicefailureports.ServiceFailureRepository `optional:"true"`
	ServiceFailures      servicefailureports.ServiceFailureEvaluator  `optional:"true"`
	Permissions          permissionports.PermissionEngine
	Validator            *Validator
	AuditService         auditports.AuditService
	EventService         shipmenteventports.ShipmentEventService
	WorkflowStarter      kernelports.WorkflowStarter
	Coordinator          *shipmentstate.Coordinator
	Commercial           *shipmentcommercial.Calculator
	OrderDerivation      orderports.OrderDerivationService
	DistanceCalculation  shipmentports.DistanceCalculationService `optional:"true"`
	TenderGuard          tenderports.TenderGuard                  `optional:"true"`
	AgentEvents          agentports.AgentEventPublisher           `optional:"true"`
	Quota                platformplanports.QuotaGuard             `optional:"true"`
	QuickFilters         shipmentports.ShipmentQuickFilterBasisResolver
	Invalidator          shipmentports.ShipmentInvalidator
}

type service struct {
	l                    *zap.Logger
	db                   kernelports.Transactor
	repo                 shipmentports.ShipmentRepository
	orderRepo            orderports.OrderRepository
	cacheRepo            customerports.ShipmentCacheRepository
	assignmentRepo       shipmentports.AssignmentRepository
	userRepo             tenantports.UserRepository
	controlRepo          tenantports.ShipmentControlRepository
	modeProfileService   modeprofileports.ModeProfileService
	permitService        permitports.PermitService
	continuityRepo       equipmentcontinuityports.EquipmentContinuityRepository
	commodityRepo        commodityports.CommodityRepository
	hazmatRuleRepo       hazmatsegregationruleports.HazmatSegregationRuleRepository
	equipmentTypeRepo    equipmenttypeports.EquipmentTypeRepository
	accessorialRepo      accessorialchargeports.AccessorialChargeRepository
	customerRepo         customerports.CustomerRepository
	chargeAllocationRepo shipmentports.ChargeAllocationRepository
	documentRepo         documentports.DocumentRepository
	billingRepo          tenantports.BillingControlRepository
	notificationService  *notificationservice.Service
	billingQueueService  billingqueueports.BillingQueueService
	serviceFailureRepo   servicefailureports.ServiceFailureRepository
	serviceFailures      servicefailureports.ServiceFailureEvaluator
	permissions          permissionports.PermissionEngine
	validator            *Validator
	auditService         auditports.AuditService
	eventService         shipmenteventports.ShipmentEventService
	workflowStarter      kernelports.WorkflowStarter
	coordinator          *shipmentstate.Coordinator
	quickFilters         shipmentports.ShipmentQuickFilterBasisResolver
	invalidator          shipmentports.ShipmentInvalidator
	commercial           *shipmentcommercial.Calculator
	orderDerivation      orderports.OrderDerivationService
	distanceCalculation  shipmentports.DistanceCalculationService
	tenderGuard          tenderports.TenderGuard
	agentEvents          agentports.AgentEventPublisher
	quota                platformplanports.QuotaGuard
	mutationObservers    []shipmentports.ShipmentMutationObserver
}

func New(p Params) *service { //nolint:gocritic // stable API shape
	return &service{
		l:                    p.Logger.Named("service.shipment"),
		db:                   p.DB,
		repo:                 p.Repo,
		orderRepo:            p.OrderRepo,
		cacheRepo:            p.CacheRepo,
		assignmentRepo:       p.AssignmentRepo,
		userRepo:             p.UserRepo,
		controlRepo:          p.ControlRepo,
		modeProfileService:   p.ModeProfileService,
		permitService:        p.PermitService,
		continuityRepo:       p.ContinuityRepo,
		commodityRepo:        p.CommodityRepo,
		hazmatRuleRepo:       p.HazmatRuleRepo,
		equipmentTypeRepo:    p.EquipmentTypeRepo,
		accessorialRepo:      p.AccessorialRepo,
		customerRepo:         p.CustomerRepo,
		chargeAllocationRepo: p.ChargeAllocationRepo,
		documentRepo:         p.DocumentRepo,
		billingRepo:          p.BillingRepo,
		notificationService:  p.NotificationService,
		billingQueueService:  p.BillingQueueService,
		serviceFailureRepo:   p.ServiceFailureRepo,
		serviceFailures:      p.ServiceFailures,
		permissions:          p.Permissions,
		validator:            p.Validator,
		auditService:         p.AuditService,
		eventService:         p.EventService,
		workflowStarter:      p.WorkflowStarter,
		coordinator:          p.Coordinator,
		commercial:           p.Commercial,
		orderDerivation:      p.OrderDerivation,
		distanceCalculation:  p.DistanceCalculation,
		tenderGuard:          p.TenderGuard,
		agentEvents:          p.AgentEvents,
		quota:                quotaservice.OrUnlimited(p.Quota),
		quickFilters:         p.QuickFilters,
		invalidator:          p.Invalidator,
	}
}

func (s *service) SetShipmentMutationObservers(observers []shipmentports.ShipmentMutationObserver) {
	s.mutationObservers = append([]shipmentports.ShipmentMutationObserver(nil), observers...)
}

func (s *service) createNotification(
	ctx context.Context,
	entity *notification.Notification,
) {
	if s.notificationService == nil || entity == nil {
		return
	}

	if _, err := s.notificationService.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create notification", zap.Error(err))
	}
}

func (s *service) List(
	ctx context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	if req.ShipmentOptions.HasQuickFilters() {
		if s.quickFilters == nil {
			return nil, errQuickFiltersUnavailable
		}
		if err := s.quickFilters.Prepare(
			ctx,
			req.Filter.TenantInfo,
			&req.ShipmentOptions,
		); err != nil {
			return nil, err
		}
	}

	return s.repo.List(ctx, req)
}

func (s *service) Get(
	ctx context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	entity, err := s.cacheRepo.GetByID(ctx, req)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, customerports.ErrCacheMiss) {
		s.l.Warn(
			"failed to load shipment from cache",
			zap.Error(err),
			zap.String("shipmentID", req.ID.String()),
		)
	}

	return s.repo.GetByID(ctx, req)
}

func (s *service) GetUnassigned(
	ctx context.Context,
	req *shipmentports.GetUnassignedShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return s.repo.GetUnassigned(ctx, req)
}

func (s *service) GetByIDs(
	ctx context.Context,
	req *shipmentports.GetShipmentsByIDsRequest,
) ([]*shipment.Shipment, error) {
	return s.repo.GetByIDs(ctx, req)
}

func (s *service) SelectOptions(
	ctx context.Context,
	req *shipmentports.ShipmentSelectOptionsRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *service) GetUIPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*shipmentports.ShipmentUIPolicy, error) {
	control, err := s.getShipmentControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	policy := &shipmentports.ShipmentUIPolicy{
		AllowMoveRemovals:      control.AllowMoveRemovals,
		CheckForDuplicateBOLs:  control.CheckForDuplicateBOLs,
		CheckHazmatSegregation: control.CheckHazmatSegregation,
		MaxShipmentWeightLimit: control.MaxShipmentWeightLimit,
	}

	if s.modeProfileService == nil {
		return policy, nil
	}

	resolved, err := s.modeProfileService.Resolve(ctx, &modeprofileports.ResolveModeProfileRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to resolve mode profile for ui policy", zap.Error(err))
		return policy, nil
	}

	policy.Profile = resolved

	return policy, nil
}

func (s *service) GetPreviousRates(
	ctx context.Context,
	req *shipmentports.GetPreviousRatesRequest,
) (*pagination.ListResult[*shipmentports.PreviousRateSummary], error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	return s.repo.GetPreviousRates(ctx, req)
}

func (s *service) Create(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	if existing, found, err := s.findIdempotentCreate(ctx, entity); err != nil || found {
		return existing, err
	}

	prepared, err := s.prepareCreate(ctx, entity, auditActor.UserID)
	if err != nil {
		return nil, err
	}

	// Generate the shipment ID before the insert so nested rows can point at it.
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("shp_")
	}

	var createdEntity *shipment.Shipment
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		created, txErr := s.persistCreate(txCtx, entity, prepared, actor)
		createdEntity = created
		return txErr
	})
	if err != nil {
		if isIdempotencyKeyConflict(err) {
			existing, found, findErr := s.findIdempotentCreate(ctx, entity)
			if findErr != nil || found {
				return existing, findErr
			}
		}
		if mapped := externalReferenceConflict(err, entity); mapped != nil {
			return nil, mapped
		}
		log.Error("failed to create shipment", zap.Error(err))
		return nil, err
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		s.announceCreated(committed, createdEntity, auditActor)
	})

	return createdEntity, nil
}

func (s *service) persistCreate(
	ctx context.Context,
	entity *shipment.Shipment,
	prepared *createPreparation,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	auditActor := actor.AuditActor()
	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	if err = s.commercial.CommitContractRating(
		ctx, created, prepared.rating, auditActor.UserID,
	); err != nil {
		return nil, err
	}

	if created.RateQuoteID != nil {
		if created, err = s.repo.Update(ctx, created); err != nil {
			return nil, fmt.Errorf("link shipment to its rate quote: %w", err)
		}
	}

	if err = s.recordCapabilityDeviations(ctx, created, prepared.advisories); err != nil {
		return nil, err
	}
	if err = s.syncPermits(ctx, created, actor); err != nil {
		return nil, err
	}

	if err = s.eventService.Record(ctx, shipmenteventservice.BuildShipmentCreated(
		tenantRefForShipment(created),
		created,
		auditActor,
	)); err != nil {
		return nil, fmt.Errorf("record shipment created event: %w", err)
	}

	return created, nil
}

func (s *service) announceCreated(
	ctx context.Context,
	created *shipment.Shipment,
	auditActor kernelports.AuditActor,
) {
	if err := s.logShipmentAction(
		created,
		auditActor,
		permission.OpCreate,
		nil,
		created,
		auditservice.WithComment("Shipment created"),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationForRecord(created, auditActor, "created", created),
	)

	agentports.PublishAgentEvent(ctx, s.agentEvents, agentports.AgentEvent{
		Kind:      agent.EventShipmentCreated,
		SubjectID: created.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: created.OrganizationID,
			BuID:  created.BusinessUnitID,
		},
	})
}

func (s *service) Update( //nolint:cyclop // legacy workflow
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
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
		&shipmentports.GetShipmentByIDRequest{
			ID: entity.GetID(),
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.GetOrganizationID(),
				BuID:  entity.GetBusinessUnitID(),
			},
			ShipmentOptions: shipmentports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		},
	)
	if err != nil {
		s.l.Error("failed to get original shipment", zap.Error(err))
		return nil, err
	}

	if multiErr := validateShipmentNotLockedForBilling(original); multiErr != nil {
		log.Warn(
			"shipment update blocked — locked for billing",
			zap.String("shipmentId", original.ID.String()),
			zap.String("billingTransferStatus", string(original.BillingTransferStatus)),
		)
		return nil, multiErr
	}

	control, err := s.getShipmentControl(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	entity.ApplyEntryMethodDefault(original)
	entity.ApplyFreightTermsDefault(original)
	shipment.FollowCustomerChange(original, entity)
	entity.NormalizeBillTo()
	entity.PreviousCustomerID = original.CustomerID
	if err = s.refuseCreditHold(ctx, entity, original); err != nil {
		return nil, err
	}
	s.restoreAssignmentsForExistingMoves(original, entity)
	s.restoreSystemOwnedAdditionalChargeFields(original, entity)
	shipment.RestoreRateOwnedFields(original, entity)
	departed := disengageAutoRating(original, entity)
	if err = s.syncOrderMembershipForUpdate(ctx, original, entity); err != nil {
		return nil, err
	}

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

	s.applyShipmentEnvelope(ctx, entity)

	if s.distanceCalculation != nil {
		if _, err = s.distanceCalculation.ResolveForShipment(ctx, entity); err != nil {
			return nil, err
		}
	}

	if err = s.commercial.Recalculate(ctx, entity, control, auditActor.UserID); err != nil {
		return nil, err
	}

	if err = s.commercial.RecordRateDeparture(
		ctx, entity, auditActor.UserID, departed,
	); err != nil {
		return nil, err
	}

	multiErr, advisories := s.validator.ValidateUpdateWithOriginalAndAdvisories(
		ctx, original, entity,
	)
	if multiErr != nil {
		return nil, multiErr
	}
	if multiErr := s.validateChargeAllocations(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	if multiErr := s.validateBillingReadinessForStatusChange(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	if err = s.ensureEquipmentAvailableForShipmentUpdate(ctx, original, entity); err != nil {
		return nil, err
	}

	req := duplicateBOLCheckRequest(entity)
	if err = s.checkDuplicateBOLsWithControl(ctx, control, req); err != nil {
		return nil, err
	}
	if err = s.checkExternalReference(ctx, entity); err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		if mapped := externalReferenceConflict(err, entity); mapped != nil {
			return nil, mapped
		}
		s.l.Error("failed to update shipment", zap.Error(err))
		return nil, err
	}

	if err = s.recordCapabilityDeviations(ctx, updatedEntity, advisories); err != nil {
		log.Error("failed to record capability deviations", zap.Error(err))
	}
	if err = s.syncPermits(ctx, updatedEntity, actor); err != nil {
		log.Error("failed to sync permit requirements; dispatch may not be blocked",
			zap.String("shipmentId", updatedEntity.ID.String()),
			zap.Error(err),
		)
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

	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"updated",
			updatedEntity,
		),
	)

	s.emitStatusChangeEvent(ctx, original, updatedEntity, auditActor)
	if err = s.recomputeOrdersForShipments(
		ctx,
		pagination.TenantInfo{
			OrgID: updatedEntity.OrganizationID,
			BuID:  updatedEntity.BusinessUnitID,
		},
		[]*shipment.Shipment{updatedEntity},
	); err != nil {
		log.Warn("failed to recompute order after shipment update", zap.Error(err))
	}
	s.evaluateServiceFailuresAfterShipmentUpdate(ctx, updatedEntity, actor)

	if updatedEntity.Status == shipment.StatusCompleted &&
		s.customerRepo != nil &&
		s.documentRepo != nil &&
		s.billingRepo != nil {
		autoMarkedEntity, autoMarkErr := s.AutoMarkReadyToInvoiceIfEligible(
			ctx,
			updatedEntity.ID,
			pagination.TenantInfo{
				OrgID: updatedEntity.OrganizationID,
				BuID:  updatedEntity.BusinessUnitID,
			},
			auditActor.UserID,
		)
		if autoMarkErr != nil {
			log.Warn(
				"failed to auto-mark shipment ready to invoice after completion",
				zap.Error(autoMarkErr),
			)
		} else if autoMarkedEntity != nil {
			updatedEntity = autoMarkedEntity
		}
	}

	s.notifyShipmentMutationObservers(ctx, original, updatedEntity, actor)

	return updatedEntity, nil
}

func (s *service) notifyShipmentMutationObservers(
	ctx context.Context,
	original *shipment.Shipment,
	updated *shipment.Shipment,
	actor *permissionports.RequestActor,
) {
	if len(s.mutationObservers) == 0 {
		return
	}
	for _, observer := range s.mutationObservers {
		if observer == nil {
			continue
		}
		if err := observer.AfterShipmentUpdate(ctx, original, updated, actor); err != nil {
			s.l.Warn(
				"shipment mutation observer failed",
				zap.String("shipmentID", updated.ID.String()),
				zap.Error(err),
			)
		}
	}
}

func (s *service) evaluateServiceFailuresAfterShipmentUpdate(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) {
	if entity == nil {
		return
	}
	err := servicefailuretrigger.EvaluateShipment(
		ctx,
		s.serviceFailures,
		entity.ID,
		pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		actor,
	)
	if err != nil {
		s.l.Warn(
			"failed to evaluate service failures after shipment update",
			zap.String("shipmentID", entity.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *service) TransferOwnership(
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

	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"ownership_transferred",
			updatedEntity,
		),
	)

	s.recordShipmentEvent(ctx, shipmenteventservice.BuildOwnershipTransferred(
		tenantRefForShipment(updatedEntity),
		updatedEntity,
		original.OwnerID,
		req.OwnerID,
		auditActor,
	))

	return updatedEntity, nil
}

func (s *service) GetDelayedShipments(
	ctx context.Context,
	req *shipmentports.GetDelayedShipmentsRequest,
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
	req *shipmentports.DelayShipmentsRequest,
	actor *permissionports.RequestActor,
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

	if err = s.recomputeOrdersForShipments(ctx, req.TenantInfo, delayedShipments); err != nil {
		return nil, err
	}

	var auditActor kernelports.AuditActor
	if actor != nil {
		auditActor = actor.AuditActor()
	}

	for _, entity := range delayedShipments {
		shipmentports.InvalidateShipments(
			ctx,
			s.invalidator,
			shipmentports.ShipmentInvalidationForRecord(entity, auditActor, "delayed", entity),
		)
	}

	return delayedShipments, nil
}

func (s *service) GetAutoCancelableShipments(
	ctx context.Context,
	req *shipmentports.GetAutoCancelableShipmentsRequest,
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
	req *shipmentports.AutoCancelShipmentsRequest,
	actor *permissionports.RequestActor,
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

	if err = s.recomputeOrdersForShipments(ctx, req.TenantInfo, canceledShipments); err != nil {
		return nil, err
	}

	var auditActor kernelports.AuditActor
	if actor != nil {
		auditActor = actor.AuditActor()
	}

	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationByActor(
			req.TenantInfo,
			auditActor,
			pulid.Nil,
			"bulk_canceled",
		),
	)

	return canceledShipments, nil
}

// validateExplicitOrder guards shipment creation that names an order directly:
// the order must exist, share the shipment's customer (invariant #4), and still
// accept new legs. Attach-after-create goes through orderservice.AttachShipments,
// which enforces the same rules.
func (s *service) validateExplicitOrder(
	ctx context.Context,
	entity *shipment.Shipment,
) error {
	if entity.OrderID.IsNil() || s.orderRepo == nil {
		return nil
	}

	ord, err := s.orderRepo.GetByID(ctx, orderports.GetOrderByIDRequest{
		ID: entity.OrderID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return errortypes.NewValidationError(
				"orderId",
				errortypes.ErrInvalid,
				"Order not found",
			)
		}
		return err
	}

	if ord.CustomerID != entity.CustomerID {
		return errortypes.NewValidationError(
			"orderId",
			errortypes.ErrInvalid,
			"The shipment's customer must match the order's customer",
		)
	}
	if !ord.Status.AllowsMembershipChange() {
		return errortypes.NewValidationError(
			"orderId",
			errortypes.ErrInvalid,
			"New legs cannot be added to a {0} order", ord.Status,
		)
	}

	return nil
}

// syncOrderMembershipForUpdate keeps order membership out of the shipment update
// path: order_id always restores from the persisted row (attach/detach are the only
// membership mutations), and a customer change either follows through to a
// single-leg order or is rejected while the shipment is one leg of many.
func (s *service) syncOrderMembershipForUpdate(
	ctx context.Context,
	original *shipment.Shipment,
	entity *shipment.Shipment,
) error {
	entity.OrderID = original.OrderID

	if entity.CustomerID == original.CustomerID || original.OrderID.IsNil() ||
		s.orderRepo == nil {
		return nil
	}

	ord, err := s.orderRepo.GetByID(ctx, orderports.GetOrderByIDRequest{
		ID: original.OrderID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeShipment: true,
	})
	if err != nil {
		return err
	}

	if len(ord.Shipments) > 1 {
		return errortypes.NewValidationError(
			"customerId",
			errortypes.ErrInvalid,
			"The customer cannot be changed while the shipment is a leg of a multi-leg order; detach it first",
		)
	}
	if !ord.Status.AllowsMembershipChange() {
		return errortypes.NewValidationError(
			"customerId",
			errortypes.ErrInvalid,
			"The customer of a leg of a {0} order cannot be changed", ord.Status,
		)
	}

	ord.CustomerID = entity.CustomerID
	if _, err = s.orderRepo.Update(ctx, ord); err != nil {
		return err
	}

	return nil
}

func (s *service) recomputeOrdersForShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipments []*shipment.Shipment,
) error {
	if s.orderDerivation == nil {
		return nil
	}

	orderIDs := make(map[pulid.ID]struct{}, len(shipments))
	for _, entity := range shipments {
		if entity == nil || entity.OrderID.IsNil() {
			continue
		}
		orderIDs[entity.OrderID] = struct{}{}
	}

	for orderID := range orderIDs {
		if err := s.orderDerivation.RecomputeOrder(ctx, tenantInfo, orderID); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) CheckForDuplicateBOLs(
	ctx context.Context,
	req *shipmentports.DuplicateBOLCheckRequest,
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
	req *shipmentports.CheckHazmatSegregationRequest,
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

	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"canceled",
			updatedEntity,
		),
	)

	s.recordShipmentEvent(ctx, shipmenteventservice.BuildShipmentCanceled(
		tenantRefForShipment(updatedEntity),
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

func (s *service) Uncancel(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	original, _, err := s.planUncancel(ctx, req)
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
			dberror.ExtractConstraintName(err) == externalReferenceConstraint {
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

	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"uncanceled",
			updatedEntity,
		),
	)

	s.recordShipmentEvent(ctx, shipmenteventservice.BuildShipmentUncanceled(
		tenantRefForShipment(updatedEntity),
		updatedEntity,
		auditActor,
	))

	return updatedEntity, nil
}

func (s *service) Duplicate(
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

func (s *service) CalculateTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipmentports.ShipmentTotalsResponse, error) {
	if entity == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("shipment", errortypes.ErrRequired, "Shipment is required")
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

	s.applyShipmentEnvelope(ctx, entity)

	if s.distanceCalculation != nil {
		if _, err = s.distanceCalculation.ResolveForShipment(ctx, entity); err != nil {
			return nil, err
		}
	}

	resp, err := s.commercial.CalculateTotals(ctx, entity, control, userID)
	if err != nil {
		s.l.Error("failed to calculate shipment totals", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

func (s *service) CalculateDistance(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipmentports.DistanceCalculationResponse, error) {
	if s.distanceCalculation == nil {
		return nil, errortypes.NewBusinessError("distance calculation service is not configured")
	}
	if err := s.hydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}
	return s.distanceCalculation.ResolveForShipment(ctx, entity)
}

func (s *service) RecalculateDistance(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentports.DistanceCalculationResponse, error) {
	if s.distanceCalculation == nil {
		return nil, errortypes.NewBusinessError("distance calculation service is not configured")
	}
	return s.distanceCalculation.RecalculateShipment(ctx, shipmentID, tenantInfo)
}

func (s *service) validateTransferActor(
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

func (s *service) validateTransferTarget(
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
