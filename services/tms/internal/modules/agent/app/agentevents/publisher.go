package agentevents

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Definitions agentdefinitionports.AgentDefinitionRepository
	RunService  agentruntimeports.AgentRunService
	Plans       platformplanports.PlanService    `optional:"true"`
	Waits       agentwaitports.AgentWaitNotifier `optional:"true"`
}

type Publisher struct {
	l           *zap.Logger
	definitions agentdefinitionports.AgentDefinitionRepository
	runService  agentruntimeports.AgentRunService
	plans       platformplanports.PlanService
	waits       agentwaitports.AgentWaitNotifier
	// attachments are bound after the graph is built, by
	// RegisterAttachmentObservers: what shows an attached file's card
	// depends, through its tools, on services that publish events.
	attachments   atomic.Pointer[[]agenteventports.AttachmentExtractionObserver]
	attachmentsMu sync.Mutex
}

func New(p Params) *Publisher {
	return &Publisher{
		l:           p.Logger.Named("service.agentevents"),
		definitions: p.Definitions,
		runService:  p.RunService,
		plans:       p.Plans,
		waits:       p.Waits,
	}
}

type ObserverParams struct {
	fx.In

	Publisher *Publisher
	Observers []agenteventports.AttachmentExtractionObserver `group:"attachment_extraction_observers"`
}

// RegisterAttachmentObservers binds the observers told when a document's
// extraction is saved.
func RegisterAttachmentObservers(p ObserverParams) {
	p.Publisher.ObserveAttachments(p.Observers...)
}

// ObserveAttachments adds observers told of every document extraction from
// now on.
func (p *Publisher) ObserveAttachments(observers ...agenteventports.AttachmentExtractionObserver) {
	if len(observers) == 0 {
		return
	}

	p.attachmentsMu.Lock()
	defer p.attachmentsMu.Unlock()

	var current []agenteventports.AttachmentExtractionObserver
	if loaded := p.attachments.Load(); loaded != nil {
		current = *loaded
	}
	next := make([]agenteventports.AttachmentExtractionObserver, 0, len(current)+len(observers))
	next = append(next, current...)
	for _, observer := range observers {
		if observer != nil {
			next = append(next, observer)
		}
	}
	p.attachments.Store(&next)
}

func (p *Publisher) Publish(ctx context.Context, event *agenteventports.AgentEvent) {
	log := p.l.With(
		zap.String("kind", string(event.Kind)),
		zap.String("subjectId", event.SubjectID.String()),
		zap.String("orgId", event.TenantInfo.OrgID.String()),
	)

	if !event.Kind.IsValid() {
		log.Warn("agent event kind is not known; nothing will run")
		return
	}
	if event.SubjectID.IsNil() || event.TenantInfo.OrgID.IsNil() || event.TenantInfo.BuID.IsNil() {
		log.Warn("agent event is missing its subject or tenant; nothing will run")
		return
	}
	ctx = dbscope.WithTenant(ctx, event.TenantInfo.DBTenant())

	// A wait is the agent's own work parked on what this event says
	// happened. It is told before the plan is consulted: an organization
	// without automated agents still has conversations that wait.
	if p.waits != nil {
		p.waits.NotifyEvent(ctx, event)
	}
	// A file read for a conversation is shown there whatever the plan says
	// of automated agents: the person asked for it by attaching it.
	if event.Kind == agenttrigger.EventDocumentExtracted {
		if observers := p.attachments.Load(); observers != nil {
			for _, observer := range *observers {
				observer.AttachmentExtracted(ctx, event)
			}
		}
	}

	allowed, err := planservice.Allows(
		ctx,
		p.plans,
		event.TenantInfo,
		platformplan.CapabilityAgentAutomation,
	)
	if err != nil {
		log.Error("failed to resolve the organization's plan for an agent event", zap.Error(err))
		return
	}
	if !allowed {
		log.Debug("the organization's plan does not include automated agents; event skipped")
		return
	}

	definitions, err := p.definitions.ListEnabledByTrigger(
		ctx,
		agentdefinitionports.ListAgentDefinitionsByTriggerRequest{
			TenantInfo: event.TenantInfo,
			Mode:       agentdefinition.TriggerEvent,
			EventKind:  event.Kind,
		},
	)
	if err != nil {
		log.Error("failed to list agents subscribed to event", zap.Error(err))
		return
	}

	for _, definition := range definitions {
		p.startFor(ctx, log, definition, event)
	}
}

func (p *Publisher) startFor(
	ctx context.Context,
	log *zap.Logger,
	definition *agentdefinition.Definition,
	event *agenteventports.AgentEvent,
) {
	log = log.With(zap.String("definitionId", definition.ID.String()))

	// A subject takes one open run at a time. That is enforced where the run
	// starts, by its workflow id, so two events arriving together cannot both
	// pass a check and start two.
	_, err := p.runService.StartForDefinition(
		ctx,
		&agentruntimeports.StartAgentRunForDefinitionRequest{
			DefinitionID: definition.ID,
			SubjectType:  event.Kind.SubjectType(),
			SubjectID:    event.SubjectID,
			Trigger:      agent.RunTriggerEvent,
			EventKind:    event.Kind,
			TenantInfo:   event.TenantInfo,
		},
		nil,
	)
	switch {
	case errors.Is(err, agentruntimeports.ErrAgentRunAlreadyOpen):
		log.Debug("agent already has an open run for this subject; event skipped")
	case err != nil:
		log.Error("failed to start agent run for event", zap.Error(err))
	}
}
