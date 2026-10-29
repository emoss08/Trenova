package toolpreview

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
)

// Build is a tool's preview: a sentence for a person and the change to each
// record, in the order the write makes them, cut to the preview's bounds.
func Build(summary string, changes ...*agentpreview.RecordChange) *agentpreview.ToolPreview {
	preview := &agentpreview.ToolPreview{
		Summary: strings.TrimSpace(summary),
		Changes: make([]agentpreview.RecordChange, 0, len(changes)),
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
	preview *agentpreview.ToolPreview,
	code agentpreview.PreviewWarningCode,
	message string,
	args ...string,
) *agentpreview.ToolPreview {
	if preview == nil {
		return nil
	}
	preview.AddWarning(agentpreview.PreviewWarning{
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
func Describe(toolName string, params map[string]any) *agentpreview.ToolPreview {
	keys := make([]string, 0, len(params))
	for key := range params {
		if !strings.HasPrefix(key, "_") {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	fields := make([]agentpreview.PreviewFieldChange, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, agentpreview.PreviewFieldChange{
			Path:  key,
			Label: assistantartifact.DisplayLabel(key, assistantartifact.DisplayText),
			Type:  assistantartifact.DisplayText,
			After: params[key],
		})
	}

	return (&agentpreview.ToolPreview{
		Summary: fmt.Sprintf(
			"Would run %s with the parameters below. This tool has no preview of its own, "+
				"so what it would change is not shown.",
			toolName,
		),
		Changes: []agentpreview.RecordChange{{
			Operation: agentpreview.PreviewOperationRun,
			Label:     toolName,
			Fields:    fields,
		}},
		Partial: true,
	}).Bounded()
}
