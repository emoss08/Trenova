//nolint:funlen // existing legacy workflow/API shape is intentionally kept stable
package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentautorate"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingreadiness"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingtransfer"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentlifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentloading"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentread"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentwrite"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazmatsegregationruleports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MutationObserverSetter interface {
	SetShipmentMutationObservers(observers []shipmentserviceports.ShipmentMutationObserver)
}

type Params struct {
	fx.In

	Logger               *zap.Logger
	DB                   kernelports.Transactor
	Repo                 shipmentports.ShipmentRepository
	OrderRepo            orderports.OrderRepository
	CacheRepo            customerports.ShipmentCacheRepository
	AssignmentRepo       assignmentports.AssignmentRepository
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
	ChargeAllocationRepo chargeallocationports.ChargeAllocationRepository
	DocumentRepo         documentports.DocumentRepository
	BillingRepo          tenantports.BillingControlRepository
	NotificationService  *notificationservice.Service
	BillingQueueService  billingqueueports.BillingQueueService        `optional:"true"`
	ServiceFailureRepo   servicefailureports.ServiceFailureRepository `optional:"true"`
	ServiceFailures      servicefailureports.ServiceFailureEvaluator  `optional:"true"`
	Permissions          permissionports.PermissionEngine
	Validator            *shipmentvalidator.Validator
	AuditService         auditports.AuditService
	EventService         shipmenteventports.ShipmentEventService
	WorkflowStarter      kernelports.WorkflowStarter
	Coordinator          *shipmentstate.Coordinator
	Commercial           *shipmentcommercial.Calculator
	OrderDerivation      orderports.OrderDerivationService
	DistanceCalculation  shipmentdistanceports.DistanceCalculationService `optional:"true"`
	TenderGuard          tenderports.TenderGuard                          `optional:"true"`
	AgentEvents          agentports.AgentEventPublisher                   `optional:"true"`
	Quota                platformplanports.QuotaGuard                     `optional:"true"`
	QuickFilters         shipmentports.ShipmentQuickFilterBasisResolver
	Invalidator          shipmentinvalidationports.ShipmentInvalidator
}

type service struct {
	l                    *zap.Logger
	db                   kernelports.Transactor
	repo                 shipmentports.ShipmentRepository
	orderRepo            orderports.OrderRepository
	cacheRepo            customerports.ShipmentCacheRepository
	assignmentRepo       assignmentports.AssignmentRepository
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
	chargeAllocationRepo chargeallocationports.ChargeAllocationRepository
	documentRepo         documentports.DocumentRepository
	billingRepo          tenantports.BillingControlRepository
	notificationService  *notificationservice.Service
	billingQueueService  billingqueueports.BillingQueueService
	serviceFailureRepo   servicefailureports.ServiceFailureRepository
	serviceFailures      servicefailureports.ServiceFailureEvaluator
	permissions          permissionports.PermissionEngine
	validator            *shipmentvalidator.Validator
	auditService         auditports.AuditService
	eventService         shipmenteventports.ShipmentEventService
	workflowStarter      kernelports.WorkflowStarter
	coordinator          *shipmentstate.Coordinator
	quickFilters         shipmentports.ShipmentQuickFilterBasisResolver
	invalidator          shipmentinvalidationports.ShipmentInvalidator
	commercial           *shipmentcommercial.Calculator
	orderDerivation      orderports.OrderDerivationService
	distanceCalculation  shipmentdistanceports.DistanceCalculationService
	tenderGuard          tenderports.TenderGuard
	agentEvents          agentports.AgentEventPublisher
	quota                platformplanports.QuotaGuard
	mutationObservers    []shipmentserviceports.ShipmentMutationObserver
	effects              *shipmenteffects.Effects
	readiness            *shipmentbillingreadiness.Service
	transfer             *shipmentbillingtransfer.Service
	autorater            *shipmentautorate.Service
	loading              *shipmentloading.Service
	lifecycle            *shipmentlifecycle.Service
	reader               *shipmentread.Service
	writer               *shipmentwrite.Service
}

