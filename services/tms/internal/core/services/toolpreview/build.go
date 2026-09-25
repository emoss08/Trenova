package toolpreview

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
)

// Build is a tool's preview: a sentence for a person and the change to each
// record, in the order the write makes them, cut to the preview's bounds.
func Build(summary string, changes ...*agent.RecordChange) *agent.ToolPreview {
	preview := &agent.ToolPreview{
		Summary: strings.TrimSpace(summary),
		Changes: make([]agent.RecordChange, 0, len(changes)),
	}
	for _, change := range changes {
		if change != nil {
			preview.Changes = append(preview.Changes, *change)
		}
	}

	return preview.Bounded()
}

// Warn adds a warning to a tool's preview.
func Warn(
	preview *agent.ToolPreview,
	code agent.PreviewWarningCode,
	message string,
	args ...string,
) *agent.ToolPreview {
	if preview == nil {
		return nil
	}
	preview.AddWarning(agent.PreviewWarning{
		Code:    code,
		Args:    args,
		Message: strings.TrimSpace(message),
	})

	return preview
}

// Describe is the preview of a tool that cannot say what it would change:
// what would be run, with what. It is the Unavailable fallback, and never
// claims more than the parameters say. Parameters the runtime writes itself
// are left out.
func Describe(toolName string, params map[string]any) *agent.ToolPreview {
	keys := make([]string, 0, len(params))
	for key := range params {
		if !strings.HasPrefix(key, "_") {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	fields := make([]agent.PreviewFieldChange, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, agent.PreviewFieldChange{
			Path:  key,
			Label: assistantartifact.DisplayLabel(key, assistantartifact.DisplayText),
			Type:  assistantartifact.DisplayText,
			After: params[key],
		})
	}

	return (&agent.ToolPreview{
		Summary: fmt.Sprintf(
			"Would run %s with the parameters below. This tool has no preview of its own, "+
				"so what it would change is not shown.",
			toolName,
		),
		Changes: []agent.RecordChange{{
			Operation: agent.PreviewOperationRun,
			Label:     toolName,
			Fields:    fields,
		}},
		Partial: true,
	}).Bounded()
}
