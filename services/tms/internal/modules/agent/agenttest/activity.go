package agenttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
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

// OpenReadScopes narrows no read.
type OpenReadScopes struct{}

func (OpenReadScopes) Resolver(*permissionports.PermissionCheckRequest) readscope.Resolver {
	return func(context.Context) (readscope.Grants, error) { return nil, nil }
}

func (QuietFollowUps) ResumeFollowUps(context.Context, assistantports.ResumeFollowUpsRequest) {}

// QuietReflections reflects on nothing.
type QuietReflections struct{}

func (QuietReflections) AfterTurn(context.Context, *agentreflectionports.ReflectOnThreadRequest) {}

func (QuietReflections) AfterRun(context.Context, *agentreflectionports.ReflectOnRunRequest) {}

// QuietChapters cues no chapter.
type QuietChapters struct{}

func (QuietChapters) Cue(context.Context, *assistantchapterports.ChapterTurnRequest) {}

// QuietQueue has nothing waiting behind a turn.
type QuietQueue struct{}

func (QuietQueue) SettleQueue(
	context.Context,
	*assistantturnports.SettleQueueRequest,
) *assistantturnports.QueuedTurn {
	return nil
}

// QuietTrajectory keeps no account of a run.
type QuietTrajectory struct{}

func (QuietTrajectory) Recorder(
	context.Context,
	pagination.TenantInfo,
	agentruntimeports.RunStepOwner,
) agentruntimeports.AgentRunEventWriter {
	return nil
}
