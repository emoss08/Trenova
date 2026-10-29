package agentreplay

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
)

// ReplayAction is one write the replay would have made: the tool, what it
// was given, and the preview of what it would have changed.
type ReplayAction struct {
	ToolName   string                `json:"toolName"`
	Arguments  map[string]any        `json:"arguments"`
	Rationale  string                `json:"rationale"`
	Tier       agent.AutonomyTier    `json:"tier"`
	Simulation *agent.ToolSimulation `json:"simulation,omitempty"`
}
