package artifactrecorder

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
)

func (r *Recorder) Written(
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

func (r *Recorder) writtenCard(
	callID, writeTool string,
	executed *agentrecord.ToolExecutionResult,
) *assistantartifact.Artifact {
	if card := toolartifacts.PortalActionArtifact(callID, executed); card != nil {
		return card
	}
	if card := r.readCard(callID, executed.Record); card != nil {
		card.Payload[toolartifacts.PayloadWritten] = executed.Action

		return card
	}

	card := toolartifacts.RecordCardArtifact(&toolartifacts.RecordCard{
		CallID:   callID,
		ToolName: writeTool,
		Entity:   executed.Record.EntityType,
		Record: map[string]any{
			toolartifacts.WrittenIDField:   executed.Record.ID,
			toolartifacts.WrittenNameField: executed.Name,
		},
		Order: []string{toolartifacts.WrittenNameField},
	})
	if card == nil {
		return nil
	}
	card.LineageKey = assistantartifact.LineageKeyFor(card.Kind, writeTool, map[string]any{
		toolartifacts.WrittenIDField: executed.Record.ID,
	})
	card.Payload[toolartifacts.PayloadWritten] = executed.Action

	return card
}

func (r *Recorder) readCard(
	callID string,
	ref *agentrecord.RecordRef,
) *assistantartifact.Artifact {
	if r.reads == nil || r.reads.Runtime == nil {
		return nil
	}
	read, ok := r.reads.Runtime.ReadRecord(r.ctx, r.reader, ref)
	if !ok {
		return nil
	}
	document, isDocument := artifactpayload.ToJSONDocument(read.Data)
	if !isDocument {
		return nil
	}
	card := toolartifacts.EntityCardArtifact(callID, read.Tool, document)
	if card == nil {
		return nil
	}
	card.LineageKey = assistantartifact.LineageKeyFor(card.Kind, read.Tool, read.Params)

	return card
}
