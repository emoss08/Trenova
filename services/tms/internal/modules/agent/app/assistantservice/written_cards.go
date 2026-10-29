package assistantservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	payloadWritten    = "written"
	decisionCallID    = "decision:"
	writtenChangeVerb = "change "
	writtenIDField    = "id"
	writtenNameField  = "name"
)

func (r *artifactRecorder) written(
	callID, writeTool string,
	executed *agentrecord.ToolExecutionResult,
) *assistantartifact.Artifact {
	if r == nil || executed == nil || executed.Record == nil ||
		strings.TrimSpace(executed.Record.ID) == "" {
		return nil
	}

	card := r.writtenCard(callID, writeTool, executed)
	if card == nil {
		return nil
	}
	saved, err := r.save(card)
	if err != nil {
		return nil
	}

	return saved
}

func (r *artifactRecorder) writtenCard(
	callID, writeTool string,
	executed *agentrecord.ToolExecutionResult,
) *assistantartifact.Artifact {
	if card := portalActionArtifact(callID, executed); card != nil {
		return card
	}
	if card := r.readCard(callID, executed.Record); card != nil {
		card.Payload[payloadWritten] = executed.Action

		return card
	}

	card := recordCardArtifact(&recordCard{
		callID:   callID,
		toolName: writeTool,
		entity:   executed.Record.EntityType,
		record: map[string]any{
			writtenIDField:   executed.Record.ID,
			writtenNameField: executed.Name,
		},
		order: []string{writtenNameField},
	})
	if card == nil {
		return nil
	}
	card.LineageKey = assistantartifact.LineageKeyFor(card.Kind, writeTool, map[string]any{
		writtenIDField: executed.Record.ID,
	})
	card.Payload[payloadWritten] = executed.Action

	return card
}

func (r *artifactRecorder) readCard(
	callID string,
	ref *agentrecord.RecordRef,
) *assistantartifact.Artifact {
	if r.svc == nil || r.svc.runtime == nil {
		return nil
	}
	read, ok := r.svc.runtime.ReadRecord(r.ctx, r.reader, ref)
	if !ok {
		return nil
	}
	document, isDocument := toJSONDocument(read.Data)
	if !isDocument {
		return nil
	}
	card := entityCardArtifact(callID, read.Tool, document)
	if card == nil {
		return nil
	}
	card.LineageKey = assistantartifact.LineageKeyFor(card.Kind, read.Tool, read.Params)

	return card
}

func writtenPhrase(artifact *assistantartifact.Artifact) string {
	action := typeutils.StringOfTrimmed(artifact.Payload[payloadWritten])
	if action == "" {
		return ""
	}

	return writtenChangeVerb + action
}

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
		saved := recorder.written(
			decisionCallID+proposal.ID.String(), proposal.ToolName, proposal.ExecutionResult,
		)
		if saved != nil {
			notes.WriteString(agentruntime.WrittenNote(shownArtifact(saved)))
		}
	}

	return notes.String()
}
