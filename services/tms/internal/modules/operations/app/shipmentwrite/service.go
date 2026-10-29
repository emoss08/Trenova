package shipmentwrite

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/allocationcheck"
	"github.com/emoss08/trenova/internal/modules/operations/app/equipmentavailabilityhelper"
	"github.com/emoss08/trenova/internal/modules/operations/app/equipmentcontinuityhelper"
	"github.com/emoss08/trenova/internal/modules/operations/app/servicefailuretrigger"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingreadiness"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingtransfer"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/app/modeprofileservice"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) recordCapabilityDeviations(
	ctx context.Context,
	entity *shipment.Shipment,
	advisories []*errortypes.AdvisoryError,
) error {
	if s.modeProfileService == nil || entity == nil {
		return nil
	}

	policy, err := shipmentvalidator.ResolveProfileForShipment(ctx, s.modeProfileService, entity)
	if errors.Is(err, modeprofileports.ErrNoModeProfileConfigured) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve mode profile for deviations: %w", err)
	}
	if policy == nil {
		return nil
	}

	if err = s.modeProfileService.RecordDeviations(ctx, &modeprofileports.RecordDeviationsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceType: modeprofileservice.ResourceTypeShipment,
		ResourceID:   entity.ID,
		Policy:       policy,
		Advisories:   advisories,
	}); err != nil {
		return fmt.Errorf("record capability deviations: %w", err)
	}
	return nil
}

func (s *Service) syncPermits(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) error {
	if s.permitService == nil || entity == nil {
		return nil
	}

	if _, err := s.permitService.Sync(ctx, entity, actor); err != nil {
		return fmt.Errorf("sync permit requirements: %w", err)
	}
	return nil
}

// validateChargeAllocations checks a save's allocations before anything is
// written. The rules live in allocationcheck so the billing queue enforces the
// same ones when a charge is reassigned.
func (s *Service) validateChargeAllocations(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	return allocationcheck.Validate(ctx, allocationcheck.Deps{
		CustomerRepo:         s.customerRepo,
		ChargeAllocationRepo: s.chargeAllocationRepo,
		Logger:               s.l,
	}, entity, "chargeAllocations")
}

func (s *Service) PreviewCreate(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentCreatePlan, error) {
	if entity == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("shipment", errortypes.ErrRequired, "Shipment is required")

		return nil, multiErr
	}

	prepared, err := s.prepareCreate(ctx, entity, actor.AuditActor().UserID)
	if err != nil {
		return nil, err
	}

	return &shipmentserviceports.ShipmentCreatePlan{
		Shipment:   entity,
		Rating:     ratingOutcome(prepared.rating),
		Advisories: prepared.advisories,
	}, nil
}