func New(p Params) *service { //nolint:gocritic // stable API shape
	s := &service{
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
	s.assemble()
	return s
}

func (s *service) SetShipmentMutationObservers(
	observers []shipmentserviceports.ShipmentMutationObserver,
) {
	s.mutationObservers = append([]shipmentserviceports.ShipmentMutationObserver(nil), observers...)
	s.writer.SetShipmentMutationObservers(s.mutationObservers)
}

func (s *service) GetBillingReadiness(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentbillingports.ShipmentBillingReadiness, error) {
	return s.readiness.GetBillingReadiness(ctx, shipmentID, tenantInfo)
}

func (s *service) AutoMarkReadyToInvoiceIfEligible(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	return s.transfer.AutoMarkReadyToInvoiceIfEligible(ctx, shipmentID, tenantInfo, userID)
}

func (s *service) TransferToBilling(
	ctx context.Context,
	req *shipmentbillingports.TransferShipmentToBillingRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	return s.transfer.TransferToBilling(ctx, req, actor)
}

func (s *service) TransferToBillingItems(
	ctx context.Context,
	req *shipmentbillingports.TransferShipmentToBillingRequest,
	actor *permissionports.RequestActor,
) (*billingqueueports.TransferToBillingResult, error) {
	return s.transfer.TransferToBillingItems(ctx, req, actor)
}

func (s *service) PlanBillingTransfers(
	ctx context.Context,
	req *shipmentbillingports.PlanBillingTransfersRequest,
) (*shipmentbillingports.BillingTransferPlan, error) {
	return s.transfer.PlanBillingTransfers(ctx, req)
}

func (s *service) ListBillingTransferCandidates(
	ctx context.Context,
	req *shipmentbillingports.ListBillingTransferCandidatesRequest,
) (*shipmentbillingports.BillingTransferCandidates, error) {
	return s.transfer.ListBillingTransferCandidates(ctx, req)
}

func (s *service) BulkTransferToBilling(
	ctx context.Context,
	req *shipmentbillingports.BulkTransferShipmentToBillingRequest,
	actor *permissionports.RequestActor,
) (*shipmentbillingports.BulkTransferToBillingResponse, error) {
	return s.transfer.BulkTransferToBilling(ctx, req, actor)
}

func (s *service) ListBillingTransferCandidateIDs(
	ctx context.Context,
	req *shipmentbillingports.ListBillingTransferCandidateIDsRequest,
) (*shipmentbillingports.BillingTransferCandidateIDsResponse, error) {
	return s.transfer.ListBillingTransferCandidateIDs(ctx, req)
}

func (s *service) ListAutoTransferCandidateIDs(
	ctx context.Context,
	req *shipmentbillingports.ListAutoTransferCandidateIDsRequest,
) ([]pulid.ID, error) {
	return s.transfer.ListAutoTransferCandidateIDs(ctx, req)
}

func (s *service) PreviewContractRate(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ContractRateApplication, error) {
	return s.autorater.PreviewContractRate(ctx, entity, actor)
}

func (s *service) AutoRate(
	ctx context.Context,
	req *shipmentserviceports.AutoRateShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, *shipmentserviceports.ContractRateApplication, error) {
	return s.autorater.AutoRate(ctx, req, actor)
}

func (s *service) PreviewAutoRate(
	ctx context.Context,
	req *shipmentserviceports.AutoRateShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentAutoRatePreview, error) {
	return s.autorater.PreviewAutoRate(ctx, req, actor)
}

func (s *service) CalculateLoadingOptimization(
	ctx context.Context,
	req *shipmentserviceports.LoadingOptimizationRequest,
) (*shipmentserviceports.LoadingOptimizationResult, error) {
	return s.loading.CalculateLoadingOptimization(ctx, req)
}

func (s *service) CheckHazmatSegregation(
	ctx context.Context,
	req *shipmentports.CheckHazmatSegregationRequest,
) error {
	return s.loading.CheckHazmatSegregation(ctx, req)
}

func (s *service) Cancel(
	ctx context.Context,
	req *shipmentports.CancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	return s.lifecycle.Cancel(ctx, req, actor)
}

func (s *service) PreviewCancel(
	ctx context.Context,
	req *shipmentports.CancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentCancelPreview, error) {
	return s.lifecycle.PreviewCancel(ctx, req, actor)
}

func (s *service) Uncancel(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	return s.lifecycle.Uncancel(ctx, req, actor)
}

func (s *service) PreviewUncancel(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
) (*shipmentserviceports.ShipmentChangePreview, error) {
	return s.lifecycle.PreviewUncancel(ctx, req)
}

func (s *service) TransferOwnership(
	ctx context.Context,
	req *shipmentports.TransferOwnershipRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	return s.lifecycle.TransferOwnership(ctx, req, actor)
}

func (s *service) PreviewTransferOwnership(
	ctx context.Context,
	req *shipmentports.TransferOwnershipRequest,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentChangePreview, error) {
	return s.lifecycle.PreviewTransferOwnership(ctx, req, actor)
}

func (s *service) Duplicate(
	ctx context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) (*shipmentports.ShipmentDuplicateWorkflowResponse, error) {
	return s.lifecycle.Duplicate(ctx, req)
}

func (s *service) PreviewDuplicate(
	ctx context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) (*shipmentserviceports.ShipmentDuplicatePreview, error) {
	return s.lifecycle.PreviewDuplicate(ctx, req)
}

func (s *service) GetDelayedShipments(
	ctx context.Context,
	req *shipmentports.GetDelayedShipmentsRequest,
) ([]*shipment.Shipment, error) {
	return s.lifecycle.GetDelayedShipments(ctx, req)
}

func (s *service) DelayShipments(
	ctx context.Context,
	req *shipmentports.DelayShipmentsRequest,
	actor *permissionports.RequestActor,
) ([]*shipment.Shipment, error) {
	return s.lifecycle.DelayShipments(ctx, req, actor)
}

func (s *service) GetAutoCancelableShipments(
	ctx context.Context,
	req *shipmentports.GetAutoCancelableShipmentsRequest,
) ([]*shipment.Shipment, error) {
	return s.lifecycle.GetAutoCancelableShipments(ctx, req)
}

func (s *service) AutoCancelShipments(
	ctx context.Context,
	req *shipmentports.AutoCancelShipmentsRequest,
	actor *permissionports.RequestActor,
) ([]*shipment.Shipment, error) {
	return s.lifecycle.AutoCancelShipments(ctx, req, actor)
}

func (s *service) List(
	ctx context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return s.reader.List(ctx, req)
}

func (s *service) Get(
	ctx context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return s.reader.Get(ctx, req)
}

func (s *service) GetUnassigned(
	ctx context.Context,
	req *shipmentports.GetUnassignedShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return s.reader.GetUnassigned(ctx, req)
}

func (s *service) GetByIDs(
	ctx context.Context,
	req *shipmentports.GetShipmentsByIDsRequest,
) ([]*shipment.Shipment, error) {
	return s.reader.GetByIDs(ctx, req)
}

func (s *service) SelectOptions(
	ctx context.Context,
	req *shipmentports.ShipmentSelectOptionsRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	return s.reader.SelectOptions(ctx, req)
}

func (s *service) GetUIPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*shipmentserviceports.ShipmentUIPolicy, error) {
	return s.reader.GetUIPolicy(ctx, tenantInfo)
}

