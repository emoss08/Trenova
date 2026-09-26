package toolpreview

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
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
