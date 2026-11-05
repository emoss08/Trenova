package assistanttest

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordanchorports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type QuietActivity struct {
	agentactivityports.AgentActivityPublisher
}

func (QuietActivity) ArtifactChanged(
	context.Context,
	*assistantartifact.Artifact,
	kernelports.AuditActor,
	string,
) {
}

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

type BareContexts struct{}

func (BareContexts) Build(
	context.Context,
	*agentruntimeports.RuntimeContextRequest,
) (agentturn.RuntimeContext, error) {
	return agentturn.RuntimeContext{}, errors.New("this test builds no context")
}

type NoAnchors struct{}

func (NoAnchors) Anchor(
	context.Context,
	*recordanchorports.RecordAnchorRequest,
) []agentturn.RuntimeAnchor {
	return nil
}

type NoCases struct{}

func (NoCases) Attach(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	[]*conversation.Thread,
) error {
	return nil
}