func (s *service) GetPreviousRates(
	ctx context.Context,
	req *shipmentports.GetPreviousRatesRequest,
) (*pagination.ListResult[*shipmentports.PreviousRateSummary], error) {
	return s.reader.GetPreviousRates(ctx, req)
}

func (s *service) CheckForDuplicateBOLs(
	ctx context.Context,
	req *shipmentports.DuplicateBOLCheckRequest,
) error {
	return s.reader.CheckForDuplicateBOLs(ctx, req)
}

func (s *service) CalculateTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipmentports.ShipmentTotalsResponse, error) {
	return s.reader.CalculateTotals(ctx, entity, userID)
}

func (s *service) CalculateDistance(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipmentdistanceports.DistanceCalculationResponse, error) {
	return s.reader.CalculateDistance(ctx, entity)
}

func (s *service) RecalculateDistance(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentdistanceports.DistanceCalculationResponse, error) {
	return s.reader.RecalculateDistance(ctx, shipmentID, tenantInfo)
}

func (s *service) PreviewRecalculateDistance(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentdistanceports.ShipmentDistancePreview, error) {
	return s.reader.PreviewRecalculateDistance(ctx, shipmentID, tenantInfo)
}

func (s *service) Create(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	return s.writer.Create(ctx, entity, actor)
}

func (s *service) PreviewCreate(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentCreatePlan, error) {
	return s.writer.PreviewCreate(ctx, entity, actor)
}

func (s *service) Update(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	return s.writer.Update(ctx, entity, actor)
}