func (s *Service) prepareCreate(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*createPreparation, error) {
	control, err := s.effects.GetShipmentControl(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	entity.ApplyEntryMethodDefault(nil)
	entity.ApplyFreightTermsDefault(nil)
	entity.NormalizeBillTo()

	if multiErr := s.coordinator.PrepareForCreateWithDelayThreshold(
		entity,
		shipmenteffects.DelayThresholdMinutes(control),
	); multiErr != nil {
		return nil, multiErr
	}

	if err = s.effects.RefuseCreditHold(ctx, entity, nil); err != nil {
		return nil, err
	}

	s.dropSystemGeneratedAdditionalChargesForCreate(entity)

	if err = s.effects.HydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}

	s.effects.ApplyShipmentEnvelope(ctx, entity)

	if s.distanceCalculation != nil {
		if _, err = s.distanceCalculation.ResolveForShipment(ctx, entity); err != nil {
			return nil, err
		}
	}

	rating, err := s.commercial.RateAndAdoptContract(ctx, entity, control, userID)
	if err != nil {
		return nil, err
	}

	if err = s.validateExplicitOrder(ctx, entity); err != nil {
		return nil, err
	}

	if multiErr := s.validateChargeAllocations(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	multiErr, advisories := s.validator.ValidateCreateWithAdvisories(ctx, entity)
	if multiErr != nil {
		return nil, multiErr
	}

	if err = s.effects.CheckDuplicateBOLsWithControl(
		ctx,
		control,
		duplicateBOLCheckRequest(entity),
	); err != nil {
		return nil, err
	}
	if err = s.checkExternalReference(ctx, entity); err != nil {
		return nil, err
	}

	return &createPreparation{rating: rating, advisories: advisories}, nil
}

type Service struct {
	l                    *zap.Logger
	db                   kernelports.Transactor
	repo                 shipmentports.ShipmentRepository
	orderRepo            orderports.OrderRepository
	assignmentRepo       assignmentports.AssignmentRepository
	modeProfileService   modeprofileports.ModeProfileService
	permitService        permitports.PermitService
	continuityRepo       equipmentcontinuityports.EquipmentContinuityRepository
	customerRepo         customerports.CustomerRepository
	chargeAllocationRepo chargeallocationports.ChargeAllocationRepository
	documentRepo         documentports.DocumentRepository
	billingRepo          tenantports.BillingControlRepository
	serviceFailures      servicefailureports.ServiceFailureEvaluator
	validator            *shipmentvalidator.Validator
	eventService         shipmenteventports.ShipmentEventService
	coordinator          *shipmentstate.Coordinator
	invalidator          shipmentinvalidationports.ShipmentInvalidator
	commercial           *shipmentcommercial.Calculator
	distanceCalculation  shipmentdistanceports.DistanceCalculationService
	agentEvents          agentports.AgentEventPublisher
	mutationObservers    []shipmentserviceports.ShipmentMutationObserver
	effects              *shipmenteffects.Effects
	readiness            *shipmentbillingreadiness.Service
	transfer             *shipmentbillingtransfer.Service
}

type Deps struct {
	Logger               *zap.Logger
	DB                   kernelports.Transactor
	Repo                 shipmentports.ShipmentRepository
	OrderRepo            orderports.OrderRepository
	AssignmentRepo       assignmentports.AssignmentRepository
	ModeProfileService   modeprofileports.ModeProfileService
	PermitService        permitports.PermitService
	ContinuityRepo       equipmentcontinuityports.EquipmentContinuityRepository
	CustomerRepo         customerports.CustomerRepository
	ChargeAllocationRepo chargeallocationports.ChargeAllocationRepository
	DocumentRepo         documentports.DocumentRepository
	BillingRepo          tenantports.BillingControlRepository
	ServiceFailures      servicefailureports.ServiceFailureEvaluator
	Validator            *shipmentvalidator.Validator
	EventService         shipmenteventports.ShipmentEventService
	Coordinator          *shipmentstate.Coordinator
	Invalidator          shipmentinvalidationports.ShipmentInvalidator
	Commercial           *shipmentcommercial.Calculator
	DistanceCalculation  shipmentdistanceports.DistanceCalculationService
	AgentEvents          agentports.AgentEventPublisher
	MutationObservers    []shipmentserviceports.ShipmentMutationObserver
	Effects              *shipmenteffects.Effects
	Readiness            *shipmentbillingreadiness.Service
	Transfer             *shipmentbillingtransfer.Service
}

func New(d Deps) *Service {
	return &Service{
		l:                    d.Logger,
		db:                   d.DB,
		repo:                 d.Repo,
		orderRepo:            d.OrderRepo,
		assignmentRepo:       d.AssignmentRepo,
		modeProfileService:   d.ModeProfileService,
		permitService:        d.PermitService,
		continuityRepo:       d.ContinuityRepo,
		customerRepo:         d.CustomerRepo,
		chargeAllocationRepo: d.ChargeAllocationRepo,
		documentRepo:         d.DocumentRepo,
		billingRepo:          d.BillingRepo,
		serviceFailures:      d.ServiceFailures,
		validator:            d.Validator,
		eventService:         d.EventService,
		coordinator:          d.Coordinator,
		invalidator:          d.Invalidator,
		commercial:           d.Commercial,
		distanceCalculation:  d.DistanceCalculation,
		agentEvents:          d.AgentEvents,
		mutationObservers:    d.MutationObservers,
		effects:              d.Effects,
		readiness:            d.Readiness,
		transfer:             d.Transfer,
	}
}

func (s *Service) SetShipmentMutationObservers(
	value []shipmentserviceports.ShipmentMutationObserver,
) {
	s.mutationObservers = value
}

func (s *Service) Create(
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
		if IsIdempotencyKeyConflict(err) {
			existing, found, findErr := s.findIdempotentCreate(ctx, entity)
			if findErr != nil || found {
				return existing, findErr
			}
		}
		if mapped := ExternalReferenceConflict(err, entity); mapped != nil {
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

func (s *Service) persistCreate(
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
		shipmenteffects.TenantRefForShipment(created),
		created,
		auditActor,
	)); err != nil {
		return nil, fmt.Errorf("record shipment created event: %w", err)
	}

	return created, nil
}

func (s *Service) announceCreated(
	ctx context.Context,
	created *shipment.Shipment,
	auditActor kernelports.AuditActor,
) {
	if err := s.effects.LogShipmentAction(
		created,
		auditActor,
		permission.OpCreate,
		nil,
		created,
		auditservice.WithComment("Shipment created"),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationForRecord(
			created,
			auditActor,
			"created",
			created,
		),
	)

	agentports.PublishAgentEvent(ctx, s.agentEvents, agentports.AgentEvent{
		Kind:      agenttrigger.EventShipmentCreated,
		SubjectID: created.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: created.OrganizationID,
			BuID:  created.BusinessUnitID,
		},
	})
}

