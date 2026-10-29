package tenderservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderbounce"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercancel"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderdispatch"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderescalation"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderlifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendernegotiation"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderresponse"
)

func (s *Service) assemble() {
	s.effects = tendereffects.New(tendereffects.Deps{
		Logger:           s.l,
		Repo:             s.repo,
		EventService:     s.eventService,
		AuditService:     s.auditService,
		Invalidator:      s.invalidator,
		Notifications:    s.notifications,
		DispatchControls: s.dispatchControls,
		IntelGate:        s.intelGate,
		Lifecycle:        s.lifecycle,
	})
	s.creator = tendercreate.New(tendercreate.Deps{
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
	s.dispatcher = tenderdispatch.New(tenderdispatch.Deps{
		Logger:       s.l,
		Repo:         s.repo,
		CarrierRepo:  s.carrierRepo,
		ShipmentRepo: s.shipmentRepo,
		OrgRepo:      s.orgRepo,
		Config:       s.cfg,
		Templates:    s.templates,
		EmailService: s.emailService,
		EDIChannel:   s.ediChannel,
		IntelGate:    s.intelGate,
		Effects:      s.effects,
	})
	s.negotiator = tendernegotiation.New(tendernegotiation.Deps{
		Logger:         s.l,
		Repo:           s.repo,
		AssignmentRepo: s.assignmentRepo,
		ShipmentRepo:   s.shipmentRepo,
		Workflows:      s.workflows,
		Permissions:    s.permissions,
		Effects:        s.effects,
	})
	s.bouncer = tenderbounce.New(tenderbounce.Deps{
		Logger:                s.l,
		Repo:                  s.repo,
		CarrierAssignmentRepo: s.carrierAssignmentRepo,
		Invalidator:           s.invalidator,
		Assigner:              s.assigner,
		Creator:               s.creator,
		Effects:               s.effects,
	})
	s.canceler = tendercancel.New(tendercancel.Deps{
		Logger:      s.l,
		Repo:        s.repo,
		Workflows:   s.workflows,
		Invalidator: s.invalidator,
		Effects:     s.effects,
	})
	s.escalator = tenderescalation.New(tenderescalation.Deps{
		Logger:  s.l,
		Creator: s.creator,
		Effects: s.effects,
	})
	s.offerLifecycle = tenderlifecycle.New(tenderlifecycle.Deps{
		Logger:                s.l,
		DB:                    s.db,
		Repo:                  s.repo,
		CarrierAssignmentRepo: s.carrierAssignmentRepo,
		Invalidator:           s.invalidator,
		Assigner:              s.assigner,
		RateConIssuer:         s.rateConIssuer,
		Canceler:              s.canceler,
		Effects:               s.effects,
		Escalator:             s.escalator,
		Negotiator:            s.negotiator,
	})
	s.responder = tenderresponse.New(tenderresponse.Deps{
		Logger:     s.l,
		Repo:       s.repo,
		Workflows:  s.workflows,
		Bouncer:    s.bouncer,
		Dispatcher: s.dispatcher,
		Effects:    s.effects,
		Negotiator: s.negotiator,
	})
}
