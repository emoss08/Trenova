package assistantservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) showWritten(
	ctx context.Context,
	thread *conversation.Thread,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
	decided []*agent.AgentProposal,
) string {
	recorder := s.newArtifactRecorder(ctx, thread, tenant, actor, nil)
	if recorder == nil {
		return ""
	}

	var notes strings.Builder
	for _, proposal := range decided {
		if proposal == nil || proposal.Status != agent.ProposalStatusExecuted {
			continue
		}
		saved := recorder.Written(
			toolartifacts.DecisionCallID+proposal.ID.String(),
			proposal.ToolName,
			proposal.ExecutionResult,
		)
		if saved != nil {
			notes.WriteString(runtimetools.WrittenNote(toolartifacts.ShownArtifact(saved)))
		}
	}

	return notes.String()
}
