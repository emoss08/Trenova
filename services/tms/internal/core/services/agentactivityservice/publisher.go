package agentactivityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// The realtime resources a Desk listens on. Proposals and runs reuse their
// permission resource; plans and artifacts have no permission of their own
// and are named here once.
const (
	ResourceAgentPlan         = "agent_plan"
	ResourceAssistantArtifact = "assistant_artifact"
)

type Params struct {
	fx.In

	Logger   *zap.Logger
	Realtime services.RealtimeService `optional:"true"`
}

// Publisher announces agent record changes to connected clients. Without a
// realtime service it announces nothing, which is how a worker without one
// still records its work.
type Publisher struct {
	l        *zap.Logger
	realtime services.RealtimeService
}

func New(p Params) services.AgentActivityPublisher {
	return &Publisher{
		l:        p.Logger.Named("service.agentactivity"),
		realtime: p.Realtime,
	}
}

type change struct {
	orgID    pulid.ID
	buID     pulid.ID
	resource string
	action   string
	recordID pulid.ID
	entity   any
}

func (p *Publisher) publish(ctx context.Context, actor services.AuditActor, c change) {
	if p == nil || p.realtime == nil {
		return
	}

	err := realtimeinvalidation.Publish(ctx, p.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: c.orgID,
		BusinessUnitID: c.buID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       c.resource,
		Action:         c.action,
		RecordID:       c.recordID,
		Entity:         c.entity,
	})
	if err != nil {
		p.l.Warn("agent activity invalidation lost",
			zap.String("resource", c.resource),
			zap.String("record", c.recordID.String()),
			zap.Error(err),
		)
	}
}

func (p *Publisher) ProposalChanged(
	ctx context.Context,
	proposal *agent.AgentProposal,
	actor services.AuditActor,
	action string,
) {
	if proposal == nil {
		return
	}
	p.publish(ctx, actor, change{
		orgID:    proposal.OrganizationID,
		buID:     proposal.BusinessUnitID,
		resource: permission.ResourceAgentProposal.String(),
		action:   action,
		recordID: proposal.ID,
		entity:   proposal,
	})
}

func (p *Publisher) PlanChanged(
	ctx context.Context,
	plan *agent.AgentPlan,
	actor services.AuditActor,
	action string,
) {
	if plan == nil {
		return
	}
	p.publish(ctx, actor, change{
		orgID:    plan.OrganizationID,
		buID:     plan.BusinessUnitID,
		resource: ResourceAgentPlan,
		action:   action,
		recordID: plan.ID,
		entity:   plan,
	})
}

func (p *Publisher) RunChanged(
	ctx context.Context,
	run *agent.AgentRun,
	actor services.AuditActor,
	action string,
) {
	if run == nil {
		return
	}
	p.publish(ctx, actor, change{
		orgID:    run.OrganizationID,
		buID:     run.BusinessUnitID,
		resource: permission.ResourceAgentRun.String(),
		action:   action,
		recordID: run.ID,
		entity:   run,
	})
}

func (p *Publisher) ArtifactChanged(
	ctx context.Context,
	artifact *assistantartifact.Artifact,
	actor services.AuditActor,
	action string,
) {
	if artifact == nil {
		return
	}
	// The payload can be large and the pane re-reads the thread anyway, so
	// the announcement carries the reference and not the rows.
	p.publish(ctx, actor, change{
		orgID:    artifact.OrganizationID,
		buID:     artifact.BusinessUnitID,
		resource: ResourceAssistantArtifact,
		action:   action,
		recordID: artifact.ID,
		entity: map[string]any{
			"id":       artifact.ID,
			"threadId": artifact.ThreadID,
			"kind":     artifact.Kind,
			"status":   artifact.Status,
		},
	})
}