func (s *Service) Update( //nolint:cyclop // legacy workflow
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

	if multiErr := shipmenteffects.ValidateShipmentNotLockedForBilling(original); multiErr != nil {
		log.Warn(
			"shipment update blocked — locked for billing",
			zap.String("shipmentId", original.ID.String()),
			zap.String("billingTransferStatus", string(original.BillingTransferStatus)),
		)
		return nil, multiErr
	}

	control, err := s.effects.GetShipmentControl(ctx, pagination.TenantInfo{
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
	if err = s.effects.RefuseCreditHold(ctx, entity, original); err != nil {
		return nil, err
	}
	s.restoreAssignmentsForExistingMoves(original, entity)
	s.restoreSystemOwnedAdditionalChargeFields(original, entity)
	shipmentrating.RestoreRateOwnedFields(original, entity)
	departed := disengageAutoRating(original, entity)
	if err = s.syncOrderMembershipForUpdate(ctx, original, entity); err != nil {
		return nil, err
	}

	if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
		original,
		entity,
		shipmenteffects.DelayThresholdMinutes(control),
	); multiErr != nil {
		return nil, multiErr
	}

	if err = s.effects.HydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}

	s.effects.ApplyShipmentEnvelope(ctx, entity)

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
	if multiErr := s.readiness.ValidateBillingReadinessForStatusChange(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	if err = s.ensureEquipmentAvailableForShipmentUpdate(ctx, original, entity); err != nil {
		return nil, err
	}

	req := duplicateBOLCheckRequest(entity)
	if err = s.effects.CheckDuplicateBOLsWithControl(ctx, control, req); err != nil {
		return nil, err
	}
	if err = s.checkExternalReference(ctx, entity); err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		if mapped := ExternalReferenceConflict(err, entity); mapped != nil {
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

	if err = s.effects.LogShipmentAction(
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

	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationForRecord(
			updatedEntity,
			auditActor,
			"updated",
			updatedEntity,
		),
	)

	s.effects.EmitStatusChangeEvent(ctx, original, updatedEntity, auditActor)
	if err = s.effects.RecomputeOrdersForShipments(
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
		autoMarkedEntity, autoMarkErr := s.transfer.AutoMarkReadyToInvoiceIfEligible(
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

func (s *Service) notifyShipmentMutationObservers(
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

func (s *Service) evaluateServiceFailuresAfterShipmentUpdate(
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

// validateExplicitOrder guards shipment creation that names an order directly:
// the order must exist, share the shipment's customer (invariant #4), and still
// accept new legs. Attach-after-create goes through orderservice.AttachShipments,
// which enforces the same rules.
func (s *Service) validateExplicitOrder(
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
func (s *Service) syncOrderMembershipForUpdate(
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

func (s *Service) checkExternalReference(ctx context.Context, entity *shipment.Shipment) error {
	reference := strings.TrimSpace(entity.ExternalReference)
	if reference == "" {
		return nil
	}

	existing, err := s.repo.FindByExternalReference(
		ctx,
		&shipmentports.ExternalReferenceCheckRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			CustomerID:        entity.CustomerID,
			ExternalReference: reference,
			ShipmentID:        entity.ID,
		},
	)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	return errDuplicateExternalReference(reference, existing.ProNumber)
}

func (s *Service) findIdempotentCreate(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, bool, error) {
	if entity.IdempotencyKey == "" {
		return nil, false, nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	id, err := s.repo.FindIDByIdempotencyKey(ctx, &shipmentports.IdempotencyKeyLookupRequest{
		TenantInfo:     tenantInfo,
		IdempotencyKey: entity.IdempotencyKey,
	})
	if err != nil || id.IsNil() {
		return nil, false, err
	}

	existing, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return nil, false, err
	}
	return existing, true, nil
}

func (s *Service) advanceContinuityForCompletedMoves(
	ctx context.Context,
	original *shipment.Shipment,
	updated *shipment.Shipment,
) error {
	if s.continuityRepo == nil || updated == nil {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: updated.OrganizationID,
		BuID:  updated.BusinessUnitID,
	}

	originalMoveCap := 0
	if original != nil {
		originalMoveCap = len(original.Moves)
	}
	originalMoves := make(map[pulid.ID]*shipment.ShipmentMove, originalMoveCap)
	if original != nil {
		for _, move := range original.Moves {
			if move == nil || move.ID.IsNil() {
				continue
			}
			originalMoves[move.ID] = move
		}
	}

	for _, move := range updated.Moves {
		if move == nil || move.ID.IsNil() || move.Assignment == nil || !move.IsCompleted() {
			continue
		}

		if previous := originalMoves[move.ID]; previous != nil && previous.IsCompleted() {
			continue
		}

		if err := equipmentcontinuityhelper.AdvanceForCompletedMove(
			ctx,
			s.continuityRepo,
			tenantInfo,
			move,
		); err != nil {
			return err
		}
	}

	return nil
}

//nolint:gocognit // existing domain workflow is intentionally branch-heavy
func (s *Service) ensureEquipmentAvailableForShipmentUpdate(
	ctx context.Context,
	original *shipment.Shipment,
	updated *shipment.Shipment,
) error {
	if s.assignmentRepo == nil || updated == nil {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: updated.OrganizationID,
		BuID:  updated.BusinessUnitID,
	}

	originalMoveCap := 0
	if original != nil {
		originalMoveCap = len(original.Moves)
	}
	originalMoves := make(map[pulid.ID]*shipment.ShipmentMove, originalMoveCap)
	if original != nil {
		for _, move := range original.Moves {
			if move == nil || move.ID.IsNil() {
				continue
			}
			originalMoves[move.ID] = move
		}
	}

	seenTractors := make(map[pulid.ID]pulid.ID)
	seenTrailers := make(map[pulid.ID]pulid.ID)

	for _, move := range updated.Moves {
		if move == nil || move.ID.IsNil() || !move.IsInTransit() {
			continue
		}

		previous := originalMoves[move.ID]
		if previous != nil && previous.IsInTransit() {
			continue
		}
		if move.Assignment == nil {
			continue
		}

		if move.Assignment.TractorID != nil {
			if priorMoveID, ok := seenTractors[*move.Assignment.TractorID]; ok {
				return errortypes.NewBusinessError("Tractor is currently in progress on another move").
					WithParam("tractorId", move.Assignment.TractorID.String()).
					WithParam("shipmentMoveId", priorMoveID.String())
			}
			seenTractors[*move.Assignment.TractorID] = move.ID
		}

		if move.Assignment.TrailerID != nil {
			if priorMoveID, ok := seenTrailers[*move.Assignment.TrailerID]; ok {
				return errortypes.NewBusinessError("Trailer is currently in progress on another move").
					WithParam("trailerId", move.Assignment.TrailerID.String()).
					WithParam("shipmentMoveId", priorMoveID.String())
			}
			seenTrailers[*move.Assignment.TrailerID] = move.ID
		}

		if err := equipmentavailabilityhelper.EnsureAssignmentEquipmentAvailable(
			ctx,
			s.assignmentRepo,
			tenantInfo,
			move.Assignment,
			move.ID,
		); err != nil {
			return err
		}
	}

	return nil
}

// dropSystemGeneratedAdditionalChargesForCreate removes the machine-owned
// charges a payload arrives carrying.
//
// A charge belongs to whichever engine placed it, and on a create none of them
// have run yet: the contract's accessorials, the fuel surcharge and the
// detention line are all placed later in this same save, from the agreement and
// the controls rather than from the request. What arrives flagged as
// system-generated is therefore a copy — the billing panel seats the ones the
// contract preview named so the rater can see what the load will cost before
// saving it — and keeping it would leave the shipment holding the copy and the
// real one, billing the customer for each accessorial twice.
//
// They are dropped rather than demoted to manual charges. A demoted copy is
// indistinguishable from a charge the rater typed, so nothing downstream can
// tell it apart from a genuine one, and the reconciliation passes append their
// own alongside it.
func (s *Service) dropSystemGeneratedAdditionalChargesForCreate(
	entity *shipment.Shipment,
) {
	if entity == nil {
		return
	}

	kept := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))

	for _, charge := range entity.AdditionalCharges {
		if charge == nil || charge.IsSystemGenerated {
			continue
		}

		charge.IsDetention = false
		charge.RateAgreementAccessorialID = nil
		charge.FuelSurchargeProgramID = nil

		kept = append(kept, charge)
	}

	entity.AdditionalCharges = kept
}

func (s *Service) restoreSystemOwnedAdditionalChargeFields(
	original *shipment.Shipment,
	updated *shipment.Shipment,
) {
	if updated == nil {
		return
	}

	var originalCharges []*shipment.AdditionalCharge
	if original != nil {
		originalCharges = original.AdditionalCharges
	}

	shipmentrating.RestoreSystemOwnedCharges(originalCharges, updated.AdditionalCharges)
}

func (s *Service) restoreAssignmentsForExistingMoves(
	original *shipment.Shipment,
	updated *shipment.Shipment,
) {
	if original == nil || updated == nil {
		return
	}

	originalMoves := make(map[pulid.ID]*shipment.ShipmentMove, len(original.Moves))
	for _, move := range original.Moves {
		if move == nil || move.ID.IsNil() {
			continue
		}

		originalMoves[move.ID] = move
	}

	for _, move := range updated.Moves {
		if move == nil || move.ID.IsNil() || move.Assignment != nil {
			continue
		}

		originalMove := originalMoves[move.ID]
		if originalMove == nil || originalMove.Assignment == nil {
			continue
		}

		move.Assignment = originalMove.Assignment
	}
}
