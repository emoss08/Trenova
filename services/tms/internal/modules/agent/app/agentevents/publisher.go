package agentevents

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
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
	EventRuns   agenteventports.EventRunQueue
	Plans       platformplanports.PlanService
	Waits       agentwaitports.AgentWaitNotifier
	Routines    []agenteventports.EventRoutine `group:"agent_event_routines"`
}

type Publisher struct {
	l           *zap.Logger
	definitions agentdefinitionports.AgentDefinitionRepository
	eventRuns   agenteventports.EventRunQueue
	plans       platformplanports.PlanService
	waits       agentwaitports.AgentWaitNotifier
	routines    []agenteventports.EventRoutine
	// attachments are bound after the graph is built, by
	// RegisterAttachmentObservers: what shows an attached file's card
	// depends, through its tools, on services that publish events.
	attachments   atomic.Pointer[[]agenteventports.AttachmentExtractionObserver]
	attachmentsMu sync.Mutex
	// observers are bound the same way, by RegisterEventObservers.
	observers atomic.Pointer[[]agenteventports.AgentEventObserver]
}

func New(p Params) *Publisher {
	return &Publisher{
		l:           p.Logger.Named("service.agentevents"),
		definitions: p.Definitions,
		eventRuns:   p.EventRuns,
		plans:       p.Plans,
		waits:       p.Waits,
		routines:    p.Routines,
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

type EventObserverParams struct {
	fx.In

	Publisher *Publisher
	Observers []agenteventports.AgentEventObserver `group:"agent_event_observers"`
}

// RegisterEventObservers binds the observers told of every event automation
// may act on.
func RegisterEventObservers(p EventObserverParams) {
	observers := make([]agenteventports.AgentEventObserver, 0, len(p.Observers))
	for _, observer := range p.Observers {
		if observer != nil {
			observers = append(observers, observer)
		}
	}
	p.Publisher.observers.Store(&observers)
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
	p.waits.NotifyEvent(ctx, event)
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

	if observers := p.observers.Load(); observers != nil {
		for _, observer := range *observers {
			observer.ObserveAgentEvent(ctx, event)
		}
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

// startFor hands the event to the agent: to a routine that does that part of
// the job in code when one does, or else to the agent's queue, which starts
// its run when it has room. A subject keeps one open run at a time; that is
// held where the run starts, so two events arriving together start one.
func (p *Publisher) startFor(
	ctx context.Context,
	log *zap.Logger,
	definition *agentdefinition.Definition,
	event *agenteventports.AgentEvent,
) {
	log = log.With(zap.String("definitionId", definition.ID.String()))

	for _, routine := range p.routines {
		if !routine.Handles(definition, event.Kind) {
			continue
		}
		if err := routine.Take(ctx, definition, event); err != nil {
			log.Error("failed to hand an agent event to its routine", zap.Error(err))
		}

		return
	}

	if err := p.eventRuns.Enqueue(ctx, &agenteventports.EventRunRequest{
		TenantInfo:   event.TenantInfo,
		DefinitionID: definition.ID,
		Kind:         event.Kind,
		SubjectID:    event.SubjectID,
	}); err != nil {
		log.Error("failed to queue an agent run for an event", zap.Error(err))
	}
}
