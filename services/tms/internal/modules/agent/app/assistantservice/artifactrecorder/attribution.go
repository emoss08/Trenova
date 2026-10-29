package artifactrecorder

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/attribution"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *Recorder) noteReader(artifact *assistantartifact.Artifact) {
	if !attribution.NamesItsReader(artifact.Kind) {
		return
	}
	reader := r.readingAgent()
	if reader == nil {
		return
	}
	artifact.Payload[toolartifacts.PayloadReadBy] = attribution.ArtifactReader{
		ID:     r.thread.AgentDefinitionID.String(),
		Name:   reader.Name,
		Icon:   reader.Icon,
		Accent: reader.Accent,
	}
}

func (r *Recorder) readingAgent() *attribution.AgentIdentity {
	if r.readBy != nil || r.reads == nil {
		return r.readBy
	}
	agentID := r.agentThread().AgentDefinitionID
	if agentID.IsNil() {
		return r.readBy
	}
	identity, found := r.reads.Attribution.AgentIdentities(r.ctx, r.tenant, []pulid.ID{agentID})[agentID]
	if found && identity.Name != "" {
		r.readBy = &identity
	}

	return r.readBy
}
