package tenderflows

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenderingconfig"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderbounce"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercancel"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderdispatch"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderescalation"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderlifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendernegotiation"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderresponse"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/zap"
)

type rig struct {
	l                     *zap.Logger
	db                    kernelports.Transactor
	repo                  tenderports.TenderRepository
	guideRepo             tenderports.RoutingGuideRepository
	carrierRepo           carrierports.CarrierRepository
	carrierAssignmentRepo carrierassignmentports.CarrierAssignmentRepository
	assignmentRepo        assignmentports.AssignmentRepository
	shipmentRepo          shipmentports.ShipmentRepository
	holdRepo              shipmentholdports.ShipmentHoldRepository
	orgRepo               tenantports.OrganizationRepository
	workflows             kernelports.WorkflowStarter
	eventService          shipmenteventports.ShipmentEventService
	auditService          auditports.AuditService
	invalidator           shipmentinvalidationports.ShipmentInvalidator
	appConfig             *appconfig.Config
	tenderingConfig       *tenderingconfig.Config
	templates             documenttemplateports.DocumentTemplateResolver
	emailService          emailports.EmailService
	notifications         *notificationservice.Service
	assigner              tenderports.CarrierMoveAssigner
	ediChannel            tenderports.EDITenderChannel
	rateConIssuer         rateconfirmationports.RateConfirmationIssuer
	rateEngine            ratequoteports.RateEngine
	dispatchControls      dispatchcontrolports.DispatchControlRepository
	permissions           permissionports.PermissionEngine
	intelGate             carrierintelports.CarrierIntelGate
	lifecycle             carrierintelports.CarrierLifecycleObserver
	effects               *tendereffects.Effects
	creator               *tendercreate.Service
	dispatcher            *tenderdispatch.Service
	negotiator            *tendernegotiation.Service
	bouncer               *tenderbounce.Service
	canceler              *tendercancel.Service
	escalator             *tenderescalation.Service
	offerLifecycle        *tenderlifecycle.Service
	responder             *tenderresponse.Service
}

func (s *rig) assemble() {
	s.effects = tendereffects.New(tendereffects.Params{
		Logger:           s.l,
		Repo:             s.repo,
		EventService:     s.eventService,
		AuditService:     s.auditService,
		Invalidator:      s.invalidator,
		Notifications:    s.notifications,
		DispatchControls: s.dispatchControls,
	})
	s.creator = tendercreate.New(tendercreate.Params{
		Logger:                s.l,
		DB:                    s.db,
		Repo:                  s.repo,
		GuideRepo:             s.guideRepo,
		CarrierRepo:           s.carrierRepo,
		CarrierAssignmentRepo: s.carrierAssignmentRepo,
		AssignmentRepo:        s.assignmentRepo,
		ShipmentRepo:          s.shipmentRepo,
		HoldRepo:              s.holdRepo,
		Workflows:             s.workflows,
		Invalidator:           s.invalidator,
		RateEngine:            s.rateEngine,
		Effects:               s.effects,
	})
	s.dispatcher = tenderdispatch.New(tenderdispatch.Params{
		Logger:          s.l,
		Repo:            s.repo,
		CarrierRepo:     s.carrierRepo,
		ShipmentRepo:    s.shipmentRepo,
		OrgRepo:         s.orgRepo,
		AppConfig:       s.appConfig,
		TenderingConfig: s.tenderingConfig,
		Templates:       s.templates,
		EmailService:    s.emailService,
		EDIChannel:      s.ediChannel,
		Effects:         s.effects,
	})
	s.negotiator = tendernegotiation.New(tendernegotiation.Params{
		Logger:         s.l,
		Repo:           s.repo,
		AssignmentRepo: s.assignmentRepo,
		ShipmentRepo:   s.shipmentRepo,
		Workflows:      s.workflows,
		Permissions:    s.permissions,
		Effects:        s.effects,
	})
	s.bouncer = tenderbounce.New(tenderbounce.Params{
		Logger:                s.l,
		Repo:                  s.repo,
		CarrierAssignmentRepo: s.carrierAssignmentRepo,
		Invalidator:           s.invalidator,
		Creator:               s.creator,
		Effects:               s.effects,
	})
	s.canceler = tendercancel.New(tendercancel.Params{
		Logger:      s.l,
		Repo:        s.repo,
		Workflows:   s.workflows,
		Invalidator: s.invalidator,
		Effects:     s.effects,
	})
	s.escalator = tenderescalation.New(tenderescalation.Params{
		Logger:  s.l,
		Creator: s.creator,
		Effects: s.effects,
	})
	s.offerLifecycle = tenderlifecycle.New(tenderlifecycle.Params{
		Logger:                s.l,
		DB:                    s.db,
		Repo:                  s.repo,
		CarrierAssignmentRepo: s.carrierAssignmentRepo,
		Invalidator:           s.invalidator,
		RateConIssuer:         s.rateConIssuer,
		Canceler:              s.canceler,
		Effects:               s.effects,
		Escalator:             s.escalator,
		Negotiator:            s.negotiator,
	})
	s.responder = tenderresponse.New(tenderresponse.Params{
		Logger:     s.l,
		Repo:       s.repo,
		Workflows:  s.workflows,
		Bouncer:    s.bouncer,
		Dispatcher: s.dispatcher,
		Effects:    s.effects,
		Negotiator: s.negotiator,
	})
	if s.intelGate != nil {
		s.effects.SetIntelGate(s.intelGate)
		s.dispatcher.SetIntelGate(s.intelGate)
	}
	if s.lifecycle != nil {
		s.effects.SetLifecycleObserver(s.lifecycle)
	}
	if s.assigner != nil {
		s.bouncer.SetCarrierMoveAssigner(s.assigner)
		s.offerLifecycle.SetCarrierMoveAssigner(s.assigner)
	}
}
