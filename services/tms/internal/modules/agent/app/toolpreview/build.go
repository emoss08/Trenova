package toolpreview

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/textcase"
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
	preview.AddWarning(&agentpreview.PreviewWarning{
		Code:    code,
		Args:    args,
		Message: strings.TrimSpace(message),
	})

	return preview
}

// Describe is the preview of a tool that cannot say what it would change:
// what would be run, with what. It is the Unavailable fallback, and never
// claims more than the parameters say. Parameters the runtime writes itself
// are left out. A parameter that holds a record's id is the record, named by
// whoever labels the preview's references; an id is never a value a person
// is shown.
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
		fields = append(fields, paramFields(key, params[key])...)
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

var idSuffixes = []string{"IDs", "Ids", "ID", "Id"}

// paramFields is one parameter as a person reads it. An id is the record it
// names, a list of ids one record a line, and an id of no kind of record the
// registry knows is left out: it names nothing anyone could recognise.
func paramFields(key string, value any) []agentpreview.PreviewFieldChange {
	switch typed := value.(type) {
	case string:
		if !isID(typed) {
			break
		}
		if ref := recordRef(typed); ref != nil {
			return []agentpreview.PreviewFieldChange{recordField(key, key, ref)}
		}

		return nil
	case []any:
		ids := make([]string, 0, len(typed))
		for _, element := range typed {
			id, ok := element.(string)
			if !ok || !isID(id) {
				ids = nil

				break
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			break
		}

		fields := make([]agentpreview.PreviewFieldChange, 0, len(ids))
		for idx, id := range ids {
			if ref := recordRef(id); ref != nil {
				fields = append(fields, recordField(fmt.Sprintf("%s[%d]", key, idx), key, ref))
			}
		}

		return fields
	}

	return []agentpreview.PreviewFieldChange{{
		Path:  key,
		Label: assistantartifact.DisplayLabel(key, assistantartifact.DisplayText),
		Type:  assistantartifact.DisplayText,
		After: value,
	}}
}

func isID(value string) bool {
	return pulid.LooksLike(strings.TrimSpace(value))
}

func recordRef(value string) *agentpreview.PreviewRef {
	id := pulid.ID(strings.TrimSpace(value))
	resource, ok := recordkind.ResourceOfIDPrefix(id.Prefix())
	if !ok {
		return nil
	}

	return &agentpreview.PreviewRef{Resource: resource, ID: id}
}

// recordField names the parameter by what it holds: "workerId" is "Worker",
// and a bare "id" the kind of record itself.
func recordField(
	path, key string,
	ref *agentpreview.PreviewRef,
) agentpreview.PreviewFieldChange {
	name := key
	for _, suffix := range idSuffixes {
		if trimmed, ok := strings.CutSuffix(key, suffix); ok {
			name = trimmed

			break
		}
	}
	label := assistantartifact.DisplayLabel(name, assistantartifact.DisplayText)
	if strings.TrimSpace(name) == "" {
		if kind, ok := recordkind.OfResource(ref.Resource); ok {
			label = textcase.CapitalizeFirst(kind.Noun())
		}
	}

	return agentpreview.PreviewFieldChange{
		Path:     path,
		Label:    label,
		Type:     assistantartifact.DisplayText,
		AfterRef: ref,
	}
}
