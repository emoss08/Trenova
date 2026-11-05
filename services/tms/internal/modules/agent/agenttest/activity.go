package agenttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// QuietActivity tells nobody what changed.
type QuietActivity struct{}

func (QuietActivity) ProposalChanged(
	context.Context,
	*agent.AgentProposal,
	kernelports.AuditActor,
	string,
) {
}

func (QuietActivity) PlanChanged(
	context.Context,
	*agentplan.AgentPlan,
	kernelports.AuditActor,
	string,
) {
}

func (QuietActivity) RunChanged(
	context.Context,
	*agent.AgentRun,
	kernelports.AuditActor,
	string,
) {
}

func (QuietActivity) ArtifactChanged(
	context.Context,
	*assistantartifact.Artifact,
	kernelports.AuditActor,
	string,
) {
}

// QuietWatchtower puts nothing on the feed.
type QuietWatchtower struct{}

func (QuietWatchtower) Upsert(context.Context, watchtowerports.WatchtowerItemInput) {}

func (QuietWatchtower) Resolve(
	context.Context,
	pagination.TenantInfo,
	watchtower.SourceKind,
	string,
) {
}

// QuietFollowUps reports nothing back to the conversation.
type QuietFollowUps struct{}

func (QuietFollowUps) FollowUp(context.Context, *assistantports.DecisionFollowUpRequest) {}
