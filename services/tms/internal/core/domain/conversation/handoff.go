package conversation

import "github.com/emoss08/trenova/shared/pulid"

const (
	// MaxHandoffFacts and MaxHandoffFactRunes bound the pinned facts one
	// hand-off carries, as a conversation bounds what it pins, since the new
	// conversation keeps them pinned; MaxHandoffArtifacts the pinned
	// artifacts.
	MaxHandoffFacts     = MaxPinnedFacts
	MaxHandoffFactRunes = MaxPinnedFactLength
	MaxHandoffArtifacts = 10
	// MaxHandoffSummaryRunes bounds the summary of the conversation.
	MaxHandoffSummaryRunes = 2000
)

// Handoff is a conversation a person took to another agent: where it went,
// and what was carried over so the other agent need not be told again.
type Handoff struct {
	FromThreadID  pulid.ID          `json:"fromThreadId"`
	ToThreadID    pulid.ID          `json:"toThreadId"`
	FromAgentID   pulid.ID          `json:"fromAgentId"`
	FromAgentName string            `json:"fromAgentName"`
	ToAgentID     pulid.ID          `json:"toAgentId"`
	ToAgentName   string            `json:"toAgentName"`
	Summary       string            `json:"summary"`
	Facts         []string          `json:"facts"`
	Artifacts     []HandoffArtifact `json:"artifacts"`
	At            int64             `json:"at"`
}

// HandoffArtifact is a pinned artifact carried over: the copy the new
// conversation holds and the one it was copied from.
type HandoffArtifact struct {
	ID       pulid.ID `json:"id"`
	SourceID pulid.ID `json:"sourceId"`
	Title    string   `json:"title"`
	Kind     string   `json:"kind"`
}
