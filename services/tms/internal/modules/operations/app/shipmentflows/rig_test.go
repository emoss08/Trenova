package shipmentflows

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
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
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazmatsegregationruleports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/zap"
)

type rigParams struct {
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
	BillingQueueService  billingqueueports.BillingQueueService
	ServiceFailureRepo   servicefailureports.ServiceFailureRepository
	ServiceFailures      servicefailureports.ServiceFailureEvaluator
	Permissions          permissionports.PermissionEngine
	Validator            *shipmentvalidator.Validator
	AuditService         auditports.AuditService
	EventService         shipmenteventports.ShipmentEventService
	WorkflowStarter      kernelports.WorkflowStarter
	Coordinator          *shipmentstate.Coordinator
	Commercial           *shipmentcommercial.Calculator
	OrderDerivation      orderports.OrderDerivationService
	DistanceCalculation  shipmentdistanceports.DistanceCalculationService
	TenderGuard          tenderports.TenderGuard
	AgentEvents          agenteventports.AgentEventPublisher
	Quota                platformplanports.QuotaGuard
	QuickFilters         shipmentports.ShipmentQuickFilterBasisResolver
	Invalidator          shipmentinvalidationports.ShipmentInvalidator
}

type rig struct {
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
	agentEvents          agenteventports.AgentEventPublisher
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

func newRig(p *rigParams) *rig {
	s := &rig{
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

func (s *rig) assemble() {
	if s.l == nil {
		s.l = zap.NewNop()
	}
	s.effects = shipmenteffects.New(shipmenteffects.Params{
		Logger:              s.l,
		Repo:                s.repo,
		ControlRepo:         s.controlRepo,
		CommodityRepo:       s.commodityRepo,
		EquipmentTypeRepo:   s.equipmentTypeRepo,
		CustomerRepo:        s.customerRepo,
		NotificationService: s.notificationService,
		AuditService:        s.auditService,
		EventService:        s.eventService,
		OrderDerivation:     s.orderDerivation,
	})
	s.readiness = shipmentbillingreadiness.New(shipmentbillingreadiness.Params{
		Logger:             s.l,
		Repo:               s.repo,
		CustomerRepo:       s.customerRepo,
		DocumentRepo:       s.documentRepo,
		BillingRepo:        s.billingRepo,
		ServiceFailureRepo: s.serviceFailureRepo,
	})
	s.transfer = shipmentbillingtransfer.New(shipmentbillingtransfer.Params{
		Logger:              s.l,
		Repo:                s.repo,
		CustomerRepo:        s.customerRepo,
		DocumentRepo:        s.documentRepo,
		BillingRepo:         s.billingRepo,
		BillingQueueService: s.billingQueueService,
		Invalidator:         s.invalidator,
		Effects:             s.effects,
		Readiness:           s.readiness,
	})
	s.autorater = shipmentautorate.New(shipmentautorate.Params{
		Logger:              s.l,
		Repo:                s.repo,
		Invalidator:         s.invalidator,
		Commercial:          s.commercial,
		DistanceCalculation: s.distanceCalculation,
		Effects:             s.effects,
	})
	s.loading = shipmentloading.New(shipmentloading.Params{
		CommodityRepo:     s.commodityRepo,
		HazmatRuleRepo:    s.hazmatRuleRepo,
		EquipmentTypeRepo: s.equipmentTypeRepo,
		Effects:           s.effects,
	})
	s.lifecycle = shipmentlifecycle.New(shipmentlifecycle.Params{
		Logger:          s.l,
		DB:              s.db,
		Repo:            s.repo,
		UserRepo:        s.userRepo,
		ContinuityRepo:  s.continuityRepo,
		BillingRepo:     s.billingRepo,
		Permissions:     s.permissions,
		WorkflowStarter: s.workflowStarter,
		Coordinator:     s.coordinator,
		Invalidator:     s.invalidator,
		TenderGuard:     s.tenderGuard,
		Quota:           s.quota,
		Effects:         s.effects,
	})
	s.reader = shipmentread.New(shipmentread.Params{
		Logger:              s.l,
		Repo:                s.repo,
		CacheRepo:           s.cacheRepo,
		ModeProfileService:  s.modeProfileService,
		QuickFilters:        s.quickFilters,
		Commercial:          s.commercial,
		DistanceCalculation: s.distanceCalculation,
		Effects:             s.effects,
	})
	s.writer = shipmentwrite.New(shipmentwrite.Params{
		Logger:               s.l,
		DB:                   s.db,
		Repo:                 s.repo,
		OrderRepo:            s.orderRepo,
		AssignmentRepo:       s.assignmentRepo,
		ModeProfileService:   s.modeProfileService,
		PermitService:        s.permitService,
		ContinuityRepo:       s.continuityRepo,
		CustomerRepo:         s.customerRepo,
		ChargeAllocationRepo: s.chargeAllocationRepo,
		DocumentRepo:         s.documentRepo,
		BillingRepo:          s.billingRepo,
		ServiceFailures:      s.serviceFailures,
		Validator:            s.validator,
		EventService:         s.eventService,
		Coordinator:          s.coordinator,
		Invalidator:          s.invalidator,
		Commercial:           s.commercial,
		DistanceCalculation:  s.distanceCalculation,
		AgentEvents:          s.agentEvents,
		Effects:              s.effects,
		Readiness:            s.readiness,
		Transfer:             s.transfer,
	})
	if s.mutationObservers != nil {
		s.writer.SetShipmentMutationObservers(s.mutationObservers)
	}
}
