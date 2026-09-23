package agentevents

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Definitions repositories.AgentDefinitionRepository
	RunService  services.AgentRunService
}

type Publisher struct {
	l           *zap.Logger
	definitions repositories.AgentDefinitionRepository
	runService  services.AgentRunService
}

func New(p Params) services.AgentEventPublisher {
	return &Publisher{
		l:           p.Logger.Named("service.agentevents"),
		definitions: p.Definitions,
		runService:  p.RunService,
	}
}

func (p *Publisher) Publish(ctx context.Context, event services.AgentEvent) {
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

	definitions, err := p.definitions.ListEnabledByTrigger(
		ctx,
		repositories.ListAgentDefinitionsByTriggerRequest{
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
	event services.AgentEvent,
) {
	log = log.With(zap.String("definitionId", definition.ID.String()))

	// A subject takes one open run at a time. That is enforced where the run
	// starts, by its workflow id, so two events arriving together cannot both
	// pass a check and start two.
	_, err := p.runService.StartForDefinition(
		ctx,
		&services.StartAgentRunForDefinitionRequest{
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
	case errors.Is(err, services.ErrAgentRunAlreadyOpen):
		log.Debug("agent already has an open run for this subject; event skipped")
	case err != nil:
		log.Error("failed to start agent run for event", zap.Error(err))
	}
}
