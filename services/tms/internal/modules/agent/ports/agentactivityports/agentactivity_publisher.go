package agentactivityports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// AgentActivityPublisher tells connected clients that an agent record
// changed, so a queue or a pane refreshes without polling. Every method is
// best effort: a lost invalidation costs a refresh, never the write.
type AgentActivityPublisher interface {
	ProposalChanged(
		ctx context.Context,
		proposal *agent.AgentProposal,
		actor kernelports.AuditActor,
		action string,
	)
	PlanChanged(
		ctx context.Context,
		plan *agentplan.AgentPlan,
		actor kernelports.AuditActor,
		action string,
	)
	RunChanged(
		ctx context.Context,
		run *agent.AgentRun,
		actor kernelports.AuditActor,
		action string,
	)
	ArtifactChanged(
		ctx context.Context,
		artifact *assistantartifact.Artifact,
		actor kernelports.AuditActor,
		action string,
	)
}

const ActivityCreated = "created"

const ActivityUpdated = "updated"

// ActivityApproving, ActivityCommitted, ActivityUndone and ActivityReopened
// follow an approval through its undo window: approved and waiting, gone
// through, taken back by its decider, or sent back to its decider because it
// could no longer go through as approved.
const ActivityApproving = "approving"

const ActivityCommitted = "committed"

const ActivityUndone = "undone"

const ActivityReopened = "reopened"
