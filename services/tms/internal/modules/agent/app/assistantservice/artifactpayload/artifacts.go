package artifactpayload

import (
	"maps"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const (
	ToolGetShipmentDraft = "intake_get_draft"
)

// StringsOf reads a JSON array of strings, dropping anything that is not one.
func StringsOf(value any) []string {
	if typed, isStrings := value.([]string); isStrings {
		return slices.DeleteFunc(slices.Clone(typed), func(text string) bool { return text == "" })
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		if text, isText := entry.(string); isText && text != "" {
			out = append(out, text)
		}
	}

	return out
}

// DraftStatus is where a draft is, read from the proposal that carries the
// decision. A draft never holds its own decision state.
func DraftStatus(status agent.ProposalStatus) assistantartifact.Status {
	switch status {
	case agent.ProposalStatusExecuted, agent.ProposalStatusSimulated:
		return assistantartifact.StatusSent
	case agent.ProposalStatusRejected, agent.ProposalStatusExpired,
		agent.ProposalStatusSuperseded, agent.ProposalStatusExecutionFailed,
		agent.ProposalStatusSkipped:
		return assistantartifact.StatusFailed
	case agent.ProposalStatusPending, agent.ProposalStatusAccepted, agent.ProposalStatusModified:
		return assistantartifact.StatusPending
	default:
		return assistantartifact.StatusPending
	}
}

func DecisionStatus(status agent.ProposalStatus) assistantartifact.Status {
	switch status {
	case agent.ProposalStatusExecuted, agent.ProposalStatusSimulated:
		return assistantartifact.StatusReady
	case agent.ProposalStatusRejected, agent.ProposalStatusExpired,
		agent.ProposalStatusSuperseded, agent.ProposalStatusExecutionFailed,
		agent.ProposalStatusSkipped:
		return assistantartifact.StatusFailed
	case agent.ProposalStatusPending, agent.ProposalStatusAccepted, agent.ProposalStatusModified:
		return assistantartifact.StatusPending
	default:
		return assistantartifact.StatusPending
	}
}

func PlanStatus(status agentplan.PlanStatus) assistantartifact.Status {
	switch status {
	case agentplan.PlanStatusCompleted:
		return assistantartifact.StatusReady
	case agentplan.PlanStatusFailed, agentplan.PlanStatusRejected, agentplan.PlanStatusExpired:
		return assistantartifact.StatusFailed
	case agentplan.PlanStatusPending, agentplan.PlanStatusApproved:
		return assistantartifact.StatusPending
	default:
		return assistantartifact.StatusPending
	}
}

func PayloadSize(payload map[string]any) int {
	encoded, err := sonic.Marshal(payload)
	if err != nil {
		return assistantartifact.MaxPayloadBytes + 1
	}

	return len(encoded)
}

// JsonDocument is a tool result in its JSON form, which is the contract the
// tool publishes; its Go type is the tool's own business. The encoding is
// kept for the one reader that needs the order the fields were written in.
type JsonDocument struct {
	Fields  map[string]any
	encoded []byte
}

func ToJSONDocument(data any) (JsonDocument, bool) {
	if m, ok := data.(map[string]any); ok {
		return JsonDocument{Fields: m}, true
	}

	encoded, err := sonic.Marshal(data)
	if err != nil {
		return JsonDocument{}, false
	}
	var out map[string]any
	if err = sonic.Unmarshal(encoded, &out); err != nil || out == nil {
		return JsonDocument{}, false
	}

	return JsonDocument{Fields: out, encoded: encoded}, true
}

// KeyOrder is the order the result declared its fields in, or alphabetical
// when it arrived as a map and so declared none.
func (d JsonDocument) KeyOrder() []string {
	if len(d.encoded) > 0 {
		if keys := jsonutils.ObjectKeyOrder(d.encoded); len(keys) == len(d.Fields) {
			return keys
		}
	}

	return slices.Sorted(maps.Keys(d.Fields))
}

func ToAssistantArtifact(
	artifact *assistantartifact.Artifact,
) assistantartifactports.AssistantArtifact {
	return assistantartifactports.AssistantArtifact{
		ID:               artifact.ID,
		ThreadID:         artifact.ThreadID,
		MessageID:        artifact.MessageID,
		RunID:            artifact.RunID,
		ProposalID:       artifact.ProposalID,
		PlanID:           artifact.PlanID,
		Kind:             artifact.Kind,
		Status:           artifact.Status,
		Title:            artifact.Title,
		Payload:          artifact.Payload,
		SourceToolCallID: artifact.SourceToolCallID,
		Pinned:           artifact.Pinned,
		LineageID:        artifact.LineageID,
		LineageSeq:       max(artifact.LineageSeq, 1),
		Slug:             artifact.Slug,
		CreatedAt:        artifact.CreatedAt,
		UpdatedAt:        artifact.UpdatedAt,
	}
}
