package agent

import "strings"

// FieldChange is one thing a write would set: the field, what it holds now
// and what it would hold after.
type FieldChange struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// ToolSimulation is what a write would have done, produced instead of the
// write when its agent is in simulation. The summary is a sentence for a
// person; the changes are for a diff, and a tool that cannot say precisely
// leaves them empty and says so in the summary.
type ToolSimulation struct {
	Summary string        `json:"summary"`
	Changes []FieldChange `json:"changes,omitempty"`
	// Previewed is false when the tool has no preview of its own and the
	// summary is only its name and parameters.
	Previewed bool `json:"previewed"`
}

// Describe writes the simulation for a transcript: the summary, then each
// change on its own line.
func (s *ToolSimulation) Describe() string {
	if s == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(s.Summary))
	for _, change := range s.Changes {
		builder.WriteString("\n- ")
		builder.WriteString(change.Field)
		builder.WriteString(": ")
		if change.From != "" {
			builder.WriteString(change.From)
			builder.WriteString(" → ")
		}
		builder.WriteString(change.To)
	}

	return builder.String()
}
