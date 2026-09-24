package agentdefinition

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/shared/stringutils"
)

func describePageDraft(draft *pagedraft.Draft) string {
	var builder strings.Builder
	switch {
	case draft.ShipmentImport != nil:
		builder.WriteString(
			"- The shipment the person is building on this page, as it stands now. It is not " +
				"saved. Change it only with the draft tools; what they set shows here on the " +
				"next turn:\n",
		)
		builder.WriteString(pageDraftOpenTag)
		writeImportDraft(&builder, draft.ShipmentImport)
	case draft.Formula != nil:
		builder.WriteString(
			"- The formula in the person's editor, which may not be saved yet. Answer about " +
				"this expression, and test it before you say what it charges:\n",
		)
		builder.WriteString(pageDraftOpenTag)
		writeFormulaDraft(&builder, draft.Formula)
	default:
		return ""
	}
	builder.WriteString("\n")
	builder.WriteString(pageDraftCloseTag)

	return builder.String()
}

func draftText(value string) string {
	return stringutils.NeutralizeCloseTag(value, pageDraftCloseTag)
}

func writeImportDraft(builder *strings.Builder, draft *pagedraft.ShipmentImport) {
	builder.WriteString("\nsurface: shipment_import")
	for _, required := range pagedraft.AllRequiredFields() {
		builder.WriteString("\nrequired ")
		builder.WriteString(string(required))
		builder.WriteString(": ")
		if value := draft.Required.Value(required); value != "" {
			builder.WriteString(draftText(value))
		} else {
			builder.WriteString("not set")
		}
	}
	for _, field := range draft.Fields {
		builder.WriteString("\nfield ")
		builder.WriteString(draftText(field.Key))
		if field.Label != "" && field.Label != field.Key {
			builder.WriteString(" (")
			builder.WriteString(draftText(field.Label))
			builder.WriteString(")")
		}
		builder.WriteString(": ")
		if field.Value == "" {
			builder.WriteString("empty")
		} else {
			builder.WriteString(draftText(field.Value))
		}
		builder.WriteString(" [")
		builder.WriteString(string(field.Status))
		builder.WriteString(", confidence ")
		builder.WriteString(strconv.FormatFloat(field.Confidence, 'f', 2, 64))
		builder.WriteString("]")
	}

	attention := 0
	for idx := range draft.Stops {
		stop := &draft.Stops[idx]
		if !stop.HasLocation() || !stop.HasSchedule() {
			attention++
		}
		writeImportStop(builder, idx, stop)
	}
	builder.WriteString("\nstops: ")
	builder.WriteString(strconv.Itoa(len(draft.Stops)))
	builder.WriteString(", needing a location or a date: ")
	builder.WriteString(strconv.Itoa(attention))
}

func writeImportStop(builder *strings.Builder, idx int, stop *pagedraft.ImportStop) {
	builder.WriteString("\nstop ")
	builder.WriteString(strconv.Itoa(idx))
	builder.WriteString(" (")
	builder.WriteString(string(stop.Role))
	builder.WriteString("): ")
	parts := make([]string, 0, 5)
	for _, part := range []string{
		stop.Name, stop.AddressLine1, stop.City, stop.State, stop.PostalCode,
	} {
		if part != "" {
			parts = append(parts, draftText(part))
		}
	}
	if len(parts) == 0 {
		builder.WriteString("no address read")
	} else {
		builder.WriteString(strings.Join(parts, ", "))
	}
	builder.WriteString("; location: ")
	if stop.HasLocation() {
		builder.WriteString(draftText(stop.LocationID))
	} else {
		builder.WriteString("not matched")
	}
	builder.WriteString("; date: ")
	if stop.Date == "" {
		builder.WriteString("none")
	} else {
		builder.WriteString(draftText(stop.Date))
	}
	if !stop.HasSchedule() {
		builder.WriteString(" (not a usable date)")
	}
	if stop.TimeWindow != "" {
		builder.WriteString("; window: ")
		builder.WriteString(draftText(stop.TimeWindow))
	}
}

func writeFormulaDraft(builder *strings.Builder, draft *pagedraft.Formula) {
	builder.WriteString("\nsurface: formula")
	builder.WriteString("\ntemplate: ")
	if draft.TemplateID == "" {
		builder.WriteString("new, not saved yet")
	} else {
		builder.WriteString(draftText(draft.TemplateID))
	}
	builder.WriteString("\nschema: ")
	builder.WriteString(draftText(draft.SchemaID))
	if draft.TemplateType != "" {
		builder.WriteString("\ntype: ")
		builder.WriteString(draftText(draft.TemplateType))
	}
	builder.WriteString("\nexpression: ")
	if strings.TrimSpace(draft.Expression) == "" {
		builder.WriteString("empty")
	} else {
		builder.WriteString("\n")
		builder.WriteString(draftText(draft.Expression))
	}
	for _, variable := range draft.Variables {
		builder.WriteString("\nvariable ")
		builder.WriteString(draftText(variable.Name))
		builder.WriteString(" (")
		builder.WriteString(string(variable.Type))
		builder.WriteString(")")
		if variable.DefaultValue != nil {
			builder.WriteString(" default ")
			builder.WriteString(draftText(formatDraftValue(variable.DefaultValue)))
		}
		if variable.Description != "" {
			builder.WriteString(": ")
			builder.WriteString(draftText(variable.Description))
		}
	}
}

func formatDraftValue(value any) string {
	switch v := value.(type) {
	case string:
		return strconv.Quote(v)
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return ""
	}
}
