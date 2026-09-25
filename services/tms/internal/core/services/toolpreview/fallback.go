package toolpreview

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
)

// Parameters is Describe for a tool whose policy names what it acts on: each
// parameter is stamped with the sensitivity of the field it names on that
// resource, and a Confidential one is left out.
func Parameters(
	toolName string,
	resource permission.Resource,
	params map[string]any,
	opts ...Option,
) *agent.ToolPreview {
	preview := Describe(toolName, params)
	o := newOptions(opts)
	for i := range preview.Changes {
		change := &preview.Changes[i]
		change.Resource = resource
		change.Fields = stamped(o, resource, change.Fields)
	}

	return preview
}

// FromSimulation reads a tool's simulation as a partial preview of the one
// record it acts on: the simulation's words, each change as a value of that
// record. A tool that simulates rather than previews says less than a
// preview does, which Partial records.
func FromSimulation(
	rec Record,
	simulation *agent.ToolSimulation,
	opts ...Option,
) *agent.ToolPreview {
	if simulation == nil {
		return nil
	}

	change := newChange(agent.PreviewOperationUpdate, &rec)
	change.Fields = make([]agent.PreviewFieldChange, 0, len(simulation.Changes))
	for _, simulated := range simulation.Changes {
		path := strings.TrimSpace(simulated.Field)
		if path == "" {
			continue
		}
		field := agent.PreviewFieldChange{
			Path:  path,
			Label: assistantartifact.DisplayLabel(camelPath(path), assistantartifact.DisplayText),
			Type:  assistantartifact.DisplayText,
			After: simulated.To,
		}
		if simulated.From != "" {
			field.Before = simulated.From
		}
		change.Fields = append(change.Fields, field)
	}
	change.Fields = stamped(newOptions(opts), rec.Resource, change.Fields)

	preview := Build(simulation.Summary, change)
	preview.Partial = true

	return preview
}

func stamped(
	o *options,
	resource permission.Resource,
	fields []agent.PreviewFieldChange,
) []agent.PreviewFieldChange {
	kept := fields[:0]
	for i := range fields {
		level := fieldsensitivity.Level(o.registry, resource, topLevel(fields[i].Path))
		if level == permission.SensitivityConfidential {
			continue
		}
		fields[i].Sensitivity = level
		kept = append(kept, fields[i])
	}

	return kept
}
