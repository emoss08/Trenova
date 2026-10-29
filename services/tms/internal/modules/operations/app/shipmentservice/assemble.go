package shipmentservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentautorate"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingreadiness"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingtransfer"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentlifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentloading"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentread"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentwrite"
)

func (s *service) assemble() {
	s.effects = shipmenteffects.New(shipmenteffects.Deps{
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
	s.readiness = shipmentbillingreadiness.New(shipmentbillingreadiness.Deps{
		Logger:             s.l,
		Repo:               s.repo,
		CustomerRepo:       s.customerRepo,
		DocumentRepo:       s.documentRepo,
		BillingRepo:        s.billingRepo,
		ServiceFailureRepo: s.serviceFailureRepo,
	})
	s.transfer = shipmentbillingtransfer.New(shipmentbillingtransfer.Deps{
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
	s.autorater = shipmentautorate.New(shipmentautorate.Deps{
		Logger:              s.l,
		Repo:                s.repo,
		Invalidator:         s.invalidator,
		Commercial:          s.commercial,
		DistanceCalculation: s.distanceCalculation,
		Effects:             s.effects,
	})
	s.loading = shipmentloading.New(shipmentloading.Deps{
		CommodityRepo:     s.commodityRepo,
		HazmatRuleRepo:    s.hazmatRuleRepo,
		EquipmentTypeRepo: s.equipmentTypeRepo,
		Effects:           s.effects,
	})
	s.lifecycle = shipmentlifecycle.New(shipmentlifecycle.Deps{
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
	s.reader = shipmentread.New(shipmentread.Deps{
		Logger:              s.l,
		Repo:                s.repo,
		CacheRepo:           s.cacheRepo,
		ModeProfileService:  s.modeProfileService,
		QuickFilters:        s.quickFilters,
		Commercial:          s.commercial,
		DistanceCalculation: s.distanceCalculation,
		Effects:             s.effects,
	})
	s.writer = shipmentwrite.New(shipmentwrite.Deps{
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
		MutationObservers:    s.mutationObservers,
		Effects:              s.effects,
		Readiness:            s.readiness,
		Transfer:             s.transfer,
	})
}
